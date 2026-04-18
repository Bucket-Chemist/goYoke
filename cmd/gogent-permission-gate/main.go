package main

import (
	"io"
	"time"

	permissiongatelib "github.com/Bucket-Chemist/goYoke/internal/hooks/permissiongate"
)

// toolEvent aliases the library type so package-internal tests continue to use
// the original unexported name.
type toolEvent = permissiongatelib.ToolEvent

// Constant and variable aliases so package-internal tests reference original names.
const (
	classAutoAllow     = permissiongatelib.ClassAutoAllow
	classNeedsApproval = permissiongatelib.ClassNeedsApproval
	classSkip          = permissiongatelib.ClassSkip
	defaultPermTimeout = permissiongatelib.DefaultPermTimeout
)

var defaultPolicy = permissiongatelib.DefaultPolicy

// Unexported shims delegate to the library so package-internal tests compile
// against the original function signatures.

func parseStdin(r io.Reader) (*toolEvent, []byte, error) {
	return permissiongatelib.ParseStdin(r)
}

func cachePath(sessionID string) string {
	return permissiongatelib.CachePath(sessionID)
}

// CheckCache and WriteCache are re-exported (capital-letter names preserved from
// original cache.go) so tests in package main can call them directly.
func CheckCache(sessionID, toolName string) (string, bool) {
	return permissiongatelib.CheckCache(sessionID, toolName)
}

func WriteCache(sessionID, toolName, decision string) {
	permissiongatelib.WriteCache(sessionID, toolName, decision)
}

func RequestPermission(toolName string, toolInputJSON []byte, sessionID string) (string, error) {
	return permissiongatelib.RequestPermission(toolName, toolInputJSON, sessionID)
}

func allow()                  { permissiongatelib.Allow() }
func denyWithReason(r string) { permissiongatelib.DenyWithReason(r) }

func extractCommand(toolInput map[string]interface{}) string {
	return permissiongatelib.ExtractCommand(toolInput)
}

func permTimeout() time.Duration {
	return permissiongatelib.PermTimeout()
}

func main() { permissiongatelib.Main() }
