package stealth

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
)

// ProxyProvider abstracts a proxy backend.
type ProxyProvider interface {
	Transport() http.RoundTripper
	Name() string
}

// ProxyRotator cycles through multiple proxy providers.
type ProxyRotator struct {
	providers []ProxyProvider
	mu        sync.Mutex
	idx       int
}

// NewProxyRotator creates a rotator from a list of providers.
// Returns nil if no providers are given.
func NewProxyRotator(providers []ProxyProvider) *ProxyRotator {
	if len(providers) == 0 {
		return nil
	}
	return &ProxyRotator{providers: providers}
}

// Next returns the next proxy provider in round-robin order.
func (p *ProxyRotator) Next() ProxyProvider {
	p.mu.Lock()
	defer p.mu.Unlock()
	provider := p.providers[p.idx%len(p.providers)]
	p.idx++
	return provider
}

// DirectProvider routes traffic directly (no proxy).
type DirectProvider struct{}

func (d *DirectProvider) Transport() http.RoundTripper {
	return &http.Transport{
		DisableKeepAlives: false,
	}
}

func (d *DirectProvider) Name() string { return "direct" }

// DecodoProvider implements Decodo residential proxy routing.
type DecodoProvider struct {
	Username     string
	Password     string
	Country      string // e.g. "id" for Indonesia
	City         string // e.g. "jakarta" (optional)
	UseUnblocker bool
}

func (d *DecodoProvider) Name() string {
	if d.UseUnblocker {
		return "decodo-unblocker"
	}
	return "decodo-rotating"
}

func (d *DecodoProvider) Transport() http.RoundTripper {
	proxyURL, _ := url.Parse(d.RotatingURL())
	return &http.Transport{
		Proxy:             http.ProxyURL(proxyURL),
		DisableKeepAlives: true, // new IP per request
	}
}

func (d *DecodoProvider) RotatingURL() string {
	user := fmt.Sprintf("user-%s-country-%s", d.Username, d.Country)
	if d.City != "" {
		user += fmt.Sprintf("-city-%s", d.City)
	}
	host := "gate.decodo.com:7000"
	if d.UseUnblocker {
		host = "unblock.decodo.com:60000"
		user = d.Username
	}
	return fmt.Sprintf("http://%s:%s@%s", user, d.Password, host)
}

func (d *DecodoProvider) StickyURL(sessionID string, durationMin int) string {
	user := fmt.Sprintf("user-%s-country-%s-session-%s-sessionduration-%d",
		d.Username, d.Country, sessionID, durationMin)
	return fmt.Sprintf("http://%s:%s@gate.decodo.com:7000", user, d.Password)
}

// HTTPProxyProvider wraps a generic HTTP/SOCKS5 proxy URL.
type HTTPProxyProvider struct {
	ProxyURL string
	Label    string
}

func (h *HTTPProxyProvider) Name() string { return h.Label }

func (h *HTTPProxyProvider) Transport() http.RoundTripper {
	proxyURL, _ := url.Parse(h.ProxyURL)
	return &http.Transport{
		Proxy:             http.ProxyURL(proxyURL),
		DisableKeepAlives: true,
	}
}
