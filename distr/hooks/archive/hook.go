// Package archive is a placeholder stub for the gogent-archive hook.
// The real implementation will be wired in DIST-002 from internal/hooks/archive.
package archive

import (
	"fmt"

	"github.com/Bucket-Chemist/GOgent-Fortress/distr/multicall"
)

func init() {
	multicall.Register("gogent-archive", Main)
}

// Main is a stub. Replaced in DIST-002.
func Main() {
	fmt.Println("not yet wired: gogent-archive")
}
