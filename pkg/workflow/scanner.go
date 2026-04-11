package workflow

import (
	"bufio"
	"io"
)

const maxWorkflowLineBytes = 10 * 1024 * 1024

func newWorkflowScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxWorkflowLineBytes)
	return scanner
}
