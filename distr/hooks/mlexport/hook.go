// Package mlexport is a placeholder stub for the goyoke-ml-export hook.
// The real implementation will be wired in DIST-002 from internal/hooks/mlexport.
package mlexport

import (
	"fmt"

	"github.com/Bucket-Chemist/goYoke/distr/multicall"
)

func init() {
	multicall.Register("goyoke-ml-export", Main)
}

// Main is a stub. Replaced in DIST-002.
func Main() {
	fmt.Println("not yet wired: goyoke-ml-export")
}
