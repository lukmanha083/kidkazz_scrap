package httputil

import "net/http"

// BrowserHeaders returns common browser-like headers.
func BrowserHeaders() http.Header {
	h := http.Header{}
	h.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	h.Set("Accept-Language", "en-US,en;q=0.9,id;q=0.8")
	h.Set("Accept-Encoding", "gzip, deflate, br")
	h.Set("Connection", "keep-alive")
	h.Set("Upgrade-Insecure-Requests", "1")
	h.Set("Sec-Fetch-Dest", "document")
	h.Set("Sec-Fetch-Mode", "navigate")
	h.Set("Sec-Fetch-Site", "none")
	h.Set("Sec-Fetch-User", "?1")
	return h
}

// TokopediaGraphQLHeaders returns headers required for Tokopedia GraphQL API.
func TokopediaGraphQLHeaders() http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("Accept", "*/*")
	h.Set("Accept-Language", "en-US,en;q=0.9,id;q=0.8")
	h.Set("Origin", "https://www.tokopedia.com")
	h.Set("Referer", "https://www.tokopedia.com/")
	h.Set("X-Device", "desktop")
	h.Set("X-Source", "tokopedia-lite")
	h.Set("X-Tkpd-Lite-Service", "zeus")
	return h
}
