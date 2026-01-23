package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/session"
)

// HookInput represents the structure of PostToolUse hook input
type HookInput struct {
	SessionID string `json:"session_id"`
	Tool      struct {
		Name   string          `json:"name"`
		Input  json.RawMessage `json:"input"`
		Result json.RawMessage `json:"result"`
	} `json:"tool"`
}

// AskUserQuestionInput represents the input structure for AskUserQuestion tool
type AskUserQuestionInput struct {
	Questions []struct {
		Question    string `json:"question"`
		Header      string `json:"header,omitempty"`
		Options     []struct {
			Label       string `json:"label"`
			Description string `json:"description,omitempty"`
		} `json:"options,omitempty"`
		MultiSelect bool `json:"multiSelect,omitempty"`
	} `json:"questions"`
}

// AskUserQuestionResult represents the result structure from AskUserQuestion tool
type AskUserQuestionResult struct {
	Answers map[string]string `json:"answers"`
}

func main() {
	if err := run(); err != nil {
		// Log error but don't fail the hook (graceful degradation)
		fmt.Fprintf(os.Stderr, "[gogent-capture-intent] %v\n", err)
		// Output empty JSON to not break hook chain
		fmt.Println("{}")
		os.Exit(0) // Don't fail - graceful degradation
	}
}

func run() error {
	// Parse hook input from STDIN
	var hookInput HookInput
	if err := json.NewDecoder(os.Stdin).Decode(&hookInput); err != nil {
		return fmt.Errorf("failed to parse hook input: %w", err)
	}

	// Verify this is an AskUserQuestion tool call
	if hookInput.Tool.Name != "AskUserQuestion" {
		// Not our tool, silent skip
		fmt.Println("{}")
		return nil
	}

	// Extract intent from AskUserQuestion result
	intent, err := extractIntent(hookInput)
	if err != nil {
		return fmt.Errorf("failed to extract intent: %w", err)
	}

	// Write to JSONL (async operation)
	if err := appendIntent(intent); err != nil {
		return fmt.Errorf("failed to write intent: %w", err)
	}

	// Output success (empty response, no hook modification needed)
	fmt.Println("{}")
	return nil
}

// extractIntent parses the hook input and constructs a UserIntent
func extractIntent(hookInput HookInput) (*session.UserIntent, error) {
	var input AskUserQuestionInput
	if err := json.Unmarshal(hookInput.Tool.Input, &input); err != nil {
		return nil, fmt.Errorf("failed to parse tool input: %w", err)
	}

	var result AskUserQuestionResult
	if err := json.Unmarshal(hookInput.Tool.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	// Handle single question case (most common)
	if len(input.Questions) == 0 {
		return nil, fmt.Errorf("no questions in input")
	}

	q := input.Questions[0]

	// Extract first answer from result
	response := ""
	for _, ans := range result.Answers {
		response = ans // Take first answer
		break
	}

	// Determine confidence based on response type
	confidence := "explicit"
	if len(q.Options) == 0 {
		confidence = "inferred" // Free-form response
	}

	// Use header as context if available
	context := q.Header

	intent := &session.UserIntent{
		Timestamp:   time.Now().Unix(),
		Question:    q.Question,
		Response:    response,
		Confidence:  confidence,
		Context:     context,
		Source:      "ask_user",
		SessionID:   hookInput.SessionID,
		ToolContext: hookInput.Tool.Name,
	}

	// GOgent-041: Add classification and keyword extraction
	intent.Category = string(session.ClassifyIntent(q.Question, response))
	intent.Keywords = session.ExtractKeywords(response)

	return intent, nil
}

// appendIntent appends a UserIntent to the JSONL file
func appendIntent(intent *session.UserIntent) error {
	// Determine project directory
	projectDir := os.Getenv("GOGENT_PROJECT_DIR")
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		projectDir = cwd
	}

	intentsPath := filepath.Join(projectDir, ".claude", "memory", "user-intents.jsonl")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(intentsPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open file for append (create if not exists)
	f, err := os.OpenFile(intentsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open intents file: %w", err)
	}
	defer f.Close()

	// Serialize intent to JSON
	data, err := json.Marshal(intent)
	if err != nil {
		return fmt.Errorf("failed to marshal intent: %w", err)
	}

	// Write JSONL line (JSON + newline)
	if _, err := f.WriteString(string(data) + "\n"); err != nil {
		return fmt.Errorf("failed to write intent: %w", err)
	}

	return nil
}
