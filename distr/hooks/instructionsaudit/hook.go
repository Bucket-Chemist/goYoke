// Package instructionsaudit registers the gogent-instructions-audit command in the multi-call dispatch table.
package instructionsaudit

import (
	"github.com/Bucket-Chemist/goYoke/distr/multicall"
	instructionsauditlib "github.com/Bucket-Chemist/goYoke/internal/hooks/instructionsaudit"
)

func init() { multicall.Register("gogent-instructions-audit", Main) }

func Main() { instructionsauditlib.Main() }
