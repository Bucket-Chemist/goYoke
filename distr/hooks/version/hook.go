// Package version registers the goyoke-version command in the multi-call dispatch table.
package version

import (
	"github.com/Bucket-Chemist/goYoke/distr/multicall"
	versionlib "github.com/Bucket-Chemist/goYoke/internal/hooks/version"
)

func init() { multicall.Register("goyoke-version", Main) }

func Main() { versionlib.Main() }
