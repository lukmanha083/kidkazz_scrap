package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/lukman83/kidkazz-scrap/config"
	"github.com/lukman83/kidkazz-scrap/internal/platform"
	"github.com/lukman83/kidkazz-scrap/internal/stealth"
	"github.com/lukman83/kidkazz-scrap/internal/tokopedia"
	"github.com/spf13/cobra"
	"golang.org/x/time/rate"
)

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "kidkazz",
	Short: "KidKazz Scrap - Marketplace scraping CLI & MCP server",
	Long:  "A Go-based CLI tool and MCP server for scraping Indonesian marketplace data.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("platform", "tokopedia", "Target marketplace platform")
	rootCmd.PersistentFlags().String("delay-profile", "normal", "Delay profile: cautious, normal, aggressive")
	rootCmd.PersistentFlags().Bool("respect-robots", true, "Respect robots.txt rules")
	rootCmd.PersistentFlags().String("proxy-mode", "direct", "Proxy mode: decodo, wireguard, custom, direct")
	rootCmd.PersistentFlags().String("wireguard-config", "", "Path to WireGuard config file")
	rootCmd.PersistentFlags().String("proxy-file", "", "Path to proxy list file")
}

func initConfig() {
	cfg = config.DefaultConfig()
	cfg.LoadFromEnv()

	// Override from flags
	if v, _ := rootCmd.PersistentFlags().GetString("platform"); v != "" {
		cfg.DefaultPlatform = v
	}
	if v, _ := rootCmd.PersistentFlags().GetString("delay-profile"); v != "" {
		cfg.DelayProfile = v
	}
	if v, _ := rootCmd.PersistentFlags().GetBool("respect-robots"); !v {
		cfg.RespectRobots = false
	}
	if v, _ := rootCmd.PersistentFlags().GetString("proxy-mode"); v != "" {
		cfg.ProxyMode = v
	}
	if v, _ := rootCmd.PersistentFlags().GetString("wireguard-config"); v != "" {
		cfg.WireGuardConfig = v
	}
	if v, _ := rootCmd.PersistentFlags().GetString("proxy-file"); v != "" {
		cfg.ProxyFile = v
	}
}

// buildHTTPClient creates the stealth-wrapped HTTP client from config.
func buildHTTPClient() *http.Client {
	fpPool := stealth.NewFingerprintPool()
	delay := stealth.NewHumanDelay(stealth.DelayProfile(cfg.DelayProfile))
	limiter := rate.NewLimiter(rate.Limit(cfg.RatePerSecond), cfg.RateBurst)

	baseTransport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
	}

	var proxyRotator *stealth.ProxyRotator
	switch cfg.ProxyMode {
	case "decodo":
		if cfg.DecodoUsername != "" && cfg.DecodoPassword != "" {
			proxyRotator = stealth.NewProxyRotator([]stealth.ProxyProvider{
				&stealth.DecodoProvider{
					Username: cfg.DecodoUsername,
					Password: cfg.DecodoPassword,
					Country:  cfg.DecodoCountry,
					City:     cfg.DecodoCity,
				},
			})
		}
	}

	robotsClient := &http.Client{}
	robots := stealth.NewRobotsChecker(robotsClient, cfg.RespectRobots)

	transport := &stealth.StealthTransport{
		Base:        baseTransport,
		Robots:      robots,
		Fingerprint: fpPool,
		Proxy:       proxyRotator,
		Delay:       delay,
		RateLimiter: limiter,
	}

	return &http.Client{Transport: transport}
}

// initPlatforms registers all available platform scrapers.
func initPlatforms() {
	client := buildHTTPClient()
	limiter := rate.NewLimiter(rate.Limit(cfg.RatePerSecond), cfg.RateBurst)
	tokScraper := tokopedia.NewScraper(client, limiter, cfg.MaxConcurrent)
	platform.Register("tokopedia", tokScraper)
}
