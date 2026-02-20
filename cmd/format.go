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
		name := p.Name
		if p.IsAd {
			name = "[AD] " + name
		}
		fmt.Fprintf(os.Stdout, " %d. %s\n", i+1, name)

		// Price line with optional original price and discount
		priceLine := "    Price: " + formatPrice(p.Price)
		if p.OriginalPrice > p.Price && p.DiscountPercent > 0 {
			priceLine += fmt.Sprintf("  (was %s, -%d%%)", formatPrice(p.OriginalPrice), p.DiscountPercent)
		}
		priceLine += "  |  Shop: " + p.Shop.Name
		if p.Shop.City != "" {
			priceLine += fmt.Sprintf(" (%s)", p.Shop.City)
		}
		if p.Shop.IsOfficial {
			priceLine += " [Official]"
		}
		fmt.Fprintln(os.Stdout, priceLine)

		if p.PriceRange != "" {
			fmt.Fprintf(os.Stdout, "    Range: %s\n", p.PriceRange)
		}
		if len(p.Labels) > 0 {
			var tags []string
			for _, l := range p.Labels {
				tags = append(tags, "["+l.Title+"]")
			}
			fmt.Fprintf(os.Stdout, "    %s\n", strings.Join(tags, " "))
		}
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
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}
