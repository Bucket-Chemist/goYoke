package main

import (
	"github.com/Bucket-Chemist/goYoke/defaults"
	loadcontextlib "github.com/Bucket-Chemist/goYoke/internal/hooks/loadcontext"
	"github.com/Bucket-Chemist/goYoke/pkg/resolve"
)

// DEFAULT_TIMEOUT aliases the library constant so package-internal tests can
// reference it by its original name.
const DEFAULT_TIMEOUT = loadcontextlib.DefaultTimeout

// outputError is an unexported shim for package-internal tests.
func outputError(message string) { loadcontextlib.OutputError(message) }

func main() {
	resolve.SetDefault(defaults.FS)
	loadcontextlib.Main()
}
