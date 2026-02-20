package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/lukman83/kidkazz-scrap/internal/models"
)

// printProductsTable prints products in a human-friendly card layout.
func printProductsTable(products []models.Product) {
	for i, p := range products {
		if i > 0 {
			fmt.Fprintln(os.Stdout)
		}
		fmt.Fprintf(os.Stdout, " %d. %s\n", i+1, p.Name)
		fmt.Fprintf(os.Stdout, "    Price: %s  |  Shop: %s", formatPrice(p.Price), p.Shop.Name)
		if p.Shop.City != "" {
			fmt.Fprintf(os.Stdout, " (%s)", p.Shop.City)
		}
		if p.Shop.IsOfficial {
			fmt.Fprint(os.Stdout, " [Official]")
		}
		fmt.Fprintln(os.Stdout)
		if p.Category != "" {
			fmt.Fprintf(os.Stdout, "    Category: %s\n", formatBreadcrumb(p.Category))
		}
		fmt.Fprintf(os.Stdout, "    %s\n", cleanURL(p.URL))
	}
}

// formatPrice formats an int64 price as "Rp 1.234.567".
func formatPrice(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return "Rp " + s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return "Rp " + strings.Join(parts, ".")
}

// cleanURL strips tracking query params (extParam, search_id, src, etc.)
// and returns just the product page URL.
func cleanURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.RawQuery = ""
	return u.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
