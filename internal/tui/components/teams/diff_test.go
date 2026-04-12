package teams

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// countLines
// ---------------------------------------------------------------------------

func TestCountLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty string", "", 0},
		{"single line no newline", "hello", 1},
		{"single line with newline", "hello\n", 1},
		{"two lines no trailing newline", "hello\nworld", 2},
		{"two lines with trailing newline", "hello\nworld\n", 2},
		{"three lines", "a\nb\nc\n", 3},
		{"only newline", "\n", 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := countLines(tc.input)
			if got != tc.want {
				t.Errorf("countLines(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// scanNDJSONForChanges
// ---------------------------------------------------------------------------

func TestScanNDJSONForChanges_MissingFile(t *testing.T) {
	files, added, removed := scanNDJSONForChanges("/nonexistent/path/stream.ndjson")
	if len(files) != 0 || added != 0 || removed != 0 {
		t.Errorf("missing file: got files=%v added=%d removed=%d, want all zero", files, added, removed)
	}
}

func TestScanNDJSONForChanges_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stream.ndjson")
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	files, added, removed := scanNDJSONForChanges(path)
	if len(files) != 0 || added != 0 || removed != 0 {
		t.Errorf("empty file: got files=%v added=%d removed=%d, want all zero", files, added, removed)
	}
}

func TestScanNDJSONForChanges_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stream.ndjson")
	if err := os.WriteFile(path, []byte("not json\n{bad\n"), 0644); err != nil {
		t.Fatal(err)
	}
	files, added, removed := scanNDJSONForChanges(path)
	if len(files) != 0 || added != 0 || removed != 0 {
		t.Errorf("malformed JSON: got files=%v added=%d removed=%d, want all zero", files, added, removed)
	}
}

func TestScanNDJSONForChanges_SkipsNonAssistantEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stream.ndjson")
	// tool_result and user events should be ignored.
	content := `{"type":"tool_result","content":"some result"}` + "\n" +
		`{"type":"user","message":{"content":"hello"}}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	files, added, removed := scanNDJSONForChanges(path)
	if len(files) != 0 || added != 0 || removed != 0 {
		t.Errorf("non-assistant events: want all zero, got files=%v added=%d removed=%d", files, added, removed)
	}
}

func TestScanNDJSONForChanges_WriteEvent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stream.ndjson")
	// Assistant event containing a Write tool_use block.
	ndjson := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Write","input":{"file_path":"/src/foo.go","content":"line1\nline2\nline3\n"}}]}}` + "\n"
	if err := os.WriteFile(path, []byte(ndjson), 0644); err != nil {
		t.Fatal(err)
	}
	files, added, removed := scanNDJSONForChanges(path)
	if _, ok := files["/src/foo.go"]; !ok {
		t.Errorf("expected /src/foo.go in files set, got %v", files)
	}
	if added != 3 {
		t.Errorf("linesAdded = %d, want 3", added)
	}
	if removed != 0 {
		t.Errorf("linesRemoved = %d, want 0", removed)
	}
}

func TestScanNDJSONForChanges_EditEvent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stream.ndjson")
	ndjson := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":{"file_path":"/src/bar.go","old_string":"old1\nold2\n","new_string":"new1\nnew2\nnew3\n"}}]}}` + "\n"
	if err := os.WriteFile(path, []byte(ndjson), 0644); err != nil {
		t.Fatal(err)
	}
	files, added, removed := scanNDJSONForChanges(path)
	if _, ok := files["/src/bar.go"]; !ok {
		t.Errorf("expected /src/bar.go in files set")
	}
	if removed != 2 {
		t.Errorf("linesRemoved = %d, want 2", removed)
	}
	if added != 3 {
		t.Errorf("linesAdded = %d, want 3", added)
	}
}

