package main

import (
	"io"
	"time"

	configguardlib "github.com/Bucket-Chemist/GOgent-Fortress/internal/hooks/configguard"
)

// configChangeEvent aliases the library type for package-internal tests.
type configChangeEvent = configguardlib.ConfigChangeEvent

func readEvent(r io.Reader, timeout time.Duration) (*configChangeEvent, error) {
	return configguardlib.ReadEvent(r, timeout)
}

func decision(event *configChangeEvent) (bool, string) {
	return configguardlib.Decision(event)
}

func main() { configguardlib.Main() }
