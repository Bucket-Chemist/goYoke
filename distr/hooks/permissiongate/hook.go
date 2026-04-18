// Package permissiongate registers the gogent-permission-gate command in the multi-call dispatch table.
package permissiongate

import (
	"github.com/Bucket-Chemist/goYoke/distr/multicall"
	permissiongatelib "github.com/Bucket-Chemist/goYoke/internal/hooks/permissiongate"
)

func init() { multicall.Register("gogent-permission-gate", Main) }

func Main() { permissiongatelib.Main() }
