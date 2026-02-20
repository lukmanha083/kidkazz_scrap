package stealth

import (
	"net/http"
	"sync"
)

// Fingerprint represents a browser identity with matching UA and headers.
type Fingerprint struct {
	UserAgent string
	Headers   http.Header
}

// FingerprintPool rotates through a set of browser fingerprints.
type FingerprintPool struct {
	fingerprints []Fingerprint
	mu           sync.Mutex
	idx          int
}

// NewFingerprintPool creates a pool with realistic browser fingerprints.
func NewFingerprintPool() *FingerprintPool {
	return &FingerprintPool{
		fingerprints: defaultFingerprints(),
	}
}

// Next returns the next fingerprint in round-robin order.
func (fp *FingerprintPool) Next() Fingerprint {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	f := fp.fingerprints[fp.idx%len(fp.fingerprints)]
	fp.idx++
	return f
}

func defaultFingerprints() []Fingerprint {
	return []Fingerprint{
		// Chrome 133 — Windows
		{
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
			Headers:   chromeHeaders("133"),
		},
		// Chrome 133 — macOS
		{
			UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
			Headers:   chromeHeaders("133"),
		},
		// Chrome 133 — Linux
		{
			UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
			Headers:   chromeHeaders("133"),
		},
		// Firefox 135 — Windows
		{
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:135.0) Gecko/20100101 Firefox/135.0",
			Headers:   firefoxHeaders(),
		},
		// Firefox 135 — macOS
		{
			UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:135.0) Gecko/20100101 Firefox/135.0",
			Headers:   firefoxHeaders(),
		},
		// Edge 133 — Windows
		{
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36 Edg/133.0.0.0",
			Headers:   chromeHeaders("133"),
		},
	}
}

func chromeHeaders(version string) http.Header {
	h := http.Header{}
	h.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	h.Set("Accept-Language", "en-US,en;q=0.9")
	h.Set("Accept-Encoding", "gzip, deflate, br")
	h.Set("Sec-Ch-Ua", `"Chromium";v="`+version+`", "Not(A:Brand";v="99", "Google Chrome";v="`+version+`"`)
	h.Set("Sec-Ch-Ua-Mobile", "?0")
	h.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	h.Set("Sec-Fetch-Dest", "document")
	h.Set("Sec-Fetch-Mode", "navigate")
	h.Set("Sec-Fetch-Site", "none")
	h.Set("Sec-Fetch-User", "?1")
	h.Set("Upgrade-Insecure-Requests", "1")
	return h
}

func firefoxHeaders() http.Header {
	h := http.Header{}
	h.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	h.Set("Accept-Language", "en-US,en;q=0.5")
	h.Set("Accept-Encoding", "gzip, deflate, br")
	h.Set("Sec-Fetch-Dest", "document")
	h.Set("Sec-Fetch-Mode", "navigate")
	h.Set("Sec-Fetch-Site", "none")
	h.Set("Sec-Fetch-User", "?1")
	h.Set("Upgrade-Insecure-Requests", "1")
	return h
}
