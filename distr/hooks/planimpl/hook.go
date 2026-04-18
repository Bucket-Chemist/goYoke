// Package planimpl is a placeholder stub for the goyoke-plan-impl hook.
// The real implementation will be wired in DIST-002 from internal/hooks/planimpl.
package planimpl

import (
	"fmt"

	"github.com/Bucket-Chemist/goYoke/distr/multicall"
)

func init() {
	multicall.Register("goyoke-plan-impl", Main)
}

// Main is a stub. Replaced in DIST-002.
func Main() {
	fmt.Println("not yet wired: goyoke-plan-impl")
}
