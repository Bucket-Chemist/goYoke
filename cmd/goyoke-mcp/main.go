// Package main is the entry point for the goyoke-mcp MCP server binary.
//
// The binary exposes goYoke agent-spawning and user-interaction
// capabilities over the Model Context Protocol using stdio transport.
//
// When the TUI is running it sets GOYOKE_SOCKET to the path of a Unix
// domain socket.  Interactive tools (ask_user, confirm_action, etc.) connect
// to this socket to relay user prompts through the TUI.  Non-interactive tools
// (test_mcp_ping, spawn_agent, team_run) work without the TUI.
//
// Usage (managed by TUI — not intended for direct invocation):
//
//	GOYOKE_SOCKET=/run/user/1000/goyoke-12345.sock goyoke-mcp
package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Bucket-Chemist/goYoke/defaults"
	tuimcp "github.com/Bucket-Chemist/goYoke/internal/tui/mcp"
	"github.com/Bucket-Chemist/goYoke/pkg/resolve"
)

func main() {
	resolve.SetDefault(defaults.FS)
	// Structured logging to stderr so it is visible via --debug on the claude
	// CLI.  MCP server stdout is reserved for the JSON-RPC framing.
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	server := mcpsdk.NewServer(
		&mcpsdk.Implementation{
			Name:    "goyoke-mcp",
			Version: "1.0.0",
		},
		nil,
	)

	// Create the UDS client.  If GOYOKE_SOCKET is not set the client
	// exists but will return ErrTUINotConnected on interactive tool calls.
	uds := tuimcp.NewUDSClient()

	// Register all 8 tools.
	tuimcp.RegisterAll(server, uds)

	slog.Info("goyoke-mcp starting", "socket", os.Getenv("GOYOKE_SOCKET"))

	if err := server.Run(context.Background(), &mcpsdk.StdioTransport{}); err != nil {
		log.Fatalf("goyoke-mcp: server error: %v", err)
	}
}
