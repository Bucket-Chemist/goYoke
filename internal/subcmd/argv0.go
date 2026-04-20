package subcmd

import (
	"path/filepath"
	"strings"
)

const mainBinaryName = "goyoke"

// DispatchByArgv0 checks whether the executable name maps to a known command.
// It strips the "goyoke-" prefix and ".exe" suffix, then searches the registry.
//
// Returns the matched RunFunc, the args to forward, and true when found.
// Returns nil, nil, false when the executable is the main binary or unrecognised.
func DispatchByArgv0(executable string, r *Registry) (RunFunc, []string, bool) {
	name := filepath.Base(executable)
	name = strings.TrimSuffix(name, ".exe")

	if name == mainBinaryName {
		return nil, nil, false
	}

	suffix := strings.TrimPrefix(name, mainBinaryName+"-")
	if suffix == name {
		// No "goyoke-" prefix — not a multicall invocation we own.
		return nil, nil, false
	}

	// Try flat command first (e.g. "goyoke-ml-export" → flat "ml-export").
	if fn, ok := r.LookupFlat(suffix); ok {
		return fn, nil, true
	}

	// Then search all groups (e.g. "goyoke-validate" → group "hook", cmd "validate").
	if fn, _, ok := r.LookupInGroups(suffix); ok {
		return fn, nil, true
	}

	return nil, nil, false
}
