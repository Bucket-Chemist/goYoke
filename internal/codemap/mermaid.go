package codemap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxMermaidTypeNodes = 20

// GenerateMermaid writes Mermaid diagram files for the graph into outputDir.
// Produces:
//   - module-dependencies.mmd  (flowchart of module imports)
//   - {module}--types.mmd      (classDiagram per module with types, if any)
func GenerateMermaid(graph *Graph, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create mermaid dir: %w", err)
	}

	if err := writeMermaidModuleDeps(graph, outputDir); err != nil {
		return fmt.Errorf("module-dependencies diagram: %w", err)
	}

	if err := writeMermaidTypesDiagrams(graph, outputDir); err != nil {
		return fmt.Errorf("type diagrams: %w", err)
	}

	return nil
}

func writeMermaidModuleDeps(graph *Graph, outputDir string) error {
	var sb strings.Builder
	sb.WriteString("graph TD\n")

	// Emit nodes.
	for _, node := range graph.Layers.ModuleDependencies.Nodes {
		id := sanitizeMermaidID(node.ID)
		fmt.Fprintf(&sb, "  %s[\"%s\"]\n", id, node.ID)
	}

	// Emit edges.
	for _, edge := range graph.Layers.ModuleDependencies.Edges {
		from := sanitizeMermaidID(edge.From)
		to := sanitizeMermaidID(edge.To)
		fmt.Fprintf(&sb, "  %s --> %s\n", from, to)
	}

	return writeMermaidFile(filepath.Join(outputDir, "module-dependencies.mmd"), sb.String())
}

func writeMermaidTypesDiagrams(graph *Graph, outputDir string) error {
	// Group type nodes by module.
	byModule := make(map[string][]TypeNode)
	for _, tn := range graph.Layers.TypeRelationships.Nodes {
		byModule[tn.Module] = append(byModule[tn.Module], tn)
	}

	for module, types := range byModule {
		if len(types) == 0 {
			continue
		}

		// If >20 types, show only exported ones.
		visible := types
		if len(types) > maxMermaidTypeNodes {
			var exported []TypeNode
			for _, tn := range types {
				// The ID is "module.TypeName"; the name is after the last dot.
				name := typeNodeName(tn.ID)
				if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
					exported = append(exported, tn)
				}
			}
			visible = exported
		}

		if len(visible) == 0 {
			continue
		}

		var sb strings.Builder
		sb.WriteString("classDiagram\n")
		for _, tn := range visible {
			name := typeNodeName(tn.ID)
			fmt.Fprintf(&sb, "  class %s\n", name)
		}

		filename := moduleToFilename(module) + "--types.mmd"
		if err := writeMermaidFile(filepath.Join(outputDir, filename), sb.String()); err != nil {
			return err
		}
	}

	return nil
}

func writeMermaidFile(path, content string) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write mermaid tmp %s: %w", path, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename mermaid %s: %w", path, err)
	}
	return nil
}

// sanitizeMermaidID replaces /, -, and . with _ for valid Mermaid node IDs.
func sanitizeMermaidID(s string) string {
	r := strings.NewReplacer("/", "_", "-", "_", ".", "_")
	return r.Replace(s)
}

// typeNodeName extracts the type name from a fully qualified ID like "module.TypeName".
func typeNodeName(id string) string {
	idx := strings.LastIndex(id, ".")
	if idx < 0 {
		return id
	}
	return id[idx+1:]
}
