package codemap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Graph is the multi-layer topology graph matching graph.schema.json.
type Graph struct {
	Version     string      `json:"version"`
	Repo        string      `json:"repo"`
	GeneratedAt string      `json:"generated_at"`
	GitCommit   string      `json:"git_commit"`
	Layers      GraphLayers `json:"layers"`
	Stats       GraphStats  `json:"stats"`
}

// GraphLayers holds the three topology layers.
type GraphLayers struct {
	ModuleDependencies ModuleDepLayer `json:"module_dependencies"`
	TypeRelationships  TypeRelLayer   `json:"type_relationships"`
	CallGraph          CallGraphLayer `json:"call_graph"`
}

// ModuleDepLayer is the module-level import dependency layer.
type ModuleDepLayer struct {
	Description string                 `json:"description"`
	Nodes       []ModuleNode           `json:"nodes"`
	Edges       []ModuleDependencyEdge `json:"edges"`
}

// TypeRelLayer holds type-level relationships.
type TypeRelLayer struct {
	Description string                `json:"description"`
	Nodes       []TypeNode            `json:"nodes"`
	Edges       []TypeRelationshipEdge `json:"edges"`
}

// CallGraphLayer is populated by CM-007; empty for now.
type CallGraphLayer struct {
	Description string          `json:"description"`
	Nodes       []CallGraphNode `json:"nodes"`
	Edges       []CallGraphEdge `json:"edges"`
}

// ModuleNode represents a module in the dependency graph.
type ModuleNode struct {
	ID           string   `json:"id"`
	Category     string   `json:"category"`
	Language     string   `json:"language"`
	SymbolCount  int      `json:"symbol_count"`
	FileCount    int      `json:"file_count"`
	Description  *string  `json:"description"`
	KeyTypes     []string `json:"key_types"`
	KeyFunctions []string `json:"key_functions"`
}

// TypeNode represents a type or interface in the type relationship layer.
type TypeNode struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Module string `json:"module"`
}

// CallGraphNode is a function node (populated by CM-007).
type CallGraphNode struct {
	ID     string `json:"id"`
	Module string `json:"module"`
}

// ModuleDependencyEdge represents an import relationship between modules.
type ModuleDependencyEdge struct {
	From              string `json:"from"`
	To                string `json:"to"`
	Type              string `json:"type"`
	SymbolsReferenced int    `json:"symbols_referenced"`
}

// TypeRelationshipEdge represents a structural relationship between types.
type TypeRelationshipEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Type  string `json:"type"`
	Field string `json:"field,omitempty"`
}

// CallGraphEdge is a call relationship (populated by CM-007).
type CallGraphEdge struct {
	From        string `json:"from"`
	To          string `json:"to"`
	CrossModule bool   `json:"cross_module"`
}

// GraphStats holds topology metrics for the whole graph.
type GraphStats struct {
	TotalModules          int         `json:"total_modules"`
	TotalSymbols          int         `json:"total_symbols"`
	TotalDependencyEdges  int         `json:"total_dependency_edges"`
	TotalTypeEdges        int         `json:"total_type_edges"`
	TotalCallEdges        int         `json:"total_call_edges"`
	AvgInDegree           float64     `json:"avg_in_degree"`
	MaxInDegree           *DegreeInfo `json:"max_in_degree"`
	MaxOutDegree          *DegreeInfo `json:"max_out_degree"`
}

// DegreeInfo identifies the module with the highest degree and its edge count.
type DegreeInfo struct {
	Module string `json:"module"`
	Degree int    `json:"degree"`
}

// moduleCategory returns the graph category for a module path.
func moduleCategory(modulePath string) string {
	switch {
	case strings.HasPrefix(modulePath, "cmd/"):
		return "command"
	case strings.HasPrefix(modulePath, "internal/"):
		return "internal"
	case strings.HasPrefix(modulePath, "pkg/"):
		return "pkg"
	case strings.HasPrefix(modulePath, "test/"):
		return "test"
	default:
		return "internal"
	}
}

// AggregateGraph builds the multi-layer graph from all module extractions.
func AggregateGraph(extractions []*ModuleExtraction, repoName, gitCommit string) *Graph {
	moduleSet := make(map[string]bool, len(extractions))
	for _, e := range extractions {
		moduleSet[e.Module] = true
	}

	modNodes, modEdges, totalSymbols := buildModuleLayer(extractions, moduleSet)
	typeNodes := buildTypeLayer(extractions)
	cgNodes, cgEdges := buildCallGraphLayer(extractions)
	inDegree, outDegree := computeDegreeStats(modNodes, modEdges)

	var avgInDegree float64
	if len(modNodes) > 0 {
		total := 0
		for _, d := range inDegree {
			total += d
		}
		avgInDegree = float64(total) / float64(len(modNodes))
	}

	maxIn := maxDegree(inDegree)
	maxOut := maxDegree(outDegree)

	return &Graph{
		Version:     "1.0",
		Repo:        repoName,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		GitCommit:   gitCommit,
		Layers: GraphLayers{
			ModuleDependencies: ModuleDepLayer{
				Description: "Which modules import which other modules",
				Nodes:       modNodes,
				Edges:       modEdges,
			},
			TypeRelationships: TypeRelLayer{
				Description: "Struct composition, interface implementation, type dependencies",
				Nodes:       typeNodes,
				Edges:       []TypeRelationshipEdge{},
			},
			CallGraph: CallGraphLayer{
				Description: "Function/method call relationships",
				Nodes:       cgNodes,
				Edges:       cgEdges,
			},
		},
		Stats: GraphStats{
			TotalModules:         len(modNodes),
			TotalSymbols:         totalSymbols,
			TotalDependencyEdges: len(modEdges),
			TotalTypeEdges:       0,
			TotalCallEdges:       len(cgEdges),
			AvgInDegree:          avgInDegree,
			MaxInDegree:          maxIn,
			MaxOutDegree:         maxOut,
		},
	}
}

