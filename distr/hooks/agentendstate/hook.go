// Package agentendstate is a placeholder stub for the gogent-agent-endstate hook.
// The real implementation will be wired in DIST-002 from internal/hooks/agentendstate.
package agentendstate

import (
	"fmt"

	"github.com/Bucket-Chemist/goYoke/distr/multicall"
)

func init() {
	multicall.Register("gogent-agent-endstate", Main)
}

// Main is a stub. Replaced in DIST-002.
func Main() {
	fmt.Println("not yet wired: gogent-agent-endstate")
}
