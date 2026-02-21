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
// cleanURL removes query parameters from the provided URL and returns the resulting URL string.
// If the input cannot be parsed as a URL, it returns the original rawURL unchanged.
func cleanURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.RawQuery = ""
	return u.String()
}

// filterAds returns a new slice containing only products that are not advertisements.
// The original slice is not modified and the relative order of kept products is preserved.
func filterAds(products []models.Product) []models.Product {
	filtered := make([]models.Product, 0, len(products))
	for _, p := range products {
		if !p.IsAd {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// truncate truncates s to at most max runes, adding "..." when max is greater than 3.
// If max is less than or equal to 0 an empty string is returned.
// If s contains max or fewer runes it is returned unchanged.
// If max is less than or equal to 3 the function returns the first max runes with no ellipsis.
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}