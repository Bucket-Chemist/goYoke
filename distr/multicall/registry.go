// Package multicall provides the dispatch table for the goyoke multi-call binary.
// It is imported by distr/main.go, and by each hook stub in distr/hooks/*.
package multicall

import "sync"

var (
	mu       sync.RWMutex
	registry = map[string]func(){}
)

// Register maps name to fn in the global command registry.
// It is intended to be called from init() functions in hook packages so that
// each hook self-registers without requiring a central list:
//
//	func init() { multicall.Register("goyoke-validate", Main) }
func Register(name string, fn func()) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = fn
}

// Lookup returns the registered function for name, if any.
func Lookup(name string) (func(), bool) {
	mu.RLock()
	defer mu.RUnlock()
	fn, ok := registry[name]
	return fn, ok
}

// All returns a shallow copy of the full registry map.
// Mutations to the returned map do not affect the registry.
func All() map[string]func() {
	mu.RLock()
	defer mu.RUnlock()
	out := make(map[string]func(), len(registry))
	for k, v := range registry {
		out[k] = v
	}
	return out
}
