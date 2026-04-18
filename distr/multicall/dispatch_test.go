package multicall_test

import (
	"testing"

	"github.com/Bucket-Chemist/goYoke/distr/multicall"
)

func TestRegisterAndLookup(t *testing.T) {
	called := false
	multicall.Register("test-register-lookup", func() { called = true })

	fn, ok := multicall.Lookup("test-register-lookup")
	if !ok {
		t.Fatal("expected to find registered command")
	}
	fn()
	if !called {
		t.Fatal("expected registered function to be called")
	}
}

func TestLookupMissing(t *testing.T) {
	_, ok := multicall.Lookup("no-such-command-xyzzy")
	if ok {
		t.Fatal("expected Lookup to return false for unknown command")
	}
}

func TestAll(t *testing.T) {
	multicall.Register("test-all-alpha", func() {})
	multicall.Register("test-all-beta", func() {})

	all := multicall.All()
	if _, ok := all["test-all-alpha"]; !ok {
		t.Error("expected test-all-alpha in All()")
	}
	if _, ok := all["test-all-beta"]; !ok {
		t.Error("expected test-all-beta in All()")
	}
}

func TestAllReturnsCopy(t *testing.T) {
	multicall.Register("test-copy-src", func() {})

	snapshot := multicall.All()
	// Mutating the snapshot must not bleed into the registry.
	snapshot["injected-key"] = func() {}

	if _, ok := multicall.Lookup("injected-key"); ok {
		t.Error("expected modifying All() result to not affect registry")
	}
}

func TestRegisterOverwrite(t *testing.T) {
	var calls []string
	multicall.Register("test-overwrite", func() { calls = append(calls, "first") })
	multicall.Register("test-overwrite", func() { calls = append(calls, "second") })

	fn, ok := multicall.Lookup("test-overwrite")
	if !ok {
		t.Fatal("expected to find overwritten command")
	}
	fn()
	if len(calls) != 1 || calls[0] != "second" {
		t.Fatalf("expected second registration to win, got %v", calls)
	}
}
