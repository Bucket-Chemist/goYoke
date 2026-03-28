// Package teamconfig defines the shared configuration types used by both
// cmd/gogent-team-run (the team runner binary) and the TUI's team display
// components. Extracting these types into a non-main package eliminates the
// type duplication that previously existed between the two and ensures changes
// to the config.json schema produce compile-time errors in both consumers.
package teamconfig

// TeamConfig is the parsed representation of a team's config.json file.
type TeamConfig struct {
	TeamName           string  `json:"team_name"`
	WorkflowType       string  `json:"workflow_type"`
	Status             string  `json:"status"`         // pending|running|completed|failed
	CreatedAt          string  `json:"created_at"`     // ISO 8601
	CompletedAt        *string `json:"completed_at"`   // ISO 8601, nil while running
	BudgetMaxUSD       float64 `json:"budget_max_usd"`
	BudgetRemainingUSD float64 `json:"budget_remaining_usd"`
	Waves              []Wave  `json:"waves"`
}

// Wave is a single execution wave within a team run.
type Wave struct {
	WaveNumber  int      `json:"wave_number"`
	Description string   `json:"description"`
	Members     []Member `json:"members"`
}

// Member is a single agent task within a wave.
type Member struct {
	Name         string  `json:"name"`
	Agent        string  `json:"agent"`
	Model        string  `json:"model"`
	Status       string  `json:"status"`       // pending|running|completed|failed
	CostUSD      float64 `json:"cost_usd"`
	ErrorMessage string  `json:"error_message"`
	StartedAt    *string `json:"started_at"`
	CompletedAt  *string `json:"completed_at"`
}
