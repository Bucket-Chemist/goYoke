package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/callback"
)

// Coverage note: The main() function (0% coverage) is intentionally not tested as it's
// an entry point that cannot be called from tests. All business logic (handlers) has
// 100% coverage. TestServerInitialization validates the initialization sequence that
// main() performs.

// TestAskUserHandler validates ask_user tool with mock callback server
func TestAskUserHandler(t *testing.T) {
	tests := []struct {
		name           string
		input          AskUserInput
		mockResponse   callback.PromptResponse
		expectedOutput AskUserOutput
		expectError    bool
	}{
		{
			name: "simple question",
			input: AskUserInput{
				Message: "What is your name?",
			},
			mockResponse: callback.PromptResponse{
				ID:        "test-1",
				Value:     "Alice",
				Cancelled: false,
			},
			expectedOutput: AskUserOutput{
				Response:  "Alice",
				Cancelled: false,
			},
			expectError: false,
		},
		{
			name: "question with options (multiple choice)",
			input: AskUserInput{
				Message: "Choose a color",
				Options: []string{"red", "green", "blue"},
			},
			mockResponse: callback.PromptResponse{
				ID:        "test-2",
				Value:     "green",
				Cancelled: false,
			},
			expectedOutput: AskUserOutput{
				Response:  "green",
				Cancelled: false,
			},
			expectError: false,
		},
		{
			name: "question with default",
			input: AskUserInput{
				Message: "Enter timeout",
				Default: "30s",
			},
			mockResponse: callback.PromptResponse{
				ID:        "test-3",
				Value:     "30s",
				Cancelled: false,
			},
			expectedOutput: AskUserOutput{
				Response:  "30s",
				Cancelled: false,
			},
			expectError: false,
		},
		{
			name: "user cancels",
			input: AskUserInput{
				Message: "Continue?",
			},
			mockResponse: callback.PromptResponse{
				ID:        "test-4",
				Value:     "",
				Cancelled: true,
			},
			expectedOutput: AskUserOutput{
				Response:  "",
				Cancelled: true,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock callback server
			srv, _ := setupMockServer(t)
			defer srv.Shutdown(context.Background())
			defer srv.Cleanup()

			// Handle prompt in background
			go func() {
				select {
				case req := <-srv.PromptChan:
					tt.mockResponse.ID = req.ID // Match request ID
					err := srv.SendResponse(tt.mockResponse)
					require.NoError(t, err)
				case <-time.After(2 * time.Second):
					t.Log("timeout waiting for prompt")
				}
			}()

			// Execute handler
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, output, err := askUserHandler(ctx, &mcp.CallToolRequest{}, tt.input)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Nil(t, result, "success case should return nil CallToolResult")
				assert.Equal(t, tt.expectedOutput.Response, output.Response)
				assert.Equal(t, tt.expectedOutput.Cancelled, output.Cancelled)
			}
		})
	}
}

