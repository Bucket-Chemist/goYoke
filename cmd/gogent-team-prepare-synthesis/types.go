package main

// EinsteinOutput represents the full stdout schema from Einstein.
// Source: .claude/schemas/teams/stdin-stdout/braintrust-einstein.json
type EinsteinOutput struct {
	AnalysisID              string                   `json:"analysis_id"`
	Status                  string                   `json:"status"`
	ExecutiveSummary        string                   `json:"executive_summary"`
	RootCauseAnalysis       *RootCauseAnalysis       `json:"root_cause_analysis"`
	ConceptualFramework     *ConceptualFramework     `json:"conceptual_framework"`
	FirstPrinciplesAnalysis *FirstPrinciplesAnalysis `json:"first_principles_analysis"`
	NovelApproaches         []NovelApproach          `json:"novel_approaches"`
	TheoreticalTradeoffs    []TheoreticalTradeoff    `json:"theoretical_tradeoffs"`
	AssumptionsSurfaced     []AssumptionSurfaced     `json:"assumptions_surfaced"`
	OpenQuestions           []OpenQuestion           `json:"open_questions"`
	HandoffNotes            string                   `json:"handoff_notes"`
}

type RootCauseAnalysis struct {
	IdentifiedCauses []IdentifiedCause `json:"identified_causes"`
}

type IdentifiedCause struct {
	Cause         string   `json:"cause"`
	Evidence      string   `json:"evidence"`
	Confidence    string   `json:"confidence"`
	AffectedScope []string `json:"affected_scope"`
}

type ConceptualFramework struct {
	FrameworkName string   `json:"framework_name"`
	Description   string   `json:"description"`
	KeyInsights   []string `json:"key_insights"`
}

type FirstPrinciplesAnalysis struct {
	AssumptionsChallenged  []AssumptionChallenged `json:"assumptions_challenged"`
	FundamentalConstraints []string               `json:"fundamental_constraints"`
}

type AssumptionChallenged struct {
	Assumption         string `json:"assumption"`
	Validity           string `json:"validity"`
	Evidence           string `json:"evidence"`
	ImplicationIfWrong string `json:"implication_if_wrong"`
}

type NovelApproach struct {
	Approach             string    `json:"approach"`
	Rationale            string    `json:"rationale"`
	Tradeoffs            Tradeoffs `json:"tradeoffs"`
	Feasibility          string    `json:"feasibility"`
	ImplementationSketch string    `json:"implementation_sketch"`
}

type Tradeoffs struct {
	Pros  []string `json:"pros"`
	Cons  []string `json:"cons"`
	Risks []string `json:"risks"`
}

type TheoreticalTradeoff struct {
	Dimension      string `json:"dimension"`
	OptionA        string `json:"option_a"`
	OptionB        string `json:"option_b"`
	Recommendation string `json:"recommendation"`
}

type AssumptionSurfaced struct {
	Assumption       string `json:"assumption"`
	Source           string `json:"source"`
	RiskIfFalse      string `json:"risk_if_false"`
	ValidationMethod string `json:"validation_method"`
}

type OpenQuestion struct {
	Question               string `json:"question"`
	Importance             string `json:"importance"`
	SuggestedInvestigation string `json:"suggested_investigation"`
}

// StaffArchOutput represents the full stdout schema from Staff-Architect.
// Source: .claude/schemas/teams/stdin-stdout/braintrust-staff-architect.json
type StaffArchOutput struct {
	ReviewID            string               `json:"review_id"`
	Status              string               `json:"status"`
	ExecutiveAssessment *ExecutiveAssessment `json:"executive_assessment"`
	IssueRegister       []Issue              `json:"issue_register"`
	AssumptionRegister  []AssumptionReg      `json:"assumption_register"`
	DependencyAnalysis  *DependencyAnalysis  `json:"dependency_analysis"`
	FailureModeAnalysis []FailureMode        `json:"failure_mode_analysis"`
	Commendations       []string             `json:"commendations"`
	Recommendations     *Recommendations     `json:"recommendations"`
	SignOff             *SignOff             `json:"sign_off"`
	HandoffNotes        string               `json:"handoff_notes"`
}

type ExecutiveAssessment struct {
	Verdict     string      `json:"verdict"`
	Confidence  string      `json:"confidence"`
	Summary     string      `json:"summary"`
	IssueCounts IssueCounts `json:"issue_counts"`
}

type IssueCounts struct {
	Critical int `json:"critical"`
	Major    int `json:"major"`
	Minor    int `json:"minor"`
}

type Issue struct {
	ID              string   `json:"id"`
	Severity        string   `json:"severity"`
	Layer           string   `json:"layer"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Evidence        string   `json:"evidence"`
	Impact          string   `json:"impact"`
	Recommendation  string   `json:"recommendation"`
	AffectedTickets []string `json:"affected_tickets"`
	AffectedFiles   []string `json:"affected_files"`
}

type AssumptionReg struct {
	ID                 string `json:"id"`
	Assumption         string `json:"assumption"`
	Source             string `json:"source"`
	Verified           bool   `json:"verified"`
	RiskIfFalse        string `json:"risk_if_false"`
	Mitigation         string `json:"mitigation"`
	VerificationMethod string `json:"verification_method"`
}

type DependencyAnalysis struct {
	CriticalPath  []string `json:"critical_path"`
	Bottlenecks   []string `json:"bottlenecks"`
	CircularRisks []string `json:"circular_risks"`
}

type FailureMode struct {
	Scenario    string `json:"scenario"`
	Probability string `json:"probability"`
	Impact      string `json:"impact"`
	Detection   string `json:"detection"`
	Mitigation  string `json:"mitigation"`
}

type Recommendations struct {
	HighPriority   []Recommendation `json:"high_priority"`
	MediumPriority []Recommendation `json:"medium_priority"`
	LowPriority    []Recommendation `json:"low_priority"`
}

type Recommendation struct {
	Action    string `json:"action"`
	Rationale string `json:"rationale"`
	Effort    string `json:"effort"`
}

type SignOff struct {
	Conditions             []string `json:"conditions"`
	PostApprovalMonitoring []string `json:"post_approval_monitoring"`
}

// EinsteinSections holds extracted sections for markdown generation.
type EinsteinSections struct {
	Status                 string
	ExecutiveSummary       string
	RootCauses             []string
	Frameworks             []string
	FirstPrinciples        []string   // NEW: challenged assumptions + fundamental constraints
	NovelApproaches        []string
	TheoreticalTradeoffs   []string   // NEW
	AssumptionsSurfaced    []string   // NEW
	OpenQuestions          []string
	HandoffNotes           string
}

// StaffArchSections holds extracted sections for markdown generation.
type StaffArchSections struct {
	Status            string
	ExecutiveVerdict  string
	CriticalIssues    []string
	MajorIssues       []string
	MinorIssues       []string
	Commendations     []string
	FailureModes      []string
	Recommendations   []string
	SignOffConditions []string
	HandoffNotes      string
}
