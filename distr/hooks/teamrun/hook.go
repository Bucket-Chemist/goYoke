// Package teamrun is a placeholder stub for the goyoke-team-run hook.
// The real implementation will be wired in DIST-002 from internal/hooks/teamrun.
package teamrun

import (
	"fmt"

	"github.com/Bucket-Chemist/goYoke/distr/multicall"
)

func init() {
	multicall.Register("goyoke-team-run", Main)
}

// Main is a stub. Replaced in DIST-002.
func Main() {
	fmt.Println("not yet wired: goyoke-team-run")
}
