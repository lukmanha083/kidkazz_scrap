package tokopedia

import (
	"context"
	"fmt"
	"log"
	"net/http"
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
	// Phase 1: Race fast strategies concurrently
	raceCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	type strategyResult struct {
		products []models.Product
		strategy string
	}
	resultCh := make(chan strategyResult, len(t.fastStrategies))

	for _, s := range t.fastStrategies {
		go func(s platform.Strategy) {
			if t.rateLimiter != nil {
				if err := t.rateLimiter.Wait(raceCtx); err != nil {
					return
				}
			}
			r, err := s.Execute(raceCtx, req)
			if err == nil && r != nil && len(r.Products) > 0 {
				resultCh <- strategyResult{products: r.Products, strategy: s.Name()}
			}
		}(s)
	}

	select {
	case r := <-resultCh:
		cancel()
		log.Printf("strategy %s succeeded with %d products", r.strategy, len(r.products))
		return r.products, nil
	case <-time.After(10 * time.Second):
		cancel()
		log.Printf("fast strategies timed out, falling back to slow strategies")
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Phase 2: Fall back to slow strategies sequentially
	for _, s := range t.slowStrategies {
		log.Printf("trying slow strategy: %s", s.Name())
		result, err := s.Execute(ctx, req)
		if err == nil && result != nil && len(result.Products) > 0 {
			log.Printf("strategy %s succeeded with %d products", s.Name(), len(result.Products))
			return result.Products, nil
		}
		if err != nil {
			log.Printf("strategy %s failed: %v", s.Name(), err)
		}
	}

	return nil, fmt.Errorf("all strategies exhausted for request: %+v", req)
}

func flatten(results [][]models.Product) []models.Product {
	var out []models.Product
	for _, r := range results {
		out = append(out, r...)
	}
	return out
}