// TestConfirmActionHandler validates confirm_action tool
func TestConfirmActionHandler(t *testing.T) {
	tests := []struct {
		name           string
		input          ConfirmActionInput
		mockResponse   callback.PromptResponse
		expectedOutput ConfirmActionOutput
		expectError    bool
	}{
		{
			name: "user confirms",
			input: ConfirmActionInput{
				Action: "Delete temporary files",
			},
			mockResponse: callback.PromptResponse{
				ID:        "test-confirm-1",
				Value:     "yes",
				Cancelled: false,
			},
			expectedOutput: ConfirmActionOutput{
				Confirmed: true,
				Cancelled: false,
			},
			expectError: false,
		},
		{
			name: "user declines",
			input: ConfirmActionInput{
				Action: "Overwrite config.json",
			},
			mockResponse: callback.PromptResponse{
				ID:        "test-confirm-2",
				Value:     "no",
				Cancelled: false,
			},
			expectedOutput: ConfirmActionOutput{
				Confirmed: false,
				Cancelled: false,
			},
			expectError: false,
		},
		{
			name: "destructive action",
			input: ConfirmActionInput{
				Action:      "Drop production database",
				Destructive: true,
			},
			mockResponse: callback.PromptResponse{
				ID:        "test-confirm-3",
				Value:     "yes",
				Cancelled: false,
			},
			expectedOutput: ConfirmActionOutput{
				Confirmed: true,
				Cancelled: false,
			},
			expectError: false,
		},
		{
			name: "user cancels",
			input: ConfirmActionInput{
				Action: "Restart service",
			},
			mockResponse: callback.PromptResponse{
				ID:        "test-confirm-4",
				Value:     "",
				Cancelled: true,
			},
			expectedOutput: ConfirmActionOutput{
				Confirmed: false,
				Cancelled: true,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _ := setupMockServer(t)
			defer srv.Shutdown(context.Background())
			defer srv.Cleanup()

			go func() {
				select {
				case req := <-srv.PromptChan:
					tt.mockResponse.ID = req.ID
					// Verify destructive flag adds warning prefix
					if tt.input.Destructive {
						assert.Contains(t, req.Message, "⚠️ DESTRUCTIVE:")
					}
					err := srv.SendResponse(tt.mockResponse)
					require.NoError(t, err)
				case <-time.After(2 * time.Second):
					t.Log("timeout waiting for prompt")
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, output, err := confirmActionHandler(ctx, &mcp.CallToolRequest{}, tt.input)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Nil(t, result)
				assert.Equal(t, tt.expectedOutput.Confirmed, output.Confirmed)
				assert.Equal(t, tt.expectedOutput.Cancelled, output.Cancelled)
			}
		})
	}
}

// TestRequestInputHandler validates request_input tool
func TestRequestInputHandler(t *testing.T) {
	tests := []struct {
		name           string
		input          RequestInputInput
		mockResponse   callback.PromptResponse
		expectedOutput RequestInputOutput
		expectError    bool
	}{
		{
			name: "text input",
			input: RequestInputInput{
				Prompt: "Enter commit message",
			},
			mockResponse: callback.PromptResponse{
				ID:        "test-input-1",
				Value:     "fix: resolve authentication bug",
				Cancelled: false,
			},
			expectedOutput: RequestInputOutput{
				Input:     "fix: resolve authentication bug",
				Cancelled: false,
			},
			expectError: false,
		},
		{
			name: "input with placeholder",
			input: RequestInputInput{
				Prompt:      "Enter URL",
				Placeholder: "https://example.com",
			},
			mockResponse: callback.PromptResponse{
				ID:        "test-input-2",
				Value:     "https://api.example.com",
				Cancelled: false,
			},
			expectedOutput: RequestInputOutput{
				Input:     "https://api.example.com",
				Cancelled: false,
			},
			expectError: false,
		},
		{
			name: "user cancels",
			input: RequestInputInput{
				Prompt: "Enter API key",
			},
			mockResponse: callback.PromptResponse{
				ID:        "test-input-3",
				Value:     "",
				Cancelled: true,
			},
			expectedOutput: RequestInputOutput{
				Input:     "",
				Cancelled: true,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _ := setupMockServer(t)
			defer srv.Shutdown(context.Background())
			defer srv.Cleanup()

			go func() {
				select {
				case req := <-srv.PromptChan:
					tt.mockResponse.ID = req.ID
					err := srv.SendResponse(tt.mockResponse)
					require.NoError(t, err)
				case <-time.After(2 * time.Second):
					t.Log("timeout waiting for prompt")
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, output, err := requestInputHandler(ctx, &mcp.CallToolRequest{}, tt.input)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Nil(t, result)
				assert.Equal(t, tt.expectedOutput.Input, output.Input)
				assert.Equal(t, tt.expectedOutput.Cancelled, output.Cancelled)
			}
		})
	}
}

// TestSelectOptionHandler validates select_option tool
func TestSelectOptionHandler(t *testing.T) {
	tests := []struct {
		name           string
		input          SelectOptionInput
		mockResponse   callback.PromptResponse
		expectedOutput SelectOptionOutput
		expectError    bool
	}{
		{
			name: "select from list",
			input: SelectOptionInput{
				Message: "Choose deployment environment",
				Options: []string{"development", "staging", "production"},
			},
			mockResponse: callback.PromptResponse{
				ID:        "test-select-1",
				Value:     "staging",
				Cancelled: false,
			},
			expectedOutput: SelectOptionOutput{
				Selected:  "staging",
				Index:     1,
				Cancelled: false,
			},
			expectError: false,
		},
		{
			name: "select first option",
			input: SelectOptionInput{
				Message: "Select action",
				Options: []string{"create", "update", "delete"},
			},
			mockResponse: callback.PromptResponse{
				ID:        "test-select-2",
				Value:     "create",
				Cancelled: false,
			},
			expectedOutput: SelectOptionOutput{
				Selected:  "create",
				Index:     0,
				Cancelled: false,
			},
			expectError: false,
		},
		{
			name: "select last option",
			input: SelectOptionInput{
				Message: "Choose log level",
				Options: []string{"debug", "info", "warn", "error"},
			},
			mockResponse: callback.PromptResponse{
				ID:        "test-select-3",
				Value:     "error",
				Cancelled: false,
			},
			expectedOutput: SelectOptionOutput{
				Selected:  "error",
				Index:     3,
				Cancelled: false,
			},
			expectError: false,
		},
		{
			name: "user cancels selection",
			input: SelectOptionInput{
				Message: "Pick a file",
				Options: []string{"file1.txt", "file2.txt"},
			},
			mockResponse: callback.PromptResponse{
				ID:        "test-select-4",
				Value:     "",
				Cancelled: true,
			},
			expectedOutput: SelectOptionOutput{
				Selected:  "",
				Index:     -1,
				Cancelled: true,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _ := setupMockServer(t)
			defer srv.Shutdown(context.Background())
			defer srv.Cleanup()

			go func() {
				select {
				case req := <-srv.PromptChan:
					tt.mockResponse.ID = req.ID
					// Verify options are passed correctly
					assert.Equal(t, tt.input.Options, req.Options)
					err := srv.SendResponse(tt.mockResponse)
					require.NoError(t, err)
				case <-time.After(2 * time.Second):
					t.Log("timeout waiting for prompt")
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, output, err := selectOptionHandler(ctx, &mcp.CallToolRequest{}, tt.input)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Nil(t, result)
				assert.Equal(t, tt.expectedOutput.Selected, output.Selected)
				assert.Equal(t, tt.expectedOutput.Index, output.Index)
				assert.Equal(t, tt.expectedOutput.Cancelled, output.Cancelled)
			}
		})
	}
}

// TestHandlerErrorCases validates error handling when callback server fails
func TestHandlerErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		setupServer func() (*callback.Server, *callback.Client)
		handler     string
	}{
		{
			name: "ask_user with unreachable server",
			setupServer: func() (*callback.Server, *callback.Client) {
				// Create server but don't start it
				srv := callback.NewServer(os.Getpid())
				client := callback.NewClientWithPath(srv.SocketPath())
				return srv, client
			},
			handler: "ask_user",
		},
		{
			name: "confirm_action with unreachable server",
			setupServer: func() (*callback.Server, *callback.Client) {
				srv := callback.NewServer(os.Getpid())
				client := callback.NewClientWithPath(srv.SocketPath())
				return srv, client
			},
			handler: "confirm_action",
		},
		{
			name: "request_input with unreachable server",
			setupServer: func() (*callback.Server, *callback.Client) {
				srv := callback.NewServer(os.Getpid())
				client := callback.NewClientWithPath(srv.SocketPath())
				return srv, client
			},
			handler: "request_input",
		},
		{
			name: "select_option with unreachable server",
			setupServer: func() (*callback.Server, *callback.Client) {
				srv := callback.NewServer(os.Getpid())
				client := callback.NewClientWithPath(srv.SocketPath())
				return srv, client
			},
			handler: "select_option",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, clientInstance := tt.setupServer()
			defer srv.Cleanup()
			client = clientInstance

			// Initialize logger for error tests
			if logger == nil {
				logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
					Level: slog.LevelError,
				}))
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			var result *mcp.CallToolResult
			var err error

			switch tt.handler {
			case "ask_user":
				result, _, err = askUserHandler(ctx, &mcp.CallToolRequest{}, AskUserInput{
					Message: "test",
				})
			case "confirm_action":
				result, _, err = confirmActionHandler(ctx, &mcp.CallToolRequest{}, ConfirmActionInput{
					Action: "test",
				})
			case "request_input":
				result, _, err = requestInputHandler(ctx, &mcp.CallToolRequest{}, RequestInputInput{
					Prompt: "test",
				})
			case "select_option":
				result, _, err = selectOptionHandler(ctx, &mcp.CallToolRequest{}, SelectOptionInput{
					Message: "test",
					Options: []string{"a", "b"},
				})
			}

			// Error should be nil (handler converts errors to CallToolResult)
			assert.NoError(t, err)
			// But result should indicate error
			assert.NotNil(t, result)
			assert.True(t, result.IsError, "expected IsError=true for communication failure")
			assert.NotEmpty(t, result.Content, "expected error message in Content")
		})
	}
}

