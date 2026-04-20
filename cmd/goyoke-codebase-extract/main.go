package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/internal/codemap"
)

var version = "dev"

func main() {
	path := flag.String("path", ".", "Project root to scan")
	output := flag.String("output", ".claude/codebase-map/extract/", "Output directory for extraction JSON files")
	lang := flag.String("lang", "auto", "Force language (auto, go)")
	files := flag.String("files", "", "Extract specific files only, comma-separated (for incremental)")
	exclude := flag.String("exclude", "", "Additional exclude patterns, comma-separated")
	includeGenerated := flag.Bool("include-generated", false, "Include generated files in extraction")
	manifest := flag.String("manifest", ".claude/codebase-map/manifest.json", "Manifest file path")
	skipCallGraph := flag.Bool("skip-call-graph", false, "Skip call graph extraction (CM-007)")
	incremental := flag.Bool("incremental", false, "Only extract changed files (CM-011)")
	render := flag.Bool("render", false, "Generate Mermaid diagram files after extraction")
	enrich := flag.Bool("enrich", false, "Enrich extraction with LLM semantic analysis (requires ANTHROPIC_API_KEY)")
	narrate := flag.Bool("narrate", false, "Generate ARCHITECTURE.md narrative (implies --enrich, requires ANTHROPIC_API_KEY)")
	budget := flag.Float64("budget", 0, "USD budget cap for LLM enrichment (0 = unlimited)")
	model := flag.String("model", "claude-sonnet-4-6", "Model to use for enrichment")
	showVersion := flag.Bool("version", false, "Print version and exit")
	verbose := flag.Bool("verbose", false, "Verbose output to stderr")

	flag.Parse()

	if *showVersion {
		fmt.Printf("goyoke-codebase-extract %s\n", version)
		os.Exit(0)
	}

	if *narrate {
		*enrich = true
	}

	if *enrich {
		if os.Getenv("ANTHROPIC_API_KEY") == "" {
			fmt.Fprintln(os.Stderr, "error: ANTHROPIC_API_KEY environment variable is required for --enrich/--narrate")
			os.Exit(1)
		}
	}

	opts := codemap.DiscoveryOpts{
		IncludeGenerated: *includeGenerated,
		Lang:             *lang,
	}
	if *files != "" {
		opts.SpecificFiles = splitComma(*files)
	}
	if *exclude != "" {
		opts.ExcludePatterns = splitComma(*exclude)
	}

	modules, totalFiles, err := runDiscovery(*path, opts, *verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectModule, err := codemap.ResolveProjectModule(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not resolve project module: %v\n", err)
		projectModule = ""
	}

	extractDir := filepath.Clean(*output)

	var extractions []*codemap.ModuleExtraction
	var totalSymbols int
	var didIncremental bool
	var existingManifest *codemap.Manifest

	if *incremental {
		extractions, totalSymbols, existingManifest, didIncremental = tryIncrementalExtraction(
			*manifest, *path, extractDir, projectModule, modules, *verbose)
	}

	if !didIncremental {
		extractions, totalSymbols = runFullExtraction(modules, *path, extractDir, projectModule, *verbose)
	}

	fmt.Fprintf(os.Stderr, "extracted %d modules, %d symbols\n", len(extractions), totalSymbols)

	if !*skipCallGraph {
		if err := runCallGraph(extractions, *path, projectModule, extractDir, *verbose); err != nil {
			fmt.Fprintf(os.Stderr, "warning: call graph skipped (%v)\n", err)
		}
	}

	graph, graphPath, err := runOutputs(
		extractions, *path, extractDir, *manifest, version,
		didIncremental, existingManifest, *render, *verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *enrich {
		if err := runEnrichment(extractions, graph, graphPath, os.Getenv("ANTHROPIC_API_KEY"), *model, *budget, *narrate, extractDir, *path, *verbose); err != nil {
			fmt.Fprintf(os.Stderr, "error: enrichment: %v\n", err)
		}
	}

	fmt.Printf("extraction complete: %d modules, %d files, %d symbols\n",
		len(extractions), totalFiles, totalSymbols)
}

// runDiscovery discovers source files and returns module map, total file count, and error.
func runDiscovery(path string, opts codemap.DiscoveryOpts, verbose bool) (map[codemap.ModuleKey][]string, int, error) {
	modules, err := codemap.DiscoverFiles(path, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("discover files: %w", err)
	}
	if len(modules) == 0 {
		return nil, 0, fmt.Errorf("no source files found in %s", path)
	}
	totalFiles := 0
	for _, fs := range modules {
		totalFiles += len(fs)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "discovered %d modules, %d files\n", len(modules), totalFiles)
		for key, fs := range modules {
			fmt.Fprintf(os.Stderr, "  %s (%s): %d files\n", key.Path, key.Language, len(fs))
		}
	} else {
		fmt.Fprintf(os.Stderr, "discovered %d modules, %d files\n", len(modules), totalFiles)
	}
	return modules, totalFiles, nil
}

// tryIncrementalExtraction attempts incremental extraction based on the existing manifest.
// Returns (extractions, totalSymbols, existingManifest, didIncremental).
func tryIncrementalExtraction(
	manifestPath, projectRoot, extractDir, projectModule string,
	allModules map[codemap.ModuleKey][]string,
	verbose bool,
) ([]*codemap.ModuleExtraction, int, *codemap.Manifest, bool) {
	em, mErr := codemap.LoadManifest(manifestPath)
	if mErr != nil || em == nil || em.GitCommit == "" || len(em.Modules) == 0 {
		if verbose {
			fmt.Fprintln(os.Stderr, "no existing manifest, performing full extraction")
		}
		return nil, 0, nil, false
	}

	changes, cErr := codemap.DetectChanges(em, projectRoot, allModules)
	switch {
	case cErr != nil:
		fmt.Fprintf(os.Stderr, "warning: change detection failed, full extraction: %v\n", cErr)
		return nil, 0, nil, false
	case changes.FullRemap:
		if verbose {
			fmt.Fprintln(os.Stderr, "incremental: full remap required")
		}
		return nil, 0, nil, false
	case len(changes.ChangedFiles)+len(changes.NewFiles)+len(changes.DeletedFiles) == 0:
		fmt.Fprintln(os.Stderr, "no changes detected since last map")
		os.Exit(0)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "incremental: %d changed, %d new, %d deleted (method: %s)\n",
			len(changes.ChangedFiles), len(changes.NewFiles), len(changes.DeletedFiles), changes.Method)
	}
	extractions, totalSymbols := runIncrementalExtraction(changes, allModules, projectRoot, extractDir, projectModule, verbose)
	return extractions, totalSymbols, em, true
}

