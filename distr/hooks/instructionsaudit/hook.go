// Package instructionsaudit registers the gogent-instructions-audit command in the multi-call dispatch table.
package instructionsaudit

import (
	"github.com/Bucket-Chemist/GOgent-Fortress/distr/multicall"
	instructionsauditlib "github.com/Bucket-Chemist/GOgent-Fortress/internal/hooks/instructionsaudit"
)

func init() { multicall.Register("gogent-instructions-audit", Main) }

func Main() { instructionsauditlib.Main() }
