package codemap

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const (
	defaultBatchSize          = 50
	approxCharsPerToken       = 4
	budgetOutputTokenEstimate = 2000
)

// modelPricing maps model IDs to [inputCostPerToken, outputCostPerToken] in USD.
var modelPricing = map[string][2]float64{
	"claude-sonnet-4-6": {3.0 / 1_000_000, 15.0 / 1_000_000},
	"claude-haiku-4-5":  {0.8 / 1_000_000, 4.0 / 1_000_000},
	"claude-opus-4-6":   {15.0 / 1_000_000, 75.0 / 1_000_000},
}

// Enricher calls the Anthropic API to add semantic enrichment to module extractions.
type Enricher struct {
	client    anthropic.Client
	model     string
	budget    float64
	spent     float64
	batchSize int
	verbose   bool
}

// NewEnricher creates an Enricher with the given API key and options.
func NewEnricher(apiKey string, opts EnrichOpts) *Enricher {
	model := opts.Model
	if model == "" {
		model = anthropic.ModelClaudeSonnet4_6
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Enricher{
		client:    client,
		model:     model,
		budget:    opts.Budget,
		spent:     0,
		batchSize: defaultBatchSize,
		verbose:   opts.Verbose,
	}
}

// Spent returns the total USD spent so far.
func (e *Enricher) Spent() float64 {
	return e.spent
}

// EnrichModule calls the LLM to enrich a single module extraction.
// API errors are returned as-is; callers should handle them per-module.
func (e *Enricher) EnrichModule(extraction *ModuleExtraction) (*EnrichedModule, error) {
	// Collect all symbols across files
	var allSymbols []Symbol
	for _, f := range extraction.Files {
		allSymbols = append(allSymbols, f.Symbols...)
	}

	// Process in batches
	symbolEnrichments := make(map[string]EnrichedSymbol)
	var moduleDesc, moduleCat string
	var keyTypes, keyFunctions []string

	for start := 0; start < len(allSymbols) || start == 0; start += e.batchSize {
		end := min(start+e.batchSize, len(allSymbols))
		batch := allSymbols[start:end]

		prompt := buildEnrichmentPrompt(extraction.Module, batch, extraction.Imports)

		resp, inputTok, outputTok, err := e.callAPI(prompt)
		if err != nil {
			return nil, fmt.Errorf("enrich module %s: %w", extraction.Module, err)
		}

		cost := e.estimateCost(inputTok, outputTok)
		e.spent += cost

		if e.verbose {
			fmt.Fprintf(os.Stderr, "  enriched batch %d-%d of %s: %d+%d tokens ($%.4f)\n",
				start, end, extraction.Module, inputTok, outputTok, cost)
		}

		parsed, err := parseEnrichmentResponse(resp)
		if err != nil {
			return nil, fmt.Errorf("parse enrichment for %s: %w", extraction.Module, err)
		}

		// Use module-level fields from first batch only
		if start == 0 {
			moduleDesc = parsed.ModuleDescription
			moduleCat = parsed.ModuleCategory
			keyTypes = parsed.KeyTypes
			keyFunctions = parsed.KeyFunctions
		}

		for _, sym := range parsed.Symbols {
			symbolEnrichments[sym.Name] = sym
		}

		// Only one batch needed if all symbols fit
		if end >= len(allSymbols) {
			break
		}
	}

	// Build enriched files with merged symbols
	enrichedFiles := make([]EnrichedFileExtract, 0, len(extraction.Files))
	for _, fe := range extraction.Files {
		merged := make([]MergedSymbol, 0, len(fe.Symbols))
		for _, sym := range fe.Symbols {
			ms := mergeSymbol(sym)
			if enr, ok := symbolEnrichments[sym.Name]; ok {
				ms.Description = enr.Description
				ms.ModuleIdentity = enr.ModuleIdentity
				ms.Complexity = enr.Complexity
				ms.IsEntrypoint = enr.IsEntrypoint
				ms.ArchitecturalRole = enr.ArchitecturalRole
				ms.Tags = enr.Tags
			}
			merged = append(merged, ms)
		}
		enrichedFiles = append(enrichedFiles, EnrichedFileExtract{
			Path:        fe.Path,
			LineCount:   fe.LineCount,
			Symbols:     merged,
			ErrorCount:  fe.ErrorCount,
			ParseErrors: fe.ParseErrors,
		})
	}

	return &EnrichedModule{
		Module:            extraction.Module,
		Language:          extraction.Language,
		Files:             enrichedFiles,
		Imports:           extraction.Imports,
		ExtractedAt:       extraction.ExtractedAt,
		ExtractorVersion:  extraction.ExtractorVersion,
		ModuleDescription: moduleDesc,
		ModuleCategory:    moduleCat,
		KeyTypes:          keyTypes,
		KeyFunctions:      keyFunctions,
		EnrichedAt:        time.Now().UTC().Format(time.RFC3339),
		EnricherModel:     e.model,
	}, nil
}

// EnrichGraph merges enrichment data into graph module nodes in place.
func (e *Enricher) EnrichGraph(graph *Graph, enrichedModules map[string]*EnrichedModule) {
	for i, node := range graph.Layers.ModuleDependencies.Nodes {
		em, ok := enrichedModules[node.ID]
		if !ok {
			continue
		}
		if em.ModuleDescription != "" {
			desc := em.ModuleDescription
			graph.Layers.ModuleDependencies.Nodes[i].Description = &desc
		}
		if len(em.KeyTypes) > 0 {
			graph.Layers.ModuleDependencies.Nodes[i].KeyTypes = em.KeyTypes
		}
		if len(em.KeyFunctions) > 0 {
			graph.Layers.ModuleDependencies.Nodes[i].KeyFunctions = em.KeyFunctions
		}
	}
}

// GenerateNarrative calls the LLM to produce an ARCHITECTURE.md narrative.
func (e *Enricher) GenerateNarrative(graph *Graph, enrichedModules map[string]*EnrichedModule) (string, error) {
	prompt := buildNarrativePrompt(graph, enrichedModules)
	text, inputTok, outputTok, err := e.callAPI(prompt)
	if err != nil {
		return "", fmt.Errorf("generate narrative: %w", err)
	}
	cost := e.estimateCost(inputTok, outputTok)
	e.spent += cost
	if e.verbose {
		fmt.Fprintf(os.Stderr, "  narrative generated: %d+%d tokens ($%.4f)\n", inputTok, outputTok, cost)
	}
	return strings.TrimSpace(text), nil
}

// pricing returns the [inputCostPerToken, outputCostPerToken] for e.model.
// Falls back to Sonnet rates with a warning for unknown models.
func (e *Enricher) pricing() [2]float64 {
	if p, ok := modelPricing[e.model]; ok {
		return p
	}
	fmt.Fprintf(os.Stderr, "warning: unknown model %q for pricing, using Sonnet rates\n", e.model)
	return modelPricing["claude-sonnet-4-6"]
}

// estimateCost computes USD cost for the given token counts using e.model's pricing.
func (e *Enricher) estimateCost(inputTokens, outputTokens int64) float64 {
	p := e.pricing()
	return float64(inputTokens)*p[0] + float64(outputTokens)*p[1]
}

// callAPI sends a prompt to the Anthropic API and returns (responseText, inputTokens, outputTokens, error).
func (e *Enricher) callAPI(prompt string) (string, int64, int64, error) {
	if e.budget > 0 {
		estimatedCost := e.estimateCost(int64(len(prompt)/approxCharsPerToken), budgetOutputTokenEstimate)
		if e.spent+estimatedCost > e.budget {
			return "", 0, 0, fmt.Errorf("budget limit $%.2f reached (spent $%.4f)", e.budget, e.spent)
		}
	}

	ctx := context.Background()
	msg, err := e.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     e.model,
		MaxTokens: 4096,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", 0, 0, fmt.Errorf("anthropic API: %w", err)
	}

	var text strings.Builder
	for _, block := range msg.Content {
		if block.Type == "text" {
			text.WriteString(block.AsText().Text)
		}
	}

	return text.String(), msg.Usage.InputTokens, msg.Usage.OutputTokens, nil
}

