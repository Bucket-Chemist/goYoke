// Package main implements the gogent-aggregate CLI for weekly learning aggregation.
// It archives learning artifact JSONL files that exceed a size threshold (1MB default)
// to weekly archive files and generates summary markdown.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
)

const (
	// MaxFileSizeBytes is the size threshold (1MB) that triggers aggregation.
	MaxFileSizeBytes = 1 * 1024 * 1024

	// WeekFormat is the ISO 8601 week format (YYYY-Www).
	WeekFormat = "2006-W02"
)

// FileSpec defines a JSONL file to aggregate with its source and archive type name.
type FileSpec struct {
	Filename    string // Base filename in memory directory
	ArchiveType string // Type name for archive file
}

// AggregationStats tracks counts of archived artifacts by type.
type AggregationStats struct {
	SharpEdges  int
	UserIntents int
	Decisions   int
	Preferences int
	Performance int
	Violations  int
}

// files defines the complete list of learning artifact JSONL files to aggregate.
var files = []FileSpec{
	{Filename: "pending-learnings.jsonl", ArchiveType: "sharp-edges"},
	{Filename: "user-intents.jsonl", ArchiveType: "user-intents"},
	{Filename: "decisions.jsonl", ArchiveType: "decisions"},
	{Filename: "preferences.jsonl", ArchiveType: "preferences"},
	{Filename: "performance.jsonl", ArchiveType: "performance"},
	{Filename: "routing-violations.jsonl", ArchiveType: "violations"},
}

func main() {
	os.Exit(run(os.Args[1:], os.Getenv("GOGENT_PROJECT_DIR")))
}

// run executes the main CLI logic with parsed args and project directory.
// Returns exit code (0 = success, 1 = error).
func run(args []string, projectDirEnv string) int {
	// Parse flags
	fs := flag.NewFlagSet("gogent-aggregate", flag.ContinueOnError)
	forceFlag := fs.Bool("force", false, "Force aggregation regardless of file size")
	dryRunFlag := fs.Bool("dry-run", false, "Show what would be aggregated without doing it")
	helpFlag := fs.Bool("help", false, "Show help message")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-aggregate] Failed to parse flags: %v\n", err)
		return 1
	}

	if *helpFlag {
		printHelp(os.Stdout)
		return 0
	}

	// Get project directory
	projectDir := projectDirEnv
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-aggregate] Failed to get working directory: %v\n", err)
			fmt.Fprintln(os.Stderr, "  Set GOGENT_PROJECT_DIR environment variable or run from project root.")
			return 1
		}
		projectDir = cwd
	}

	memoryDir := config.ProjectMemoryDir(projectDir)
	archiveDir := filepath.Join(memoryDir, "archive")

	// Build full paths for all files
	filePaths := buildFilePaths(memoryDir, files)

	// Check which files need aggregation
	filesToAggregate, err := checkAggregationNeeded(filePaths, *forceFlag, *dryRunFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-aggregate] Error checking files: %v\n", err)
		return 1
	}

	if len(filesToAggregate) == 0 {
		fmt.Println("No aggregation needed (all files < 1MB)")
		return 0
	}

	if *dryRunFlag {
		fmt.Println("\n[dry-run] No files were modified")
		return 0
	}

	// Ensure archive directory exists
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-aggregate] Failed to create archive directory: %v\n", err)
		return 1
	}

	// Perform aggregation
	week := time.Now().Format(WeekFormat)
	stats := aggregateFiles(filePaths, archiveDir, week)

	// Generate summary
	summaryPath := filepath.Join(archiveDir, week+"-summary.md")
	if err := generateSummary(summaryPath, week, stats); err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-aggregate] Failed to generate summary: %v\n", err)
		return 1
	}

	// Print completion message
	fmt.Printf("Aggregation complete: %s\n", week)
	fmt.Printf("  Sharp Edges: %d\n", stats.SharpEdges)
	fmt.Printf("  User Intents: %d\n", stats.UserIntents)
	fmt.Printf("  Decisions: %d\n", stats.Decisions)
	fmt.Printf("  Preferences: %d\n", stats.Preferences)
	fmt.Printf("  Performance: %d\n", stats.Performance)
	fmt.Printf("  Violations: %d\n", stats.Violations)
	fmt.Printf("  Summary: %s\n", summaryPath)

	return 0
}

// buildFilePaths creates full paths for all file specs.
func buildFilePaths(memoryDir string, specs []FileSpec) []string {
	paths := make([]string, len(specs))
	for i, spec := range specs {
		paths[i] = filepath.Join(memoryDir, spec.Filename)
	}
	return paths
}

// checkAggregationNeeded determines which files need aggregation.
// Returns list of files to aggregate, or error if file check fails.
func checkAggregationNeeded(filePaths []string, force, dryRun bool) ([]string, error) {
	var needsAggregation []string

	for _, path := range filePaths {
		size, err := getFileSize(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Skip missing files
			}
			return nil, fmt.Errorf("failed to stat %s: %w", path, err)
		}

		if size > MaxFileSizeBytes || force {
			needsAggregation = append(needsAggregation, path)
			if dryRun {
				fmt.Printf("[dry-run] Would aggregate %s (size: %d bytes)\n", filepath.Base(path), size)
			}
		}
	}

	return needsAggregation, nil
}

