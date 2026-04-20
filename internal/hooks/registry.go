// Package hooks provides a registry wiring all hook implementations to the
// dispatch framework in internal/subcmd.
package hooks

import (
	"github.com/Bucket-Chemist/goYoke/internal/hooks/agentendstate"
	"github.com/Bucket-Chemist/goYoke/internal/hooks/archive"
	"github.com/Bucket-Chemist/goYoke/internal/hooks/configguard"
	"github.com/Bucket-Chemist/goYoke/internal/hooks/directimplcheck"
	"github.com/Bucket-Chemist/goYoke/internal/hooks/instructionsaudit"
	"github.com/Bucket-Chemist/goYoke/internal/hooks/loadcontext"
	"github.com/Bucket-Chemist/goYoke/internal/hooks/orchestratorguard"
	"github.com/Bucket-Chemist/goYoke/internal/hooks/permissiongate"
	"github.com/Bucket-Chemist/goYoke/internal/hooks/sharpedge"
	"github.com/Bucket-Chemist/goYoke/internal/hooks/skillguard"
	"github.com/Bucket-Chemist/goYoke/internal/hooks/validate"
	"github.com/Bucket-Chemist/goYoke/internal/subcmd"
)

// RegisterAll wires all 11 hook implementations into the dispatch registry.
func RegisterAll(r *subcmd.Registry) {
	r.RegisterGroup("hook", map[string]subcmd.RunFunc{
		"load-context":       subcmd.WrapMain(loadcontext.Main),
		"validate":           subcmd.WrapMain(validate.Main),
		"skill-guard":        subcmd.WrapMain(skillguard.Main),
		"direct-impl-check":  subcmd.WrapMain(directimplcheck.Main),
		"permission-gate":    subcmd.WrapMain(permissiongate.Main),
		"sharp-edge":         subcmd.WrapMain(sharpedge.Main),
		"agent-endstate":     subcmd.WrapMain(agentendstate.Main),
		"orchestrator-guard": subcmd.WrapMain(orchestratorguard.Main),
		"archive":            subcmd.WrapMain(archive.Main),
		"config-guard":       subcmd.WrapMain(configguard.Main),
		"instructions-audit": subcmd.WrapMain(instructionsaudit.Main),
	})
}
