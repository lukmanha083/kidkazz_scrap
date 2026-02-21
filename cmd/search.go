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

var searchCmd = &cobra.Command{
	Use:   "search [keyword]",
	Short: "Search products by keyword",
	Args:  cobra.ExactArgs(1),
	RunE:  runSearch,
}

func init() {
	searchCmd.Flags().Int("page", 1, "Page number")
	searchCmd.Flags().Int("limit", 20, "Products per page")
	searchCmd.Flags().String("format", "json", "Output format: json, table")
	searchCmd.Flags().Bool("no-ads", false, "Exclude ad/promoted products")
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	initPlatforms()

	keyword := args[0]
	page, _ := cmd.Flags().GetInt("page")
	limit, _ := cmd.Flags().GetInt("limit")
	format, _ := cmd.Flags().GetString("format")
	noAds, _ := cmd.Flags().GetBool("no-ads")
	platformName, _ := cmd.Flags().GetString("platform")

	scraper, err := platform.Get(platformName)
	if err != nil {
		return err
	}

	spin := ui.NewSpinner()
	spin.Start(fmt.Sprintf("Searching '%s' on %s...", keyword, platformName))
	ctx := platform.WithProgress(context.Background(), spin.Update)
	products, err := scraper.Search(ctx, keyword, platform.SearchOpts{
		Page:  page,
		Limit: limit,
	})
	spin.Stop()
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if noAds {
		before := len(products)
		products = filterAds(products)
		if len(products) < before {
			fmt.Fprintf(os.Stderr, "Note: %d ad(s) filtered, showing %d of %d results\n", before-len(products), len(products), before)
		}
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
