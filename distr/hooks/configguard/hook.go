// Package configguard registers the goyoke-config-guard command in the multi-call dispatch table.
package configguard

import (
	"github.com/Bucket-Chemist/goYoke/distr/multicall"
	configguardlib "github.com/Bucket-Chemist/goYoke/internal/hooks/configguard"
)

func init() { multicall.Register("goyoke-config-guard", Main) }

func Main() { configguardlib.Main() }