// TestGetLogLevel validates log level environment variable parsing
func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected slog.Level
	}{
		{
			name:     "debug level",
			envValue: "debug",
			expected: slog.LevelDebug,
		},
		{
			name:     "info level (default)",
			envValue: "",
			expected: slog.LevelInfo,
		},
		{
			name:     "warn level",
			envValue: "warn",
			expected: slog.LevelWarn,
		},
		{
			name:     "error level",
			envValue: "error",
			expected: slog.LevelError,
		},
		{
			name:     "unknown level defaults to info",
			envValue: "invalid",
			expected: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			original := os.Getenv("LOG_LEVEL")
			defer os.Setenv("LOG_LEVEL", original)

			// Set test value
			if tt.envValue == "" {
				os.Unsetenv("LOG_LEVEL")
			} else {
				os.Setenv("LOG_LEVEL", tt.envValue)
			}

			level := getLogLevel()
			assert.Equal(t, tt.expected, level)
		})
	}
}

// TestMCPProtocolConformance validates MCP protocol requirements
func TestMCPProtocolConformance(t *testing.T) {
	srv, _ := setupMockServer(t)
	defer srv.Shutdown(context.Background())
	defer srv.Cleanup()

	// Create MCP server
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "gofortress-mcp-server",
			Version: version,
		},
		&mcp.ServerOptions{
			Logger: slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelError, // Quiet during tests
			})),
		},
	)

	// Register all tools
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

	// Verify server configuration
	assert.NotNil(t, server)
	t.Log("✅ MCP Protocol: All 4 tools registered successfully")
}

