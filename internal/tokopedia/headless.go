package tokopedia

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/lukman83/kidkazz-scrap/internal/models"
	"github.com/lukman83/kidkazz-scrap/internal/platform"
)

// HeadlessBrowserStrategy uses rod to render pages with JS execution.
type HeadlessBrowserStrategy struct {
	launcherURL string // optional remote launcher URL
}

func NewHeadlessBrowserStrategy() *HeadlessBrowserStrategy {
	return &HeadlessBrowserStrategy{}
}

func (h *HeadlessBrowserStrategy) Name() string { return "headless" }

func (h *HeadlessBrowserStrategy) Execute(ctx context.Context, req platform.Request) (*platform.Result, error) {
	switch req.Type {
	case platform.SearchRequest, platform.TrendingRequest:
		return h.search(ctx, req)
	case platform.ProductDetailRequest:
		return h.productDetail(ctx, req)
	default:
		return nil, fmt.Errorf("headless strategy does not support request type %d", req.Type)
	}
}

func (h *HeadlessBrowserStrategy) search(ctx context.Context, req platform.Request) (*platform.Result, error) {
	searchURL := fmt.Sprintf("https://www.tokopedia.com/search?q=%s&page=%d", url.QueryEscape(req.Keyword), req.Page)

	page, cleanup, err := h.openPage(ctx, searchURL)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// Wait for page to stabilize
	timedPage := page.Timeout(15 * time.Second)
	if err := timedPage.WaitStable(time.Second); err == nil {
		_ = timedPage.WaitDOMStable(2*time.Second, 0.1)
	}

	htmlContent, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("get page HTML: %w", err)
	}

	// Try to extract JSON-LD from the rendered page
	products, err := extractJSONLD(htmlContent)
	if err == nil && len(products) > 0 {
		for i := range products {
			products[i].Strategy = "headless"
		}
		return &platform.Result{
			Products: products,
			Strategy: h.Name(),
		}, nil
	}

	// Fallback: try to extract from page's JavaScript data
	products, err = h.extractFromDOM(page)
	if err != nil {
		return nil, fmt.Errorf("headless extraction failed: %w", err)
	}

	return &platform.Result{
		Products: products,
		Strategy: h.Name(),
	}, nil
}

func (h *HeadlessBrowserStrategy) productDetail(ctx context.Context, req platform.Request) (*platform.Result, error) {
	page, cleanup, err := h.openPage(ctx, req.URL)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	timedPage := page.Timeout(15 * time.Second)
	if err := timedPage.WaitStable(time.Second); err == nil {
		_ = timedPage.WaitDOMStable(2*time.Second, 0.1)
	}

	htmlContent, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("get page HTML: %w", err)
	}

	products, err := extractJSONLD(htmlContent)
	if err != nil || len(products) == 0 {
		return nil, fmt.Errorf("no product data extracted from headless page")
	}

	for i := range products {
		products[i].Strategy = "headless"
	}

	return &platform.Result{
		Products: products,
		Strategy: h.Name(),
	}, nil
}

func (h *HeadlessBrowserStrategy) openPage(ctx context.Context, pageURL string) (*rod.Page, func(), error) {
	var l *launcher.Launcher
	if h.launcherURL != "" {
		l = launcher.MustNewManaged(h.launcherURL)
	} else {
		l = launcher.New().Headless(true).Logger(io.Discard)
	}
	if bin := os.Getenv("ROD_BROWSER_BIN"); bin != "" {
		l = l.Bin(bin)
	}
	controlURL, err := l.Launch()
	if err != nil {
		return nil, nil, fmt.Errorf("launch browser: %w", err)
	}

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		l.Kill()
		return nil, nil, fmt.Errorf("connect browser: %w", err)
	}

	// Set viewport to desktop size
	page, err := browser.Page(proto.TargetCreateTarget{URL: pageURL})
	if err != nil {
		browser.Close()
		return nil, nil, fmt.Errorf("open page: %w", err)
	}

	err = page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  1920,
		Height: 1080,
	})
	if err != nil {
		browser.Close()
		return nil, nil, fmt.Errorf("set viewport: %w", err)
	}

	cleanup := func() {
		page.Close()
		browser.Close()
		l.Cleanup()
	}

	return page, cleanup, nil
}

func (h *HeadlessBrowserStrategy) extractFromDOM(page *rod.Page) ([]models.Product, error) {
	// Try to evaluate JavaScript to extract product data from the page's state
	result, err := page.Eval(`() => {
		// Try to find product data in window.__data or similar
		const scripts = document.querySelectorAll('script');
		for (const script of scripts) {
			const text = script.textContent;
			if (text.includes('"products"') || text.includes('"product_name"')) {
				return text;
			}
		}
		return '';
	}`)
	if err != nil || result.Value.Str() == "" {
		return nil, fmt.Errorf("no embedded product data found")
	}

	// Attempt to find JSON objects in the script content
	content := result.Value.Str()
	products, err := tryExtractProductsFromScript(content)
	if err != nil {
		return nil, err
	}

	return products, nil
}

func tryExtractProductsFromScript(content string) ([]models.Product, error) {
	// Look for product arrays in the script content
	start := strings.Index(content, `"products":[`)
	if start == -1 {
		return nil, fmt.Errorf("no products array found")
	}

	// Find the array bounds
	start = strings.Index(content[start:], "[") + start
	depth := 0
	end := start
	for i := start; i < len(content); i++ {
		if content[i] == '[' {
			depth++
		} else if content[i] == ']' {
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
	}

	if end <= start {
		return nil, fmt.Errorf("malformed products array")
	}

	var rawProducts []json.RawMessage
	if err := json.Unmarshal([]byte(content[start:end]), &rawProducts); err != nil {
		return nil, fmt.Errorf("parse products array: %w", err)
	}

	var products []models.Product
	for _, raw := range rawProducts {
		var gp graphqlProduct
		if err := json.Unmarshal(raw, &gp); err != nil {
			continue
		}
		if gp.Name == "" {
			continue
		}
		p := models.Product{
			ID:       gp.ID.String(),
			Name:     gp.Name,
			Price:    parsePrice(gp.Price),
			ImageURL: gp.ImageURL,
			URL:      gp.URL,
			Platform: "tokopedia",
			ScrapedAt: time.Now(),
			Strategy: "headless",
			Shop: models.Shop{
				ID:   gp.Shop.ID.String(),
				Name: gp.Shop.Name,
				City: gp.Shop.City,
			},
		}
		products = append(products, p)
	}

	if len(products) == 0 {
		return nil, fmt.Errorf("no valid products parsed from DOM")
	}

	return products, nil
}
