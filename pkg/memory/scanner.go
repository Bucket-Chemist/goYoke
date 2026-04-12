package memory

import (
	"bufio"
	"io"
)

const maxFailureTrackerLineBytes = 10 * 1024 * 1024

func newFailureScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxFailureTrackerLineBytes)
	return scanner
}
