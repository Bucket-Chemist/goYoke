package archive

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/session"
)

// showPerformance displays performance metrics with optional filtering.
func showPerformance() {
	perfFlags := flag.NewFlagSet("performance", flag.ExitOnError)
	byOperationFlag := perfFlags.Bool("by-operation", false, "Group metrics by operation type (summary view)")
	slowOnlyFlag := perfFlags.Bool("slow-only", false, "Show only slow metrics (>1000ms)")
	sinceFlag := perfFlags.String("since", "", "Filter since duration (7d) or date (YYYY-MM-DD)")
	perfFlags.Parse(os.Args[2:])

	projectDir := getProjectDir()
	q := session.NewQuery(projectDir)

	filters := session.PerformanceFilters{}
	if *slowOnlyFlag {
		filters.SlowOnly = true
	}
	if *sinceFlag != "" {
		since := parseSinceFilter(*sinceFlag)
		timestamp := since.Unix()
		filters.Since = &timestamp
	}

	if *byOperationFlag {
		summaries, err := q.QueryPerformanceSummary(filters)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-archive] Failed to query performance summary: %v\n", err)
			os.Exit(1)
		}

		if len(summaries) == 0 {
			fmt.Println("No performance metrics recorded.")
			return
		}

		fmt.Println("Operation                      | Count | Success | Failed | Avg (ms) | Min (ms) | Max (ms)")
		fmt.Println("-------------------------------|-------|---------|--------|----------|----------|----------")

		var totalOps int
		var totalSuccess int
		var totalFailed int

		for _, s := range summaries {
			operation := truncateForTable(s.Operation, 30)
			fmt.Printf("%-30s | %5d | %7d | %6d | %8.1f | %8d | %8d\n",
				operation, s.Count, s.SuccessCount, s.FailCount, s.AvgMs, s.MinMs, s.MaxMs)
			totalOps += s.Count
			totalSuccess += s.SuccessCount
			totalFailed += s.FailCount
		}

		fmt.Printf("\nTotal: %d operation(s) (%d success, %d failed)\n", totalOps, totalSuccess, totalFailed)
		return
	}

	metrics, err := q.QueryPerformance(filters)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-archive] Failed to query performance metrics: %v\n", err)
		os.Exit(1)
	}

	if len(metrics) == 0 {
		fmt.Println("No performance metrics recorded.")
		return
	}

	fmt.Println("Timestamp  | Operation                      | Duration | Memory      | Success | Context")
	fmt.Println("-----------|--------------------------------|----------|-------------|---------|--------------------")

	for _, m := range metrics {
		timestamp := time.Unix(m.Timestamp, 0).Format("2006-01-02")
		operation := truncateForTable(m.Operation, 30)
		duration := fmt.Sprintf("%dms", m.DurationMs)
		memory := formatBytes(m.MemoryBytes)
		success := "Yes"
		if !m.Success {
			success = "No"
		}
		context := truncateForTable(m.Context, 18)
		fmt.Printf("%s | %-30s | %8s | %11s | %-7s | %s\n",
			timestamp, operation, duration, memory, success, context)
	}

	fmt.Printf("\nTotal: %d metric(s)\n", len(metrics))

	var totalMs int64
	var successCount int
	for _, m := range metrics {
		totalMs += m.DurationMs
		if m.Success {
			successCount++
		}
	}
	avgMs := float64(totalMs) / float64(len(metrics))
	successRate := float64(successCount) / float64(len(metrics)) * 100
	fmt.Printf("Average duration: %.1fms | Success rate: %.1f%%\n", avgMs, successRate)
}

// formatBytes formats byte count for human-readable display.
func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "-"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// truncateForTable truncates string for table display.
func truncateForTable(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// parseSinceFilter parses duration (e.g., "7d") or date (YYYY-MM-DD) into time.Time.
func parseSinceFilter(since string) time.Time {
	now := time.Now()

	if daysStr, ok := strings.CutSuffix(since, "d"); ok {
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-archive] Invalid --since format '%s'\n", since)
			fmt.Fprintln(os.Stderr, "  Use duration format (e.g., '7d', '30d') or date format (YYYY-MM-DD)")
			os.Exit(1)
		}
		return now.AddDate(0, 0, -days)
	}

	parsedDate, err := time.Parse("2006-01-02", since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-archive] Invalid --since date format '%s'\n", since)
		fmt.Fprintln(os.Stderr, "  Expected YYYY-MM-DD format (e.g., '2026-01-15')")
		os.Exit(1)
	}
	return parsedDate
}
