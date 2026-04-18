// Package doctheater registers the goyoke-doc-theater command in the multi-call dispatch table.
package doctheater

import (
	"github.com/Bucket-Chemist/goYoke/distr/multicall"
	doctheaterlib "github.com/Bucket-Chemist/goYoke/internal/hooks/doctheater"
)

func init() { multicall.Register("goyoke-doc-theater", Main) }

func Main() { doctheaterlib.Main() }
