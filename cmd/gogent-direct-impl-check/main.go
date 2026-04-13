package main

import (
	directimplchecklib "github.com/Bucket-Chemist/GOgent-Fortress/internal/hooks/directimplcheck"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// Unexported shims delegate to the library so that package-internal tests
// continue to call the original function signatures.

func suggestAgent(filePath, content string) string {
	return directimplchecklib.SuggestAgent(filePath, content)
}

func isExcluded(filePath string, excludePatterns []string) bool {
	return directimplchecklib.IsExcluded(filePath, excludePatterns)
}

func isImplementationFile(filePath string, config *routing.DirectImplCheckConfig) bool {
	return directimplchecklib.IsImplementationFile(filePath, config)
}

func countLines(content string) int {
	return directimplchecklib.CountLines(content)
}

func main() { directimplchecklib.Main() }
