package platform

import (
	"context"
	"encoding/json"

	"github.com/lukman83/kidkazz-scrap/internal/models"
)

type RequestType int

const (
	SearchRequest RequestType = iota
	TrendingRequest
	ProductDetailRequest
)

type Request struct {
	Type    RequestType
	Keyword string
	URL     string
	Page    int
	Limit   int
}

type Result struct {
	Products  []models.Product
	TotalData int
	Strategy  string
	Raw       json.RawMessage
}

type SearchOpts struct {
	Page  int
	Limit int
}

type TrendingOpts struct {
	Category string
	Limit    int
}

type Strategy interface {
	Name() string
	Execute(ctx context.Context, req Request) (*Result, error)
}

type Scraper interface {
	Search(ctx context.Context, keyword string, opts SearchOpts) ([]models.Product, error)
	Trending(ctx context.Context, opts TrendingOpts) ([]models.Product, error)
	ProductDetail(ctx context.Context, url string) (*models.Product, error)
}
