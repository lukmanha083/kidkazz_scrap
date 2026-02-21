package mcp

import (
	"fmt"
	"log"
	"net/http"
	"strings"

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

	log.Printf("KidKazz MCP HTTP server listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

func bearerAuth(apiKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, fmt.Sprintf(`{"error":"missing Authorization header"}`), http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == auth || token != apiKey {
			http.Error(w, fmt.Sprintf(`{"error":"invalid token"}`), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
