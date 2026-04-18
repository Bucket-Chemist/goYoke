package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/routing"
	"github.com/Bucket-Chemist/goYoke/pkg/telemetry"
)

const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "routing":
		routingCmd := flag.NewFlagSet("routing", flag.ExitOnError)
		format := routingCmd.String("format", "csv", "Output format: csv, json")
		since := routingCmd.String("since", "7d", "Time filter: 1d, 7d, 30d")
		output := routingCmd.String("output", "-", "Output file (- for stdout)")
		routingCmd.Parse(os.Args[2:])
		exportRoutingDecisions(*format, *since, *output)

	case "sequences":
		sequencesCmd := flag.NewFlagSet("sequences", flag.ExitOnError)
		format := sequencesCmd.String("format", "json", "Output format: json, csv")
		successOnly := sequencesCmd.Bool("successful-only", false, "Only successful sequences")
		output := sequencesCmd.String("output", "-", "Output file (- for stdout)")
		sequencesCmd.Parse(os.Args[2:])
		exportToolSequences(*format, *successOnly, *output)

	case "collaborations":
		collaborationsCmd := flag.NewFlagSet("collaborations", flag.ExitOnError)
		format := collaborationsCmd.String("format", "json", "Output format: json, csv")
		output := collaborationsCmd.String("output", "-", "Output file (- for stdout)")
		collaborationsCmd.Parse(os.Args[2:])
		exportCollaborations(*format, *output)

	case "training-dataset":
		trainingCmd := flag.NewFlagSet("training-dataset", flag.ExitOnError)
		outDir := trainingCmd.String("output", "ml-data/", "Output directory")
		trainingCmd.Parse(os.Args[2:])
		exportTrainingDataset(*outDir)

	case "review-findings":
		reviewCmd := flag.NewFlagSet("review-findings", flag.ExitOnError)
		output := reviewCmd.String("output", "-", "Output file (- for stdout)")
		reviewCmd.Parse(os.Args[2:])
		exportReviewFindings(*output)

	case "review-stats":
		statsCmd := flag.NewFlagSet("review-stats", flag.ExitOnError)
		statsCmd.Parse(os.Args[2:])
		exportReviewStats()

	case "sharp-edge-hits":
		hitsCmd := flag.NewFlagSet("sharp-edge-hits", flag.ExitOnError)
		output := hitsCmd.String("output", "-", "Output file (- for stdout)")
		hitsCmd.Parse(os.Args[2:])
		exportSharpEdgeHits(*output)

	default:
		printUsage()
		os.Exit(1)
	}
}

// exportRoutingDecisions exports routing decisions from PostToolEvent logs
// CLI layer - delegates to filterAndExportRouting
func exportRoutingDecisions(format, since, output string) {
	events, err := telemetry.ReadMLToolEvents()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ml-export] Failed to read ML tool events: %v. Check telemetry data exists.\n", err)
		os.Exit(1)
	}

	var out *os.File
	if output == "-" {
		out = os.Stdout
	} else {
		var err error
		out, err = os.Create(output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ml-export] Failed to create output file %s: %v. Check directory permissions.\n", output, err)
			os.Exit(1)
		}
		defer out.Close()
	}

	count, err := filterAndExportRouting(events, format, since, out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ml-export] %v\n", err)
		os.Exit(1)
	}

	if output != "-" {
		fmt.Printf("[ml-export] Exported %d routing decisions to %s\n", count, output)
	}
}

// filterAndExportRouting filters and exports routing events (testable)
func filterAndExportRouting(events []routing.PostToolEvent, format, since string, out io.Writer) (int, error) {
	// Filter by time window
	cutoff := time.Now().Add(-parseDuration(since))
	filtered := make([]routing.PostToolEvent, 0)
	for _, event := range events {
		eventTime := time.Unix(event.CapturedAt, 0)
		if eventTime.After(cutoff) && event.SelectedTier != "" {
			filtered = append(filtered, event)
		}
	}

	switch format {
	case "csv":
		writeRoutingCSV(out, filtered)
	case "json":
		writeJSON(out, filtered)
	default:
		return 0, fmt.Errorf("[ml-export] Unknown format: %s. Supported: csv, json", format)
	}

	return len(filtered), nil
}

