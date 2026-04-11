package session

import (
	"bufio"
	"io"
)

const maxSessionLineBytes = 10 * 1024 * 1024

func newSessionScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxSessionLineBytes)
	return scanner
}