// WriteEnrichedModule writes an enriched module JSON to outputDir atomically.
func WriteEnrichedModule(outputDir string, enriched *EnrichedModule) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create enriched dir: %w", err)
	}

	data, err := json.MarshalIndent(enriched, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal enriched module %s: %w", enriched.Module, err)
	}

	outPath := filepath.Join(outputDir, moduleToFilename(enriched.Module)+".json")
	tmpPath := outPath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write enriched tmp for %s: %w", enriched.Module, err)
	}

	if err := os.Rename(tmpPath, outPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename enriched tmp for %s: %w", enriched.Module, err)
	}

	return nil
}

// WriteNarrative writes the narrative markdown to path atomically.
func WriteNarrative(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create narrative dir: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write narrative tmp: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename narrative tmp: %w", err)
	}

	return nil
}

// mergeSymbol converts a Symbol into a MergedSymbol (enrichment fields empty).
func mergeSymbol(sym Symbol) MergedSymbol {
	return MergedSymbol{
		Name:       sym.Name,
		Kind:       sym.Kind,
		Signature:  sym.Signature,
		Params:     sym.Params,
		Returns:    sym.Returns,
		Receiver:   sym.Receiver,
		LineStart:  sym.LineStart,
		LineEnd:    sym.LineEnd,
		Exported:   sym.Exported,
		Decorators: sym.Decorators,
		Calls:      sym.Calls,
		CalledBy:   sym.CalledBy,
	}
}
