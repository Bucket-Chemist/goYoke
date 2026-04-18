// goyoke-mcp-poc is a minimal Go MCP server for TUI-002 spike validation.
// It registers a single tool (test_mcp_ping) and runs over stdio transport
// to verify Claude Code CLI can discover and invoke Go MCP tools.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PingInput is intentionally empty — the tool takes no arguments.
type PingInput struct{}

// PingOutput is the structured response from test_mcp_ping.
type PingOutput struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

func pingHandler(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input PingInput,
) (*mcp.CallToolResult, PingOutput, error) {
	return nil, PingOutput{
		Status:    "pong",
		Timestamp: time.Now().Unix(),
	}, nil
}

func main() {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "goyoke-mcp-poc",
			Version: "0.1.0",
		},
		nil,
	)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "test_mcp_ping",
		Description: "Returns a pong response with timestamp. Used for MCP connectivity validation.",
	}, pingHandler)

	fmt.Fprintln(log.Default().Writer(), "goyoke-mcp-poc: starting on stdio")
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}
