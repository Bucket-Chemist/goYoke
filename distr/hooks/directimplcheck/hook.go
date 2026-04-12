// Package directimplcheck is a placeholder stub for the gogent-direct-impl-check hook.
// The real implementation will be wired in DIST-002 from internal/hooks/directimplcheck.
package directimplcheck

import (
	"fmt"

	"github.com/Bucket-Chemist/GOgent-Fortress/distr/multicall"
)

func init() {
	multicall.Register("gogent-direct-impl-check", Main)
}

// Main is a stub. Replaced in DIST-002.
func Main() {
	fmt.Println("not yet wired: gogent-direct-impl-check")
}
