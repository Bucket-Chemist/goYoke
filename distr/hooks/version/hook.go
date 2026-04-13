// Package version registers the gogent-version command in the multi-call dispatch table.
package version

import (
	"github.com/Bucket-Chemist/GOgent-Fortress/distr/multicall"
	versionlib "github.com/Bucket-Chemist/GOgent-Fortress/internal/hooks/version"
)

func init() { multicall.Register("gogent-version", Main) }

func Main() { versionlib.Main() }
