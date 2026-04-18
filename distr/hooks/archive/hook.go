// Package archive is a placeholder stub for the goyoke-archive hook.
// The real implementation will be wired in DIST-002 from internal/hooks/archive.
package archive

import (
	"fmt"

	"github.com/Bucket-Chemist/goYoke/distr/multicall"
)

func init() {
	multicall.Register("goyoke-archive", Main)
}

// Main is a stub. Replaced in DIST-002.
func Main() {
	fmt.Println("not yet wired: goyoke-archive")
}
