package utils

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/internal/codemap"
)

// RunCodebaseExtract implements the goyoke-codebase-extract utility.
func RunCodebaseExtract(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("codebase-extract", flag.ContinueOnError)
	path := fs.String("path", ".", "Project root to scan")
	output := fs.String("output", ".claude/codebase-map/extract/", "Output directory")
	lang := fs.String("lang", "auto", "Force language (auto, go)")
	files := fs.String("files", "", "Extract specific files only, comma-separated")
	exclude := fs.String("exclude", "", "Additional exclude patterns, comma-separated")
	includeGenerated := fs.Bool("include-generated", false, "Include generated files")
	manifest := fs.String("manifest", ".claude/codebase-map/manifest.json", "Manifest file path")
	skipCallGraph := fs.Bool("skip-call-graph", false, "Skip call graph extraction")
	incremental := fs.Bool("incremental", false, "Only extract changed files")
	render := fs.Bool("render", false, "Generate Mermaid diagram files after extraction")
	enrich := fs.Bool("enrich", false, "Enrich extraction with LLM semantic analysis")
	narrate := fs.Bool("narrate", false, "Generate ARCHITECTURE.md narrative (implies --enrich)")
	budget := fs.Float64("budget", 0, "USD budget cap for LLM enrichment (0 = unlimited)")
	model := fs.String("model", "claude-sonnet-4-6", "Model to use for enrichment")
	showVersion := fs.Bool("version", false, "Print version and exit")
	verbose := fs.Bool("verbose", false, "Verbose output to stderr")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("codebase-extract: %w", err)
	}

	if *showVersion {
		fmt.Fprintln(stdout, "goyoke-codebase-extract dev")
		return nil
	}

	if *narrate {
		*enrich = true
	}

	if *enrich {
		if os.Getenv("ANTHROPIC_API_KEY") == "" {
			return fmt.Errorf("codebase-extract: ANTHROPIC_API_KEY environment variable is required for --enrich/--narrate")
		}
	}

	opts := codemap.DiscoveryOpts{
		IncludeGenerated: *includeGenerated,
		Lang:             *lang,
	}
	if *files != "" {
		opts.SpecificFiles = cbeSpitComma(*files)
	}
	if *exclude != "" {
		opts.ExcludePatterns = cbeSpitComma(*exclude)
	}

	modules, totalFiles, err := cbeRunDiscovery(*path, opts, *verbose)
	if err != nil {
		return fmt.Errorf("codebase-extract: %w", err)
	}

	projectModule, err := codemap.ResolveProjectModule(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "codebase-extract warning: could not resolve project module: %v\n", err)
		projectModule = ""
	}

	extractDir := filepath.Clean(*output)

	var extractions []*codemap.ModuleExtraction
	var totalSymbols int
	var didIncremental bool
	var existingManifest *codemap.Manifest

	if *incremental {
		extractions, totalSymbols, existingManifest, didIncremental = cbeTryIncrementalExtraction(
			*manifest, *path, extractDir, projectModule, modules, *verbose)
	}

	if !didIncremental {
		extractions, totalSymbols = cbeRunFullExtraction(modules, *path, extractDir, projectModule, *verbose)
	}

	fmt.Fprintf(os.Stderr, "extracted %d modules, %d symbols\n", len(extractions), totalSymbols)

	if !*skipCallGraph {
		if err := cbeRunCallGraph(extractions, *path, projectModule, extractDir, *verbose); err != nil {
			fmt.Fprintf(os.Stderr, "codebase-extract warning: call graph skipped (%v)\n", err)
		}
	}

	graph, graphPath, err := cbeRunOutputs(
		extractions, *path, extractDir, *manifest, "dev",
		didIncremental, existingManifest, *render, *verbose)
	if err != nil {
		return fmt.Errorf("codebase-extract: %w", err)
	}

	if *enrich {
		if err := cbeRunEnrichment(extractions, graph, graphPath, os.Getenv("ANTHROPIC_API_KEY"), *model, *budget, *narrate, extractDir, *path, *verbose); err != nil {
			fmt.Fprintf(os.Stderr, "codebase-extract error: enrichment: %v\n", err)
		}
	}

	fmt.Fprintf(stdout, "extraction complete: %d modules, %d files, %d symbols\n",
		len(extractions), totalFiles, totalSymbols)
	return nil
}

