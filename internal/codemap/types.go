package codemap

// ModuleExtraction is the top-level output per module, matching extraction.schema.json.
type ModuleExtraction struct {
	Module           string       `json:"module"`
	Language         string       `json:"language"`
	Files            []FileExtract `json:"files"`
	Imports          ImportGraph  `json:"imports"`
	ExtractedAt      string       `json:"extracted_at"`
	ExtractorVersion string       `json:"extractor_version"`
}

// FileExtract holds per-file extraction output.
type FileExtract struct {
	Path        string       `json:"path"`
	LineCount   int          `json:"line_count"`
	Symbols     []Symbol     `json:"symbols"`
	ErrorCount  int          `json:"error_count"`
	ParseErrors []ParseError `json:"parse_errors"`
}

// ParseError describes a parse error found during extraction.
type ParseError struct {
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

// Symbol represents a top-level declaration extracted from a source file.
type Symbol struct {
	Name       string      `json:"name"`
	Kind       string      `json:"kind"` // "function", "method", "type", "interface", "const", "var"
	Signature  string      `json:"signature"`
	Params     []Param     `json:"params"`
	Returns    []ReturnVal `json:"returns"`
	Receiver   *string     `json:"receiver"`
	LineStart  int         `json:"line_start"`
	LineEnd    int         `json:"line_end"`
	Exported   bool        `json:"exported"`
	Decorators []string    `json:"decorators"`
	Calls      []string    `json:"calls"`
	CalledBy   []string    `json:"called_by"`
}

// Param is a function parameter with name and type.
type Param struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ReturnVal is a return value type.
type ReturnVal struct {
	Type string `json:"type"`
}

// ImportGraph aggregates imports for a module split by origin.
type ImportGraph struct {
	Internal []string `json:"internal"`
	External []string `json:"external"`
	Stdlib   []string `json:"stdlib"`
}

// LanguageExtractor abstracts language-specific extraction logic.
type LanguageExtractor interface {
	Language() string
	Extensions() []string
	IsExported(name string) bool
	// ClassifyImport returns "internal", "external", or "stdlib".
	ClassifyImport(importPath string, projectModule string) string
}

// ModuleKey identifies a logical module by directory and language.
// A single directory can contain files of multiple languages, each
// forming its own ModuleKey group.
type ModuleKey struct {
	Path     string // directory relative to project root (empty = repo root)
	Language string // "go", "rust", "typescript", "python", "r"
}

// DiscoveryOpts controls file discovery behaviour.
type DiscoveryOpts struct {
	ExcludePatterns  []string
	IncludeGenerated bool
	SpecificFiles    []string
	Lang             string // "auto", "go", etc.
}
