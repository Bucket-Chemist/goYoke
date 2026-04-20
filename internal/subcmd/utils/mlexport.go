package utils

import (
	"context"
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

// RunMLExport implements the goyoke-ml-export utility.
// args[0] is the subcommand (routing, sequences, collaborations, etc.).
func RunMLExport(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) < 1 {
		mlexportPrintUsage(stdout)
		return fmt.Errorf("ml-export: no command specified")
	}

	switch args[0] {
	case "routing":
		fs := flag.NewFlagSet("routing", flag.ContinueOnError)
		format := fs.String("format", "csv", "Output format: csv, json")
		since := fs.String("since", "7d", "Time filter: 1d, 7d, 30d")
		output := fs.String("output", "-", "Output file (- for stdout)")
		if err := fs.Parse(args[1:]); err != nil {
			return fmt.Errorf("ml-export routing: %w", err)
		}
		return mlexportRoutingDecisions(*format, *since, *output, stdout)

	case "sequences":
		fs := flag.NewFlagSet("sequences", flag.ContinueOnError)
		format := fs.String("format", "json", "Output format: json, csv")
		successOnly := fs.Bool("successful-only", false, "Only successful sequences")
		output := fs.String("output", "-", "Output file (- for stdout)")
		if err := fs.Parse(args[1:]); err != nil {
			return fmt.Errorf("ml-export sequences: %w", err)
		}
		return mlexportToolSequences(*format, *successOnly, *output, stdout)

	case "collaborations":
		fs := flag.NewFlagSet("collaborations", flag.ContinueOnError)
		format := fs.String("format", "json", "Output format: json, csv")
		output := fs.String("output", "-", "Output file (- for stdout)")
		if err := fs.Parse(args[1:]); err != nil {
			return fmt.Errorf("ml-export collaborations: %w", err)
		}
		return mlexportCollaborations(*format, *output, stdout)

	case "training-dataset":
		fs := flag.NewFlagSet("training-dataset", flag.ContinueOnError)
		outDir := fs.String("output", "ml-data/", "Output directory")
		if err := fs.Parse(args[1:]); err != nil {
			return fmt.Errorf("ml-export training-dataset: %w", err)
		}
		return mlexportTrainingDataset(*outDir, stdout)

	case "review-findings":
		fs := flag.NewFlagSet("review-findings", flag.ContinueOnError)
		output := fs.String("output", "-", "Output file (- for stdout)")
		if err := fs.Parse(args[1:]); err != nil {
			return fmt.Errorf("ml-export review-findings: %w", err)
		}
		return mlexportReviewFindings(*output, stdout)

	case "review-stats":
		fs := flag.NewFlagSet("review-stats", flag.ContinueOnError)
		if err := fs.Parse(args[1:]); err != nil {
			return fmt.Errorf("ml-export review-stats: %w", err)
		}
		return mlexportReviewStats(stdout)

	case "sharp-edge-hits":
		fs := flag.NewFlagSet("sharp-edge-hits", flag.ContinueOnError)
		output := fs.String("output", "-", "Output file (- for stdout)")
		if err := fs.Parse(args[1:]); err != nil {
			return fmt.Errorf("ml-export sharp-edge-hits: %w", err)
		}
		return mlexportSharpEdgeHits(*output, stdout)

	default:
		mlexportPrintUsage(stdout)
		return fmt.Errorf("ml-export: unknown command %q", args[0])
	}
}

func mlexportRoutingDecisions(format, since, output string, defaultOut io.Writer) error {
	events, err := telemetry.ReadMLToolEvents()
	if err != nil {
		return fmt.Errorf("[ml-export] failed to read ML tool events: %w", err)
	}

	out, closer, err := mlexportOpenOutput(output, defaultOut)
	if err != nil {
		return err
	}
	defer closer()

	count, err := mlexportFilterAndExportRouting(events, format, since, out)
	if err != nil {
		return fmt.Errorf("[ml-export] %w", err)
	}

	if output != "-" {
		fmt.Fprintf(defaultOut, "[ml-export] Exported %d routing decisions to %s\n", count, output)
	}
	return nil
}

func mlexportFilterAndExportRouting(events []routing.PostToolEvent, format, since string, out io.Writer) (int, error) {
	cutoff := time.Now().Add(-mlexportParseDuration(since))
	filtered := make([]routing.PostToolEvent, 0)
	for _, event := range events {
		eventTime := time.Unix(event.CapturedAt, 0)
		if eventTime.After(cutoff) && event.SelectedTier != "" {
			filtered = append(filtered, event)
		}
	}

	switch format {
	case "csv":
		mlexportWriteRoutingCSV(out, filtered)
	case "json":
		mlexportWriteJSON(out, filtered)
	default:
		return 0, fmt.Errorf("unknown format: %s. Supported: csv, json", format)
	}
	return len(filtered), nil
}

