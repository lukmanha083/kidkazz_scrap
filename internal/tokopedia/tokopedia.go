package tokopedia

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lukman83/kidkazz-scrap/internal/models"
	"github.com/lukman83/kidkazz-scrap/internal/platform"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// Scraper implements platform.Scraper for Tokopedia.
type Scraper struct {
	fastStrategies []platform.Strategy // Static, GraphQL — raced concurrently
	slowStrategies []platform.Strategy // Headless — tried sequentially as fallback
	rateLimiter    *rate.Limiter
	maxConcurrent  int
}

// NewScraper creates a new Tokopedia scraper with the full strategy chain.
func NewScraper(client *http.Client, rateLimiter *rate.Limiter, maxConcurrent int) *Scraper {
	return &Scraper{
		fastStrategies: []platform.Strategy{
			NewStaticPageStrategy(client),
			NewGraphQLStrategy(client),
		},
		slowStrategies: []platform.Strategy{
			NewHeadlessBrowserStrategy(),
		},
		rateLimiter:   rateLimiter,
		maxConcurrent: maxConcurrent,
	}
}

func (t *Scraper) Search(ctx context.Context, keyword string, opts platform.SearchOpts) ([]models.Product, error) {
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	req := platform.Request{
		Type:    platform.SearchRequest,
		Keyword: keyword,
		Page:    opts.Page,
		Limit:   opts.Limit,
	}

	return t.executeWithFallback(ctx, req)
}

func (t *Scraper) Trending(ctx context.Context, opts platform.TrendingOpts) ([]models.Product, error) {
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	keyword := "trending"
	if opts.Category != "" {
		keyword = opts.Category
	}

	req := platform.Request{
		Type:    platform.TrendingRequest,
		Keyword: keyword,
		Limit:   opts.Limit,
		Page:    1,
	}

	return t.executeWithFallback(ctx, req)
}

func (t *Scraper) ProductDetail(ctx context.Context, url string) (*models.Product, error) {
	req := platform.Request{
		Type: platform.ProductDetailRequest,
		URL:  url,
	}

	products, err := t.executeWithFallback(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return nil, fmt.Errorf("no product detail found for: %s", url)
	}
	return &products[0], nil
}

// SearchAll fetches multiple pages concurrently with rate limiting.
func (t *Scraper) SearchAll(ctx context.Context, keyword string, pages, perPage int) ([]models.Product, error) {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(t.maxConcurrent)

	results := make([][]models.Product, pages)
	for i := 0; i < pages; i++ {
		i := i
		g.Go(func() error {
			if err := t.rateLimiter.Wait(ctx); err != nil {
				return err
			}
			products, err := t.Search(ctx, keyword, platform.SearchOpts{Page: i + 1, Limit: perPage})
			if err != nil {
				return err
			}
			results[i] = products
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return flatten(results), nil
}

// executeWithFallback races fast strategies concurrently, then falls back to slow strategies.
func (t *Scraper) executeWithFallback(ctx context.Context, req platform.Request) ([]models.Product, error) {
	var strategyErrors []string

	// Phase 1: Race fast strategies concurrently
	raceCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	type strategyResult struct {
		products []models.Product
		strategy string
		err      error
	}
	resultCh := make(chan strategyResult, len(t.fastStrategies))

	for _, s := range t.fastStrategies {
		go func(s platform.Strategy) {
			if t.rateLimiter != nil {
				if err := t.rateLimiter.Wait(raceCtx); err != nil {
					resultCh <- strategyResult{strategy: s.Name(), err: err}
					return
				}
			}
			r, err := s.Execute(raceCtx, req)
			if err != nil {
				resultCh <- strategyResult{strategy: s.Name(), err: err}
				return
			}
			if r == nil || len(r.Products) == 0 {
				resultCh <- strategyResult{strategy: s.Name(), err: fmt.Errorf("no products returned")}
				return
			}
			resultCh <- strategyResult{products: r.Products, strategy: s.Name()}
		}(s)
	}

	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	fastRemaining := len(t.fastStrategies)

fastLoop:
	for fastRemaining > 0 {
		select {
		case r := <-resultCh:
			fastRemaining--
			if r.err == nil && len(r.products) > 0 {
				cancel()
				platform.ReportProgress(ctx, fmt.Sprintf("Found %d products via %s", len(r.products), r.strategy))
				return r.products, nil
			}
			if r.err != nil {
				strategyErrors = append(strategyErrors, fmt.Sprintf("%s: %v", r.strategy, r.err))
				platform.ReportProgress(ctx, fmt.Sprintf("Strategy %s failed, trying next...", r.strategy))
			}
		case <-timer.C:
			cancel()
			strategyErrors = append(strategyErrors, fmt.Sprintf("fast strategies: timed out after 10s (%d still pending)", fastRemaining))
			platform.ReportProgress(ctx, "Fast strategies timed out, trying headless browser...")
			break fastLoop
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Phase 2: Fall back to slow strategies sequentially
	for _, s := range t.slowStrategies {
		platform.ReportProgress(ctx, fmt.Sprintf("Trying %s strategy...", s.Name()))
		result, err := s.Execute(ctx, req)
		if err == nil && result != nil && len(result.Products) > 0 {
			platform.ReportProgress(ctx, fmt.Sprintf("Found %d products via %s", len(result.Products), s.Name()))
			return result.Products, nil
		}
		if err != nil {
			strategyErrors = append(strategyErrors, fmt.Sprintf("%s: %v", s.Name(), err))
			platform.ReportProgress(ctx, fmt.Sprintf("Strategy %s failed, trying next...", s.Name()))
		}
	}

	return nil, fmt.Errorf("all strategies exhausted for %q:\n  %s", req.Keyword, strings.Join(strategyErrors, "\n  "))
}

func flatten(results [][]models.Product) []models.Product {
	var out []models.Product
	for _, r := range results {
		out = append(out, r...)
	}
	return out
}
