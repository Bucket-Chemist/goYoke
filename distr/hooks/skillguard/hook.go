// Package skillguard is a placeholder stub for the gogent-skill-guard hook.
// The real implementation will be wired in DIST-002 from internal/hooks/skillguard.
package skillguard

import (
	"fmt"

	"github.com/Bucket-Chemist/goYoke/distr/multicall"
)

func init() {
	multicall.Register("gogent-skill-guard", Main)
}

// Main is a stub. Replaced in DIST-002.
func Main() {
	fmt.Println("not yet wired: gogent-skill-guard")
}
