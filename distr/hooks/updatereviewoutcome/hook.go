// Package updatereviewoutcome registers the goyoke-update-review-outcome command in the multi-call dispatch table.
package updatereviewoutcome

import (
	"github.com/Bucket-Chemist/goYoke/distr/multicall"
	updatereviewoutcomelib "github.com/Bucket-Chemist/goYoke/internal/hooks/updatereviewoutcome"
)

func init() { multicall.Register("goyoke-update-review-outcome", Main) }

func Main() { updatereviewoutcomelib.Main() }
