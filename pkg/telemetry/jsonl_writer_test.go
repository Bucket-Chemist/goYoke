package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func setupJSONLTestDir(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GOGENT_PROJECT_DIR", dir)

	// Create minimal .claude structure
	os.MkdirAll(filepath.Join(dir, ".claude", "memory"), 0755)
	os.MkdirAll(filepath.Join(dir, ".claude", "agents"), 0755)
	os.MkdirAll(filepath.Join(dir, ".gogent"), 0755)

	return func() { /* TempDir auto-cleans */ }
}

func TestAppendJSONL_Basic(t *testing.T) {
	cleanup := setupJSONLTestDir(t)
	defer cleanup()

	path := filepath.Join(os.Getenv("GOGENT_PROJECT_DIR"), ".gogent", "test.jsonl")

	data := []byte(`{"id":1,"msg":"hello"}`)
	err := AppendJSONL(path, data)
	if err != nil {
		t.Fatalf("AppendJSONL failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expected := `{"id":1,"msg":"hello"}` + "\n"
	if string(content) != expected {
		t.Errorf("Content mismatch.\nGot: %q\nWant: %q", string(content), expected)
	}
}

func TestAppendJSONL_CreatesDir(t *testing.T) {
	cleanup := setupJSONLTestDir(t)
	defer cleanup()

	// Path with nested non-existent directories
	path := filepath.Join(os.Getenv("GOGENT_PROJECT_DIR"), "deep", "nested", "path", "test.jsonl")

	data := []byte(`{"test":"creates-dir"}`)
	err := AppendJSONL(path, data)
	if err != nil {
		t.Fatalf("AppendJSONL failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("File was not created")
	}
}

func TestAppendJSONL_Concurrent(t *testing.T) {
	cleanup := setupJSONLTestDir(t)
	defer cleanup()

	path := filepath.Join(os.Getenv("GOGENT_PROJECT_DIR"), ".gogent", "concurrent.jsonl")

	// Spawn 100 goroutines writing simultaneously
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			data := fmt.Sprintf(`{"id":%d}`, n)
			err := AppendJSONL(path, []byte(data))
			if err != nil {
				t.Errorf("AppendJSONL failed for goroutine %d: %v", n, err)
			}
		}(i)
	}
	wg.Wait()

	// Verify: 100 valid JSON lines, no corruption
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 100 {
		t.Errorf("Expected 100 lines, got %d", len(lines))
	}

	for i, line := range lines {
		var obj map[string]int
		err := json.Unmarshal([]byte(line), &obj)
		if err != nil {
			t.Errorf("Line %d is not valid JSON: %s (error: %v)", i, line, err)
		}
	}
}

func TestAppendJSONL_MultipleAppends(t *testing.T) {
	cleanup := setupJSONLTestDir(t)
	defer cleanup()

	path := filepath.Join(os.Getenv("GOGENT_PROJECT_DIR"), ".gogent", "multi.jsonl")

	// Append multiple lines sequentially
	for i := 0; i < 5; i++ {
		data := fmt.Sprintf(`{"seq":%d}`, i)
		if err := AppendJSONL(path, []byte(data)); err != nil {
			t.Fatalf("AppendJSONL failed at iteration %d: %v", i, err)
		}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(lines))
	}
}
