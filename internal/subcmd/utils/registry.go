package utils

import (
	"github.com/Bucket-Chemist/goYoke/internal/hooks/logreview"
	"github.com/Bucket-Chemist/goYoke/internal/hooks/updatereviewoutcome"
	"github.com/Bucket-Chemist/goYoke/internal/hooks/version"
	"github.com/Bucket-Chemist/goYoke/internal/subcmd"
	"github.com/Bucket-Chemist/goYoke/internal/subcmd/utils/harness"
	"github.com/Bucket-Chemist/goYoke/internal/subcmd/utils/teampreparesynth"
	"github.com/Bucket-Chemist/goYoke/internal/subcmd/utils/teamrun"
)

// RegisterAll registers all utility commands with the dispatch registry.
func RegisterAll(r *subcmd.Registry) {
	// Already extracted in internal/hooks/ (wire via WrapMain)
	r.Register("version", subcmd.WrapMain(version.Main))
	r.Register("log-review", subcmd.WrapMain(logreview.Main))
	r.Register("update-review-outcome", subcmd.WrapMain(updatereviewoutcome.Main))

	// Multi-file sub-packages (wire via WrapMain)
	r.Register("team-run", subcmd.WrapMain(teamrun.Main))
	r.Register("team-prepare-synthesis", subcmd.WrapMain(teampreparesynth.Main))

	// Single-file utilities in this package
	r.Register("scout", RunScout)
	r.Register("plan-impl", RunPlanImpl)
	r.Register("aggregate", RunAggregate)
	r.Register("ml-export", RunMLExport)
	r.Register("validate-schemas", RunValidateSchemas)
	r.Register("capture-intent", RunCaptureIntent)
	r.Register("codebase-extract", RunCodebaseExtract)

	// Harness command group: goyoke harness <subcommand>
	r.RegisterGroup("harness", harness.Commands())
}
