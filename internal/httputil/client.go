package httputil

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/andybalholm/brotli"
)

// NewHTTPClient creates an HTTP client with sensible defaults.
// An optional RoundTripper (e.g. StealthTransport) can be injected.
func NewHTTPClient(transport http.RoundTripper) *http.Client {
	if transport == nil {
		transport = &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		}
	}
	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

// DoWithRetry performs an HTTP request with retry logic.
// On retry, the request body is reset via req.GetBody if available.
func DoWithRetry(client *http.Client, req *http.Request, maxRetries int) (*http.Response, error) {
	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		if i > 0 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("reset request body for retry: %w", err)
			}
			req.Body = body
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
			continue
		}
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}

// ReadBody reads and decompresses an HTTP response body.
func ReadBody(resp *http.Response) ([]byte, error) {
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		var err error
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}
		defer reader.Close()
	case "br":
		reader = io.NopCloser(brotli.NewReader(resp.Body))
	default:
		reader = resp.Body
	}
	return io.ReadAll(reader)
}
