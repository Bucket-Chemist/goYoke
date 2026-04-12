// Package configguard is a placeholder stub for the gogent-config-guard hook.
// The real implementation will be wired in DIST-002 from internal/hooks/configguard.
package configguard

import (
	"fmt"

	"github.com/Bucket-Chemist/GOgent-Fortress/distr/multicall"
)

func init() {
	multicall.Register("gogent-config-guard", Main)
}

// Main is a stub. Replaced in DIST-002.
func Main() {
	fmt.Println("not yet wired: gogent-config-guard")
}
