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
type DirectProvider struct {
	transport http.RoundTripper
}

func (d *DirectProvider) Transport() http.RoundTripper { return d.transport }
func (d *DirectProvider) Name() string                 { return "direct" }

// DecodoProvider implements Decodo residential proxy routing.
type DecodoProvider struct {
	Username     string
	Password     string
	Country      string // e.g. "id" for Indonesia
	City         string // e.g. "jakarta" (optional)
	UseUnblocker bool
	transport    http.RoundTripper
	once         sync.Once
}

func (d *DecodoProvider) Name() string {
	if d.UseUnblocker {
		return "decodo-unblocker"
	}
	return "decodo-rotating"
}

func (d *DecodoProvider) Transport() http.RoundTripper {
	d.once.Do(func() {
		proxyURL := d.buildProxyURL()
		d.transport = &http.Transport{
			Proxy:             http.ProxyURL(proxyURL),
			DisableKeepAlives: true, // new IP per request
		}
	})
	return d.transport
}

func (d *DecodoProvider) buildProxyURL() *url.URL {
	user := fmt.Sprintf("user-%s-country-%s", d.Username, d.Country)
	if d.City != "" {
		user += fmt.Sprintf("-city-%s", d.City)
	}
	host := "gate.decodo.com:7000"
	if d.UseUnblocker {
		host = "unblock.decodo.com:60000"
		user = d.Username
	}
	return &url.URL{
		Scheme: "http",
		User:   url.UserPassword(user, d.Password),
		Host:   host,
	}
}

func (d *DecodoProvider) StickyURL(sessionID string, durationMin int) *url.URL {
	user := fmt.Sprintf("user-%s-country-%s-session-%s-sessionduration-%d",
		d.Username, d.Country, sessionID, durationMin)
	return &url.URL{
		Scheme: "http",
		User:   url.UserPassword(user, d.Password),
		Host:   "gate.decodo.com:7000",
	}
}

// HTTPProxyProvider wraps a generic HTTP/SOCKS5 proxy URL.
type HTTPProxyProvider struct {
	RawURL    string
	Label     string
	transport http.RoundTripper
	once      sync.Once
	parseErr  error
}

func (h *HTTPProxyProvider) Name() string { return h.Label }

func (h *HTTPProxyProvider) Transport() http.RoundTripper {
	h.once.Do(func() {
		proxyURL, err := url.Parse(h.RawURL)
		if err != nil {
			h.parseErr = err
			h.transport = http.DefaultTransport
			return
		}
		h.transport = &http.Transport{
			Proxy:             http.ProxyURL(proxyURL),
			DisableKeepAlives: true,
		}
	})
	return h.transport
}

// Err returns any error from parsing the proxy URL.
// Must be called after Transport() to ensure initialization.
func (h *HTTPProxyProvider) Err() error {
	h.once.Do(func() {}) // ensure init ran
	return h.parseErr
}
