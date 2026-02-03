package main

import "encoding/json"

// ScoutReport is the unified output structure for scout operations.
type ScoutReport struct {
	SchemaVersion string `json:"schema_version"`

	Backend   string `json:"backend"` // "native" or "synthetic_fallback"
	Target    string `json:"target"`
	Timestamp string `json:"timestamp"` // RFC3339 format

	ScopeMetrics          *ScopeMetrics          `json:"scope_metrics"`
	ComplexitySignals     *ComplexitySignals     `json:"complexity_signals"`
	RoutingRecommendation *RoutingRecommendation `json:"routing_recommendation"`
	KeyFiles              []KeyFile              `json:"key_files"`
	Warnings              []string               `json:"warnings"`
}

// ScopeMetrics captures basic scope measurements.
type ScopeMetrics struct {
	TotalFiles        int            `json:"total_files"`
	TotalLines        int            `json:"total_lines"`
	EstimatedTokens   int            `json:"estimated_tokens"`
	Languages         []string       `json:"languages"`
	FileTypes         map[string]int `json:"file_types"` // extension -> count
	MaxFileLines      int            `json:"max_file_lines"`
	FilesOver500Lines int            `json:"files_over_500_lines"`
}

// ComplexitySignals provides semantic analysis.
type ComplexitySignals struct {
	Available             bool    `json:"available"`
	ImportDensity         *string `json:"import_density,omitempty"`          // "low", "medium", "high"
	CrossFileDependencies *int    `json:"cross_file_dependencies,omitempty"` // Number of cross-file deps
	TestCoveragePresent   bool    `json:"test_coverage_present"`
	Note                  string  `json:"note,omitempty"`
}

// RoutingRecommendation suggests tier based on scope.
type RoutingRecommendation struct {
	RecommendedTier     string  `json:"recommended_tier"` // "haiku", "sonnet", "external"
	Confidence          string  `json:"confidence"`       // "high", "medium", "low"
	Reasoning           string  `json:"reasoning"`
	ClarificationNeeded *string `json:"clarification_needed,omitempty"`
}

// KeyFile identifies notable files (by size or complexity).
type KeyFile struct {
	Path      string `json:"path"`
	Lines     int    `json:"lines"`
	Relevance string `json:"relevance"`
}

// FileInfo holds metadata about a single file during scanning.
type FileInfo struct {
	Path     string
	Lines    int
	Language string
	IsTest   bool
}

// MarshalJSON ensures consistent JSON output.
func (sr *ScoutReport) MarshalJSON() ([]byte, error) {
	type Alias ScoutReport
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(sr),
	})
}
