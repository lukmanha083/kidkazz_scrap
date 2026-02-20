package models

import "time"

type Label struct {
	Title    string `json:"title"`
	Position string `json:"position,omitempty"`
	Type     string `json:"type,omitempty"`
}

type Product struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Price           int64     `json:"price"`
	OriginalPrice   int64     `json:"original_price,omitempty"`
	PriceRange      string    `json:"price_range,omitempty"`
	DiscountPercent int       `json:"discount_percent,omitempty"`
	ImageURL        string    `json:"image_url,omitempty"`
	URL             string    `json:"url"`
	Category        string    `json:"category,omitempty"`
	Shop            Shop      `json:"shop"`
	ReviewCount     int       `json:"review_count,omitempty"`
	IsAd            bool      `json:"is_ad,omitempty"`
	Labels          []Label   `json:"labels,omitempty"`
	Wishlist        bool      `json:"wishlist,omitempty"`
	Platform        string    `json:"platform"`
	ScrapedAt       time.Time `json:"scraped_at"`
	Strategy        string    `json:"strategy"`
}

type Shop struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	City       string `json:"city,omitempty"`
	IsOfficial bool   `json:"is_official,omitempty"`
}
