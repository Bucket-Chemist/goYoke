// Package version registers the gogent-version command in the multi-call dispatch table.
package version

import (
	"github.com/Bucket-Chemist/goYoke/distr/multicall"
	versionlib "github.com/Bucket-Chemist/goYoke/internal/hooks/version"
)

func init() { multicall.Register("gogent-version", Main) }

func Main() { versionlib.Main() }
