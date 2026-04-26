// Package harnesses provides the embedded asset tree for harness adapters.
// This package is managed separately from the main defaults package so that
// harness templates remain independent of the Claude runtime asset tree.
package harnesses

import "embed"

//go:embed manual hermes
var FS embed.FS
