package codemap

import (
	"regexp"
	"sort"
	"strings"
)

// nodeBuiltins lists Node.js built-in module names for TypeScript stdlib classification.
var nodeBuiltins = map[string]bool{
	"assert": true, "buffer": true, "child_process": true, "cluster": true,
	"console": true, "constants": true, "crypto": true, "dgram": true,
	"dns": true, "domain": true, "events": true, "fs": true, "http": true,
	"https": true, "module": true, "net": true, "os": true, "path": true,
	"perf_hooks": true, "process": true, "punycode": true, "querystring": true,
	"readline": true, "repl": true, "stream": true, "string_decoder": true,
	"timers": true, "tls": true, "trace_events": true, "tty": true, "url": true,
	"util": true, "v8": true, "vm": true, "worker_threads": true, "zlib": true,
}

// pyStdlib lists Python standard library top-level module names.
var pyStdlib = map[string]bool{
	"abc": true, "ast": true, "asyncio": true, "builtins": true,
	"codecs": true, "collections": true, "concurrent": true, "contextlib": true,
	"copy": true, "csv": true, "ctypes": true, "dataclasses": true,
	"datetime": true, "decimal": true, "difflib": true, "dis": true,
	"email": true, "enum": true, "errno": true, "functools": true,
	"gc": true, "glob": true, "gzip": true, "hashlib": true, "heapq": true,
	"hmac": true, "html": true, "http": true, "importlib": true,
	"inspect": true, "io": true, "ipaddress": true, "itertools": true,
	"json": true, "linecache": true, "locale": true, "logging": true,
	"math": true, "mimetypes": true, "multiprocessing": true, "operator": true,
	"os": true, "pathlib": true, "pickle": true, "platform": true,
	"pprint": true, "queue": true, "random": true, "re": true,
	"shlex": true, "shutil": true, "signal": true, "socket": true,
	"sqlite3": true, "ssl": true, "stat": true, "statistics": true,
	"string": true, "struct": true, "subprocess": true, "sys": true,
	"tempfile": true, "threading": true, "time": true, "tomllib": true,
	"traceback": true, "types": true, "typing": true, "unicodedata": true,
	"unittest": true, "urllib": true, "uuid": true, "warnings": true,
	"weakref": true, "xml": true, "zipfile": true, "zlib": true,
	"zoneinfo": true,
}

// rBasePackages lists R base/recommended packages treated as stdlib.
var rBasePackages = map[string]bool{
	"base": true, "compiler": true, "datasets": true, "grDevices": true,
	"graphics": true, "grid": true, "methods": true, "parallel": true,
	"splines": true, "stats": true, "stats4": true, "tools": true,
	"translations": true, "utils": true,
}

var (
	rustUseRe  = regexp.MustCompile(`(?m)^use\s+([\w:]+(?:::\w+)*)`)
	tsImportRe = regexp.MustCompile(`(?m)^import\s+(?:.*?\s+from\s+)?['"]([^'"]+)['"]`)
	pyFromRe   = regexp.MustCompile(`(?m)^from\s+([\w.]+)\s+import`)
	pyImportRe = regexp.MustCompile(`(?m)^import\s+([\w.]+(?:\s*,\s*[\w.]+)*)`)
	rLibraryRe = regexp.MustCompile(`(?m)(?:library|require)\s*\(\s*["']?([\w.]+)["']?\s*\)`)
)

// ExtractImports extracts import classifications for a non-Go source file using regex.
func ExtractImports(source []byte, lang string, projectName string) ImportGraph {
	switch lang {
	case "rust":
		return extractRustImports(source, projectName)
	case "typescript":
		return extractTypeScriptImports(source)
	case "python":
		return extractPythonImports(source, projectName)
	case "r":
		return extractRImports(source)
	}
	return ImportGraph{}
}

