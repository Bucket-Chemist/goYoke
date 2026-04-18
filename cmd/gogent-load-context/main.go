package main

import (
	loadcontextlib "github.com/Bucket-Chemist/goYoke/internal/hooks/loadcontext"
)

// DEFAULT_TIMEOUT aliases the library constant so package-internal tests can
// reference it by its original name.
const DEFAULT_TIMEOUT = loadcontextlib.DefaultTimeout

// outputError is an unexported shim for package-internal tests.
func outputError(message string) { loadcontextlib.OutputError(message) }

func main() { loadcontextlib.Main() }
