// Package directimplcheck registers the goyoke-direct-impl-check command in the multi-call dispatch table.
package directimplcheck

import (
	"github.com/Bucket-Chemist/goYoke/distr/multicall"
	directimplchecklib "github.com/Bucket-Chemist/goYoke/internal/hooks/directimplcheck"
)

func init() { multicall.Register("goyoke-direct-impl-check", Main) }

func Main() { directimplchecklib.Main() }
