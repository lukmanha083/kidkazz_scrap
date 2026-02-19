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

	params := BuildSearchParams(req.Keyword, page, limit)

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

	products, err := parseSearchResponse(respBody)
	if err != nil {
		return nil, err
	}

	return &platform.Result{
		Products: products,
		Strategy: g.Name(),
		Raw:      json.RawMessage(respBody),
	}, nil
}

func (g *GraphQLStrategy) trending(ctx context.Context, req platform.Request) (*platform.Result, error) {
	// Trending uses the same search endpoint with a popularity sort
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	return g.search(ctx, platform.Request{
		Type:    platform.SearchRequest,
		Keyword: req.Keyword,
		Page:    1,
		Limit:   limit,
	})
}

// graphqlResponse represents the GraphQL response structure.
type graphqlResponse []struct {
	Data struct {
		AceSearchProductV4 struct {
			Header struct {
				TotalData    int    `json:"totalData"`
				ResponseCode string `json:"responseCode"`
			} `json:"header"`
			Data struct {
				Products []graphqlProduct `json:"products"`
			} `json:"data"`
		} `json:"ace_search_product_v4"`
	} `json:"data"`
}

type graphqlProduct struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price struct {
		Text   string `json:"text"`
		Number int64  `json:"number"`
	} `json:"price"`
	OriginalPrice      string  `json:"originalPrice"`
	DiscountPercentage int     `json:"discountPercentage"`
	ImageURL           map[string]string `json:"imageUrl"`
	URL                string  `json:"url"`
	Shop               struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		City       string `json:"city"`
		IsOfficial bool   `json:"isOfficial"`
	} `json:"shop"`
	RatingAverage string `json:"ratingAverage"`
	CountReview   string `json:"countReview"`
}

func parseSearchResponse(data []byte) ([]models.Product, error) {
	var resp graphqlResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal graphql response: %w", err)
	}

	if len(resp) == 0 {
		return nil, fmt.Errorf("empty graphql response")
	}

	gqlProducts := resp[0].Data.AceSearchProductV4.Data.Products
	if len(gqlProducts) == 0 {
		return nil, fmt.Errorf("no products in graphql response")
	}

	products := make([]models.Product, 0, len(gqlProducts))
	for _, gp := range gqlProducts {
		p := models.Product{
			ID:              gp.ID,
			Name:            gp.Name,
			Price:           gp.Price.Number,
			DiscountPercent: gp.DiscountPercentage,
			URL:             gp.URL,
			Platform:        "tokopedia",
			ScrapedAt:       time.Now(),
			Strategy:        "graphql",
			Shop: models.Shop{
				ID:         gp.Shop.ID,
				Name:       gp.Shop.Name,
				City:       gp.Shop.City,
				IsOfficial: gp.Shop.IsOfficial,
			},
		}

		// Image URL from the map
		if imgURL, ok := gp.ImageURL["300"]; ok {
			p.ImageURL = imgURL
		}

		// Parse rating
		if gp.RatingAverage != "" {
			var rating float64
			fmt.Sscanf(gp.RatingAverage, "%f", &rating)
			p.Rating = rating
		}

		// Parse review count
		if gp.CountReview != "" {
			var count int
			fmt.Sscanf(gp.CountReview, "%d", &count)
			p.ReviewCount = count
		}

		products = append(products, p)
	}

	return products, nil
}
