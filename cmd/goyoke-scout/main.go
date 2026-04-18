package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Parse arguments
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: goyoke-scout <target> \"<instruction>\" [--output=<file>]\n")
		fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
		fmt.Fprintf(os.Stderr, "  SCOUT_OUTPUT=<file>          Output file path\n")
		os.Exit(1)
	}

	target := os.Args[1]
	instruction := os.Args[2]
	var inputFiles []string

	// Handle piped input (stdin file list)
	if target == "-" {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				inputFiles = append(inputFiles, line)
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to read stdin file list: %v\n", err)
			os.Exit(2)
		}
		if len(inputFiles) == 0 {
			fmt.Fprintf(os.Stderr, "Error: no files provided via stdin\n")
			os.Exit(2)
		}
		target = commonRoot(inputFiles)
	}

	// Validate target exists
	if _, err := os.Stat(target); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: target not found: %s\n", target)
		os.Exit(2)
	}

	// Run native scout
	report, err := runScout(target, instruction, inputFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Determine output path
	outputPath := os.Getenv("SCOUT_OUTPUT")
	if outputPath == "" {
		// Check for --output flag
		for i, arg := range os.Args {
			if strings.HasPrefix(arg, "--output=") {
				outputPath = strings.TrimPrefix(arg, "--output=")
			} else if arg == "--output" && i+1 < len(os.Args) {
				outputPath = os.Args[i+1]
			}
		}
	}

	// Marshal JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshalling JSON: %v\n", err)
		os.Exit(1)
	}

	// Write to file if specified (atomic write)
	if outputPath != "" {
		if err := atomicWrite(outputPath, data); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
	}

	// Always output to stdout as well
	fmt.Println(string(data))
}

// runScout executes the native scout with synthetic fallback.
func runScout(target, instruction string, files []string) (*ScoutReport, error) {
	// Run native scout
	scout := &NativeScout{Target: target, Instruction: instruction, Files: files}
	report, err := scout.Run()

	if err == nil {
		return report, nil
	}

	// Native failed - synthetic fallback (always succeeds)
	log.Printf("Native scout failed: %v, generating synthetic report", err)
	return generateSyntheticReport(target, err, nil), nil
}

func commonRoot(paths []string) string {
	if len(paths) == 0 {
		return "."
	}

	common := normalizePathRoot(paths[0])
	for _, path := range paths[1:] {
		common = sharedPathPrefix(common, normalizePathRoot(path))
		if common == "." || common == string(filepath.Separator) {
			break
		}
	}
	return common
}

func normalizePathRoot(path string) string {
	clean := filepath.Clean(path)
	if info, err := os.Stat(clean); err == nil && !info.IsDir() {
		return filepath.Dir(clean)
	}
	return clean
}

func sharedPathPrefix(a, b string) string {
	if a == b {
		return a
	}

	aParts := strings.Split(filepath.Clean(a), string(filepath.Separator))
	bParts := strings.Split(filepath.Clean(b), string(filepath.Separator))
	maxParts := len(aParts)
	if len(bParts) < maxParts {
		maxParts = len(bParts)
	}

	var shared []string
	for i := 0; i < maxParts; i++ {
		if aParts[i] != bParts[i] {
			break
		}
		shared = append(shared, aParts[i])
	}

	switch len(shared) {
	case 0:
		return "."
	case 1:
		if shared[0] == "" {
			return string(filepath.Separator)
		}
	}

	return filepath.Join(shared...)
}

// atomicWrite writes data to a file atomically using temp + rename.
func atomicWrite(path string, data []byte) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temp file
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // Clean up temp file
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
