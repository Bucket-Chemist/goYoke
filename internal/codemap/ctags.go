package codemap

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CtagsTag represents one tag from ctags --output-format=json.
type CtagsTag struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Language  string `json:"language"`
	Kind      string `json:"kind"`
	Line      int    `json:"line"`
	End       int    `json:"end"`
	Signature string `json:"signature"`
	Access    string `json:"access"`
	Scope     string `json:"scope"`
	ScopeKind string `json:"scopeKind"`
}

// CtagsAvailable returns true if the ctags binary is installed.
func CtagsAvailable() bool {
	_, err := exec.LookPath("ctags")
	return err == nil
}

// RunCtags invokes universal-ctags on the given files and returns parsed tags.
// files should be paths relative to projectRoot.
func RunCtags(files []string, projectRoot string) ([]CtagsTag, error) {
	if len(files) == 0 {
		return nil, nil
	}
	args := []string{
		"--output-format=json",
		"--fields=+SneKaz", // Signature, end-line, Kind-long, access, scope
		"--kinds-all=*",
		"--extras=-F", // no file-scope tags
		"-L", "-",     // read file list from stdin
	}
	cmd := exec.Command("ctags", args...)
	cmd.Dir = projectRoot
	cmd.Stdin = strings.NewReader(strings.Join(files, "\n"))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ctags: %w\nstderr: %s", err, stderr.String())
	}

	var tags []CtagsTag
	scanner := bufio.NewScanner(&stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		// Skip ctags metadata lines (e.g. {"_type":"ptag",...}).
		if bytes.Contains(line, []byte(`"_type":"ptag"`)) {
			continue
		}
		var tag CtagsTag
		if err := json.Unmarshal(line, &tag); err != nil {
			continue // skip malformed lines
		}
		tags = append(tags, tag)
	}
	return tags, scanner.Err()
}

// CtagsToSymbol converts a CtagsTag to a Symbol.
func CtagsToSymbol(tag CtagsTag, lang string) Symbol {
	sym := Symbol{
		Name:      tag.Name,
		Kind:      mapCtagsKind(tag.Kind, lang),
		Signature: tag.Signature,
		LineStart: tag.Line,
		LineEnd:   tag.End,
		Exported:  isCtagsExported(tag, lang),
	}
	if tag.ScopeKind == "class" || tag.ScopeKind == "struct" || tag.ScopeKind == "impl" {
		sym.Kind = "method"
		scope := tag.Scope
		sym.Receiver = &scope
	}
	return sym
}

func mapCtagsKind(kind, _ string) string {
	switch kind {
	case "function", "func":
		return "function"
	case "method":
		return "method"
	case "class", "struct":
		return "type"
	case "enum":
		return "enum"
	case "trait":
		return "interface"
	case "interface":
		return "interface"
	case "type", "typedef", "alias":
		return "type"
	case "constant", "const":
		return "const"
	case "variable", "var", "field", "member", "property":
		return "var"
	case "use", "import":
		return "import"
	case "module", "namespace", "package":
		return "module"
	}
	return kind
}

func isCtagsExported(tag CtagsTag, lang string) bool {
	switch lang {
	case "rust":
		return tag.Access == "public"
	case "typescript":
		return tag.Access == "public" || tag.Access == "export"
	case "python":
		// Dunder names (__init__, __str__) are considered exported.
		if strings.HasPrefix(tag.Name, "__") && strings.HasSuffix(tag.Name, "__") {
			return true
		}
		return !strings.HasPrefix(tag.Name, "_")
	case "r":
		return true // R has no private concept outside NAMESPACE
	}
	return true
}

// ExtractModuleCtags extracts symbols and imports for a non-Go module using ctags + regex.
func ExtractModuleCtags(modulePath string, filePaths []string, lang string, projectRoot string, projectName string, verbose bool) (*ModuleExtraction, error) {
	tags, err := RunCtags(filePaths, projectRoot)
	if err != nil {
		return nil, fmt.Errorf("ctags: %w", err)
	}

	// Group tags by relative file path.
	byFile := make(map[string][]CtagsTag, len(filePaths))
	for _, tag := range tags {
		byFile[tag.Path] = append(byFile[tag.Path], tag)
	}

	var files []FileExtract
	internalSet := make(map[string]bool)
	externalSet := make(map[string]bool)
	stdlibSet := make(map[string]bool)

	for _, relPath := range filePaths {
		absPath := filepath.Join(projectRoot, relPath)
		source, readErr := os.ReadFile(absPath)
		if readErr != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "  warning: read %s: %v\n", relPath, readErr)
			}
			continue
		}

		lineCount := bytes.Count(source, []byte("\n")) + 1

		var symbols []Symbol
		for _, tag := range byFile[relPath] {
			symbols = append(symbols, CtagsToSymbol(tag, lang))
		}

		if lang == "python" {
			extractPythonDecorators(source, symbols)
		}

		fileImports := ExtractImports(source, lang, projectName)
		for _, p := range fileImports.Internal {
			internalSet[p] = true
		}
		for _, p := range fileImports.External {
			externalSet[p] = true
		}
		for _, p := range fileImports.Stdlib {
			stdlibSet[p] = true
		}

		files = append(files, FileExtract{
			Path:      relPath,
			LineCount: lineCount,
			Symbols:   symbols,
		})
	}

	if lang == "r" {
		symbolCount := 0
		for _, fe := range files {
			symbolCount += len(fe.Symbols)
		}
		if symbolCount == 0 {
			fmt.Fprintf(os.Stderr, "warning: no R symbols extracted for %s — ctags may not support R or files may be empty\n", modulePath)
		}
	}

	graph := ImportGraph{
		Internal: sortedKeys(internalSet),
		External: sortedKeys(externalSet),
		Stdlib:   sortedKeys(stdlibSet),
	}

	return &ModuleExtraction{
		Module:           modulePath,
		Language:         lang,
		Files:            files,
		Imports:          graph,
		ExtractedAt:      time.Now().UTC().Format(time.RFC3339),
		ExtractorVersion: Version,
	}, nil
}

// extractPythonDecorators scans lines above function/method definitions for @decorator.
func extractPythonDecorators(source []byte, symbols []Symbol) {
	lines := bytes.Split(source, []byte("\n"))
	for i, sym := range symbols {
		if sym.Kind != "function" && sym.Kind != "method" {
			continue
		}
		for lineIdx := sym.LineStart - 2; lineIdx >= 0; lineIdx-- {
			line := bytes.TrimSpace(lines[lineIdx])
			if bytes.HasPrefix(line, []byte("@")) {
				// Prepend so decorators are in top-down order.
				symbols[i].Decorators = append([]string{string(line)}, symbols[i].Decorators...)
			} else {
				break
			}
		}
	}
}

func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
