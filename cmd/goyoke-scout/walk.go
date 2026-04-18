package main

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// walkTarget is a helper that walks a directory and calls fn for each supported file.
// This avoids code duplication between native scout and routing score calculation.
func walkTarget(target string, fn func(FileInfo)) error {
	return filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip unreadable paths
		}

		if d.IsDir() {
			// Skip hidden directories and common vendor directories
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(path)
		if lang, ok := SupportedExtensions[ext]; ok {
			lines, _ := countLines(path)
			fn(FileInfo{
				Path:     path,
				Lines:    lines,
				Language: lang,
				IsTest:   isTestFile(path),
			})
		}
		return nil
	})
}
