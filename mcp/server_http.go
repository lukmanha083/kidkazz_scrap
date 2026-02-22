package mcp

import (
	"crypto/subtle"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// ServeHTTP starts the MCP server over HTTP with optional Bearer token auth.
func ServeHTTP(addr, apiKey string) error {
	s := server.NewMCPServer(
		"kidkazz-scrap",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	registerTools(s)

	httpServer := server.NewStreamableHTTPServer(s, server.WithStateLess(true))

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	var mcpHandler http.Handler = httpServer
	if apiKey != "" {
		mcpHandler = bearerAuth(apiKey, httpServer)
	}
	mux.Handle("/mcp", mcpHandler)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("KidKazz MCP HTTP server listening on %s", addr)
	return srv.ListenAndServe()
}

func bearerAuth(apiKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.Header().Set("WWW-Authenticate", `Bearer realm="mcp"`)
			http.Error(w, `{"error":"missing Authorization header"}`, http.StatusUnauthorized)
			return
		}
		token, found := strings.CutPrefix(auth, "Bearer ")
		if !found || subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="mcp", error="invalid_token"`)
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
