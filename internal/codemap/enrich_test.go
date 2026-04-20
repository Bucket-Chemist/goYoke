package codemap

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildEnrichmentPrompt(t *testing.T) {
	symbols := []Symbol{
		{Name: "NewClient", Kind: "function", Signature: "func NewClient(apiKey string) *Client", LineStart: 10, LineEnd: 20},
		{Name: "Client", Kind: "type", Signature: "type Client struct", LineStart: 30, LineEnd: 50},
		{Name: "Fetch", Kind: "method", Signature: "func (c *Client) Fetch(url string) ([]byte, error)", LineStart: 55, LineEnd: 80},
	}
	imports := ImportGraph{
		Internal: []string{"github.com/example/project/internal/config"},
		External: []string{"net/http"},
	}

	prompt := buildEnrichmentPrompt("internal/httpclient", symbols, imports)

	// Must contain module path
	assert.Contains(t, prompt, "internal/httpclient")

	// Must contain all symbol names
	assert.Contains(t, prompt, "NewClient")
	assert.Contains(t, prompt, "Client")
	assert.Contains(t, prompt, "Fetch")

	// Must contain enum values for module_identity
	assert.Contains(t, prompt, "core")
	assert.Contains(t, prompt, "helper")
	assert.Contains(t, prompt, "glue")

	// Must contain enum values for complexity
	assert.Contains(t, prompt, "low")
	assert.Contains(t, prompt, "medium")
	assert.Contains(t, prompt, "high")

	// Must contain enum values for architectural_role
	assert.Contains(t, prompt, "handler")
	assert.Contains(t, prompt, "service")
	assert.Contains(t, prompt, "utility")

	// Must contain module_category values
	assert.Contains(t, prompt, "command")
	assert.Contains(t, prompt, "library")
	assert.Contains(t, prompt, "internal")

	// Must contain imports
	assert.Contains(t, prompt, "internal/config")

	// Rough token estimate: under 8K tokens (32K chars as proxy)
	assert.Less(t, len(prompt), 32000, "prompt should be under 8K tokens estimate")
}

func TestParseEnrichmentResponseValid(t *testing.T) {
	raw := `{
  "module_description": "Handles HTTP communication with the API.",
  "module_category": "internal",
  "key_types": ["Client"],
  "key_functions": ["NewClient", "Fetch"],
  "symbols": [
    {
      "name": "NewClient",
      "description": "Creates a new HTTP client with the given API key.",
      "module_identity": "core",
      "complexity": "low",
      "is_entrypoint": true,
      "architectural_role": "factory",
      "tags": ["initialization"]
    },
    {
      "name": "Fetch",
      "description": "Fetches a URL and returns the response body.",
      "module_identity": "core",
      "complexity": "medium",
      "is_entrypoint": false,
      "architectural_role": "service",
      "tags": ["io", "http"]
    }
  ]
}`

	resp, err := parseEnrichmentResponse(raw)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "Handles HTTP communication with the API.", resp.ModuleDescription)
	assert.Equal(t, "internal", resp.ModuleCategory)
	assert.Equal(t, []string{"Client"}, resp.KeyTypes)
	assert.Equal(t, []string{"NewClient", "Fetch"}, resp.KeyFunctions)
	require.Len(t, resp.Symbols, 2)

	assert.Equal(t, "NewClient", resp.Symbols[0].Name)
	assert.Equal(t, "core", resp.Symbols[0].ModuleIdentity)
	assert.Equal(t, "low", resp.Symbols[0].Complexity)
	assert.True(t, resp.Symbols[0].IsEntrypoint)
	assert.Equal(t, "factory", resp.Symbols[0].ArchitecturalRole)

	assert.Equal(t, "Fetch", resp.Symbols[1].Name)
	assert.Equal(t, []string{"io", "http"}, resp.Symbols[1].Tags)
}

func TestParseEnrichmentResponseWithCodeFences(t *testing.T) {
	raw := "```json\n" + `{
  "module_description": "Test module.",
  "module_category": "test",
  "key_types": [],
  "key_functions": [],
  "symbols": []
}` + "\n```"

	resp, err := parseEnrichmentResponse(raw)
	require.NoError(t, err)
	assert.Equal(t, "Test module.", resp.ModuleDescription)
}

func TestParseEnrichmentResponseInvalid(t *testing.T) {
	_, err := parseEnrichmentResponse("no json here")
	assert.Error(t, err)

	_, err = parseEnrichmentResponse("")
	assert.Error(t, err)

	_, err = parseEnrichmentResponse("{invalid json}")
	assert.Error(t, err)
}

