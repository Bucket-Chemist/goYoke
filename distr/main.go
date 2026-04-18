// Package main is the entry point for the gofortress multi-call binary.
//
// The binary dispatches based on the name it is invoked with (os.Args[0]).
// When installed, each hook is a symlink to this binary:
//
//	ln -s gofortress gogent-validate
//	ln -s gofortress gogent-archive
//	# …etc.
//
// Each blank import below triggers the hook package's init() function, which
// calls multicall.Register.  Adding a new hook in DIST-002 through DIST-006
// only requires adding one import here — no central map literal to merge.
package main

import (
	"github.com/Bucket-Chemist/goYoke/distr/multicall"

	// Hook packages — each init() calls multicall.Register("<binary-name>", Main).
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/agentendstate"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/archive"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/configguard"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/directimplcheck"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/doctheater"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/instructionsaudit"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/loadcontext"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/logreview"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/mlexport"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/orchestratorguard"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/permissiongate"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/planimpl"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/scout"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/sharpedge"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/skillguard"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/teamrun"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/updatereviewoutcome"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/validate"
	_ "github.com/Bucket-Chemist/goYoke/distr/hooks/version"
)

func main() {
	multicall.Dispatch()
}