func mlexportWriteRoutingCSV(out io.Writer, events []routing.PostToolEvent) {
	w := csv.NewWriter(out)
	defer w.Flush()

	header := []string{
		"timestamp", "task_type", "task_domain", "context_window",
		"recent_success_rate", "selected_tier", "selected_agent",
		"outcome_success", "outcome_cost", "escalation_required",
	}
	if err := w.Write(header); err != nil {
		return
	}

	for _, event := range events {
		cost := telemetry.EstimatedCost(&event)
		escalation := event.Tier != event.SelectedTier
		row := []string{
			time.Unix(event.CapturedAt, 0).Format(time.RFC3339),
			event.TaskType, event.TaskDomain,
			fmt.Sprint(event.InputTokens + event.OutputTokens),
			"0.0000",
			event.SelectedTier, event.SelectedAgent,
			fmt.Sprint(event.Success),
			fmt.Sprintf("%.6f", cost),
			fmt.Sprint(escalation),
		}
		_ = w.Write(row)
	}
}

func mlexportWriteJSON(out io.Writer, data any) {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(data)
}

type mlexportToolSequence struct {
	SequenceID string   `json:"sequence_id"`
	Tools      []string `json:"tools"`
	Successful bool     `json:"successful"`
	DurationMs int64    `json:"duration_ms"`
	TokenCount int      `json:"token_count"`
	Cost       float64  `json:"cost"`
}

func mlexportToolSequences(format string, successOnly bool, output string, defaultOut io.Writer) error {
	events, err := telemetry.ReadMLToolEvents()
	if err != nil {
		return fmt.Errorf("[ml-export] failed to read tool events: %w", err)
	}

	sequences := mlexportBuildSequences(events)
	if successOnly {
		filtered := make([]mlexportToolSequence, 0)
		for _, seq := range sequences {
			if seq.Successful {
				filtered = append(filtered, seq)
			}
		}
		sequences = filtered
	}

	out, closer, err := mlexportOpenOutput(output, defaultOut)
	if err != nil {
		return err
	}
	defer closer()

	switch format {
	case "json":
		mlexportWriteJSON(out, sequences)
	case "csv":
		mlexportWriteSequencesCSV(out, sequences)
	default:
		return fmt.Errorf("[ml-export] unknown format: %s. Supported: json, csv", format)
	}

	if output != "-" {
		fmt.Fprintf(defaultOut, "[ml-export] Exported %d tool sequences to %s\n", len(sequences), output)
	}
	return nil
}

func mlexportBuildSequences(events []routing.PostToolEvent) []mlexportToolSequence {
	sessionSequences := make(map[string][]routing.PostToolEvent)
	for _, event := range events {
		sessionSequences[event.SessionID] = append(sessionSequences[event.SessionID], event)
	}

	sequences := make([]mlexportToolSequence, 0)
	for sessionID, sessionEvents := range sessionSequences {
		if len(sessionEvents) == 0 {
			continue
		}
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

		sequences = append(sequences, mlexportToolSequence{
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

func mlexportWriteSequencesCSV(out io.Writer, sequences []mlexportToolSequence) {
	w := csv.NewWriter(out)
	defer w.Flush()

	header := []string{"sequence_id", "tool_chain", "successful", "duration_ms", "token_count", "cost"}
	_ = w.Write(header)

	for _, seq := range sequences {
		row := []string{
			seq.SequenceID,
			strings.Join(seq.Tools, " → "),
			fmt.Sprint(seq.Successful),
			fmt.Sprint(seq.DurationMs),
			fmt.Sprint(seq.TokenCount),
			fmt.Sprintf("%.6f", seq.Cost),
		}
		_ = w.Write(row)
	}
}

type mlexportCollaborationEdge struct {
	SourceAgent      string  `json:"source_agent"`
	TargetAgent      string  `json:"target_agent"`
	InteractionCount int     `json:"interaction_count"`
	SuccessRate      float64 `json:"success_rate"`
	AvgCost          float64 `json:"avg_cost"`
}

func mlexportCollaborations(format, output string, defaultOut io.Writer) error {
	collabs, err := telemetry.ReadCollaborationLogs()
	if err != nil {
		return fmt.Errorf("[ml-export] failed to read collaborations: %w", err)
	}

	edges := mlexportAggregateCollaborations(collabs)

	out, closer, err := mlexportOpenOutput(output, defaultOut)
	if err != nil {
		return err
	}
	defer closer()

	switch format {
	case "json":
		mlexportWriteJSON(out, edges)
	case "csv":
		mlexportWriteCollaborationsCSV(out, edges)
	default:
		return fmt.Errorf("[ml-export] unknown format: %s. Supported: json, csv", format)
	}

	if output != "-" {
		fmt.Fprintf(defaultOut, "[ml-export] Exported %d collaboration edges to %s\n", len(edges), output)
	}
	return nil
}

func mlexportAggregateCollaborations(collabs []telemetry.AgentCollaboration) []mlexportCollaborationEdge {
	edgeMap := make(map[string]*mlexportCollaborationEdge)
	for _, c := range collabs {
		key := c.ParentAgent + " → " + c.ChildAgent
		edge, exists := edgeMap[key]
		if !exists {
			edge = &mlexportCollaborationEdge{SourceAgent: c.ParentAgent, TargetAgent: c.ChildAgent}
			edgeMap[key] = edge
		}
		edge.InteractionCount++
		if c.ChildSuccess {
			edge.SuccessRate += 1.0
		}
	}

	edges := make([]mlexportCollaborationEdge, 0, len(edgeMap))
	for _, edge := range edgeMap {
		if edge.InteractionCount > 0 {
			edge.SuccessRate = edge.SuccessRate / float64(edge.InteractionCount)
		}
		edges = append(edges, *edge)
	}
	return edges
}

func mlexportWriteCollaborationsCSV(out io.Writer, edges []mlexportCollaborationEdge) {
	w := csv.NewWriter(out)
	defer w.Flush()

	header := []string{"source_agent", "target_agent", "interaction_count", "success_rate", "avg_cost"}
	_ = w.Write(header)

	for _, edge := range edges {
		row := []string{
			edge.SourceAgent, edge.TargetAgent,
			fmt.Sprint(edge.InteractionCount),
			fmt.Sprintf("%.4f", edge.SuccessRate),
			fmt.Sprintf("%.6f", edge.AvgCost),
		}
		_ = w.Write(row)
	}
}

func mlexportTrainingDataset(outputDir string, stdout io.Writer) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("[ml-export] failed to create output directory %s: %w", outputDir, err)
	}

	fmt.Fprintln(stdout, "[ml-export] Exporting routing decisions...")
	if err := mlexportRoutingDecisions("csv", "30d", filepath.Join(outputDir, "routing.csv"), stdout); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "[ml-export] Exporting tool sequences...")
	if err := mlexportToolSequences("json", true, filepath.Join(outputDir, "sequences.json"), stdout); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "[ml-export] Exporting collaborations...")
	if err := mlexportCollaborations("json", filepath.Join(outputDir, "collaborations.json"), stdout); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "[ml-export] Training dataset complete at %s/\n", outputDir)
	fmt.Fprintf(stdout, "[ml-export] Files: routing.csv, sequences.json, collaborations.json\n")
	return nil
}

