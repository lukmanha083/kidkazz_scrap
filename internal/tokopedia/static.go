package tokopedia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lukman83/kidkazz-scrap/internal/httputil"
	"github.com/lukman83/kidkazz-scrap/internal/models"
	"github.com/lukman83/kidkazz-scrap/internal/platform"
	"golang.org/x/net/html"
)

// StaticPageStrategy fetches raw HTML and extracts JSON-LD structured data.
type StaticPageStrategy struct {
	client *http.Client
}

func NewStaticPageStrategy(client *http.Client) *StaticPageStrategy {
	return &StaticPageStrategy{client: client}
}

func (s *StaticPageStrategy) Name() string { return "static" }

func (s *StaticPageStrategy) Execute(ctx context.Context, req platform.Request) (*platform.Result, error) {
	switch req.Type {
	case platform.SearchRequest:
		return s.search(ctx, req)
	case platform.ProductDetailRequest:
		return s.productDetail(ctx, req)
	default:
		return nil, fmt.Errorf("static strategy does not support request type %d", req.Type)
	}
}

func (s *StaticPageStrategy) search(ctx context.Context, req platform.Request) (*platform.Result, error) {
	searchURL := fmt.Sprintf("https://www.tokopedia.com/search?q=%s&page=%d", url.QueryEscape(req.Keyword), req.Page)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range httputil.BrowserHeaders() {
		httpReq.Header[k] = v
	}

	resp, err := httputil.DoWithRetry(s.client, httpReq, 2)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := httputil.ReadBody(resp)
	if err != nil {
		return nil, err
	}

	products, err := extractJSONLD(string(body))
	if err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return nil, fmt.Errorf("no JSON-LD product data found")
	}

	return &platform.Result{
		Products: products,
		Strategy: s.Name(),
	}, nil
}

func (s *StaticPageStrategy) productDetail(ctx context.Context, req platform.Request) (*platform.Result, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", req.URL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range httputil.BrowserHeaders() {
		httpReq.Header[k] = v
	}

	resp, err := httputil.DoWithRetry(s.client, httpReq, 2)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := httputil.ReadBody(resp)
	if err != nil {
		return nil, err
	}

	products, err := extractJSONLD(string(body))
	if err != nil {
		return nil, fmt.Errorf("extract JSON-LD: %w", err)
	}
	if len(products) == 0 {
		return nil, fmt.Errorf("no JSON-LD product data found in page")
	}

	return &platform.Result{
		Products: products,
		Strategy: s.Name(),
	}, nil
}

// extractJSONLD parses HTML and extracts Product data from JSON-LD script tags.
func extractJSONLD(htmlContent string) ([]models.Product, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	var products []models.Product
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			for _, attr := range n.Attr {
				if attr.Key == "type" && attr.Val == "application/ld+json" {
					if n.FirstChild != nil {
						if p, err := parseJSONLDProduct(n.FirstChild.Data); err == nil {
							products = append(products, p...)
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return products, nil
}

// jsonLDItem represents a generic JSON-LD object.
type jsonLDItem struct {
	Type        string      `json:"@type"`
	Name        string      `json:"name"`
	URL         string      `json:"url"`
	Image       interface{} `json:"image"`
	Description string      `json:"description"`
	Offers      *jsonLDOffer `json:"offers"`
	AggregateRating *jsonLDAggregateRating `json:"aggregateRating"`
	ItemListElement []jsonLDListElement     `json:"itemListElement"`
}

type jsonLDOffer struct {
	Type         string `json:"@type"`
	Price        json.Number `json:"price"`
	PriceCurrency string `json:"priceCurrency"`
	Seller       *jsonLDSeller `json:"seller"`
}

type jsonLDSeller struct {
	Type string `json:"@type"`
	Name string `json:"name"`
}

type jsonLDAggregateRating struct {
	RatingValue json.Number `json:"ratingValue"`
	ReviewCount json.Number `json:"reviewCount"`
}

type jsonLDListElement struct {
	Type string      `json:"@type"`
	Item *jsonLDItem `json:"item"`
}

func parseJSONLDProduct(data string) ([]models.Product, error) {
	data = strings.TrimSpace(data)

	// Try as single object
	var item jsonLDItem
	if err := json.Unmarshal([]byte(data), &item); err == nil {
		if p, ok := jsonLDToProduct(&item); ok {
			return []models.Product{p}, nil
		}
		// Check for ItemList
		if item.Type == "ItemList" && len(item.ItemListElement) > 0 {
			var products []models.Product
			for _, elem := range item.ItemListElement {
				if elem.Item != nil {
					if p, ok := jsonLDToProduct(elem.Item); ok {
						products = append(products, p)
					}
				}
			}
			return products, nil
		}
	}

	// Try as array
	var items []jsonLDItem
	if err := json.Unmarshal([]byte(data), &items); err == nil {
		var products []models.Product
		for _, it := range items {
			if p, ok := jsonLDToProduct(&it); ok {
				products = append(products, p)
			}
		}
		return products, nil
	}

	return nil, fmt.Errorf("no product data in JSON-LD")
}

func jsonLDToProduct(item *jsonLDItem) (models.Product, bool) {
	if item.Type != "Product" {
		return models.Product{}, false
	}

	p := models.Product{
		Name:      item.Name,
		URL:       item.URL,
		Platform:  "tokopedia",
		ScrapedAt: time.Now(),
		Strategy:  "static",
	}

	if item.Offers != nil {
		if price, err := item.Offers.Price.Int64(); err == nil {
			p.Price = price
		}
		if item.Offers.Seller != nil {
			p.Shop.Name = item.Offers.Seller.Name
		}
	}

	if item.AggregateRating != nil {
		if r, err := item.AggregateRating.RatingValue.Float64(); err == nil {
			p.Rating = r
		}
		if rc, err := item.AggregateRating.ReviewCount.Int64(); err == nil {
			p.ReviewCount = int(rc)
		}
	}

	// Extract image URL
	switch img := item.Image.(type) {
	case string:
		p.ImageURL = img
	case []interface{}:
		if len(img) > 0 {
			if s, ok := img[0].(string); ok {
				p.ImageURL = s
			}
		}
	}

	return p, true
}
