package main

import (
	orchestratorguardlib "github.com/Bucket-Chemist/GOgent-Fortress/internal/hooks/orchestratorguard"
)

// Unexported shims delegate to the library so that package-internal tests
// continue to call the original function signatures.

func outputAllow(reason string)  { orchestratorguardlib.OutputAllow(reason) }
func outputError(message string) { orchestratorguardlib.OutputError(message) }
func escapeJSON(s string) string { return orchestratorguardlib.EscapeJSON(s) }

func main() { orchestratorguardlib.Main() }
