// Package mcpcmd implements the goYoke MCP server as an importable subcmd.
package mcpcmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	tuimcp "github.com/Bucket-Chemist/goYoke/internal/tui/mcp"
)

// Run starts the goYoke MCP server using stdio transport.
//
// NOTE: The stdin and stdout parameters exist for RunFunc interface
// compatibility but are UNUSED. The MCP SDK (modelcontextprotocol/go-sdk)
// hardcodes os.Stdin and os.Stdout in its StdioTransport. The dispatch layer
// must NOT consume stdin before calling this function.
//
// resolve.SetDefault is intentionally omitted here; the caller (unified binary
// main or cmd/goyoke-mcp/main.go) handles that once at startup.
func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
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

	// Create the UDS client. If GOYOKE_SOCKET is not set the client exists but
	// returns ErrTUINotConnected on interactive tool calls.
	uds := tuimcp.NewUDSClient()

	tuimcp.RegisterAll(server, uds)

	slog.Info("goyoke-mcp starting", "socket", os.Getenv("GOYOKE_SOCKET"))

	if err := server.Run(ctx, &mcpsdk.StdioTransport{}); err != nil {
		return fmt.Errorf("goyoke-mcp: server error: %w", err)
	}
	return nil
}