func cbeRunDiscovery(path string, opts codemap.DiscoveryOpts, verbose bool) (map[codemap.ModuleKey][]string, int, error) {
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

func cbeTryIncrementalExtraction(
	manifestPath, projectRoot, extractDir, projectModule string,
	allModules map[codemap.ModuleKey][]string,
	verbose bool,
) ([]*codemap.ModuleExtraction, int, *codemap.Manifest, bool) {
	em, mErr := codemap.LoadManifest(manifestPath)
	if mErr != nil || em == nil || em.GitCommit == "" || len(em.Modules) == 0 {
		return nil, 0, nil, false
	}

	changes, cErr := codemap.DetectChanges(em, projectRoot, allModules)
	switch {
	case cErr != nil:
		fmt.Fprintf(os.Stderr, "codebase-extract warning: change detection failed, full extraction: %v\n", cErr)
		return nil, 0, nil, false
	case changes.FullRemap:
		return nil, 0, nil, false
	case len(changes.ChangedFiles)+len(changes.NewFiles)+len(changes.DeletedFiles) == 0:
		fmt.Fprintln(os.Stderr, "no changes detected since last map")
		return nil, 0, nil, false // caller should handle exit
	}

	extractions, totalSymbols := cbeRunIncrementalExtraction(changes, allModules, projectRoot, extractDir, projectModule, verbose)
	return extractions, totalSymbols, em, true
}

func cbeRunFullExtraction(
	modules map[codemap.ModuleKey][]string,
	projectRoot, extractDir, projectModule string,
	verbose bool,
) ([]*codemap.ModuleExtraction, int) {
	var extractions []*codemap.ModuleExtraction
	var totalSymbols int
	ctagsWarned := false

	for key, filePaths := range modules {
		var extraction *codemap.ModuleExtraction
		var err error

		switch key.Language {
		case "go":
			extraction, err = codemap.ExtractModule(key.Path, filePaths, projectRoot, projectModule, verbose)
		default:
			if !codemap.CtagsAvailable() {
				if !ctagsWarned {
					fmt.Fprintln(os.Stderr, "codebase-extract warning: ctags not installed — skipping non-Go files")
					ctagsWarned = true
				}
				continue
			}
			projectName, _ := codemap.ResolveProjectName(projectRoot, key.Language)
			extraction, err = codemap.ExtractModuleCtags(key.Path, filePaths, key.Language, projectRoot, projectName, verbose)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "codebase-extract warning: module %s (%s): %v\n", key.Path, key.Language, err)
			continue
		}

		symbolCount := 0
		for _, f := range extraction.Files {
			symbolCount += len(f.Symbols)
		}
		totalSymbols += symbolCount

		if err := codemap.WriteModuleExtraction(extractDir, extraction); err != nil {
			fmt.Fprintf(os.Stderr, "codebase-extract error: write extraction %s: %v\n", extraction.Module, err)
			continue
		}
		extractions = append(extractions, extraction)
	}

	fmt.Fprintf(os.Stderr, "wrote %d extraction files to %s\n", len(extractions), extractDir)
	return extractions, totalSymbols
}

func cbeRunCallGraph(extractions []*codemap.ModuleExtraction, projectRoot, projectModule, extractDir string, verbose bool) error {
	var goExtractions []*codemap.ModuleExtraction
	for _, e := range extractions {
		if e.Language == "go" {
			goExtractions = append(goExtractions, e)
		}
	}
	if len(goExtractions) == 0 {
		return nil
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

	for _, e := range goExtractions {
		if err := codemap.WriteModuleExtraction(extractDir, e); err != nil {
			return fmt.Errorf("re-write extraction %s: %w", e.Module, err)
		}
	}
	return nil
}

func cbeRunOutputs(
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

	if render {
		mermaidDir := filepath.Join(filepath.Dir(extractDir), "mermaid")
		if err := codemap.GenerateMermaid(graph, mermaidDir); err != nil {
			return nil, "", fmt.Errorf("generate mermaid: %w", err)
		}
	}

	_ = verbose
	return graph, graphPath, nil
}

func cbeRunEnrichment(
	extractions []*codemap.ModuleExtraction,
	graph *codemap.Graph, graphPath string,
	apiKey, modelStr string, budget float64,
	narrate bool, extractDir, projectRoot string, verbose bool,
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
			fmt.Fprintf(os.Stderr, "codebase-extract warning: enrich module %s: %v\n", e.Module, err)
			continue
		}
		enrichedModules[e.Module] = em
		if err := codemap.WriteEnrichedModule(enrichedDir, em); err != nil {
			fmt.Fprintf(os.Stderr, "codebase-extract error: write enriched %s: %v\n", e.Module, err)
		}
	}

	enricher.EnrichGraph(graph, enrichedModules)
	if err := codemap.WriteGraph(graphPath, graph); err != nil {
		fmt.Fprintf(os.Stderr, "codebase-extract error: write enriched graph: %v\n", err)
	}

	if narrate {
		narrative, err := enricher.GenerateNarrative(graph, enrichedModules)
		if err != nil {
			fmt.Fprintf(os.Stderr, "codebase-extract error: generate narrative: %v\n", err)
		} else {
			repoName := codemap.ResolveRepoName(projectRoot)
			narrativePath := filepath.Join(projectRoot, "docs", "codebase-architecture", repoName, "ARCHITECTURE.md")
			if err := codemap.WriteNarrative(narrativePath, narrative); err != nil {
				fmt.Fprintf(os.Stderr, "codebase-extract error: write narrative: %v\n", err)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "codebase-extract enrichment cost: $%.4f\n", enricher.Spent())
	return nil
}

func cbeRunIncrementalExtraction(
	changes *codemap.ChangeSet,
	allModules map[codemap.ModuleKey][]string,
	projectRoot, extractDir, projectModule string,
	verbose bool,
) ([]*codemap.ModuleExtraction, int) {
	changedByMod := make(map[string][]string)
	for _, f := range append(changes.ChangedFiles, changes.NewFiles...) {
		mod := cbeFileModule(f)
		changedByMod[mod] = append(changedByMod[mod], f)
	}
	deletedByMod := make(map[string][]string)
	for _, f := range changes.DeletedFiles {
		mod := cbeFileModule(f)
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
				fmt.Fprintf(os.Stderr, "codebase-extract warning: incremental extract %s: %v\n", mod, err)
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
			fmt.Fprintf(os.Stderr, "codebase-extract error: write incremental %s: %v\n", mod, err)
			continue
		}
		for _, f := range result.Files {
			totalSymbols += len(f.Symbols)
		}
		extractions = append(extractions, result)
	}

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
			continue
		}
		extractions = append(extractions, existing)
	}

	return extractions, totalSymbols
}

func cbeSpitComma(s string) []string {
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

func cbeFileModule(relPath string) string {
	dir := filepath.ToSlash(filepath.Dir(relPath))
	if dir == "." {
		return ""
	}
	return dir
}
