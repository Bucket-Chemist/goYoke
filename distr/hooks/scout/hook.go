// Package scout is a placeholder stub for the goyoke-scout hook.
// The real implementation will be wired in DIST-002 from internal/hooks/scout.
package scout

import (
	"fmt"

	"github.com/Bucket-Chemist/goYoke/distr/multicall"
)

func init() {
	multicall.Register("goyoke-scout", Main)
}

// Main is a stub. Replaced in DIST-002.
func Main() {
	fmt.Println("not yet wired: goyoke-scout")
}
