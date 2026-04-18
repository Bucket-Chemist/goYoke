// Package loadcontext registers the goyoke-load-context command in the multi-call dispatch table.
package loadcontext

import (
	"github.com/Bucket-Chemist/goYoke/distr/multicall"
	loadcontextlib "github.com/Bucket-Chemist/goYoke/internal/hooks/loadcontext"
)

func init() { multicall.Register("goyoke-load-context", Main) }

func Main() { loadcontextlib.Main() }
