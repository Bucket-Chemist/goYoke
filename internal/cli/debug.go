package cli

import (
	"fmt"
	"os"
)

// debugSubprocess controls whether subprocess debug logs are emitted to stderr.
// Enable with: GOFORTRESS_DEBUG_SUBPROCESS=1
var debugSubprocess = os.Getenv("GOFORTRESS_DEBUG_SUBPROCESS") != ""

// subprocessDebugLog writes debug output to stderr when GOFORTRESS_DEBUG_SUBPROCESS is set.
// Used for subprocess lifecycle tracing without polluting the TUI.
func subprocessDebugLog(format string, args ...interface{}) {
	if debugSubprocess {
		fmt.Fprintf(os.Stderr, "[SUBPROCESS] "+format, args...)
	}
}