// writeRoutingCSV writes routing decisions as CSV with ML features
func writeRoutingCSV(out io.Writer, events []routing.PostToolEvent) {
	w := csv.NewWriter(out)
	defer w.Flush()

	// Header: ML features + action + reward
	header := []string{
		"timestamp",
		"task_type",
		"task_domain",
		"context_window",
		"recent_success_rate",
		"selected_tier",
		"selected_agent",
		"outcome_success",
		"outcome_cost",
		"escalation_required",
	}
	if err := w.Write(header); err != nil {
		fmt.Fprintf(os.Stderr, "[ml-export] Failed to write CSV header: %v\n", err)
		return
	}

	for _, event := range events {
		// Calculate estimated cost
		cost := telemetry.EstimatedCost(&event)

		// Determine if escalation was needed (check tier progression)
		escalation := event.Tier != event.SelectedTier

		row := []string{
			time.Unix(event.CapturedAt, 0).Format(time.RFC3339),
			event.TaskType,
			event.TaskDomain,
			fmt.Sprint(event.InputTokens + event.OutputTokens), // context window
			"0.0000", // Recent success rate not available in current implementation
			event.SelectedTier,
			event.SelectedAgent,
			fmt.Sprint(event.Success),
			fmt.Sprintf("%.6f", cost),
			fmt.Sprint(escalation),
		}
		if err := w.Write(row); err != nil {
			fmt.Fprintf(os.Stderr, "[ml-export] Failed to write CSV row: %v\n", err)
			return
		}
	}
}

// writeJSON writes data as formatted JSON
func writeJSON(out io.Writer, data interface{}) {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		fmt.Fprintf(os.Stderr, "[ml-export] Failed to write JSON: %v\n", err)
		return
	}
}

// ToolSequence represents a sequence of tool invocations
type ToolSequence struct {
	SequenceID string   `json:"sequence_id"`
	Tools      []string `json:"tools"`
	Successful bool     `json:"successful"`
	DurationMs int64    `json:"duration_ms"`
	TokenCount int      `json:"token_count"`
	Cost       float64  `json:"cost"`
}

// exportToolSequences exports tool sequences from PostToolEvent logs
func exportToolSequences(format string, successOnly bool, output string) {
	events, err := telemetry.ReadMLToolEvents()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ml-export] Failed to read tool events: %v. Check telemetry data exists.\n", err)
		os.Exit(1)
	}

	// Build sequences from events with SequenceIndex
	sequences := buildSequences(events)

	// Filter successful sequences if requested
	if successOnly {
		filtered := make([]ToolSequence, 0)
		for _, seq := range sequences {
			if seq.Successful {
				filtered = append(filtered, seq)
			}
		}
		sequences = filtered
	}

	var out *os.File
	if output == "-" {
		out = os.Stdout
	} else {
		var err error
		out, err = os.Create(output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ml-export] Failed to create output file %s: %v. Check directory permissions.\n", output, err)
			os.Exit(1)
		}
		defer out.Close()
	}

	switch format {
	case "json":
		writeJSON(out, sequences)
	case "csv":
		writeSequencesCSV(out, sequences)
	default:
		fmt.Fprintf(os.Stderr, "[ml-export] Unknown format: %s. Supported: json, csv\n", format)
		os.Exit(1)
	}

	if output != "-" {
		fmt.Printf("[ml-export] Exported %d tool sequences to %s\n", len(sequences), output)
	}
}

// buildSequences constructs tool sequences from PostToolEvent logs
func buildSequences(events []routing.PostToolEvent) []ToolSequence {
	// Group events by session ID
	sessionSequences := make(map[string][]routing.PostToolEvent)
	for _, event := range events {
		sessionSequences[event.SessionID] = append(sessionSequences[event.SessionID], event)
	}

	sequences := make([]ToolSequence, 0)
	for sessionID, sessionEvents := range sessionSequences {
		if len(sessionEvents) == 0 {
			continue
		}

		// Calculate sequence metrics
		tools := make([]string, 0)
		var totalDuration int64
		var totalTokens int
		var totalCost float64
		allSuccessful := true

		for _, event := range sessionEvents {
			tools = append(tools, event.ToolName)
			totalDuration += event.DurationMs
			totalTokens += event.InputTokens + event.OutputTokens
			totalCost += telemetry.EstimatedCost(&event)
			if !event.Success {
				allSuccessful = false
			}
		}

		sequences = append(sequences, ToolSequence{
			SequenceID: sessionID,
			Tools:      tools,
			Successful: allSuccessful,
			DurationMs: totalDuration,
			TokenCount: totalTokens,
			Cost:       totalCost,
		})
	}

	return sequences
}

