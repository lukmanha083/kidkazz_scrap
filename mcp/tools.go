package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lukman83/kidkazz-scrap/internal/platform"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerTools(s *server.MCPServer) {
	// search_products
	searchTool := mcp.NewTool("search_products",
		mcp.WithDescription("Search products by keyword on a marketplace platform"),
		mcp.WithString("keyword",
			mcp.Required(),
			mcp.Description("Search keyword"),
		),
		mcp.WithString("platform",
			mcp.Description("Target platform (default: tokopedia)"),
		),
		mcp.WithNumber("page",
			mcp.Description("Page number (default: 1)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Products per page (default: 20)"),
		),
	)
	s.AddTool(searchTool, handleSearchProducts)

	// get_trending
	trendingTool := mcp.NewTool("get_trending",
		mcp.WithDescription("Get trending/popular products on a marketplace platform"),
		mcp.WithString("platform",
			mcp.Description("Target platform (default: tokopedia)"),
		),
		mcp.WithString("category",
			mcp.Description("Category filter"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Number of products (default: 10)"),
		),
	)
	s.AddTool(trendingTool, handleGetTrending)

	// product_detail
	detailTool := mcp.NewTool("product_detail",
		mcp.WithDescription("Get full product details by URL"),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("Product page URL"),
		),
	)
	s.AddTool(detailTool, handleProductDetail)
}

func handleSearchProducts(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyword := request.GetString("keyword", "")
	if keyword == "" {
		return mcp.NewToolResultError("keyword is required"), nil
	}

	platformName := request.GetString("platform", "tokopedia")
	page := request.GetInt("page", 1)
	limit := request.GetInt("limit", 20)

	scraper, err := platform.Get(platformName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("platform error: %v", err)), nil
	}

	products, err := scraper.Search(ctx, keyword, platform.SearchOpts{
		Page:  page,
		Limit: limit,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search error: %v", err)), nil
	}

	data, _ := json.MarshalIndent(products, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetTrending(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	platformName := request.GetString("platform", "tokopedia")
	category := request.GetString("category", "")
	limit := request.GetInt("limit", 10)

	scraper, err := platform.Get(platformName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("platform error: %v", err)), nil
	}

	products, err := scraper.Trending(ctx, platform.TrendingOpts{
		Category: category,
		Limit:    limit,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("trending error: %v", err)), nil
	}

	data, _ := json.MarshalIndent(products, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleProductDetail(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url := request.GetString("url", "")
	if url == "" {
		return mcp.NewToolResultError("url is required"), nil
	}

	platformName := "tokopedia"

	scraper, err := platform.Get(platformName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("platform error: %v", err)), nil
	}

	product, err := scraper.ProductDetail(ctx, url)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("detail error: %v", err)), nil
	}

	data, _ := json.MarshalIndent(product, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
