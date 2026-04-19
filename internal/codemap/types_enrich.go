package codemap

// EnrichedSymbol is the LLM enrichment delta for a single symbol, matched by Name.
type EnrichedSymbol struct {
	Name              string   `json:"name"`
	Description       string   `json:"description,omitempty"`
	ModuleIdentity    string   `json:"module_identity,omitempty"`
	Complexity        string   `json:"complexity,omitempty"`
	IsEntrypoint      bool     `json:"is_entrypoint"`
	ArchitecturalRole string   `json:"architectural_role,omitempty"`
	Tags              []string `json:"tags,omitempty"`
}

// enrichmentResponse is the JSON structure expected from the LLM per module.
type enrichmentResponse struct {
	ModuleDescription string           `json:"module_description"`
	ModuleCategory    string           `json:"module_category"`
	KeyTypes          []string         `json:"key_types"`
	KeyFunctions      []string         `json:"key_functions"`
	Symbols           []EnrichedSymbol `json:"symbols"`
}

// MergedSymbol combines base Symbol fields with LLM enrichment fields for the final JSON output.
type MergedSymbol struct {
	Name        string      `json:"name"`
	Kind        string      `json:"kind"`
	Signature   string      `json:"signature"`
	Params      []Param     `json:"params"`
	Returns     []ReturnVal `json:"returns"`
	Receiver    *string     `json:"receiver"`
	LineStart   int         `json:"line_start"`
	LineEnd     int         `json:"line_end"`
	Exported    bool        `json:"exported"`
	Decorators  []string    `json:"decorators"`
	Calls       []string    `json:"calls"`
	CalledBy    []string    `json:"called_by"`
	// LLM enrichment (omitempty keeps unenriched symbols clean)
	Description       string   `json:"description,omitempty"`
	ModuleIdentity    string   `json:"module_identity,omitempty"`
	Complexity        string   `json:"complexity,omitempty"`
	IsEntrypoint      bool     `json:"is_entrypoint,omitempty"`
	ArchitecturalRole string   `json:"architectural_role,omitempty"`
	Tags              []string `json:"tags,omitempty"`
}

// EnrichedFileExtract is a FileExtract with merged (base + enrichment) symbols.
type EnrichedFileExtract struct {
	Path        string         `json:"path"`
	LineCount   int            `json:"line_count"`
	Symbols     []MergedSymbol `json:"symbols"`
	ErrorCount  int            `json:"error_count"`
	ParseErrors []ParseError   `json:"parse_errors"`
}

// EnrichedModule is the full enriched extraction output.
// It mirrors ModuleExtraction but with EnrichedFileExtract and module-level LLM fields.
type EnrichedModule struct {
	Module           string               `json:"module"`
	Language         string               `json:"language"`
	Files            []EnrichedFileExtract `json:"files"`
	Imports          ImportGraph          `json:"imports"`
	ExtractedAt      string               `json:"extracted_at"`
	ExtractorVersion string               `json:"extractor_version"`
	// LLM enrichment fields
	ModuleDescription string   `json:"module_description,omitempty"`
	ModuleCategory    string   `json:"module_category,omitempty"`
	KeyTypes          []string `json:"key_types,omitempty"`
	KeyFunctions      []string `json:"key_functions,omitempty"`
	EnrichedAt        string   `json:"enriched_at,omitempty"`
	EnricherModel     string   `json:"enricher_model,omitempty"`
}

// EnrichOpts controls LLM enrichment behaviour.
type EnrichOpts struct {
	Model   string
	Budget  float64
	Verbose bool
}
