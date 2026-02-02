package telemetry

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
	"github.com/google/uuid"
)

// SharpEdgeHit tracks when a reviewer catches a known sharp edge pattern
type SharpEdgeHit struct {
	HitID           string  `json:"hit_id"`
	Timestamp       int64   `json:"timestamp"`
	SessionID       string  `json:"session_id"`
	SharpEdgeID     string  `json:"sharp_edge_id"`     // From sharp-edges.yaml (validated)
	AgentID         string  `json:"agent_id"`          // Which agent owns the sharp edge
	ReviewerID      string  `json:"reviewer_id"`       // Which reviewer caught it
	FindingID       string  `json:"finding_id"`        // Links to ReviewFinding
	File            string  `json:"file"`
	Line            int     `json:"line,omitempty"`
	MatchConfidence float64 `json:"match_confidence"`  // 0.0-1.0
	WasActioned     bool    `json:"was_actioned"`      // Did user fix it
}

// NewSharpEdgeHit creates a new hit record
// Returns error if sharpEdgeID is not in the registry
func NewSharpEdgeHit(sessionID, sharpEdgeID, agentID, reviewerID, findingID, file string, line int) (*SharpEdgeHit, error) {
	// Validate sharp edge ID against registry
	if !IsValidSharpEdgeID(sharpEdgeID) {
		return nil, fmt.Errorf("invalid sharp_edge_id: %s", sharpEdgeID)
	}

	return &SharpEdgeHit{
		HitID:           uuid.New().String(),
		Timestamp:       time.Now().Unix(),
		SessionID:       sessionID,
		SharpEdgeID:     sharpEdgeID,
		AgentID:         agentID,
		ReviewerID:      reviewerID,
		FindingID:       findingID,
		File:            file,
		Line:            line,
		MatchConfidence: 1.0, // Default to exact match; can be overridden
	}, nil
}

// LogSharpEdgeHit writes hit to JSONL storage (concurrency-safe)
func LogSharpEdgeHit(hit *SharpEdgeHit) error {
	path := config.GetSharpEdgeHitsPathWithProjectDir()

	data, err := json.Marshal(hit)
	if err != nil {
		return fmt.Errorf("[sharp-edge-hit] marshal: %w", err)
	}

	return AppendJSONL(path, data)
}
