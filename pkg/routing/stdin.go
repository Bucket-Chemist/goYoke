package routing

import (
	"bufio"
	"fmt"
	"io"
	"time"
)

// ReadStdin reads all data from reader with timeout protection.
// Returns error if no data received within timeout duration.
// This prevents hooks from hanging indefinitely (fixes M-6).
func ReadStdin(r io.Reader, timeout time.Duration) ([]byte, error) {
	type result struct {
		data []byte
		err  error
	}

	ch := make(chan result, 1)
	go func() {
		data, err := io.ReadAll(bufio.NewReader(r))
		ch <- result{data, err}
	}()

	select {
	case res := <-ch:
		if res.err != nil {
			return nil, fmt.Errorf("[stdin-reader] Failed to read input: %w. Ensure hook receives valid data.", res.err)
		}
		if len(res.data) == 0 {
			return nil, fmt.Errorf("[stdin-reader] No data received. Hook may not be receiving STDIN input.")
		}
		return res.data, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("[stdin-reader] Read timeout after %v. Hook did not receive input. Check hook configuration.", timeout)
	}
}
