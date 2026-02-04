package claude

import (
	"fmt"
	"strings"
)

// CommandResult represents the outcome of a command execution.
// It provides structured feedback about what the command did and
// whether any user action (like process restart) is required.
type CommandResult struct {
	Message         string // Display message for user
	Error           error  // Error if command failed
	RequiresRestart bool   // True if process needs restart
	NewModel        string // New model name if changing
}

// CommandContext provides access to current session state.
// This enables commands to display relevant information and make
// context-aware decisions without requiring access to the full process state.
type CommandContext struct {
	SessionID    string  // Current session identifier
	CurrentModel string  // Currently active model name
	MessageCount int     // Number of messages exchanged
	TotalCost    float64 // Total cost in USD
}

// CommandHandler is the signature for command handler functions.
// Handlers receive parsed arguments and context, returning a CommandResult
// that describes the outcome and any required actions.
type CommandHandler func(args []string, ctx CommandContext) CommandResult

// commandRegistration pairs a handler with its metadata.
type commandRegistration struct {
	Handler     CommandHandler
	Description string
	Usage       string
}

// nativeCommands is the registry of all native TUI commands.
// These commands are intercepted and handled locally instead of
// being passed to Claude.
var nativeCommands = map[string]commandRegistration{
	"/model": {
		Handler:     handleModelCommand,
		Description: "View or change the active Claude model",
		Usage:       "/model [opus|sonnet|haiku]",
	},
	"/context": {
		Handler:     handleContextCommand,
		Description: "Display current session context",
		Usage:       "/context",
	},
}

// IsNativeCommand checks if the given input is a native TUI command.
// Returns true if the input starts with a registered command name.
func IsNativeCommand(input string) bool {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return false
	}

	// Extract command name (first word)
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false
	}

	cmd := strings.ToLower(parts[0])
	_, exists := nativeCommands[cmd]
	return exists
}

// ExecuteCommand executes a native TUI command and returns the result.
// The input string is parsed to extract the command and arguments,
// then dispatched to the appropriate handler.
func ExecuteCommand(input string, ctx CommandContext) CommandResult {
	input = strings.TrimSpace(input)
	if input == "" {
		return CommandResult{
			Error:   fmt.Errorf("empty command"),
			Message: "Error: Empty command",
		}
	}

	// Parse command and arguments
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return CommandResult{
			Error:   fmt.Errorf("empty command"),
			Message: "Error: Empty command",
		}
	}

	cmdName := strings.ToLower(parts[0])
	args := parts[1:]

	// Look up command
	registration, exists := nativeCommands[cmdName]
	if !exists {
		return CommandResult{
			Error:   fmt.Errorf("unknown command: %s", cmdName),
			Message: fmt.Sprintf("Error: Unknown command '%s'", cmdName),
		}
	}

	// Execute handler
	return registration.Handler(args, ctx)
}

// NormalizeModelName normalizes model name aliases to canonical short names.
// Supports case-insensitive matching of common model aliases.
//
// Supported mappings:
//   - "opus", "claude-3-opus", "claude-opus-4", "claude-opus-4-5" → "opus"
//   - "sonnet", "claude-3-sonnet", "claude-sonnet-4", "claude-sonnet-4-5" → "sonnet"
//   - "haiku", "claude-3-haiku", "claude-haiku-3.5" → "haiku"
//
// Returns empty string if the input is not a recognized model name.
func NormalizeModelName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))

	// Opus variations
	switch name {
	case "opus", "claude-3-opus", "claude-opus-4", "claude-opus-4-5",
		"claude-3-opus-20240229", "opus-4", "opus-4-5":
		return "opus"
	}

	// Sonnet variations
	switch name {
	case "sonnet", "claude-3-sonnet", "claude-sonnet-4", "claude-sonnet-4-5",
		"claude-3-sonnet-20240229", "sonnet-4", "sonnet-4-5":
		return "sonnet"
	}

	// Haiku variations
	switch name {
	case "haiku", "claude-3-haiku", "claude-haiku-3.5",
		"claude-3-haiku-20240307", "haiku-3.5":
		return "haiku"
	}

	return "" // Unrecognized model
}

// handleModelCommand handles the /model command.
// With no arguments: displays current model information.
// With arguments: validates the model name and returns restart instructions.
func handleModelCommand(args []string, ctx CommandContext) CommandResult {
	// No arguments - display current model
	if len(args) == 0 {
		return CommandResult{
			Message: fmt.Sprintf("Current model: %s\n\nTo change models, use: /model [opus|sonnet|haiku]\n\nNote: Changing models requires a process restart.",
				ctx.CurrentModel),
		}
	}

	// Parse requested model
	requestedModel := strings.Join(args, " ")
	normalizedModel := NormalizeModelName(requestedModel)

	if normalizedModel == "" {
		return CommandResult{
			Error: fmt.Errorf("invalid model: %s", requestedModel),
			Message: fmt.Sprintf("Error: Unknown model '%s'\n\nSupported models:\n  - opus (Claude Opus 4.5)\n  - sonnet (Claude Sonnet 4.5)\n  - haiku (Claude Haiku 3.5)",
				requestedModel),
		}
	}

	// Check if already using this model
	if normalizedModel == strings.ToLower(ctx.CurrentModel) {
		return CommandResult{
			Message: fmt.Sprintf("Already using model: %s", ctx.CurrentModel),
		}
	}

	// Valid model change - requires restart
	return CommandResult{
		Message: fmt.Sprintf("Model change requested: %s → %s\n\nRestart required. Press Ctrl+C to exit, then restart with:\n  claude-cli --model=%s",
			ctx.CurrentModel, normalizedModel, normalizedModel),
		RequiresRestart: true,
		NewModel:        normalizedModel,
	}
}

// handleContextCommand handles the /context command.
// Displays formatted session information including ID, message count,
// total cost, and active model.
func handleContextCommand(args []string, ctx CommandContext) CommandResult {
	message := fmt.Sprintf(`Session Context:
  Session ID:    %s
  Model:         %s
  Messages:      %d
  Total Cost:    $%.4f`,
		ctx.SessionID,
		ctx.CurrentModel,
		ctx.MessageCount,
		ctx.TotalCost,
	)

	return CommandResult{
		Message: message,
	}
}