// writeSequencesCSV writes tool sequences as CSV
func writeSequencesCSV(out io.Writer, sequences []ToolSequence) {
	w := csv.NewWriter(out)
	defer w.Flush()

	header := []string{
		"sequence_id",
		"tool_chain",
		"successful",
		"duration_ms",
		"token_count",
		"cost",
	}
	if err := w.Write(header); err != nil {
		fmt.Fprintf(os.Stderr, "[ml-export] Failed to write CSV header: %v\n", err)
		return
	}

	for _, seq := range sequences {
		row := []string{
			seq.SequenceID,
			strings.Join(seq.Tools, " → "),
			fmt.Sprint(seq.Successful),
			fmt.Sprint(seq.DurationMs),
			fmt.Sprint(seq.TokenCount),
			fmt.Sprintf("%.6f", seq.Cost),
		}
		if err := w.Write(row); err != nil {
			fmt.Fprintf(os.Stderr, "[ml-export] Failed to write CSV row: %v\n", err)
			return
		}
	}
}

// CollaborationEdge represents aggregated collaboration between two agents
type CollaborationEdge struct {
	SourceAgent      string  `json:"source_agent"`
	TargetAgent      string  `json:"target_agent"`
	InteractionCount int     `json:"interaction_count"`
	SuccessRate      float64 `json:"success_rate"`
	AvgCost          float64 `json:"avg_cost"`
}

// exportCollaborations exports agent collaboration data
func exportCollaborations(format string, output string) {
	collabs, err := telemetry.ReadCollaborationLogs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ml-export] Failed to read collaborations: %v. Check telemetry data exists.\n", err)
		os.Exit(1)
	}

	// Aggregate collaborations into edges
	edges := aggregateCollaborations(collabs)

	var out *os.File
	if output == "-" {
		out = os.Stdout
	} else {
		var err error
		out, err = os.Create(output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ml-export] Failed to create output file %s: %v. Check directory permissions.\n", output, err)
			os.Exit(1)
		}
		defer out.Close()
	}

	switch format {
	case "json":
		writeJSON(out, edges)
	case "csv":
		writeCollaborationsCSV(out, edges)
	default:
		fmt.Fprintf(os.Stderr, "[ml-export] Unknown format: %s. Supported: json, csv\n", format)
		os.Exit(1)
	}

	if output != "-" {
		fmt.Printf("[ml-export] Exported %d collaboration edges to %s\n", len(edges), output)
	}
}

// aggregateCollaborations converts individual collaboration logs into aggregated edges
func aggregateCollaborations(collabs []telemetry.AgentCollaboration) []CollaborationEdge {
	edgeMap := make(map[string]*CollaborationEdge)

	for _, c := range collabs {
		key := c.ParentAgent + " → " + c.ChildAgent
		edge, exists := edgeMap[key]

		if !exists {
			edge = &CollaborationEdge{
				SourceAgent:      c.ParentAgent,
				TargetAgent:      c.ChildAgent,
				InteractionCount: 0,
				SuccessRate:      0.0,
				AvgCost:          0.0,
			}
			edgeMap[key] = edge
		}

		edge.InteractionCount++
		if c.ChildSuccess {
			edge.SuccessRate += 1.0
		}
		// Note: Cost not available in AgentCollaboration struct currently
	}

	// Calculate final success rates
	edges := make([]CollaborationEdge, 0, len(edgeMap))
	for _, edge := range edgeMap {
		if edge.InteractionCount > 0 {
			edge.SuccessRate = edge.SuccessRate / float64(edge.InteractionCount)
		}
		edges = append(edges, *edge)
	}

	return edges
}

// writeCollaborationsCSV writes collaboration edges as CSV
func writeCollaborationsCSV(out io.Writer, edges []CollaborationEdge) {
	w := csv.NewWriter(out)
	defer w.Flush()

	header := []string{
		"source_agent",
		"target_agent",
		"interaction_count",
		"success_rate",
		"avg_cost",
	}
	if err := w.Write(header); err != nil {
		fmt.Fprintf(os.Stderr, "[ml-export] Failed to write CSV header: %v\n", err)
		return
	}

	for _, edge := range edges {
		row := []string{
			edge.SourceAgent,
			edge.TargetAgent,
			fmt.Sprint(edge.InteractionCount),
			fmt.Sprintf("%.4f", edge.SuccessRate),
			fmt.Sprintf("%.6f", edge.AvgCost),
		}
		if err := w.Write(row); err != nil {
			fmt.Fprintf(os.Stderr, "[ml-export] Failed to write CSV row: %v\n", err)
			return
		}
	}
}