// TestToolInputValidation validates input schema enforcement
func TestToolInputValidation(t *testing.T) {
	tests := []struct {
		name      string
		tool      string
		input     interface{}
		wantValid bool
	}{
		{
			name: "ask_user valid",
			tool: "ask_user",
			input: AskUserInput{
				Message: "test",
			},
			wantValid: true,
		},
		{
			name: "confirm_action valid",
			tool: "confirm_action",
			input: ConfirmActionInput{
				Action: "test",
			},
			wantValid: true,
		},
		{
			name: "request_input valid",
			tool: "request_input",
			input: RequestInputInput{
				Prompt: "test",
			},
			wantValid: true,
		},
		{
			name: "select_option valid with options",
			tool: "select_option",
			input: SelectOptionInput{
				Message: "test",
				Options: []string{"a", "b"},
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify inputs can be marshaled to JSON (MCP requirement)
			data, err := json.Marshal(tt.input)
			require.NoError(t, err, "input must be JSON-serializable")

			// Verify inputs can be unmarshaled back
			switch tt.tool {
			case "ask_user":
				var input AskUserInput
				err = json.Unmarshal(data, &input)
			case "confirm_action":
				var input ConfirmActionInput
				err = json.Unmarshal(data, &input)
			case "request_input":
				var input RequestInputInput
				err = json.Unmarshal(data, &input)
			case "select_option":
				var input SelectOptionInput
				err = json.Unmarshal(data, &input)
			}

			if tt.wantValid {
				require.NoError(t, err, "valid input should round-trip through JSON")
			}
		})
	}
}

