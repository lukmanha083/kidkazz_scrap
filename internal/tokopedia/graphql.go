package tokopedia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lukman83/kidkazz-scrap/internal/httputil"
	"github.com/lukman83/kidkazz-scrap/internal/models"
	"github.com/lukman83/kidkazz-scrap/internal/platform"
)

// GraphQLStrategy calls Tokopedia's internal GraphQL API.
type GraphQLStrategy struct {
	client *http.Client
}

func NewGraphQLStrategy(client *http.Client) *GraphQLStrategy {
	return &GraphQLStrategy{client: client}
}

func (g *GraphQLStrategy) Name() string { return "graphql" }

func (g *GraphQLStrategy) Execute(ctx context.Context, req platform.Request) (*platform.Result, error) {
	switch req.Type {
	case platform.SearchRequest:
		return g.search(ctx, req)
	case platform.TrendingRequest:
		return g.trending(ctx, req)
	default:
		return nil, fmt.Errorf("graphql strategy does not support request type %d", req.Type)
	}
}

func (g *GraphQLStrategy) search(ctx context.Context, req platform.Request) (*platform.Result, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	return g.executeSearch(ctx, req.Keyword, page, limit, SortBestMatch)
}

func (g *GraphQLStrategy) trending(ctx context.Context, req platform.Request) (*platform.Result, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	return g.executeSearch(ctx, req.Keyword, 1, limit, SortBestSeller)
}

func (g *GraphQLStrategy) executeSearch(ctx context.Context, keyword string, page, limit, sort int) (*platform.Result, error) {
	params := BuildSearchParams(keyword, page, limit, sort)

	payload := []map[string]interface{}{
		{
			"operationName": "SearchProductQueryV4",
			"query":         searchProductQuery,
			"variables": map[string]interface{}{
				"params": params,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", graphQLEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, v := range httputil.TokopediaGraphQLHeaders() {
		httpReq.Header[k] = v
	}

	resp, err := httputil.DoWithRetry(g.client, httpReq, 2)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := httputil.ReadBody(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graphql response status %d: %s", resp.StatusCode, string(respBody))
	}

	products, totalData, err := parseSearchResponse(respBody)
	if err != nil {
		return nil, err
	}

	return &platform.Result{
		Products:  products,
		TotalData: totalData,
		Strategy:  g.Name(),
		Raw:       json.RawMessage(respBody),
	}, nil
}

// graphqlResponse represents the GraphQL response structure.
type graphqlResponse []struct {
	Data struct {
		AceSearchProductV4 struct {
			Header struct {
				TotalData    int             `json:"totalData"`
				ResponseCode json.Number     `json:"responseCode"`
			} `json:"header"`
			Data struct {
				Products []graphqlProduct `json:"products"`
			} `json:"data"`
		} `json:"ace_search_product_v4"`
	} `json:"data"`
}

type graphqlProduct struct {
	ID                  json.Number `json:"id"`
	Name                string      `json:"name"`
	Price               string      `json:"price"`
	OriginalPrice       string      `json:"originalPrice"`
	PriceRange          string      `json:"priceRange"`
	DiscountPercentage  int         `json:"discountPercentage"`
	CategoryBreadcrumb  string      `json:"categoryBreadcrumb"`
	ImageURL            string      `json:"imageUrl"`
	URL                 string      `json:"url"`
	CountReview         json.Number `json:"countReview"`
	Wishlist            bool        `json:"wishlist"`
	Ads struct {
		ID string `json:"id"`
	} `json:"ads"`
	LabelGroups []struct {
		Position string `json:"position"`
		Title    string `json:"title"`
		Type     string `json:"type"`
	} `json:"labelGroups"`
	Shop struct {
		ID         json.Number `json:"id"`
		Name       string      `json:"name"`
		URL        string      `json:"url"`
		City       string      `json:"city"`
		IsOfficial bool        `json:"isOfficial"`
	} `json:"shop"`
}

func parseSearchResponse(data []byte) ([]models.Product, int, error) {
	var resp graphqlResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, 0, fmt.Errorf("unmarshal graphql response: %w", err)
	}

	if len(resp) == 0 {
		return nil, 0, fmt.Errorf("empty graphql response")
	}

	ace := resp[0].Data.AceSearchProductV4
	rc, err := ace.Header.ResponseCode.Int64()
	if err != nil {
		return nil, 0, fmt.Errorf("invalid graphql responseCode %q: %w", ace.Header.ResponseCode.String(), err)
	}
	if rc != 0 {
		return nil, 0, fmt.Errorf("graphql error responseCode %d", rc)
	}
	totalData := ace.Header.TotalData
	gqlProducts := ace.Data.Products
	if len(gqlProducts) == 0 {
		return []models.Product{}, totalData, nil
	}

	products := make([]models.Product, 0, len(gqlProducts))
	for _, gp := range gqlProducts {
		isAd := gp.Ads.ID != "" && gp.Ads.ID != "0"

		var labels []models.Label
		for _, lg := range gp.LabelGroups {
			if lg.Title == "" {
				continue
			}
			labels = append(labels, models.Label{
				Title:    lg.Title,
				Position: lg.Position,
				Type:     lg.Type,
			})
		}

		p := models.Product{
			ID:              gp.ID.String(),
			Name:            gp.Name,
			Price:           parsePrice(gp.Price),
			OriginalPrice:   parsePrice(gp.OriginalPrice),
			PriceRange:      gp.PriceRange,
			DiscountPercent: gp.DiscountPercentage,
			Category:        gp.CategoryBreadcrumb,
			ImageURL:        gp.ImageURL,
			URL:             gp.URL,
			IsAd:            isAd,
			Labels:          labels,
			Wishlist:        gp.Wishlist,
			Platform:        "tokopedia",
			ScrapedAt:       time.Now(),
			Strategy:        "graphql",
			Shop: models.Shop{
				ID:         gp.Shop.ID.String(),
				Name:       gp.Shop.Name,
				City:       gp.Shop.City,
				IsOfficial: gp.Shop.IsOfficial,
			},
		}

		if rc, err := gp.CountReview.Int64(); err == nil {
			p.ReviewCount = int(rc)
		}

		products = append(products, p)
	}

	return products, totalData, nil
}

// parsePrice extracts a numeric price from strings like "Rp100.000" or "Rp 1.234.567".
func parsePrice(s string) int64 {
	var digits []byte
	for _, c := range s {
		if c >= '0' && c <= '9' {
			digits = append(digits, byte(c))
		}
	}
	var n int64
	for _, d := range digits {
		n = n*10 + int64(d-'0')
	}
	return n
}
