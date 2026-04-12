// Package version is a placeholder stub for the gogent-version hook.
// The real implementation will be wired in DIST-002 from internal/hooks/version.
package version

import (
	"fmt"

	"github.com/Bucket-Chemist/GOgent-Fortress/distr/multicall"
)

func init() {
	multicall.Register("gogent-version", Main)
}

// Main is a stub. Replaced in DIST-002.
func Main() {
	fmt.Println("not yet wired: gogent-version")
}