func TestBudgetEnforcement(t *testing.T) {
	e := &Enricher{
		model:     "claude-sonnet-4-6",
		budget:    0.01,
		spent:     0.009, // nearly at limit
		batchSize: defaultBatchSize,
	}

	// Build a prompt long enough that the estimated cost pushes over budget
	longPrompt := strings.Repeat("analyze this symbol: func Example() error { return nil }\n", 200)
	_, _, _, err := e.callAPI(longPrompt)

	// Should hit budget before calling API (no API key set, but budget fires first)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "budget limit")
}

func TestEnrichGraph(t *testing.T) {
	graph := &Graph{
		Layers: GraphLayers{
			ModuleDependencies: ModuleDepLayer{
				Nodes: []ModuleNode{
					{ID: "internal/auth", Category: "internal", SymbolCount: 5},
					{ID: "cmd/server", Category: "command", SymbolCount: 3},
				},
			},
		},
	}

	enrichedModules := map[string]*EnrichedModule{
		"internal/auth": {
			ModuleDescription: "Handles authentication and authorization.",
			KeyTypes:          []string{"Token", "Claims"},
			KeyFunctions:      []string{"Validate", "Parse"},
		},
		// cmd/server intentionally missing — should not panic
	}

	e := &Enricher{model: "claude-sonnet-4-6"}
	e.EnrichGraph(graph, enrichedModules)

	// internal/auth should be enriched
	authNode := graph.Layers.ModuleDependencies.Nodes[0]
	require.NotNil(t, authNode.Description)
	assert.Equal(t, "Handles authentication and authorization.", *authNode.Description)
	assert.Equal(t, []string{"Token", "Claims"}, authNode.KeyTypes)
	assert.Equal(t, []string{"Validate", "Parse"}, authNode.KeyFunctions)

	// cmd/server should be untouched
	serverNode := graph.Layers.ModuleDependencies.Nodes[1]
	assert.Nil(t, serverNode.Description)
}

func TestEstimateCost(t *testing.T) {
	e := &Enricher{model: "claude-sonnet-4-6"}
	cost := e.estimateCost(1000, 1000)
	// 1K input * $3/MTok + 1K output * $15/MTok = $0.018
	assert.InDelta(t, 0.018, cost, 0.0001)
}

func TestNewEnricher_Defaults(t *testing.T) {
	e := NewEnricher("test-key", EnrichOpts{})
	assert.NotNil(t, e)
	assert.Equal(t, "claude-sonnet-4-6", e.model)
	assert.Equal(t, 0.0, e.budget)
	assert.False(t, e.verbose)
	assert.Equal(t, defaultBatchSize, e.batchSize)
}

func TestNewEnricher_CustomModel(t *testing.T) {
	e := NewEnricher("key", EnrichOpts{Model: "claude-haiku-4-5", Budget: 0.5, Verbose: true})
	assert.Equal(t, "claude-haiku-4-5", e.model)
	assert.Equal(t, 0.5, e.budget)
	assert.True(t, e.verbose)
}

func TestSpent(t *testing.T) {
	e := &Enricher{spent: 0.123}
	assert.InDelta(t, 0.123, e.Spent(), 0.0001)
}

func TestPricing_KnownModel(t *testing.T) {
	e := &Enricher{model: "claude-haiku-4-5"}
	p := e.pricing()
	assert.Equal(t, modelPricing["claude-haiku-4-5"], p)
}

func TestPricing_UnknownModelFallback(t *testing.T) {
	e := &Enricher{model: "claude-unknown-99"}
	p := e.pricing()
	// Falls back to Sonnet rates
	assert.Equal(t, modelPricing["claude-sonnet-4-6"], p)
}

func TestMergeSymbol(t *testing.T) {
	recv := "Client"
	sym := Symbol{
		Name:       "Fetch",
		Kind:       "method",
		Signature:  "func (c *Client) Fetch() error",
		Receiver:   &recv,
		LineStart:  10,
		LineEnd:    20,
		Exported:   true,
		Decorators: []string{},
		Calls:      []string{"internal/net.Dial"},
		CalledBy:   []string{},
	}

	ms := mergeSymbol(sym)
	assert.Equal(t, sym.Name, ms.Name)
	assert.Equal(t, sym.Kind, ms.Kind)
	assert.Equal(t, sym.Receiver, ms.Receiver)
	assert.Equal(t, sym.Calls, ms.Calls)
	// Enrichment fields should be zero
	assert.Empty(t, ms.Description)
	assert.Empty(t, ms.ModuleIdentity)
}
