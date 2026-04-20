package utils

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
)

// RunAggregate implements the goyoke-aggregate utility.
// args receives remaining CLI arguments after the subcommand name is stripped.
func RunAggregate(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("aggregate", flag.ContinueOnError)
	forceFlag := fs.Bool("force", false, "Force aggregation regardless of file size")
	dryRunFlag := fs.Bool("dry-run", false, "Show what would be aggregated without doing it")
	helpFlag := fs.Bool("help", false, "Show help message")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("aggregate: failed to parse flags: %w", err)
	}

	if *helpFlag {
		aggPrintHelp(stdout)
		return nil
	}

	projectDir := os.Getenv("GOYOKE_PROJECT_DIR")
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("aggregate: failed to get working directory: %w", err)
		}
		projectDir = cwd
	}

	memoryDir := config.ProjectMemoryDir(projectDir)
	archiveDir := filepath.Join(memoryDir, "archive")

	filePaths := aggBuildFilePaths(memoryDir, aggFiles)

	filesToAggregate, err := aggCheckNeeded(filePaths, *forceFlag, *dryRunFlag)
	if err != nil {
		return fmt.Errorf("aggregate: error checking files: %w", err)
	}

	if len(filesToAggregate) == 0 {
		fmt.Fprintln(stdout, "No aggregation needed (all files < 1MB)")
		return nil
	}

	if *dryRunFlag {
		fmt.Fprintln(stdout, "\n[dry-run] No files were modified")
		return nil
	}

	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("aggregate: failed to create archive directory: %w", err)
	}

	week := time.Now().Format(aggWeekFormat)
	stats := aggAggregateFiles(filePaths, archiveDir, week)

	summaryPath := filepath.Join(archiveDir, week+"-summary.md")
	if err := aggGenerateSummary(summaryPath, week, stats); err != nil {
		return fmt.Errorf("aggregate: failed to generate summary: %w", err)
	}

	fmt.Fprintf(stdout, "Aggregation complete: %s\n", week)
	fmt.Fprintf(stdout, "  Sharp Edges: %d\n", stats.SharpEdges)
	fmt.Fprintf(stdout, "  User Intents: %d\n", stats.UserIntents)
	fmt.Fprintf(stdout, "  Decisions: %d\n", stats.Decisions)
	fmt.Fprintf(stdout, "  Preferences: %d\n", stats.Preferences)
	fmt.Fprintf(stdout, "  Performance: %d\n", stats.Performance)
	fmt.Fprintf(stdout, "  Violations: %d\n", stats.Violations)
	fmt.Fprintf(stdout, "  Summary: %s\n", summaryPath)

	return nil
}

const (
	aggMaxFileSizeBytes = 1 * 1024 * 1024
	aggWeekFormat       = "2006-W02"
)

type aggFileSpec struct {
	Filename    string
	ArchiveType string
}

type aggAggregationStats struct {
	SharpEdges  int
	UserIntents int
	Decisions   int
	Preferences int
	Performance int
	Violations  int
}

var aggFiles = []aggFileSpec{
	{Filename: "pending-learnings.jsonl", ArchiveType: "sharp-edges"},
	{Filename: "user-intents.jsonl", ArchiveType: "user-intents"},
	{Filename: "decisions.jsonl", ArchiveType: "decisions"},
	{Filename: "preferences.jsonl", ArchiveType: "preferences"},
	{Filename: "performance.jsonl", ArchiveType: "performance"},
	{Filename: "routing-violations.jsonl", ArchiveType: "violations"},
}

func aggBuildFilePaths(memoryDir string, specs []aggFileSpec) []string {
	paths := make([]string, len(specs))
	for i, spec := range specs {
		paths[i] = filepath.Join(memoryDir, spec.Filename)
	}
	return paths
}

func aggCheckNeeded(filePaths []string, force, dryRun bool) ([]string, error) {
	var needsAggregation []string
	for _, path := range filePaths {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to stat %s: %w", path, err)
		}
		size := info.Size()
		if size > aggMaxFileSizeBytes || force {
			needsAggregation = append(needsAggregation, path)
			if dryRun {
				fmt.Printf("[dry-run] Would aggregate %s (size: %d bytes)\n", filepath.Base(path), size)
			}
		}
	}
	return needsAggregation, nil
}

func aggAggregateFiles(srcPaths []string, archiveDir, week string) aggAggregationStats {
	stats := aggAggregationStats{}
	for _, srcPath := range srcPaths {
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue
		}
		basename := filepath.Base(srcPath)
		fileType := aggGetFileType(basename)
		if fileType == "" {
			continue
		}
		count, err := aggCountNonEmptyLines(srcPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-aggregate] Warning: Failed to count lines in %s: %v\n", srcPath, err)
			count = 0
		}
		aggUpdateStats(&stats, fileType, count)
		archivePath := filepath.Join(archiveDir, week+"-"+fileType+".jsonl")
		if err := os.Rename(srcPath, archivePath); err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-aggregate] Warning: Failed to archive %s: %v\n", srcPath, err)
			continue
		}
		if err := os.WriteFile(srcPath, []byte(""), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-aggregate] Warning: Failed to create replacement %s: %v\n", srcPath, err)
		}
	}
	return stats
}

func aggGetFileType(basename string) string {
	for _, spec := range aggFiles {
		if basename == spec.Filename {
			return spec.ArchiveType
		}
	}
	return ""
}

func aggUpdateStats(stats *aggAggregationStats, fileType string, count int) {
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

func aggCountNonEmptyLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			count++
		}
	}
	return count, scanner.Err()
}

func aggGenerateSummary(summaryPath, week string, stats aggAggregationStats) error {
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
- sharp edges: `+"`"+`.goyoke/memory/archive/%s-sharp-edges.jsonl`+"`"+`
- user intents: `+"`"+`.goyoke/memory/archive/%s-user-intents.jsonl`+"`"+`
- decisions: `+"`"+`.goyoke/memory/archive/%s-decisions.jsonl`+"`"+`
- preferences: `+"`"+`.goyoke/memory/archive/%s-preferences.jsonl`+"`"+`
- performance: `+"`"+`.goyoke/memory/archive/%s-performance.jsonl`+"`"+`
- violations: `+"`"+`.goyoke/memory/archive/%s-violations.jsonl`+"`"+`
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
		week, week, week, week, week, week,
	)
	return os.WriteFile(summaryPath, []byte(summary), 0644)
}

func aggPrintHelp(w io.Writer) {
	fmt.Fprintln(w, "goyoke-aggregate - Weekly learning aggregation")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  aggregate                Check files and aggregate if >1MB")
	fmt.Fprintln(w, "  aggregate --force        Force aggregation regardless of size")
	fmt.Fprintln(w, "  aggregate --dry-run      Show what would be aggregated")
	fmt.Fprintln(w, "  aggregate --help         Show this help")
}
