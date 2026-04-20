package subcmd

import (
	"bytes"
	"context"
	"io"
	"testing"
)

// callRecorder captures whether a RunFunc was invoked and the args it received.
type callRecorder struct {
	called bool
	args   []string
}

func (rec *callRecorder) fn() RunFunc {
	return func(_ context.Context, args []string, _ io.Reader, _ io.Writer) error {
		rec.called = true
		rec.args = args
		return nil
	}
}

func newCtx() context.Context { return context.Background() }
func noIO() (io.Reader, io.Writer) {
	return bytes.NewReader(nil), io.Discard
}

// --- Dispatch tests ---

func TestDispatch_EmptyArgs(t *testing.T) {
	r := NewRegistry()
	stdin, stdout := noIO()
	err := r.Dispatch(newCtx(), nil, stdin, stdout)
	if err != ErrNoCommand {
		t.Fatalf("expected ErrNoCommand, got %v", err)
	}
}

func TestDispatch_UnknownCommand(t *testing.T) {
	r := NewRegistry()
	stdin, stdout := noIO()
	err := r.Dispatch(newCtx(), []string{"unknown"}, stdin, stdout)
	if err != ErrUnknownCommand {
		t.Fatalf("expected ErrUnknownCommand, got %v", err)
	}
}

func TestDispatch_FlatCommand(t *testing.T) {
	r := NewRegistry()
	rec := &callRecorder{}
	r.Register("greet", rec.fn())

	stdin, stdout := noIO()
	err := r.Dispatch(newCtx(), []string{"greet", "world"}, stdin, stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rec.called {
		t.Fatal("expected command to be called")
	}
	if len(rec.args) != 1 || rec.args[0] != "world" {
		t.Fatalf("expected args [world], got %v", rec.args)
	}
}

func TestDispatch_GroupCommand(t *testing.T) {
	r := NewRegistry()
	rec := &callRecorder{}
	r.RegisterGroup("hook", map[string]RunFunc{
		"validate": rec.fn(),
	})

	stdin, stdout := noIO()
	err := r.Dispatch(newCtx(), []string{"hook", "validate", "arg1"}, stdin, stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rec.called {
		t.Fatal("expected command to be called")
	}
	if len(rec.args) != 1 || rec.args[0] != "arg1" {
		t.Fatalf("expected args [arg1], got %v", rec.args)
	}
}

func TestDispatch_GroupPrefixWithoutSubcommand(t *testing.T) {
	r := NewRegistry()
	r.RegisterGroup("hook", map[string]RunFunc{
		"validate": func(_ context.Context, _ []string, _ io.Reader, _ io.Writer) error { return nil },
	})

	stdin, stdout := noIO()
	err := r.Dispatch(newCtx(), []string{"hook"}, stdin, stdout)
	if err != ErrNoCommand {
		t.Fatalf("expected ErrNoCommand, got %v", err)
	}
}

func TestDispatch_GroupUnknownSubcommand(t *testing.T) {
	r := NewRegistry()
	r.RegisterGroup("hook", map[string]RunFunc{
		"validate": func(_ context.Context, _ []string, _ io.Reader, _ io.Writer) error { return nil },
	})

	stdin, stdout := noIO()
	err := r.Dispatch(newCtx(), []string{"hook", "nope"}, stdin, stdout)
	if err != ErrUnknownCommand {
		t.Fatalf("expected ErrUnknownCommand, got %v", err)
	}
}

// --- WrapMain tests ---

func TestWrapMain_CallsFunc(t *testing.T) {
	called := false
	fn := WrapMain(func() { called = true })

	stdin, stdout := noIO()
	err := fn(newCtx(), nil, stdin, stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected wrapped func to be called")
	}
}

// --- argv0 tests ---

func buildArgv0Registry() *Registry {
	r := NewRegistry()
	r.Register("ml-export", func(_ context.Context, _ []string, _ io.Reader, _ io.Writer) error { return nil })
	r.RegisterGroup("hook", map[string]RunFunc{
		"validate": func(_ context.Context, _ []string, _ io.Reader, _ io.Writer) error { return nil },
	})
	return r
}

func TestDispatchByArgv0(t *testing.T) {
	tests := []struct {
		executable string
		wantFound  bool
	}{
		{"goyoke", false},
		{"/usr/bin/goyoke", false},
		{"goyoke-validate", true},
		{"goyoke-validate.exe", true},
		{"goyoke-ml-export", true},
		{"goyoke-ml-export.exe", true},
		{"unknown-binary", false},
		{"other-tool", false},
	}

	r := buildArgv0Registry()

	for _, tc := range tests {
		t.Run(tc.executable, func(t *testing.T) {
			fn, _, found := DispatchByArgv0(tc.executable, r)
			if found != tc.wantFound {
				t.Fatalf("DispatchByArgv0(%q): found=%v, want %v", tc.executable, found, tc.wantFound)
			}
			if tc.wantFound && fn == nil {
				t.Fatal("expected non-nil RunFunc when found=true")
			}
		})
	}
}

// --- Registry lookup tests ---

func TestLookupFlat(t *testing.T) {
	r := NewRegistry()
	rec := &callRecorder{}
	r.Register("cmd", rec.fn())

	fn, ok := r.LookupFlat("cmd")
	if !ok || fn == nil {
		t.Fatal("expected flat lookup to succeed")
	}
	if _, ok := r.LookupFlat("missing"); ok {
		t.Fatal("expected flat lookup to fail for unknown name")
	}
}

func TestLookupInGroups(t *testing.T) {
	r := NewRegistry()
	rec := &callRecorder{}
	r.RegisterGroup("hook", map[string]RunFunc{"validate": rec.fn()})

	fn, group, ok := r.LookupInGroups("validate")
	if !ok || fn == nil {
		t.Fatal("expected group lookup to succeed")
	}
	if group != "hook" {
		t.Fatalf("expected group 'hook', got %q", group)
	}
	if _, _, ok := r.LookupInGroups("missing"); ok {
		t.Fatal("expected group lookup to fail for unknown name")
	}
}
