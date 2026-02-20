package stealth

import (
	"fmt"
	"net/http"

	"golang.org/x/time/rate"
)

// StealthTransport is an http.RoundTripper that applies the full stealth pipeline:
// RobotsCheck → RateLimiter → HumanDelay → Fingerprint → Proxy → Send
type StealthTransport struct {
	Base        http.RoundTripper
	Robots      *RobotsChecker
	Fingerprint *FingerprintPool
	Proxy       *ProxyRotator
	Delay       *HumanDelay
	RateLimiter *rate.Limiter
}

func (t *StealthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone request to avoid mutating the caller's request (http.RoundTripper contract)
	clone := req.Clone(req.Context())

	// 1. Apply fingerprint (UA + headers)
	fp := t.Fingerprint.Next()
	clone.Header.Set("User-Agent", fp.UserAgent)
	for key, vals := range fp.Headers {
		if clone.Header.Get(key) == "" {
			for _, v := range vals {
				clone.Header.Add(key, v)
			}
		}
	}

	// 2. Check robots.txt
	if t.Robots != nil {
		allowed, err := t.Robots.IsAllowed(fp.UserAgent, clone.URL.String())
		if err == nil && !allowed {
			return nil, fmt.Errorf("blocked by robots.txt: %s", clone.URL.Path)
		}
	}

	// 3. Wait for rate limiter token
	if t.RateLimiter != nil {
		if err := t.RateLimiter.Wait(clone.Context()); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}
	}

	// 4. Apply human-like delay
	if t.Delay != nil {
		if err := t.Delay.Wait(clone.Context()); err != nil {
			return nil, fmt.Errorf("delay: %w", err)
		}
	}

	// 5. Route through proxy if configured
	transport := t.Base
	if t.Proxy != nil {
		transport = t.Proxy.Next().Transport()
	}
	if transport == nil {
		transport = http.DefaultTransport
	}

	return transport.RoundTrip(clone)
}
