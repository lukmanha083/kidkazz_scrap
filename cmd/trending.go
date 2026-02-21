package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/lukman83/kidkazz-scrap/internal/platform"
	"github.com/lukman83/kidkazz-scrap/internal/ui"
	"github.com/spf13/cobra"
)

var trendingCmd = &cobra.Command{
	Use:   "trending",
	Short: "Get trending/popular products",
	RunE:  runTrending,
}

func init() {
	trendingCmd.Flags().Int("limit", 10, "Number of products")
	trendingCmd.Flags().String("category", "", "Category filter")
	trendingCmd.Flags().String("format", "json", "Output format: json, table")
	trendingCmd.Flags().Bool("no-ads", false, "Exclude ad/promoted products")
	rootCmd.AddCommand(trendingCmd)
}

func runTrending(cmd *cobra.Command, args []string) error {
	initPlatforms()

	limit, _ := cmd.Flags().GetInt("limit")
	category, _ := cmd.Flags().GetString("category")
	format, _ := cmd.Flags().GetString("format")
	noAds, _ := cmd.Flags().GetBool("no-ads")
	platformName, _ := cmd.Flags().GetString("platform")

	scraper, err := platform.Get(platformName)
	if err != nil {
		return err
	}

	spin := ui.NewSpinner()
	spin.Start("Fetching trending products...")
	ctx := platform.WithProgress(context.Background(), spin.Update)
	products, err := scraper.Trending(ctx, platform.TrendingOpts{
		Category: category,
		Limit:    limit,
	})
	spin.Stop()
	if err != nil {
		return fmt.Errorf("trending failed: %w", err)
	}

	if noAds {
		products = filterAds(products)
	}

	switch format {
	case "table":
		printProductsTable(products)
	default:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(products)
	}

	return nil
}
