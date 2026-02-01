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

func TestAppendJSONL_Basic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	data := []byte(`{"id":1}`)
	if err := AppendJSONL(path, data); err != nil {
		t.Fatalf("AppendJSONL failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(content) != "{\"id\":1}\n" {
		t.Errorf("Unexpected content: %q", content)
	}
}

func TestAppendJSONL_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "test.jsonl")

	if err := AppendJSONL(path, []byte(`{"nested":true}`)); err != nil {
		t.Fatalf("AppendJSONL failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("File was not created")
	}
}

func TestAppendJSONL_Concurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "concurrent.jsonl")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			data := fmt.Sprintf(`{"id":%d}`, n)
			if err := AppendJSONL(path, []byte(data)); err != nil {
				t.Errorf("AppendJSONL failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	// Verify: 100 valid JSON lines, no corruption
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 100 {
		t.Errorf("Expected 100 lines, got %d", len(lines))
	}

	for i, line := range lines {
		var obj map[string]int
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("Line %d is not valid JSON: %s", i, line)
		}
	}
}
