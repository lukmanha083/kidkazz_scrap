package cmd

import (
	"fmt"

	mcpserver "github.com/lukman83/kidkazz-scrap/mcp"
	"github.com/spf13/cobra"
)

var serveHTTPCmd = &cobra.Command{
	Use:   "serve-http",
	Short: "Start MCP HTTP server",
	Long:  "Start the MCP server over HTTP for remote access (e.g. from Fly.io).",
	RunE:  runServeHTTP,
}

func init() {
	serveHTTPCmd.Flags().String("port", "", "HTTP port (default from $PORT or 8080)")
	rootCmd.AddCommand(serveHTTPCmd)
}

func runServeHTTP(cmd *cobra.Command, args []string) error {
	initPlatforms()

	port := cfg.HTTPPort
	if p, _ := cmd.Flags().GetString("port"); p != "" {
		port = p
	}

	addr := fmt.Sprintf(":%s", port)
	return mcpserver.ServeHTTP(addr, cfg.APIKey)
}