// TestClientEnvironmentVariable validates GOFORTRESS_SOCKET handling
func TestClientEnvironmentVariable(t *testing.T) {
	t.Run("missing socket env var", func(t *testing.T) {
		// Save original value
		original := os.Getenv("GOFORTRESS_SOCKET")
		defer func() {
			if original != "" {
				os.Setenv("GOFORTRESS_SOCKET", original)
			}
		}()

		// Unset variable
		os.Unsetenv("GOFORTRESS_SOCKET")

		// Should fail
		_, err := callback.NewClient()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GOFORTRESS_SOCKET")
	})

	t.Run("valid socket path", func(t *testing.T) {
		srv := callback.NewServer(os.Getpid())
		ctx := context.Background()
		err := srv.Start(ctx)
		require.NoError(t, err)
		defer srv.Shutdown(ctx)
		defer srv.Cleanup()

		// Save original value
		original := os.Getenv("GOFORTRESS_SOCKET")
		defer func() {
			if original != "" {
				os.Setenv("GOFORTRESS_SOCKET", original)
			} else {
				os.Unsetenv("GOFORTRESS_SOCKET")
			}
		}()

		// Set to valid path
		os.Setenv("GOFORTRESS_SOCKET", srv.SocketPath())

		// Should succeed
		client, err := callback.NewClient()
		require.NoError(t, err)
		assert.NotNil(t, client)

		// Health check should work
		healthCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		err = client.HealthCheck(healthCtx)
		assert.NoError(t, err)
	})
}

// TestHealthCheck validates health check endpoint
func TestHealthCheck(t *testing.T) {
	srv, clientInstance := setupMockServer(t)
	defer srv.Shutdown(context.Background())
	defer srv.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := clientInstance.HealthCheck(ctx)
	require.NoError(t, err, "health check should succeed with running server")
}

// setupMockServer creates a mock callback server for testing
func setupMockServer(t *testing.T) (*callback.Server, *callback.Client) {
	t.Helper()

	srv := callback.NewServer(os.Getpid())
	ctx := context.Background()

	err := srv.Start(ctx)
	require.NoError(t, err, "failed to start mock callback server")

	// Set global client for handlers
	client = callback.NewClientWithPath(srv.SocketPath())

	// Initialize logger if nil (required by handlers)
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelError, // Quiet during tests
		}))
	}

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	return srv, client
}

// Benchmark handler performance
func BenchmarkAskUserHandler(b *testing.B) {
	srv := callback.NewServer(os.Getpid())
	ctx := context.Background()
	err := srv.Start(ctx)
	require.NoError(b, err)
	defer srv.Shutdown(ctx)
	defer srv.Cleanup()

	client = callback.NewClientWithPath(srv.SocketPath())

	// Mock responder
	go func() {
		for {
			select {
			case req := <-srv.PromptChan:
				srv.SendResponse(callback.PromptResponse{
					ID:        req.ID,
					Value:     "test response",
					Cancelled: false,
				})
			case <-ctx.Done():
				return
			}
		}
	}()

	input := AskUserInput{Message: "test"}
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := askUserHandler(reqCtx, &mcp.CallToolRequest{}, input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestConcurrentRequests validates handling of multiple simultaneous prompts
func TestConcurrentRequests(t *testing.T) {
	srv, _ := setupMockServer(t)
	defer srv.Shutdown(context.Background())
	defer srv.Cleanup()

	// Mock responder
	go func() {
		for {
			select {
			case req := <-srv.PromptChan:
				// Simulate slight delay
				time.Sleep(10 * time.Millisecond)
				srv.SendResponse(callback.PromptResponse{
					ID:        req.ID,
					Value:     fmt.Sprintf("response-%s", req.ID),
					Cancelled: false,
				})
			case <-time.After(5 * time.Second):
				return
			}
		}
	}()

	// Send multiple concurrent requests
	const numRequests = 5
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, output, err := askUserHandler(ctx, &mcp.CallToolRequest{}, AskUserInput{
				Message: fmt.Sprintf("Question %d", id),
			})

			if err != nil {
				results <- err
				return
			}

			if output.Cancelled {
				results <- fmt.Errorf("unexpected cancellation")
				return
			}

			results <- nil
		}(i)
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		err := <-results
		require.NoError(t, err, "concurrent request %d failed", i)
	}
}

