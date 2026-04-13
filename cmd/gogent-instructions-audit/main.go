package main

import (
	"io"
	"time"

	instructionsauditlib "github.com/Bucket-Chemist/GOgent-Fortress/internal/hooks/instructionsaudit"
)

// instructionsLoadedEvent aliases the library type for package-internal tests.
type instructionsLoadedEvent = instructionsauditlib.InstructionsLoadedEvent

// auditRecord aliases the library type for package-internal tests.
type auditRecord = instructionsauditlib.AuditRecord

func readEvent(r io.Reader, timeout time.Duration) (*instructionsLoadedEvent, error) {
	return instructionsauditlib.ReadEvent(r, timeout)
}

func auditLogPath() string {
	return instructionsauditlib.AuditLogPath()
}

func appendAuditRecord(path string, record auditRecord) error {
	return instructionsauditlib.AppendAuditRecord(path, record)
}

func main() { instructionsauditlib.Main() }