func extractRustImports(source []byte, projectName string) ImportGraph {
	seen := make(map[string]bool)
	var internal, external, stdlib []string

	for _, match := range rustUseRe.FindAllSubmatch(source, -1) {
		if len(match) < 2 {
			continue
		}
		path := string(match[1])
		root := strings.SplitN(path, "::", 2)[0]
		if seen[root] {
			continue
		}
		seen[root] = true

		switch {
		case root == "crate" || root == "super" || root == "self":
			internal = append(internal, root)
		case root == "std" || root == "core" || root == "alloc":
			stdlib = append(stdlib, root)
		case projectName != "" && strings.EqualFold(root, projectName):
			internal = append(internal, root)
		default:
			external = append(external, root)
		}
	}
	sort.Strings(internal)
	sort.Strings(external)
	sort.Strings(stdlib)
	return ImportGraph{Internal: internal, External: external, Stdlib: stdlib}
}

func extractTypeScriptImports(source []byte) ImportGraph {
	seen := make(map[string]bool)
	var internal, external, stdlib []string

	for _, match := range tsImportRe.FindAllSubmatch(source, -1) {
		if len(match) < 2 {
			continue
		}
		path := string(match[1])

		// Normalise to the package name (drop sub-paths).
		var pkg string
		if strings.HasPrefix(path, "@") {
			parts := strings.SplitN(path, "/", 3)
			if len(parts) >= 2 {
				pkg = parts[0] + "/" + parts[1]
			} else {
				pkg = path
			}
		} else {
			pkg = strings.SplitN(path, "/", 2)[0]
		}

		if seen[pkg] {
			continue
		}
		seen[pkg] = true

		switch {
		case strings.HasPrefix(path, "."):
			internal = append(internal, path)
		case nodeBuiltins[pkg] || strings.HasPrefix(path, "node:"):
			stdlib = append(stdlib, pkg)
		default:
			external = append(external, pkg)
		}
	}
	sort.Strings(internal)
	sort.Strings(external)
	sort.Strings(stdlib)
	return ImportGraph{Internal: internal, External: external, Stdlib: stdlib}
}

func extractPythonImports(source []byte, projectName string) ImportGraph {
	seen := make(map[string]bool)
	var internal, external, stdlib []string

	add := func(rawPath string) {
		rawPath = strings.TrimSpace(rawPath)
		if seen[rawPath] {
			return
		}
		seen[rawPath] = true
		root := strings.SplitN(rawPath, ".", 2)[0]

		switch {
		case rawPath == "" || strings.HasPrefix(rawPath, "."):
			internal = append(internal, rawPath)
		case pyStdlib[root]:
			stdlib = append(stdlib, root)
		case projectName != "" && root == projectName:
			internal = append(internal, rawPath)
		default:
			external = append(external, root)
		}
	}

	for _, match := range pyFromRe.FindAllSubmatch(source, -1) {
		if len(match) >= 2 {
			add(string(match[1]))
		}
	}
	for _, match := range pyImportRe.FindAllSubmatch(source, -1) {
		if len(match) >= 2 {
			for mod := range strings.SplitSeq(string(match[1]), ",") {
				add(strings.TrimSpace(mod))
			}
		}
	}

	sort.Strings(internal)
	sort.Strings(external)
	sort.Strings(stdlib)
	return ImportGraph{Internal: internal, External: external, Stdlib: stdlib}
}

func extractRImports(source []byte) ImportGraph {
	seen := make(map[string]bool)
	var external, stdlib []string

	for _, match := range rLibraryRe.FindAllSubmatch(source, -1) {
		if len(match) < 2 {
			continue
		}
		pkg := string(match[1])
		if seen[pkg] {
			continue
		}
		seen[pkg] = true
		if rBasePackages[pkg] {
			stdlib = append(stdlib, pkg)
		} else {
			external = append(external, pkg)
		}
	}
	sort.Strings(external)
	sort.Strings(stdlib)
	return ImportGraph{External: external, Stdlib: stdlib}
}
