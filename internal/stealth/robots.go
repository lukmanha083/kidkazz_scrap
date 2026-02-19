package stealth

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

// RobotsChecker caches and checks robots.txt rules per domain.
type RobotsChecker struct {
	rules    map[string]*robotstxt.RobotsData
	expiry   map[string]time.Time
	mu       sync.RWMutex
	client   *http.Client
	cacheTTL time.Duration
	enabled  bool
}

// NewRobotsChecker creates a new robots.txt checker.
func NewRobotsChecker(client *http.Client, enabled bool) *RobotsChecker {
	return &RobotsChecker{
		rules:    make(map[string]*robotstxt.RobotsData),
		expiry:   make(map[string]time.Time),
		client:   client,
		cacheTTL: 1 * time.Hour,
		enabled:  enabled,
	}
}

// IsAllowed checks if the given URL is allowed by robots.txt.
func (r *RobotsChecker) IsAllowed(userAgent, rawURL string) (bool, error) {
	if !r.enabled {
		return true, nil
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return false, err
	}

	domain := u.Scheme + "://" + u.Host
	data, err := r.getRobots(domain)
	if err != nil {
		// If we can't fetch robots.txt, allow the request
		return true, nil
	}

	group := data.FindGroup(userAgent)
	return group.Test(u.Path), nil
}

// CrawlDelay returns the crawl delay specified for the user agent.
func (r *RobotsChecker) CrawlDelay(userAgent, domain string) time.Duration {
	if !r.enabled {
		return 0
	}

	data, err := r.getRobots(domain)
	if err != nil {
		return 0
	}

	group := data.FindGroup(userAgent)
	return group.CrawlDelay
}

func (r *RobotsChecker) getRobots(domain string) (*robotstxt.RobotsData, error) {
	r.mu.RLock()
	data, ok := r.rules[domain]
	exp, expOk := r.expiry[domain]
	r.mu.RUnlock()

	if ok && expOk && time.Now().Before(exp) {
		return data, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if data, ok := r.rules[domain]; ok {
		if exp, ok := r.expiry[domain]; ok && time.Now().Before(exp) {
			return data, nil
		}
	}

	robotsURL := domain + "/robots.txt"
	resp, err := r.client.Get(robotsURL)
	if err != nil {
		return nil, fmt.Errorf("fetch robots.txt: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read robots.txt: %w", err)
	}

	data, err = robotstxt.FromBytes(body)
	if err != nil {
		return nil, fmt.Errorf("parse robots.txt: %w", err)
	}

	r.rules[domain] = data
	r.expiry[domain] = time.Now().Add(r.cacheTTL)
	return data, nil
}
