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
	"github.com/Bucket-Chemist/GOgent-Fortress/distr/multicall"

	// Hook packages — each init() calls multicall.Register("<binary-name>", Main).
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/agentendstate"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/archive"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/configguard"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/directimplcheck"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/doctheater"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/instructionsaudit"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/loadcontext"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/logreview"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/mlexport"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/orchestratorguard"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/permissiongate"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/planimpl"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/scout"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/sharpedge"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/skillguard"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/teamrun"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/updatereviewoutcome"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/validate"
	_ "github.com/Bucket-Chemist/GOgent-Fortress/distr/hooks/version"
)

func main() {
	multicall.Dispatch()
}
