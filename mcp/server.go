package mcp

import (
	"github.com/mark3labs/mcp-go/server"
)

// Serve starts the MCP stdio server with all tools registered.
func Serve() error {
	s := server.NewMCPServer(
		"kidkazz-scrap",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	registerTools(s)

	return server.ServeStdio(s)
}
