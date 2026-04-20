package resolve

import (
	"errors"
	"fmt"
	"io/fs"
)

var criticalFiles = []string{
	"agents/agents-index.json",
	"routing-schema.json",
	"CLAUDE.md",
	"settings-template.json",
}

var criticalDirs = []string{
	"agents",
	"conventions",
	"rules",
}

func ValidateEmbeddedFS(embedFS fs.ReadFileFS) error {
	if embedFS == nil {
		return fmt.Errorf("resolve: embedded FS is nil")
	}

	var errs []error

	for _, path := range criticalFiles {
		if _, err := embedFS.ReadFile(path); err != nil {
			errs = append(errs, fmt.Errorf("missing critical file: %s", path))
		}
	}

	for _, dir := range criticalDirs {
		entries, err := fs.ReadDir(embedFS, dir)
		if err != nil || len(entries) == 0 {
			errs = append(errs, fmt.Errorf("empty or missing critical directory: %s", dir))
		}
	}

	return errors.Join(errs...)
}
