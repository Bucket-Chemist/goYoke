// Package loadcontext is a placeholder stub for the gogent-load-context hook.
// The real implementation will be wired in DIST-002 from internal/hooks/loadcontext.
package loadcontext

import (
	"fmt"

	"github.com/Bucket-Chemist/GOgent-Fortress/distr/multicall"
)

func init() {
	multicall.Register("gogent-load-context", Main)
}

// Main is a stub. Replaced in DIST-002.
func Main() {
	fmt.Println("not yet wired: gogent-load-context")
}