// runFullExtraction extracts all modules and writes extraction JSON files.
func runFullExtraction(
	modules map[codemap.ModuleKey][]string,
	projectRoot, extractDir, projectModule string,
	verbose bool,
) ([]*codemap.ModuleExtraction, int) {
	var extractions []*codemap.ModuleExtraction
	var totalSymbols int
	var err error
	ctagsWarned := false

	for key, filePaths := range modules {
		var extraction *codemap.ModuleExtraction

		switch key.Language {
		case "go":
			extraction, err = codemap.ExtractModule(key.Path, filePaths, projectRoot, projectModule, verbose)
		default:
			if !codemap.CtagsAvailable() {
				if !ctagsWarned {
					fmt.Fprintln(os.Stderr, "warning: ctags not installed — skipping non-Go files. Install: pacman -S ctags")
					ctagsWarned = true
				}
				continue
			}
			projectName, _ := codemap.ResolveProjectName(projectRoot, key.Language)
			extraction, err = codemap.ExtractModuleCtags(key.Path, filePaths, key.Language, projectRoot, projectName, verbose)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: module %s (%s): %v\n", key.Path, key.Language, err)
			continue
		}

		symbolCount := 0
		for _, f := range extraction.Files {
			symbolCount += len(f.Symbols)
		}
		totalSymbols += symbolCount

		if verbose {
			fmt.Fprintf(os.Stderr, "  extracted %s (%s): %d files, %d symbols\n",
				key.Path, key.Language, len(filePaths), symbolCount)
		}

		if err := codemap.WriteModuleExtraction(extractDir, extraction); err != nil {
			fmt.Fprintf(os.Stderr, "error: write extraction %s: %v\n", extraction.Module, err)
			os.Exit(1)
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "  wrote %s\n", extraction.Module)
		}
		extractions = append(extractions, extraction)
	}

	fmt.Fprintf(os.Stderr, "wrote %d extraction files to %s\n", len(extractions), extractDir)
	return extractions, totalSymbols
}

