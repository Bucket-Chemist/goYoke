package telemetry

import (
	"bufio"
	"io"
)

const maxTelemetryLineBytes = 10 * 1024 * 1024

func newTelemetryScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxTelemetryLineBytes)
	return scanner
}
