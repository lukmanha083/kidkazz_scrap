package cmd

import (
	"fmt"
	"log"

	mcpserver "github.com/lukman83/kidkazz-scrap/mcp"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP stdio server",
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	initPlatforms()

	fmt.Fprintln(cmd.ErrOrStderr(), "Starting KidKazz MCP server on stdio...")

	if err := mcpserver.Serve(); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
	return nil
}
