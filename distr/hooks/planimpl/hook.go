// Package planimpl is a placeholder stub for the gogent-plan-impl hook.
// The real implementation will be wired in DIST-002 from internal/hooks/planimpl.
package planimpl

import (
	"fmt"

	"github.com/Bucket-Chemist/GOgent-Fortress/distr/multicall"
)

func init() {
	multicall.Register("gogent-plan-impl", Main)
}

// Main is a stub. Replaced in DIST-002.
func Main() {
	fmt.Println("not yet wired: gogent-plan-impl")
}
