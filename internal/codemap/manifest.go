package codemap

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Manifest tracks extraction state and per-file timestamps. Matches manifest.schema.json.
type Manifest struct {
	Version          string                     `json:"version"`
	RepoName         string                     `json:"repo_name"`
	ProjectRoot      string                     `json:"project_root"`
	Languages        []string                   `json:"languages"`
	Modules          map[string]*ManifestModule `json:"modules"`
	LastFullMap      *string                    `json:"last_full_map"`
	LastIncremental  *string                    `json:"last_incremental"`
	Stats            ManifestStats              `json:"stats"`
	ExtractorVersion string                     `json:"extractor_version"`
	GitCommit        string                     `json:"git_commit"`
}

// ManifestModule holds per-module tracking state.
type ManifestModule struct {
	Files         map[string]*ManifestFile `json:"files"`
	LastExtracted string                   `json:"last_extracted"`
	LastEnriched  *string                  `json:"last_enriched"`
	Status        string                   `json:"status"` // extracted, enriched, stale, error
}

// ManifestFile holds per-file timestamps and metrics.
type ManifestFile struct {
	Mtime       string  `json:"mtime"`
	ExtractedAt string  `json:"extracted_at"`
	EnrichedAt  *string `json:"enriched_at"`
	LineCount   int     `json:"line_count"`
	SymbolCount int     `json:"symbol_count"`
}

// ManifestStats aggregates extraction metrics.
type ManifestStats struct {
	TotalFiles     int            `json:"total_files"`
	TotalModules   int            `json:"total_modules"`
	TotalSymbols   int            `json:"total_symbols"`
	TotalFunctions int            `json:"total_functions"`
	TotalTypes     int            `json:"total_types"`
	Languages      map[string]int `json:"languages"`
}

// LoadManifest loads a manifest from disk. Returns an empty manifest if path does not exist.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Manifest{
			Version: "1.0",
			Modules: make(map[string]*ManifestModule),
			Stats:   ManifestStats{Languages: make(map[string]int)},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

// BuildManifest constructs a fresh manifest from extraction results.
func BuildManifest(projectRoot string, extractions []*ModuleExtraction, version string) (*Manifest, error) {
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}

	repoName := ResolveRepoName(absRoot)
	gitCommit := ResolveGitCommit(absRoot)
	now := time.Now().UTC().Format(time.RFC3339)

	modules := make(map[string]*ManifestModule, len(extractions))
	stats := ManifestStats{
		Languages: make(map[string]int),
	}

	langSet := make(map[string]bool)

	for _, e := range extractions {
		files := make(map[string]*ManifestFile, len(e.Files))
		symbolCount := 0

		for _, fe := range e.Files {
			sc := len(fe.Symbols)
			symbolCount += sc

			mtime := statMtime(filepath.Join(absRoot, fe.Path))

			files[fe.Path] = &ManifestFile{
				Mtime:       mtime,
				ExtractedAt: e.ExtractedAt,
				LineCount:   fe.LineCount,
				SymbolCount: sc,
			}

			stats.TotalFiles++
			stats.Languages[e.Language]++
			langSet[e.Language] = true

			for _, sym := range fe.Symbols {
				stats.TotalSymbols++
				switch sym.Kind {
				case "function", "method":
					stats.TotalFunctions++
				case "type", "interface":
					stats.TotalTypes++
				}
			}
		}

		modules[e.Module] = &ManifestModule{
			Files:         files,
			LastExtracted: e.ExtractedAt,
			Status:        "extracted",
		}

		stats.TotalModules++
	}

	langs := make([]string, 0, len(langSet))
	for l := range langSet {
		langs = append(langs, l)
	}

	m := &Manifest{
		Version:          "1.0",
		RepoName:         repoName,
		ProjectRoot:      absRoot,
		Languages:        langs,
		Modules:          modules,
		LastFullMap:      &now,
		Stats:            stats,
		ExtractorVersion: version,
		GitCommit:        gitCommit,
	}
	return m, nil
}

// WriteManifest writes the manifest to path atomically.
func WriteManifest(path string, m *Manifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create manifest dir: %w", err)
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write manifest tmp: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename manifest tmp: %w", err)
	}

	return nil
}

// ResolveRepoName tries git remote, falls back to directory basename.
func ResolveRepoName(root string) string {
	out, err := exec.Command("git", "-C", root, "remote", "get-url", "origin").Output()
	if err == nil {
		remote := strings.TrimSpace(string(out))
		// strip .git suffix and take last path segment
		remote = strings.TrimSuffix(remote, ".git")
		parts := strings.Split(remote, "/")
		if len(parts) > 0 && parts[len(parts)-1] != "" {
			return parts[len(parts)-1]
		}
	}
	return filepath.Base(root)
}

// ResolveGitCommit returns the current HEAD commit SHA, or empty string.
func ResolveGitCommit(root string) string {
	out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// statMtime returns the mtime of path formatted as RFC3339, or empty string on error.
func statMtime(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return info.ModTime().UTC().Format(time.RFC3339)
}
