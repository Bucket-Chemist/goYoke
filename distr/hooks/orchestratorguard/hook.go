// Package orchestratorguard registers the gogent-orchestrator-guard command in the multi-call dispatch table.
package orchestratorguard

import (
	"github.com/Bucket-Chemist/goYoke/distr/multicall"
	orchestratorguardlib "github.com/Bucket-Chemist/goYoke/internal/hooks/orchestratorguard"
)

func init() { multicall.Register("gogent-orchestrator-guard", Main) }

func Main() { orchestratorguardlib.Main() }
