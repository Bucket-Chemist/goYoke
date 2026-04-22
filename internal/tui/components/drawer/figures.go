package drawer

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// DiagramEntry describes a single discovered diagram file.
type DiagramEntry struct {
	// Name is the display name derived from the file basename without extension.
	Name string
	// Path is the absolute path to the diagram source file.
	Path string
	// Type is the diagram format: "mermaid" for .mmd files, "text" otherwise.
	Type string
	// Source identifies which tool produced the diagram (e.g. "codebase-map").
	Source string
}

// FiguresState tracks the list of discovered diagrams and which one is selected.
type FiguresState struct {
	Diagrams []DiagramEntry
	Selected int
}

// FiguresContentMsg is sent to the Bubbletea event loop when DiscoverDiagrams
// completes so the AppModel can populate the figures drawer.
type FiguresContentMsg struct {
	Diagrams []DiagramEntry
}

// mermaidHTMLTmpl is the minimal HTML template used when opening a diagram in
// the browser. The {{.Content}} placeholder receives the raw Mermaid source.
const mermaidHTMLTmpl = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>{{.Name}}</title>
<!-- NOTE: no SRI — internal tool only -->
<script src="https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js"></script>
<script>mermaid.initialize({startOnLoad:true});</script>
<style>body{margin:2rem;font-family:sans-serif;}</style>
</head>
<body>
<h2>{{.Name}}</h2>
<div class="mermaid">
{{safe .Content}}
</div>
</body>
</html>`

// DiscoverDiagrams walks well-known locations relative to projectRoot and
// returns a sorted slice of DiagramEntry for every .mmd file found.
//
// Searched paths:
//   - docs/codebase-architecture/*/diagrams/*.mmd  (codebase-map structured output)
//   - .claude/codebase-map/mermaid/*.mmd           (CM-005c runtime output)
func DiscoverDiagrams(projectRoot string) []DiagramEntry {
	var entries []DiagramEntry

	patterns := []struct {
		glob   string
		source string
	}{
		{filepath.Join(projectRoot, "docs", "codebase-architecture", "*", "diagrams", "*.mmd"), "codebase-map"},
		{filepath.Join(projectRoot, ".claude", "codebase-map", "mermaid", "*.mmd"), "codebase-map"},
	}

	seen := map[string]struct{}{}

	for _, p := range patterns {
		matches, err := filepath.Glob(p.glob)
		if err != nil {
			continue
		}
		for _, match := range matches {
			abs, err := filepath.Abs(match)
			if err != nil {
				abs = match
			}
			if _, ok := seen[abs]; ok {
				continue
			}
			seen[abs] = struct{}{}

			name := strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))
			entries = append(entries, DiagramEntry{
				Name:   name,
				Path:   abs,
				Type:   "mermaid",
				Source: p.source,
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return entries
}

// OpenInBrowser writes content to a temporary HTML file embedding the Mermaid
// CDN renderer, then opens the file with xdg-open. It does not wait for the
// browser process to exit.
func OpenInBrowser(name, content string) error {
	f, err := os.CreateTemp("", "goyoke-diagram-*.html")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := f.Name()

	if err := renderMermaidHTML(f, name, content); err != nil {
		_ = f.Close()
		return fmt.Errorf("render html: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", tmpPath)
	case "darwin":
		cmd = exec.Command("open", tmpPath)
	default:
		cmd = exec.Command("xdg-open", tmpPath)
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	go func() {
		_ = cmd.Wait()
		time.Sleep(5 * time.Second)
		os.Remove(tmpPath)
	}()
	return nil
}

// renderMermaidHTML writes the Mermaid HTML template to w.
func renderMermaidHTML(w io.Writer, name, content string) error {
	tmpl, err := template.New("mermaid").Funcs(template.FuncMap{
		"safe": func(s string) template.HTML { return template.HTML(s) }, //nolint:gosec
	}).Parse(mermaidHTMLTmpl)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, struct {
		Name    string
		Content string
	}{Name: name, Content: content})
}

// readDiagramContent reads the file at path and returns its content.
// Returns an error string as content if reading fails so the drawer still shows something.
func readDiagramContent(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("(could not read %s: %v)", path, err)
	}
	return string(b)
}

// FormatFiguresContent renders the figures drawer content string for a given
// FiguresState. The content shows a navigable list of diagrams with the
// currently selected diagram's source displayed below.
func FormatFiguresContent(state FiguresState) string {
	if len(state.Diagrams) == 0 {
		return "No diagrams found.\n\nRun /codebase-map to generate diagrams."
	}

	var sb strings.Builder

	// Diagram index header.
	fmt.Fprintf(&sb, "Diagrams (%d)\n\n", len(state.Diagrams))
	for i, d := range state.Diagrams {
		if i == state.Selected {
			fmt.Fprintf(&sb, "  ▸ %s  [%s]\n", d.Name, d.Source)
		} else {
			fmt.Fprintf(&sb, "    %s\n", d.Name)
		}
	}

	// Show source of selected diagram.
	selected := state.Diagrams[state.Selected]
	content := readDiagramContent(selected.Path)
	sb.WriteString("\n── diagram source ──\n\n")
	sb.WriteString(content)

	return sb.String()
}
