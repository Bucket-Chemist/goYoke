// Package main is the entry point for the gofortress-mcp-standalone MCP server binary.
//
// This binary exposes GOgent-Fortress agent-spawning capabilities over the
// Model Context Protocol using stdio transport. Unlike gofortress-mcp, it does
// not require a running TUI or GOFORTRESS_SOCKET — it is fully standalone.
//
// Usage:
//
//	gofortress-mcp-standalone
package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// Structured logging to stderr so it is visible via --debug on the claude
	// CLI. MCP server stdout is reserved for the JSON-RPC framing.
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	server := mcpsdk.NewServer(
		&mcpsdk.Implementation{
			Name:    "gofortress-mcp-standalone",
			Version: "1.0.0",
		},
		nil,
	)

	RegisterAll(server)

	slog.Info("gofortress-mcp-standalone starting")

	if err := server.Run(context.Background(), &mcpsdk.StdioTransport{}); err != nil {
		log.Fatalf("gofortress-mcp-standalone: server error: %v", err)
	}
}