// exportTrainingDataset exports complete ML training dataset to directory
func exportTrainingDataset(outputDir string) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[ml-export] Failed to create output directory %s: %v. Check permissions.\n", outputDir, err)
		os.Exit(1)
	}

	// Export routing decisions (30 days of data)
	fmt.Println("[ml-export] Exporting routing decisions...")
	exportRoutingDecisions("csv", "30d", filepath.Join(outputDir, "routing.csv"))

	// Export successful tool sequences
	fmt.Println("[ml-export] Exporting tool sequences...")
	exportToolSequences("json", true, filepath.Join(outputDir, "sequences.json"))

	// Export collaboration edges
	fmt.Println("[ml-export] Exporting collaborations...")
	exportCollaborations("json", filepath.Join(outputDir, "collaborations.json"))

	fmt.Printf("[ml-export] Training dataset complete at %s/\n", outputDir)
	fmt.Printf("[ml-export] Files: routing.csv, sequences.json, collaborations.json\n")
}

// exportReviewFindings exports all review findings
func exportReviewFindings(output string) {
	findings, err := telemetry.ReadReviewFindings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ml-export] Failed to read review findings: %v\n", err)
		os.Exit(1)
	}

	var out *os.File
	if output == "-" {
		out = os.Stdout
	} else {
		var err error
		out, err = os.Create(output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ml-export] Failed to create output file %s: %v\n", output, err)
			os.Exit(1)
		}
		defer out.Close()
	}

	writeJSON(out, findings)

	if output != "-" {
		fmt.Printf("[ml-export] Exported %d review findings to %s\n", len(findings), output)
	}
}

// exportReviewStats shows aggregate review statistics
func exportReviewStats() {
	findings, err := telemetry.ReadReviewFindings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ml-export] Failed to read review findings: %v\n", err)
		os.Exit(1)
	}

	stats := telemetry.CalculateReviewStats(findings)
	writeJSON(os.Stdout, stats)
}

// exportSharpEdgeHits exports sharp edge correlations
func exportSharpEdgeHits(output string) {
	hits, err := telemetry.ReadSharpEdgeHits()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ml-export] Failed to read sharp edge hits: %v\n", err)
		os.Exit(1)
	}

	var out *os.File
	if output == "-" {
		out = os.Stdout
	} else {
		var err error
		out, err = os.Create(output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ml-export] Failed to create output file %s: %v\n", output, err)
			os.Exit(1)
		}
		defer out.Close()
	}

	writeJSON(out, hits)

	if output != "-" {
		fmt.Printf("[ml-export] Exported %d sharp edge hits to %s\n", len(hits), output)
	}
}

// parseDuration converts time string to duration
func parseDuration(since string) time.Duration {
	switch since {
	case "1d":
		return 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	default:
		return 7 * 24 * time.Hour
	}
}

// printUsage prints CLI usage information
func printUsage() {
	fmt.Println(`Usage: gogent-ml-export <command> [options]

Commands:
  routing          Export routing decisions as CSV/JSON
  sequences        Export tool sequences as JSON/CSV
  collaborations   Export agent collaboration data as JSON/CSV
  training-dataset Export complete ML training dataset to directory
  review-findings  Export all review findings as JSON
  review-stats     Show aggregate review statistics
  sharp-edge-hits  Export sharp edge correlations as JSON

Routing options:
  --format string      Output format: csv, json (default "csv")
  --since string       Time filter: 1d, 7d, 30d (default "7d")
  --output string      Output file (default "-" for stdout)

Sequences options:
  --format string      Output format: json, csv (default "json")
  --successful-only    Only export successful sequences (default false)
  --output string      Output file (default "-" for stdout)

Collaborations options:
  --format string      Output format: json, csv (default "json")
  --output string      Output file (default "-" for stdout)

Training-dataset options:
  --output string      Output directory (default "ml-data/")

Review-findings options:
  --output string      Output file (default "-" for stdout)

Sharp-edge-hits options:
  --output string      Output file (default "-" for stdout)

Examples:
  gogent-ml-export routing --format csv --since 7d --output routing.csv
  gogent-ml-export sequences --successful-only --output sequences.json
  gogent-ml-export collaborations --format json
  gogent-ml-export training-dataset --output ./ml-data/
  gogent-ml-export review-findings --output findings.json
  gogent-ml-export review-stats
  gogent-ml-export sharp-edge-hits --output hits.json`)
}
