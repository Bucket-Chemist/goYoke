// Package agentendstate is a placeholder stub for the goyoke-agent-endstate hook.
// The real implementation will be wired in DIST-002 from internal/hooks/agentendstate.
package agentendstate

import (
	"fmt"

	"github.com/Bucket-Chemist/goYoke/distr/multicall"
)

func init() {
	multicall.Register("goyoke-agent-endstate", Main)
}

// Main is a stub. Replaced in DIST-002.
func Main() {
	fmt.Println("not yet wired: goyoke-agent-endstate")
}
