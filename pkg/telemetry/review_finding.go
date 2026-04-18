package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/google/uuid"
)

// ReviewFinding captures a single code review finding for ML analysis
type ReviewFinding struct {
	FindingID      string `json:"finding_id"`
	Timestamp      int64  `json:"timestamp"`
	SessionID      string `json:"session_id"`
	ReviewScope    string `json:"review_scope"`
	FilesReviewed  int    `json:"files_reviewed"`
	Severity       string `json:"severity"`
	Reviewer       string `json:"reviewer"`
	Category       string `json:"category"`
	File           string `json:"file"`
	Line           int    `json:"line,omitempty"`
	Message        string `json:"message"`
	Recommendation string `json:"recommendation,omitempty"`
	SharpEdgeID    string `json:"sharp_edge_id,omitempty"`
	WasFixed       bool   `json:"was_fixed,omitempty"`
	FixCommit      string `json:"fix_commit,omitempty"`
}

// ReviewOutcomeUpdate represents an outcome update (append-only)
type ReviewOutcomeUpdate struct {
	FindingID       string `json:"finding_id"`
	Resolution      string `json:"resolution"`
	ResolutionMs    int64  `json:"resolution_ms"`
	TicketID        string `json:"ticket_id,omitempty"`
	CommitHash      string `json:"commit_hash,omitempty"`
	UpdateTimestamp int64  `json:"update_timestamp"`
}

// NewReviewFinding creates a new finding record
func NewReviewFinding(sessionID, reviewer, severity, category, file string, line int, message string) *ReviewFinding {
	return &ReviewFinding{
		FindingID: uuid.New().String(),
		Timestamp: time.Now().Unix(),
		SessionID: sessionID,
		Reviewer:  reviewer,
		Severity:  severity,
		Category:  category,
		File:      file,
		Line:      line,
		Message:   truncateMessage(message, 1000),
	}
}

// LogReviewFinding writes finding to JSONL storage (concurrency-safe)
func LogReviewFinding(finding *ReviewFinding) error {
	path := config.GetReviewFindingsPathWithProjectDir()
	data, err := json.Marshal(finding)
	if err != nil {
		return fmt.Errorf("[review-finding] marshal: %w", err)
	}
	return AppendJSONL(path, data)
}

// UpdateReviewFindingOutcome appends outcome update (concurrency-safe)
func UpdateReviewFindingOutcome(findingID, resolution, ticketID, commitHash string, resolutionMs int64) error {
	update := ReviewOutcomeUpdate{
		FindingID:       findingID,
		Resolution:      resolution,
		ResolutionMs:    resolutionMs,
		TicketID:        ticketID,
		CommitHash:      commitHash,
		UpdateTimestamp: time.Now().Unix(),
	}
	path := config.GetReviewOutcomesPathWithProjectDir()
	data, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("[review-outcome] marshal: %w", err)
	}
	return AppendJSONL(path, data)
}

// LookupFindingTimestamp retrieves the original timestamp for a finding
// Used to calculate accurate resolution time
func LookupFindingTimestamp(findingID string) (int64, error) {
	findings, err := ReadReviewFindings()
	if err != nil {
		return 0, err
	}
	for _, f := range findings {
		if f.FindingID == findingID {
			return f.Timestamp, nil
		}
	}
	return 0, fmt.Errorf("finding not found: %s", findingID)
}

// ReadReviewFindings reads all findings from storage
func ReadReviewFindings() ([]ReviewFinding, error) {
	path := config.GetReviewFindingsPathWithProjectDir()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []ReviewFinding{}, nil
		}
		return nil, fmt.Errorf("[review-finding] read: %w", err)
	}

	var findings []ReviewFinding
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var f ReviewFinding
		if err := json.Unmarshal([]byte(line), &f); err != nil {
			continue // Skip malformed lines
		}
		findings = append(findings, f)
	}
	return findings, nil
}

// CalculateReviewStats returns aggregate metrics
func CalculateReviewStats(findings []ReviewFinding) map[string]interface{} {
	bySeverity := make(map[string]int)
	byReviewer := make(map[string]int)
	byCategory := make(map[string]int)

	for _, f := range findings {
		bySeverity[f.Severity]++
		byReviewer[f.Reviewer]++
		byCategory[f.Category]++
	}

	return map[string]interface{}{
		"total_findings": len(findings),
		"by_severity":    bySeverity,
		"by_reviewer":    byReviewer,
		"by_category":    byCategory,
	}
}

func truncateMessage(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
