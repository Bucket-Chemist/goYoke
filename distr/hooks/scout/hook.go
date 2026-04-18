// Package scout is a placeholder stub for the gogent-scout hook.
// The real implementation will be wired in DIST-002 from internal/hooks/scout.
package scout

import (
	"fmt"

	"github.com/Bucket-Chemist/goYoke/distr/multicall"
)

func init() {
	multicall.Register("gogent-scout", Main)
}

// Main is a stub. Replaced in DIST-002.
func Main() {
	fmt.Println("not yet wired: gogent-scout")
}
