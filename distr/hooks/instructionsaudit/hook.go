// Package instructionsaudit is a placeholder stub for the gogent-instructions-audit hook.
// The real implementation will be wired in DIST-002 from internal/hooks/instructionsaudit.
package instructionsaudit

import (
	"fmt"

	"github.com/Bucket-Chemist/GOgent-Fortress/distr/multicall"
)

func init() {
	multicall.Register("gogent-instructions-audit", Main)
}

// Main is a stub. Replaced in DIST-002.
func Main() {
	fmt.Println("not yet wired: gogent-instructions-audit")
}
