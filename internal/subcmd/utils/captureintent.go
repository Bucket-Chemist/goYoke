package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/session"
)

// RunCaptureIntent implements the goyoke-capture-intent utility.
// Reads hook input from stdin (JSON), captures user intent from AskUserQuestion results.
func RunCaptureIntent(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	if err := captureIntentRun(stdin, stdout); err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-capture-intent] %v\n", err)
		// Graceful degradation: output empty JSON on error
		fmt.Fprintln(stdout, "{}")
		return nil
	}
	return nil
}

func captureIntentRun(stdin io.Reader, stdout io.Writer) error {
	var hookInput ciHookInput
	if err := json.NewDecoder(stdin).Decode(&hookInput); err != nil {
		return fmt.Errorf("failed to parse hook input: %w", err)
	}

	if hookInput.Tool.Name != "AskUserQuestion" {
		fmt.Fprintln(stdout, "{}")
		return nil
	}

	intent, err := captureExtractIntent(hookInput)
	if err != nil {
		return fmt.Errorf("failed to extract intent: %w", err)
	}

	if err := captureAppendIntent(intent); err != nil {
		return fmt.Errorf("failed to write intent: %w", err)
	}

	fmt.Fprintln(stdout, "{}")
	return nil
}

type ciHookInput struct {
	SessionID string `json:"session_id"`
	Tool      struct {
		Name   string          `json:"name"`
		Input  json.RawMessage `json:"input"`
		Result json.RawMessage `json:"result"`
	} `json:"tool"`
}

type ciAskUserQuestionInput struct {
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

type ciAskUserQuestionResult struct {
	Answers map[string]string `json:"answers"`
}

func captureExtractIntent(hookInput ciHookInput) (*session.UserIntent, error) {
	var input ciAskUserQuestionInput
	if err := json.Unmarshal(hookInput.Tool.Input, &input); err != nil {
		return nil, fmt.Errorf("failed to parse tool input: %w", err)
	}

	var result ciAskUserQuestionResult
	if err := json.Unmarshal(hookInput.Tool.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	if len(input.Questions) == 0 {
		return nil, fmt.Errorf("no questions in input")
	}

	q := input.Questions[0]

	response := ""
	for _, ans := range result.Answers {
		response = ans
		break
	}

	confidence := "explicit"
	if len(q.Options) == 0 {
		confidence = "inferred"
	}

	intent := &session.UserIntent{
		Timestamp:   time.Now().Unix(),
		Question:    q.Question,
		Response:    response,
		Confidence:  confidence,
		Context:     q.Header,
		Source:      "ask_user",
		SessionID:   hookInput.SessionID,
		ToolContext: hookInput.Tool.Name,
	}

	intent.Category = string(session.ClassifyIntent(q.Question, response))
	intent.Keywords = session.ExtractKeywords(response)

	return intent, nil
}

func captureAppendIntent(intent *session.UserIntent) error {
	projectDir := os.Getenv("GOYOKE_PROJECT_DIR")
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		projectDir = cwd
	}

	intentsPath := filepath.Join(config.ProjectMemoryDir(projectDir), "user-intents.jsonl")

	if err := os.MkdirAll(filepath.Dir(intentsPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	f, err := os.OpenFile(intentsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open intents file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(intent)
	if err != nil {
		return fmt.Errorf("failed to marshal intent: %w", err)
	}

	if _, err := f.WriteString(string(data) + "\n"); err != nil {
		return fmt.Errorf("failed to write intent: %w", err)
	}

	return nil
}
