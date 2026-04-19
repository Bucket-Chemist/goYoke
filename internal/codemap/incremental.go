package codemap

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ChangeSet describes what changed since the last extraction.
type ChangeSet struct {
	FullRemap    bool
	ChangedFiles []string
	NewFiles     []string
	DeletedFiles []string
	Method       string // "git" or "mtime"
}

// DetectChanges compares manifest state against current filesystem/git state.
// Returns FullRemap=true when the manifest is unusable or missing.
func DetectChanges(manifest *Manifest, projectRoot string, discoveredFiles map[ModuleKey][]string) (*ChangeSet, error) {
	if manifest == nil || manifest.GitCommit == "" || len(manifest.Modules) == 0 {
		return &ChangeSet{FullRemap: true}, nil
	}

	// Flatten discovered files into a lookup set.
	allDiscovered := make(map[string]bool)
	for _, files := range discoveredFiles {
		for _, f := range files {
			allDiscovered[f] = true
		}
	}

	// Git-based detection when possible.
	if isGitRepo(projectRoot) {
		cs, err := gitChangedFiles(projectRoot, manifest.GitCommit, allSupportedExts())
		if err == nil {
			cs.ChangedFiles = filterToDiscovered(cs.ChangedFiles, allDiscovered)
			cs.NewFiles = filterToDiscovered(cs.NewFiles, allDiscovered)
			// Deleted = files tracked in manifest that are no longer discovered.
			cs.DeletedFiles = manifestOnlyFiles(manifest, allDiscovered)
			return cs, nil
		}
		// git failed — fall through to mtime.
	}

	return mtimeChanges(manifest, projectRoot, allDiscovered)
}

// isGitRepo reports whether projectRoot is inside a git repository.
func isGitRepo(root string) bool {
	out, err := exec.Command("git", "-C", root, "rev-parse", "--git-dir").Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}

// gitChangedFiles runs git diff --name-status <sinceCommit> HEAD and parses the output.
// Only files with extensions in supportedExts are included in the result.
func gitChangedFiles(projectRoot, sinceCommit string, supportedExts []string) (*ChangeSet, error) {
	if err := exec.Command("git", "-C", projectRoot, "cat-file", "-e", sinceCommit).Run(); err != nil {
		return nil, fmt.Errorf("commit %s not found: %w", sinceCommit, err)
	}

	out, err := exec.Command("git", "-C", projectRoot, "diff", "--name-status", sinceCommit, "HEAD").Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}

	extSet := make(map[string]bool, len(supportedExts))
	for _, ext := range supportedExts {
		extSet[ext] = true
	}
	supported := func(path string) bool {
		return extSet[filepath.Ext(path)]
	}

	cs := &ChangeSet{Method: "git"}
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		status := parts[0]

		switch {
		case status == "A":
			if supported(parts[1]) {
				cs.NewFiles = append(cs.NewFiles, parts[1])
			}
		case status == "M":
			if supported(parts[1]) {
				cs.ChangedFiles = append(cs.ChangedFiles, parts[1])
			}
		case status == "D":
			if supported(parts[1]) {
				cs.DeletedFiles = append(cs.DeletedFiles, parts[1])
			}
		case strings.HasPrefix(status, "R") && len(parts) >= 3:
			if supported(parts[1]) {
				cs.DeletedFiles = append(cs.DeletedFiles, parts[1])
			}
			if supported(parts[2]) {
				cs.NewFiles = append(cs.NewFiles, parts[2])
			}
		case strings.HasPrefix(status, "C") && len(parts) >= 3:
			if supported(parts[2]) {
				cs.NewFiles = append(cs.NewFiles, parts[2])
			}
		}
	}
	return cs, nil
}

// filterToDiscovered keeps only paths that appear in the discovered set.
func filterToDiscovered(files []string, discovered map[string]bool) []string {
	var out []string
	for _, f := range files {
		if discovered[f] {
			out = append(out, f)
		}
	}
	return out
}

