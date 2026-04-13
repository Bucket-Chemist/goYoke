// Package directimplcheck registers the gogent-direct-impl-check command in the multi-call dispatch table.
package directimplcheck

import (
	"github.com/Bucket-Chemist/GOgent-Fortress/distr/multicall"
	directimplchecklib "github.com/Bucket-Chemist/GOgent-Fortress/internal/hooks/directimplcheck"
)

func init() { multicall.Register("gogent-direct-impl-check", Main) }

func Main() { directimplchecklib.Main() }
