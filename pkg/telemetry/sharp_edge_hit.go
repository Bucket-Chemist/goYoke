package telemetry

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
	"github.com/google/uuid"
)

// SharpEdgeHit tracks when a reviewer catches a known sharp edge pattern
type SharpEdgeHit struct {
	HitID           string  `json:"hit_id"`
	Timestamp       int64   `json:"timestamp"`
	SessionID       string  `json:"session_id"`
	SharpEdgeID     string  `json:"sharp_edge_id"`
	AgentID         string  `json:"agent_id"`
	ReviewerID      string  `json:"reviewer_id"`
	FindingID       string  `json:"finding_id"`
	File            string  `json:"file"`
	Line            int     `json:"line,omitempty"`
	MatchConfidence float64 `json:"match_confidence"`
	WasActioned     bool    `json:"was_actioned"`
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
		MatchConfidence: 1.0, // Default to exact match
	}, nil
}

// LogSharpEdgeHit writes hit to JSONL storage
func LogSharpEdgeHit(hit *SharpEdgeHit) error {
	path := config.GetSharpEdgeHitsPathWithProjectDir()
	data, err := json.Marshal(hit)
	if err != nil {
		return fmt.Errorf("[sharp-edge-hit] marshal: %w", err)
	}
	return AppendJSONL(path, data)
}

// ReadSharpEdgeHits reads all hits from storage
func ReadSharpEdgeHits() ([]SharpEdgeHit, error) {
	path := config.GetSharpEdgeHitsPathWithProjectDir()
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []SharpEdgeHit{}, nil
		}
		return nil, fmt.Errorf("[sharp-edge-hit] open: %w", err)
	}
	defer file.Close()

	var hits []SharpEdgeHit
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var h SharpEdgeHit
		if err := json.Unmarshal(scanner.Bytes(), &h); err != nil {
			continue // Skip malformed lines
		}
		hits = append(hits, h)
	}
	return hits, scanner.Err()
}