func buildModuleLayer(extractions []*ModuleExtraction, moduleSet map[string]bool) (nodes []ModuleNode, edges []ModuleDependencyEdge, totalSymbols int) {
	edgeSet := make(map[string]bool)
	for _, e := range extractions {
		symCount := 0
		for _, fe := range e.Files {
			symCount += len(fe.Symbols)
		}
		totalSymbols += symCount
		nodes = append(nodes, ModuleNode{
			ID:          e.Module,
			Category:    moduleCategory(e.Module),
			Language:    e.Language,
			SymbolCount: symCount,
			FileCount:   len(e.Files),
		})
		for _, importPath := range e.Imports.Internal {
			target := resolveModulePath(importPath, moduleSet)
			if target == "" || target == e.Module {
				continue
			}
			key := e.Module + "→" + target
			if !edgeSet[key] {
				edgeSet[key] = true
				edges = append(edges, ModuleDependencyEdge{
					From: e.Module,
					To:   target,
					Type: "imports",
				})
			}
		}
	}
	return
}

func buildTypeLayer(extractions []*ModuleExtraction) []TypeNode {
	var nodes []TypeNode
	for _, e := range extractions {
		for _, fe := range e.Files {
			for _, sym := range fe.Symbols {
				if sym.Kind != "type" && sym.Kind != "interface" {
					continue
				}
				nodes = append(nodes, TypeNode{
					ID:     e.Module + "." + sym.Name,
					Kind:   sym.Kind,
					Module: e.Module,
				})
			}
		}
	}
	return nodes
}

func buildCallGraphLayer(extractions []*ModuleExtraction) (nodes []CallGraphNode, edges []CallGraphEdge) {
	nodes = []CallGraphNode{} // initialized as empty slice (not nil) to serialize as [] in JSON
	edges = []CallGraphEdge{} // initialized as empty slice (not nil) to serialize as [] in JSON
	nodeSet := make(map[string]bool)
	edgeSet := make(map[string]bool)
	for _, e := range extractions {
		for _, fe := range e.Files {
			for _, sym := range fe.Symbols {
				if sym.Kind != "function" && sym.Kind != "method" {
					continue
				}
				if len(sym.Calls) == 0 && len(sym.CalledBy) == 0 {
					continue
				}
				fqn := e.Module + "." + sym.Name
				if !nodeSet[fqn] {
					nodeSet[fqn] = true
					nodes = append(nodes, CallGraphNode{ID: fqn, Module: e.Module})
				}
				for _, callee := range sym.Calls {
					if !nodeSet[callee] {
						nodeSet[callee] = true
						nodes = append(nodes, CallGraphNode{ID: callee, Module: fqnModule(callee)})
					}
					edgeKey := fqn + "→" + callee
					if !edgeSet[edgeKey] {
						edgeSet[edgeKey] = true
						edges = append(edges, CallGraphEdge{
							From:        fqn,
							To:          callee,
							CrossModule: fqnModule(fqn) != fqnModule(callee),
						})
					}
				}
			}
		}
	}
	return
}

func computeDegreeStats(nodes []ModuleNode, edges []ModuleDependencyEdge) (inDegree, outDegree map[string]int) {
	inDegree = make(map[string]int, len(nodes))
	outDegree = make(map[string]int, len(nodes))
	for _, e := range edges {
		outDegree[e.From]++
		inDegree[e.To]++
	}
	return
}

// WriteGraph writes the graph to path atomically.
func WriteGraph(path string, g *Graph) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create graph dir: %w", err)
	}

	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal graph: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write graph tmp: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename graph tmp: %w", err)
	}

	return nil
}

// resolveModulePath maps a full import path to a module path in the module set.
// e.g. "github.com/Bucket-Chemist/goYoke/internal/codemap" → "internal/codemap"
func resolveModulePath(importPath string, moduleSet map[string]bool) string {
	for mod := range moduleSet {
		if strings.HasSuffix(importPath, "/"+mod) {
			return mod
		}
	}
	return ""
}

// maxDegree returns the module with the highest degree, or nil if empty.
func maxDegree(degrees map[string]int) *DegreeInfo {
	if len(degrees) == 0 {
		return nil
	}
	var best DegreeInfo
	for mod, d := range degrees {
		if d > best.Degree {
			best = DegreeInfo{Module: mod, Degree: d}
		}
	}
	return &best
}