func mlexportReviewFindings(output string, defaultOut io.Writer) error {
	findings, err := telemetry.ReadReviewFindings()
	if err != nil {
		return fmt.Errorf("[ml-export] failed to read review findings: %w", err)
	}

	out, closer, err := mlexportOpenOutput(output, defaultOut)
	if err != nil {
		return err
	}
	defer closer()

	mlexportWriteJSON(out, findings)

	if output != "-" {
		fmt.Fprintf(defaultOut, "[ml-export] Exported %d review findings to %s\n", len(findings), output)
	}
	return nil
}

func mlexportReviewStats(stdout io.Writer) error {
	findings, err := telemetry.ReadReviewFindings()
	if err != nil {
		return fmt.Errorf("[ml-export] failed to read review findings: %w", err)
	}
	stats := telemetry.CalculateReviewStats(findings)
	mlexportWriteJSON(stdout, stats)
	return nil
}

func mlexportSharpEdgeHits(output string, defaultOut io.Writer) error {
	hits, err := telemetry.ReadSharpEdgeHits()
	if err != nil {
		return fmt.Errorf("[ml-export] failed to read sharp edge hits: %w", err)
	}

	out, closer, err := mlexportOpenOutput(output, defaultOut)
	if err != nil {
		return err
	}
	defer closer()

	mlexportWriteJSON(out, hits)

	if output != "-" {
		fmt.Fprintf(defaultOut, "[ml-export] Exported %d sharp edge hits to %s\n", len(hits), output)
	}
	return nil
}

func mlexportOpenOutput(output string, defaultOut io.Writer) (io.Writer, func(), error) {
	if output == "-" {
		return defaultOut, func() {}, nil
	}
	f, err := os.Create(output)
	if err != nil {
		return nil, nil, fmt.Errorf("[ml-export] failed to create output file %s: %w", output, err)
	}
	return f, func() { f.Close() }, nil
}

func mlexportParseDuration(since string) time.Duration {
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

func mlexportPrintUsage(w io.Writer) {
	fmt.Fprintln(w, `Usage: goyoke ml-export <command> [options]

Commands:
  routing          Export routing decisions as CSV/JSON
  sequences        Export tool sequences as JSON/CSV
  collaborations   Export agent collaboration data as JSON/CSV
  training-dataset Export complete ML training dataset to directory
  review-findings  Export all review findings as JSON
  review-stats     Show aggregate review statistics
  sharp-edge-hits  Export sharp edge correlations as JSON`)
}
