package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/lukman83/kidkazz-scrap/internal/platform"
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
	rootCmd.AddCommand(trendingCmd)
}

func runTrending(cmd *cobra.Command, args []string) error {
	initPlatforms()

	limit, _ := cmd.Flags().GetInt("limit")
	category, _ := cmd.Flags().GetString("category")
	format, _ := cmd.Flags().GetString("format")
	platformName, _ := cmd.Flags().GetString("platform")

	scraper, err := platform.Get(platformName)
	if err != nil {
		return err
	}

	ctx := context.Background()
	products, err := scraper.Trending(ctx, platform.TrendingOpts{
		Category: category,
		Limit:    limit,
	})
	if err != nil {
		return fmt.Errorf("trending failed: %w", err)
	}

	switch format {
	case "table":
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tPRICE\tSHOP\tRATING\tSTRATEGY")
		for _, p := range products {
			fmt.Fprintf(w, "%s\tRp%d\t%s\t%.1f\t%s\n",
				truncate(p.Name, 50), p.Price, p.Shop.Name, p.Rating, p.Strategy)
		}
		w.Flush()
	default:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(products)
	}

	return nil
}
