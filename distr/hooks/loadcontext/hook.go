// Package loadcontext registers the gogent-load-context command in the multi-call dispatch table.
package loadcontext

import (
	"github.com/Bucket-Chemist/GOgent-Fortress/distr/multicall"
	loadcontextlib "github.com/Bucket-Chemist/GOgent-Fortress/internal/hooks/loadcontext"
)

func init() { multicall.Register("gogent-load-context", Main) }

func Main() { loadcontextlib.Main() }