// getFileSize returns file size in bytes, error if file doesn't exist or can't be accessed.
func getFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// aggregateFiles moves JSONL files to archive with week prefix.
// It handles errors gracefully, logging warnings but continuing with other files.
func aggregateFiles(srcPaths []string, archiveDir, week string) AggregationStats {
	stats := AggregationStats{}

	for _, srcPath := range srcPaths {
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue // Skip missing files
		}

		// Determine file type from name
		basename := filepath.Base(srcPath)
		fileType := getFileType(basename)
		if fileType == "" {
			continue // Unknown file type
		}

		// Count lines before archiving
		count, err := countLines(srcPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-aggregate] Warning: Failed to count lines in %s: %v\n", srcPath, err)
			count = 0
		}

		// Update stats by type
		updateStats(&stats, fileType, count)

		// Archive path
		archivePath := filepath.Join(archiveDir, week+"-"+fileType+".jsonl")

		// Move file to archive (rename is atomic on same filesystem)
		if err := os.Rename(srcPath, archivePath); err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-aggregate] Warning: Failed to archive %s: %v\n", srcPath, err)
			continue
		}

		// Create empty replacement file
		if err := os.WriteFile(srcPath, []byte(""), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-aggregate] Warning: Failed to create replacement %s: %v\n", srcPath, err)
		}
	}

	return stats
}

// getFileType returns the archive type name for a given filename.
func getFileType(basename string) string {
	for _, spec := range files {
		if basename == spec.Filename {
			return spec.ArchiveType
		}
	}
	return ""
}

// updateStats updates the appropriate stat counter based on file type.
func updateStats(stats *AggregationStats, fileType string, count int) {
	switch fileType {
	case "sharp-edges":
		stats.SharpEdges = count
	case "user-intents":
		stats.UserIntents = count
	case "decisions":
		stats.Decisions = count
	case "preferences":
		stats.Preferences = count
	case "performance":
		stats.Performance = count
	case "violations":
		stats.Violations = count
	}
}

// countLines counts non-empty lines in a file.
func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return count, nil
}

// generateSummary creates a markdown summary of the archived week.
func generateSummary(summaryPath, week string, stats AggregationStats) error {
	summary := fmt.Sprintf(`# Weekly Learning Summary - %s

## Overview

Generated: %s
Week: %s

## Artifact Counts

- **Sharp Edges**: %d
- **User Intents**: %d
- **Decisions**: %d
- **Preferences**: %d
- **Performance Metrics**: %d
- **Routing Violations**: %d

## Archive Location

All artifacts for this week are archived in:
- sharp edges: `+"`"+`.gogent/memory/archive/%s-sharp-edges.jsonl`+"`"+`
- user intents: `+"`"+`.gogent/memory/archive/%s-user-intents.jsonl`+"`"+`
- decisions: `+"`"+`.gogent/memory/archive/%s-decisions.jsonl`+"`"+`
- preferences: `+"`"+`.gogent/memory/archive/%s-preferences.jsonl`+"`"+`
- performance: `+"`"+`.gogent/memory/archive/%s-performance.jsonl`+"`"+`
- violations: `+"`"+`.gogent/memory/archive/%s-violations.jsonl`+"`"+`

## Usage

Query archived decisions:
`+"```bash"+`
# Example: Find architectural decisions from this week
grep '"category":"architecture"' .gogent/memory/archive/%s-decisions.jsonl
`+"```"+`

Review sharp edges:
`+"```bash"+`
# Example: Count error types
jq '.error_type' .gogent/memory/archive/%s-sharp-edges.jsonl | sort | uniq -c
`+"```"+`

---

**Note**: This is an automated weekly aggregation. Original JSONL files have been archived and reset.
`,
		week,
		time.Now().Format("2006-01-02 15:04:05"),
		week,
		stats.SharpEdges,
		stats.UserIntents,
		stats.Decisions,
		stats.Preferences,
		stats.Performance,
		stats.Violations,
		week, week, week, week, week, week, // 6 archive paths
		week, // decisions query example
		week, // sharp edges query example
	)

	if err := os.WriteFile(summaryPath, []byte(summary), 0644); err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	return nil
}

// printHelp displays usage information to the given writer.
func printHelp(w *os.File) {
	fmt.Fprintln(w, "gogent-aggregate - Weekly learning aggregation")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  gogent-aggregate                Check files and aggregate if >1MB")
	fmt.Fprintln(w, "  gogent-aggregate --force        Force aggregation regardless of size")
	fmt.Fprintln(w, "  gogent-aggregate --dry-run      Show what would be aggregated")
	fmt.Fprintln(w, "  gogent-aggregate --help         Show this help")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Files checked:")
	fmt.Fprintln(w, "  - .gogent/memory/pending-learnings.jsonl   (SharpEdges)")
	fmt.Fprintln(w, "  - .gogent/memory/user-intents.jsonl        (UserIntents)")
	fmt.Fprintln(w, "  - .gogent/memory/decisions.jsonl           (Decisions)")
	fmt.Fprintln(w, "  - .gogent/memory/preferences.jsonl         (Preferences)")
	fmt.Fprintln(w, "  - .gogent/memory/performance.jsonl         (Performance)")
	fmt.Fprintln(w, "  - .gogent/memory/routing-violations.jsonl  (Violations)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Archive location:")
	fmt.Fprintln(w, "  .gogent/memory/archive/YYYY-Www-{type}.jsonl")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  gogent-aggregate                    # Auto-aggregate if needed")
	fmt.Fprintln(w, "  gogent-aggregate --dry-run          # Preview aggregation")
	fmt.Fprintln(w, "  gogent-aggregate --force            # Force weekly rollup")
}