func TestScanNDJSONForChanges_DeduplicatesSameFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stream.ndjson")
	// Two Write events to the same file path.
	ndjson := `{"type":"assistant","message":{"content":[` +
		`{"type":"tool_use","name":"Write","input":{"file_path":"/src/same.go","content":"a\n"}},` +
		`{"type":"tool_use","name":"Write","input":{"file_path":"/src/same.go","content":"b\nc\n"}}` +
		`]}}` + "\n"
	if err := os.WriteFile(path, []byte(ndjson), 0644); err != nil {
		t.Fatal(err)
	}
	files, added, _ := scanNDJSONForChanges(path)
	if len(files) != 1 {
		t.Errorf("files len = %d, want 1 (deduplicated)", len(files))
	}
	// Both Write contents are counted toward linesAdded.
	if added != 3 {
		t.Errorf("linesAdded = %d, want 3 (1+2)", added)
	}
}

func TestScanNDJSONForChanges_SkipsNonFileTools(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stream.ndjson")
	ndjson := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"go build ./..."}}]}}` + "\n"
	if err := os.WriteFile(path, []byte(ndjson), 0644); err != nil {
		t.Fatal(err)
	}
	files, added, removed := scanNDJSONForChanges(path)
	if len(files) != 0 || added != 0 || removed != 0 {
		t.Errorf("Bash event should be ignored: got files=%v added=%d removed=%d", files, added, removed)
	}
}

func TestScanNDJSONForChanges_WriteWithEmptyFilePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stream.ndjson")
	// Write with empty file_path should be skipped.
	ndjson := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Write","input":{"file_path":"","content":"line1\n"}}]}}` + "\n"
	if err := os.WriteFile(path, []byte(ndjson), 0644); err != nil {
		t.Fatal(err)
	}
	files, added, _ := scanNDJSONForChanges(path)
	if len(files) != 0 || added != 0 {
		t.Errorf("empty file_path should be skipped, got files=%v added=%d", files, added)
	}
}

// ---------------------------------------------------------------------------
// computeDiffSummary
// ---------------------------------------------------------------------------

func TestComputeDiffSummary_NilTeam(t *testing.T) {
	s := computeDiffSummary(nil)
	if s.FilesChanged != 0 || s.LinesAdded != 0 || s.LinesRemoved != 0 || s.TotalCost != 0 {
		t.Errorf("nil team: want all zero, got %+v", s)
	}
}

func TestComputeDiffSummary_NoStreamFiles(t *testing.T) {
	dir := t.TempDir()
	ts := &TeamState{
		Dir: dir,
		Config: TeamConfig{
			TeamName: "test",
			Status:   "completed",
			Waves: []Wave{
				{WaveNumber: 1, Members: []Member{
					{Name: "worker", Agent: "go-pro", Status: "completed", CostUSD: 1.5},
				}},
			},
		},
	}
	s := computeDiffSummary(ts)
	// Stream files don't exist → file/line counts are zero, cost is correct.
	if s.FilesChanged != 0 {
		t.Errorf("FilesChanged = %d, want 0 (no stream files)", s.FilesChanged)
	}
	if s.TotalCost < 1.499 || s.TotalCost > 1.501 {
		t.Errorf("TotalCost = %.3f, want 1.500", s.TotalCost)
	}
}