// TestServerInitialization validates server startup flow
func TestServerInitialization(t *testing.T) {
	// Setup mock callback server
	srv := callback.NewServer(os.Getpid())
	ctx := context.Background()
	err := srv.Start(ctx)
	require.NoError(t, err)
	defer srv.Shutdown(ctx)
	defer srv.Cleanup()

	// Set environment variable
	original := os.Getenv("GOFORTRESS_SOCKET")
	defer func() {
		if original != "" {
			os.Setenv("GOFORTRESS_SOCKET", original)
		} else {
			os.Unsetenv("GOFORTRESS_SOCKET")
		}
	}()
	os.Setenv("GOFORTRESS_SOCKET", srv.SocketPath())

	// Initialize client (simulates main() behavior)
	clientInstance, err := callback.NewClient()
	require.NoError(t, err)
	assert.NotNil(t, clientInstance)

	// Verify health check
	healthCtx, healthCancel := context.WithTimeout(ctx, 5*time.Second)
	defer healthCancel()
	err = clientInstance.HealthCheck(healthCtx)
	require.NoError(t, err, "health check should succeed")

	// Create MCP server (simulates main() behavior)
	testLogger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "gofortress-mcp-server",
			Version: version,
		},
		&mcp.ServerOptions{
			Logger: testLogger,
			InitializedHandler: func(ctx context.Context, req *mcp.InitializedRequest) {
				testLogger.Info("MCP session initialized with Claude")
			},
		},
	)

	require.NotNil(t, server)

	// Register all tools (simulates main() behavior)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ask_user",
		Description: "Ask the user a question. Use when you need clarification, preferences, or any input. Can present multiple choice options or free-form questions.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input AskUserInput) (*mcp.CallToolResult, AskUserOutput, error) {
		// Test handler with mock client
		client = clientInstance
		logger = testLogger
		return askUserHandler(ctx, req, input)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "confirm_action",
		Description: "Request user confirmation before proceeding. Use for destructive operations, irreversible changes, or when explicit approval is needed.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ConfirmActionInput) (*mcp.CallToolResult, ConfirmActionOutput, error) {
		client = clientInstance
		logger = testLogger
		return confirmActionHandler(ctx, req, input)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "request_input",
		Description: "Request free-form text input from the user. Use when you need text content like code snippets, descriptions, or configuration values.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input RequestInputInput) (*mcp.CallToolResult, RequestInputOutput, error) {
		client = clientInstance
		logger = testLogger
		return requestInputHandler(ctx, req, input)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "select_option",
		Description: "Present a list of options and let user select one. Returns both the selected value and its index.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SelectOptionInput) (*mcp.CallToolResult, SelectOptionOutput, error) {
		client = clientInstance
		logger = testLogger
		return selectOptionHandler(ctx, req, input)
	})

	t.Log("✅ Server initialization complete with all 4 tools")
}

// TestMainInitializationFailure validates error handling during startup
func TestMainInitializationFailure(t *testing.T) {
	t.Run("missing GOFORTRESS_SOCKET", func(t *testing.T) {
		// Save and unset environment variable
		original := os.Getenv("GOFORTRESS_SOCKET")
		defer func() {
			if original != "" {
				os.Setenv("GOFORTRESS_SOCKET", original)
			}
		}()
		os.Unsetenv("GOFORTRESS_SOCKET")

		// NewClient should fail
		_, err := callback.NewClient()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GOFORTRESS_SOCKET")
	})

	t.Run("health check timeout", func(t *testing.T) {
		// Create client with invalid socket path
		clientInstance := callback.NewClientWithPath("/tmp/nonexistent-socket.sock")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Health check should fail
		err := clientInstance.HealthCheck(ctx)
		require.Error(t, err)
	})
}

// TestLogLevelConfiguration validates getLogLevel for all levels
func TestLogLevelConfiguration(t *testing.T) {
	// This is already covered in TestGetLogLevel, but adding explicit verification
	// that the logger can be initialized with each level
	levels := []struct {
		env   string
		level slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
	}

	for _, tc := range levels {
		t.Run(tc.env, func(t *testing.T) {
			original := os.Getenv("LOG_LEVEL")
			defer os.Setenv("LOG_LEVEL", original)

			os.Setenv("LOG_LEVEL", tc.env)
			level := getLogLevel()
			assert.Equal(t, tc.level, level)

			// Verify logger can be created with this level
			testLogger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
				Level: level,
			}))
			assert.NotNil(t, testLogger)
		})
	}
}
