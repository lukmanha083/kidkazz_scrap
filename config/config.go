package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	// General
	DefaultPlatform string
	RespectRobots   bool
	DelayProfile    string // "cautious", "normal", "aggressive"

	// Rate limiting
	RatePerSecond float64
	RateBurst     int
	MaxConcurrent int

	// HTTP server
	HTTPPort string
	APIKey   string

	// Proxy
	ProxyMode       string // "decodo", "wireguard", "custom", "direct"
	DecodoUsername   string
	DecodoPassword   string
	DecodoCountry   string
	DecodoCity      string
	WireGuardConfig string
	ProxyFile       string // file with proxy list for custom mode
}

// DefaultConfig returns configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		DefaultPlatform: "tokopedia",
		RespectRobots:   true,
		DelayProfile:    "normal",
		RatePerSecond:   2.0,
		RateBurst:       3,
		MaxConcurrent:   5,
		ProxyMode:       "direct",
		DecodoCountry:   "id",
		HTTPPort:        "8080",
	}
}

// LoadFromEnv loads .env file (if present) then overrides config from environment variables.
func (c *Config) LoadFromEnv() {
	// Auto-load .env file; silently ignored if missing
	_ = godotenv.Load()

	if v := os.Getenv("KIDKAZZ_PLATFORM"); v != "" {
		c.DefaultPlatform = v
	}
	if v := os.Getenv("KIDKAZZ_DELAY_PROFILE"); v != "" {
		c.DelayProfile = v
	}
	if v := os.Getenv("KIDKAZZ_RATE_PER_SECOND"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.RatePerSecond = f
		}
	}
	if v := os.Getenv("KIDKAZZ_RATE_BURST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.RateBurst = n
		}
	}
	if v := os.Getenv("KIDKAZZ_MAX_CONCURRENT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxConcurrent = n
		}
	}
	if v := os.Getenv("KIDKAZZ_PROXY_MODE"); v != "" {
		c.ProxyMode = v
	}
	if v := os.Getenv("DECODO_USERNAME"); v != "" {
		c.DecodoUsername = v
	}
	if v := os.Getenv("DECODO_PASSWORD"); v != "" {
		c.DecodoPassword = v
	}
	if v := os.Getenv("DECODO_COUNTRY"); v != "" {
		c.DecodoCountry = v
	}
	if v := os.Getenv("DECODO_CITY"); v != "" {
		c.DecodoCity = v
	}
	if v := os.Getenv("KIDKAZZ_WG_CONFIG"); v != "" {
		c.WireGuardConfig = v
	}
	if v := os.Getenv("KIDKAZZ_PROXIES"); v != "" {
		c.ProxyFile = v
	}
	if v := os.Getenv("KIDKAZZ_RESPECT_ROBOTS"); v == "false" {
		c.RespectRobots = false
	}
	if v := os.Getenv("PORT"); v != "" {
		c.HTTPPort = v
	}
	if v := os.Getenv("KIDKAZZ_API_KEY"); v != "" {
		c.APIKey = v
	}
}