func TestComputeDiffSummary_WithStreamFile(t *testing.T) {
	dir := t.TempDir()
	// Write a stream file for the "go-pro" agent.
	ndjson := `{"type":"assistant","message":{"content":[` +
		`{"type":"tool_use","name":"Write","input":{"file_path":"/src/a.go","content":"x\ny\n"}},` +
		`{"type":"tool_use","name":"Edit","input":{"file_path":"/src/b.go","old_string":"old\n","new_string":"new1\nnew2\n"}}` +
		`]}}` + "\n"
	streamPath := filepath.Join(dir, "stream_go-pro.ndjson")
	if err := os.WriteFile(streamPath, []byte(ndjson), 0644); err != nil {
		t.Fatal(err)
	}

	ts := &TeamState{
		Dir: dir,
		Config: TeamConfig{
			Status: "completed",
			Waves: []Wave{{
				WaveNumber: 1,
				Members: []Member{
					{Name: "worker", Agent: "go-pro", CostUSD: 0.75},
				},
			}},
		},
	}
	s := computeDiffSummary(ts)
	if s.FilesChanged != 2 {
		t.Errorf("FilesChanged = %d, want 2", s.FilesChanged)
	}
	// Write: 2 lines added. Edit: 1 removed, 2 added → total 4 added, 1 removed.
	if s.LinesAdded != 4 {
		t.Errorf("LinesAdded = %d, want 4", s.LinesAdded)
	}
	if s.LinesRemoved != 1 {
		t.Errorf("LinesRemoved = %d, want 1", s.LinesRemoved)
	}
	if s.TotalCost < 0.749 || s.TotalCost > 0.751 {
		t.Errorf("TotalCost = %.3f, want 0.750", s.TotalCost)
	}
}

func TestComputeDiffSummary_DeduplicatesAcrossMembers(t *testing.T) {
	dir := t.TempDir()
	// Two members both modify the same file.
	ndjsonA := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Write","input":{"file_path":"/shared.go","content":"a\n"}}]}}` + "\n"
	ndjsonB := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Write","input":{"file_path":"/shared.go","content":"b\n"}}]}}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "stream_agent-a.ndjson"), []byte(ndjsonA), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stream_agent-b.ndjson"), []byte(ndjsonB), 0644); err != nil {
		t.Fatal(err)
	}

	ts := &TeamState{
		Dir: dir,
		Config: TeamConfig{
			Status: "completed",
			Waves: []Wave{{
				WaveNumber: 1,
				Members: []Member{
					{Name: "a", Agent: "agent-a"},
					{Name: "b", Agent: "agent-b"},
				},
			}},
		},
	}
	s := computeDiffSummary(ts)
	if s.FilesChanged != 1 {
		t.Errorf("FilesChanged = %d, want 1 (shared.go deduplicated)", s.FilesChanged)
	}
}

// ---------------------------------------------------------------------------
// renderCompletionSummary
// ---------------------------------------------------------------------------

func TestRenderCompletionSummary_NoFiles(t *testing.T) {
	s := DiffSummary{FilesChanged: 0, TotalCost: 1.23}
	got := renderCompletionSummary(s, 120)
	if !strings.Contains(got, "no file changes") {
		t.Errorf("want 'no file changes' in %q", got)
	}
	if !strings.Contains(got, "1.23") {
		t.Errorf("want cost '1.23' in %q", got)
	}
	if !strings.Contains(got, "✓ done") {
		t.Errorf("want '✓ done' in %q", got)
	}
}

func TestRenderCompletionSummary_WithFilesAndLines(t *testing.T) {
	s := DiffSummary{FilesChanged: 5, LinesAdded: 42, LinesRemoved: 10, TotalCost: 2.50}
	got := renderCompletionSummary(s, 120)
	if !strings.Contains(got, "5 file(s) modified") {
		t.Errorf("want '5 file(s) modified' in %q", got)
	}
	if !strings.Contains(got, "+42") {
		t.Errorf("want '+42' in %q", got)
	}
	if !strings.Contains(got, "-10") {
		t.Errorf("want '-10' in %q", got)
	}
	if !strings.Contains(got, "2.50") {
		t.Errorf("want '2.50' in %q", got)
	}
}

func TestRenderCompletionSummary_FilesNoLineData(t *testing.T) {
	s := DiffSummary{FilesChanged: 3, LinesAdded: 0, LinesRemoved: 0, TotalCost: 0.75}
	got := renderCompletionSummary(s, 120)
	if !strings.Contains(got, "3 file(s) modified") {
		t.Errorf("want '3 file(s) modified' in %q", got)
	}
	// No line delta rendered when both are zero.
	if strings.Contains(got, "+0") {
		t.Errorf("unexpected '+0' in %q", got)
	}
}