// runCallGraph extracts Go call graph data and re-writes affected extraction files.
func runCallGraph(
	extractions []*codemap.ModuleExtraction,
	projectRoot, projectModule, extractDir string,
	verbose bool,
) error {
	var goExtractions []*codemap.ModuleExtraction
	for _, e := range extractions {
		if e.Language == "go" {
			goExtractions = append(goExtractions, e)
		}
	}
	if len(goExtractions) == 0 {
		return nil
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "loading type-checked packages for call graph...\n")
	}
	pkgs, err := codemap.LoadTypeCheckedPackages(projectRoot)
	if err != nil {
		return fmt.Errorf("type checking failed: %w", err)
	}

	var allSites []codemap.CallSite
	for _, pkg := range pkgs {
		sites := codemap.ExtractCallSites(pkg.Fset, pkg, projectModule)
		allSites = append(allSites, sites...)
	}
	codemap.ResolveCallGraph(goExtractions, allSites)

	if verbose {
		fmt.Fprintf(os.Stderr, "call graph: %d call sites resolved\n", len(allSites))
	}

	for _, e := range goExtractions {
		if err := codemap.WriteModuleExtraction(extractDir, e); err != nil {
			return fmt.Errorf("re-write extraction %s: %w", e.Module, err)
		}
	}
	return nil
}

// runOutputs writes graph.json, manifest, and optionally Mermaid diagrams.
// Returns the graph, graph file path, and any fatal error.
func runOutputs(
	extractions []*codemap.ModuleExtraction,
	projectRoot, extractDir, manifestPath, ver string,
	didIncremental bool, existingManifest *codemap.Manifest,
	render bool, verbose bool,
) (*codemap.Graph, string, error) {
	repoName := codemap.ResolveRepoName(projectRoot)
	gitCommit := codemap.ResolveGitCommit(projectRoot)
	graph := codemap.AggregateGraph(extractions, repoName, gitCommit)

	graphPath := filepath.Join(filepath.Dir(extractDir), "graph.json")
	if err := codemap.WriteGraph(graphPath, graph); err != nil {
		return nil, "", fmt.Errorf("write graph: %w", err)
	}
	fmt.Fprintf(os.Stderr, "wrote graph.json (%d module nodes, %d edges)\n",
		len(graph.Layers.ModuleDependencies.Nodes),
		len(graph.Layers.ModuleDependencies.Edges))

	m, err := codemap.BuildManifest(projectRoot, extractions, ver)
	if err != nil {
		return nil, "", fmt.Errorf("build manifest: %w", err)
	}
	if didIncremental && existingManifest != nil {
		m.LastFullMap = existingManifest.LastFullMap
		now := time.Now().UTC().Format(time.RFC3339)
		m.LastIncremental = &now
	}
	if err := codemap.WriteManifest(manifestPath, m); err != nil {
		return nil, "", fmt.Errorf("write manifest: %w", err)
	}
	fmt.Fprintf(os.Stderr, "wrote manifest (%d modules, %d files, %d symbols)\n",
		m.Stats.TotalModules, m.Stats.TotalFiles, m.Stats.TotalSymbols)

	if render {
		mermaidDir := filepath.Join(filepath.Dir(extractDir), "mermaid")
		if err := codemap.GenerateMermaid(graph, mermaidDir); err != nil {
			return nil, "", fmt.Errorf("generate mermaid: %w", err)
		}
		fmt.Fprintf(os.Stderr, "wrote mermaid diagrams to %s\n", mermaidDir)
	}

	_ = verbose
	return graph, graphPath, nil
}

