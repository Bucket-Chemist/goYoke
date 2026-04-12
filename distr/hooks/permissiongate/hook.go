// Package permissiongate is a placeholder stub for the gogent-permission-gate hook.
// The real implementation will be wired in DIST-002 from internal/hooks/permissiongate.
package permissiongate

import (
	"fmt"

	"github.com/Bucket-Chemist/GOgent-Fortress/distr/multicall"
)

func init() {
	multicall.Register("gogent-permission-gate", Main)
}

// Main is a stub. Replaced in DIST-002.
func Main() {
	fmt.Println("not yet wired: gogent-permission-gate")
}
