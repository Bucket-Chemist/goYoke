package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/callback"
)

var (
	version = "1.0.0"
	client  *callback.Client
	logger  *slog.Logger
)

// Tool input types
type AskUserInput struct {
	Message string   `json:"message" jsonschema:"question to ask the user,required"`
	Options []string `json:"options,omitempty" jsonschema:"optional list of choices"`
	Default string   `json:"default,omitempty" jsonschema:"default value if user doesn't respond"`
}

type ConfirmActionInput struct {
	Action      string `json:"action" jsonschema:"description of action requiring confirmation,required"`
	Destructive bool   `json:"destructive,omitempty" jsonschema:"whether action is destructive"`
}

type RequestInputInput struct {
	Prompt      string `json:"prompt" jsonschema:"input prompt to display,required"`
	Placeholder string `json:"placeholder,omitempty" jsonschema:"placeholder text"`
}

type SelectOptionInput struct {
	Message string   `json:"message" jsonschema:"selection prompt,required"`
	Options []string `json:"options" jsonschema:"options to choose from,required"`
}

// Tool output types
type AskUserOutput struct {
	Response  string `json:"response"`
	Cancelled bool   `json:"cancelled"`
}

type ConfirmActionOutput struct {
	Confirmed bool `json:"confirmed"`
	Cancelled bool `json:"cancelled"`
}

type RequestInputOutput struct {
	Input     string `json:"input"`
	Cancelled bool   `json:"cancelled"`
}

type SelectOptionOutput struct {
	Selected  string `json:"selected"`
	Index     int    `json:"index"`
	Cancelled bool   `json:"cancelled"`
}

// Tool handlers
func askUserHandler(ctx context.Context, req *mcp.CallToolRequest, input AskUserInput) (
	*mcp.CallToolResult, AskUserOutput, error,
) {
	promptType := "ask"
	if len(input.Options) > 0 {
		promptType = "select"
	}

	resp, err := client.SendPrompt(ctx, callback.PromptRequest{
		ID:      fmt.Sprintf("ask-%d", time.Now().UnixNano()),
		Type:    promptType,
		Message: input.Message,
		Options: input.Options,
		Default: input.Default,
	})

	if err != nil {
		logger.Error("failed to get user response", "error", err)
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "Failed to communicate with TUI: " + err.Error()}},
		}, AskUserOutput{}, nil
	}

	return nil, AskUserOutput{
		Response:  resp.Value,
		Cancelled: resp.Cancelled,
	}, nil
}

func confirmActionHandler(ctx context.Context, req *mcp.CallToolRequest, input ConfirmActionInput) (
	*mcp.CallToolResult, ConfirmActionOutput, error,
) {
	message := input.Action
	if input.Destructive {
		message = "⚠️ DESTRUCTIVE: " + message
	}

	resp, err := client.SendPrompt(ctx, callback.PromptRequest{
		ID:      fmt.Sprintf("confirm-%d", time.Now().UnixNano()),
		Type:    "confirm",
		Message: message,
	})

	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "Failed to get confirmation: " + err.Error()}},
		}, ConfirmActionOutput{}, nil
	}

	return nil, ConfirmActionOutput{
		Confirmed: resp.Value == "yes",
		Cancelled: resp.Cancelled,
	}, nil
}

func requestInputHandler(ctx context.Context, req *mcp.CallToolRequest, input RequestInputInput) (
	*mcp.CallToolResult, RequestInputOutput, error,
) {
	resp, err := client.SendPrompt(ctx, callback.PromptRequest{
		ID:      fmt.Sprintf("input-%d", time.Now().UnixNano()),
		Type:    "input",
		Message: input.Prompt,
		Default: input.Placeholder,
	})

	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "Failed to get user input: " + err.Error()}},
		}, RequestInputOutput{}, nil
	}

	return nil, RequestInputOutput{
		Input:     resp.Value,
		Cancelled: resp.Cancelled,
	}, nil
}

func selectOptionHandler(ctx context.Context, req *mcp.CallToolRequest, input SelectOptionInput) (
	*mcp.CallToolResult, SelectOptionOutput, error,
) {
	resp, err := client.SendPrompt(ctx, callback.PromptRequest{
		ID:      fmt.Sprintf("select-%d", time.Now().UnixNano()),
		Type:    "select",
		Message: input.Message,
		Options: input.Options,
	})

	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "Failed to get selection: " + err.Error()}},
		}, SelectOptionOutput{}, nil
	}

	// Find index of selected option
	index := -1
	for i, opt := range input.Options {
		if opt == resp.Value {
			index = i
			break
		}
	}

	return nil, SelectOptionOutput{
		Selected:  resp.Value,
		Index:     index,
		Cancelled: resp.Cancelled,
	}, nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Configure logging to stderr (stdout reserved for MCP protocol)
	logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: getLogLevel(),
	}))

	// Initialize callback client
	var err error
	client, err = callback.NewClient()
	if err != nil {
		logger.Error("failed to initialize callback client", "error", err)
		os.Exit(1)
	}

	// Verify TUI is reachable
	healthCtx, healthCancel := context.WithTimeout(ctx, 5*time.Second)
	defer healthCancel()
	if err := client.HealthCheck(healthCtx); err != nil {
		logger.Error("TUI health check failed", "error", err)
		os.Exit(1)
	}
	logger.Info("connected to TUI callback server")

	// Create MCP server
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "gofortress-mcp-server",
			Version: version,
		},
		&mcp.ServerOptions{
			Logger: logger,
			InitializedHandler: func(ctx context.Context, req *mcp.InitializedRequest) {
				logger.Info("MCP session initialized with Claude")
			},
		},
	)

	// Register tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ask_user",
		Description: "Ask the user a question. Use when you need clarification, preferences, or any input. Can present multiple choice options or free-form questions.",
	}, askUserHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "confirm_action",
		Description: "Request user confirmation before proceeding. Use for destructive operations, irreversible changes, or when explicit approval is needed.",
	}, confirmActionHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "request_input",
		Description: "Request free-form text input from the user. Use when you need text content like code snippets, descriptions, or configuration values.",
	}, requestInputHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "select_option",
		Description: "Present a list of options and let user select one. Returns both the selected value and its index.",
	}, selectOptionHandler)

	// Run server
	logger.Info("starting MCP server on stdio")
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		if !errors.Is(err, mcp.ErrConnectionClosed) && !errors.Is(err, context.Canceled) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}
	logger.Info("MCP server shutdown complete")
}

func getLogLevel() slog.Level {
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
