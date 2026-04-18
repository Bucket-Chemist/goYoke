// Package logreview registers the goyoke-log-review command in the multi-call dispatch table.
package logreview

import (
	"github.com/Bucket-Chemist/goYoke/distr/multicall"
	logreviewlib "github.com/Bucket-Chemist/goYoke/internal/hooks/logreview"
)

func init() { multicall.Register("goyoke-log-review", Main) }

func Main() { logreviewlib.Main() }
