package codemap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteModuleExtraction writes a single module's extraction to JSON in outputDir.
// Uses atomic write (write to .tmp then rename).
func WriteModuleExtraction(outputDir string, extraction *ModuleExtraction) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	data, err := json.MarshalIndent(extraction, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal extraction for %s: %w", extraction.Module, err)
	}

	base := moduleToFilename(extraction.Module)
	if extraction.Language != "" && extraction.Language != "go" {
		base = base + "." + extraction.Language
	}
	outPath := filepath.Join(outputDir, base+".json")
	tmpPath := outPath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write tmp file for %s: %w", extraction.Module, err)
	}

	if err := os.Rename(tmpPath, outPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename tmp to output for %s: %w", extraction.Module, err)
	}

	return nil
}