// runEnrichment performs LLM semantic enrichment and optionally generates ARCHITECTURE.md.
func runEnrichment(
	extractions []*codemap.ModuleExtraction,
	graph *codemap.Graph, graphPath string,
	apiKey, modelStr string, budget float64,
	narrate bool,
	extractDir, projectRoot string,
	verbose bool,
) error {
	enricher := codemap.NewEnricher(apiKey, codemap.EnrichOpts{
		Model:   modelStr,
		Budget:  budget,
		Verbose: verbose,
	})

	enrichedDir := filepath.Join(filepath.Dir(extractDir), "enriched")
	enrichedModules := make(map[string]*codemap.EnrichedModule, len(extractions))

	for _, e := range extractions {
		if e.Language != "go" {
			continue
		}
		em, err := enricher.EnrichModule(e)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: enrich module %s: %v\n", e.Module, err)
			continue
		}
		enrichedModules[e.Module] = em
		if err := codemap.WriteEnrichedModule(enrichedDir, em); err != nil {
			fmt.Fprintf(os.Stderr, "error: write enriched %s: %v\n", e.Module, err)
		}
	}

	fmt.Fprintf(os.Stderr, "enriched %d/%d modules to %s\n", len(enrichedModules), len(extractions), enrichedDir)

	enricher.EnrichGraph(graph, enrichedModules)
	if err := codemap.WriteGraph(graphPath, graph); err != nil {
		fmt.Fprintf(os.Stderr, "error: write enriched graph: %v\n", err)
	}

	if narrate {
		narrative, err := enricher.GenerateNarrative(graph, enrichedModules)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: generate narrative: %v\n", err)
		} else {
			repoName := codemap.ResolveRepoName(projectRoot)
			narrativePath := filepath.Join(projectRoot, "docs", "codebase-architecture", repoName, "ARCHITECTURE.md")
			if err := codemap.WriteNarrative(narrativePath, narrative); err != nil {
				fmt.Fprintf(os.Stderr, "error: write narrative: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "wrote %s\n", narrativePath)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "enrichment cost: $%.4f\n", enricher.Spent())
	return nil
}

func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// fileModule returns the module path for a relative file path (its parent directory).
func fileModule(relPath string) string {
	dir := filepath.ToSlash(filepath.Dir(relPath))
	if dir == "." {
		return ""
	}
	return dir
}

// runIncrementalExtraction loads, merges, and writes only affected modules.
// Unchanged modules are loaded from existing extraction JSON files.
// Returns the full set of extractions (unchanged + updated) and total symbol count.
// Note: incremental extraction currently handles Go modules only.
func runIncrementalExtraction(
	changes *codemap.ChangeSet,
	allModules map[codemap.ModuleKey][]string,
	projectRoot, extractDir, projectModule string,
	verbose bool,
) ([]*codemap.ModuleExtraction, int) {
	changedByMod := make(map[string][]string)
	for _, f := range append(changes.ChangedFiles, changes.NewFiles...) {
		mod := fileModule(f)
		changedByMod[mod] = append(changedByMod[mod], f)
	}
	deletedByMod := make(map[string][]string)
	for _, f := range changes.DeletedFiles {
		mod := fileModule(f)
		deletedByMod[mod] = append(deletedByMod[mod], f)
	}

	affectedMods := make(map[string]bool)
	for mod := range changedByMod {
		affectedMods[mod] = true
	}
	for mod := range deletedByMod {
		affectedMods[mod] = true
	}

	goModules := make(map[string][]string)
	for key, files := range allModules {
		if key.Language == "go" {
			goModules[key.Path] = files
		}
	}

	var extractions []*codemap.ModuleExtraction
	var totalSymbols int

	for mod := range goModules {
		if !affectedMods[mod] {
			existing, err := codemap.LoadExistingExtraction(extractDir, mod)
			if err != nil {
				if verbose {
					fmt.Fprintf(os.Stderr, "  unchanged module %s: no existing extraction, skipping\n", mod)
				}
				continue
			}
			for _, f := range existing.Files {
				totalSymbols += len(f.Symbols)
			}
			extractions = append(extractions, existing)
			continue
		}

		var result *codemap.ModuleExtraction

		if changedFiles := changedByMod[mod]; len(changedFiles) > 0 {
			partial, err := codemap.ExtractModule(mod, changedFiles, projectRoot, projectModule, verbose)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: incremental extract %s: %v\n", mod, err)
				if existing, loadErr := codemap.LoadExistingExtraction(extractDir, mod); loadErr == nil {
					extractions = append(extractions, existing)
				}
				continue
			}
			if existing, err := codemap.LoadExistingExtraction(extractDir, mod); err == nil {
				result = codemap.MergeExtraction(existing, partial)
			} else {
				result = partial
			}
		} else {
			existing, err := codemap.LoadExistingExtraction(extractDir, mod)
			if err != nil {
				continue
			}
			result = existing
		}

		if deleted := deletedByMod[mod]; len(deleted) > 0 {
			codemap.RemoveDeletedFiles(result, deleted)
		}

		if len(result.Files) == 0 {
			continue
		}

		if err := codemap.WriteModuleExtraction(extractDir, result); err != nil {
			fmt.Fprintf(os.Stderr, "error: write incremental %s: %v\n", mod, err)
			continue // skip module from extractions on write failure
		}
		for _, f := range result.Files {
			totalSymbols += len(f.Symbols)
		}
		extractions = append(extractions, result)
		if verbose {
			fmt.Fprintf(os.Stderr, "  incremental: %s: %d files\n", mod, len(result.Files))
		}
	}

	// Handle modules entirely removed from discovery.
	for mod, deleted := range deletedByMod {
		if _, exists := goModules[mod]; exists {
			continue
		}
		existing, err := codemap.LoadExistingExtraction(extractDir, mod)
		if err != nil {
			continue
		}
		codemap.RemoveDeletedFiles(existing, deleted)
		if len(existing.Files) == 0 {
			continue
		}
		if err := codemap.WriteModuleExtraction(extractDir, existing); err != nil {
			fmt.Fprintf(os.Stderr, "error: write incremental %s: %v\n", mod, err)
			continue // skip module from extractions on write failure
		}
		extractions = append(extractions, existing)
	}

	return extractions, totalSymbols
}
