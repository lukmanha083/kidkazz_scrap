package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/lukman83/kidkazz-scrap/internal/platform"
	"github.com/lukman83/kidkazz-scrap/internal/ui"
	"github.com/spf13/cobra"
)

var categoriesCmd = &cobra.Command{
	Use:   "categories [keyword]",
	Short: "Show popular categories for a keyword",
	Args:  cobra.ExactArgs(1),
	RunE:  runCategories,
}

func init() {
	categoriesCmd.Flags().Int("limit", 60, "Number of products to sample")
	rootCmd.AddCommand(categoriesCmd)
}

func runCategories(cmd *cobra.Command, args []string) error {
	initPlatforms()

	keyword := args[0]
	limit, _ := cmd.Flags().GetInt("limit")
	platformName, _ := cmd.Flags().GetString("platform")

	scraper, err := platform.Get(platformName)
	if err != nil {
		return err
	}

	spin := ui.NewSpinner()
	spin.Start(fmt.Sprintf("Discovering best-seller categories for '%s'...", keyword))
	ctx := platform.WithProgress(context.Background(), spin.Update)
	products, err := scraper.Trending(ctx, platform.TrendingOpts{
		Category: keyword,
		Limit:    limit,
	})
	spin.Stop()
	if err != nil {
		return fmt.Errorf("trending failed: %w", err)
	}

	// Aggregate categories
	counts := make(map[string]int)
	for _, p := range products {
		if p.Category != "" {
			counts[p.Category]++
		}
	}

	if len(counts) == 0 {
		fmt.Println("No categories found.")
		return nil
	}

	// Sort by count descending
	type entry struct {
		category string
		count    int
	}
	entries := make([]entry, 0, len(counts))
	for cat, n := range counts {
		entries = append(entries, entry{cat, n})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	fmt.Printf("Popular categories for \"%s\" (%d products sampled):\n\n", keyword, len(products))
	for i, e := range entries {
		// Format breadcrumb: "mainan-hobi/figure/action-figure" → "Mainan Hobi > Figure > Action Figure"
		fmt.Printf(" %2d. %-50s  (%d products)\n", i+1, formatBreadcrumb(e.category), e.count)
	}

	return nil
}

// formatBreadcrumb converts "mainan-hobi/figure/action-figure" to "Mainan Hobi > Figure > Action Figure".
func formatBreadcrumb(s string) string {
	parts := strings.Split(s, "/")
	for i, p := range parts {
		words := strings.Split(p, "-")
		for j, w := range words {
			if len(w) > 0 {
				words[j] = strings.ToUpper(w[:1]) + w[1:]
			}
		}
		parts[i] = strings.Join(words, " ")
	}
	return strings.Join(parts, " > ")
}