// manifestOnlyFiles returns paths tracked in the manifest that are no longer discovered.
func manifestOnlyFiles(manifest *Manifest, discovered map[string]bool) []string {
	var deleted []string
	for _, mod := range manifest.Modules {
		for path := range mod.Files {
			if !discovered[path] {
				deleted = append(deleted, path)
			}
		}
	}
	sort.Strings(deleted)
	return deleted
}

// mtimeChanges falls back to comparing file mtimes against manifest extracted_at timestamps.
func mtimeChanges(manifest *Manifest, projectRoot string, discovered map[string]bool) (*ChangeSet, error) {
	cs := &ChangeSet{Method: "mtime"}

	// Build path → extracted_at from manifest.
	manifestFiles := make(map[string]string)
	for _, mod := range manifest.Modules {
		for path, mf := range mod.Files {
			manifestFiles[path] = mf.ExtractedAt
		}
	}

	for path := range discovered {
		extractedAt, inManifest := manifestFiles[path]
		if !inManifest {
			cs.NewFiles = append(cs.NewFiles, path)
			continue
		}
		absPath := filepath.Join(projectRoot, path)
		info, err := os.Stat(absPath)
		if err != nil {
			continue
		}
		t, err := time.Parse(time.RFC3339, extractedAt)
		if err != nil || info.ModTime().UTC().After(t) {
			cs.ChangedFiles = append(cs.ChangedFiles, path)
		}
	}

	for path := range manifestFiles {
		if !discovered[path] {
			cs.DeletedFiles = append(cs.DeletedFiles, path)
		}
	}

	sort.Strings(cs.ChangedFiles)
	sort.Strings(cs.NewFiles)
	sort.Strings(cs.DeletedFiles)
	return cs, nil
}

// MergeExtraction updates an existing module extraction with new file data from partial.
// Files present in partial replace their counterparts in existing; other files are kept.
// Import graphs are unioned (approximation — may retain stale entries for deleted imports).
func MergeExtraction(existing, partial *ModuleExtraction) *ModuleExtraction {
	fileMap := make(map[string]FileExtract, len(existing.Files))
	for _, f := range existing.Files {
		fileMap[f.Path] = f
	}
	for _, f := range partial.Files {
		fileMap[f.Path] = f
	}

	files := make([]FileExtract, 0, len(fileMap))
	for _, f := range fileMap {
		files = append(files, f)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	return &ModuleExtraction{
		Module:           existing.Module,
		Language:         existing.Language,
		Files:            files,
		Imports:          unionImports(existing.Imports, partial.Imports),
		ExtractedAt:      partial.ExtractedAt,
		ExtractorVersion: partial.ExtractorVersion,
	}
}

// RemoveDeletedFiles removes file entries that match deletedFiles from extraction.
// Import graph is not modified (approximation — may retain stale entries).
func RemoveDeletedFiles(extraction *ModuleExtraction, deletedFiles []string) {
	deleteSet := make(map[string]bool, len(deletedFiles))
	for _, f := range deletedFiles {
		deleteSet[f] = true
	}
	kept := extraction.Files[:0]
	for _, f := range extraction.Files {
		if !deleteSet[f.Path] {
			kept = append(kept, f)
		}
	}
	extraction.Files = kept
}

// LoadExistingExtraction reads a previously written extraction JSON file from outputDir.
func LoadExistingExtraction(outputDir, modulePath string) (*ModuleExtraction, error) {
	path := filepath.Join(outputDir, moduleToFilename(modulePath)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read extraction %s: %w", modulePath, err)
	}
	var e ModuleExtraction
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("parse extraction %s: %w", modulePath, err)
	}
	return &e, nil
}

// unionImports merges two ImportGraphs, deduplicating each slice.
func unionImports(a, b ImportGraph) ImportGraph {
	return ImportGraph{
		Internal: dedupSorted(append(a.Internal, b.Internal...)),
		External: dedupSorted(append(a.External, b.External...)),
		Stdlib:   dedupSorted(append(a.Stdlib, b.Stdlib...)),
	}
}

func dedupSorted(ss []string) []string {
	if len(ss) == 0 {
		return nil
	}
	sort.Strings(ss)
	out := ss[:1]
	for _, s := range ss[1:] {
		if s != out[len(out)-1] {
			out = append(out, s)
		}
	}
	return out
}
