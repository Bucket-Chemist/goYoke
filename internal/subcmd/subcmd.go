package subcmd

import (
	"context"
	"errors"
	"io"
	"maps"
)

// RunFunc is the universal command signature for all subcmd commands.
type RunFunc func(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error

var (
	ErrNoCommand      = errors.New("no command specified")
	ErrUnknownCommand = errors.New("unknown command")
)

// Registry holds registered flat commands and command groups.
// Safe for concurrent reads after setup; do not register during dispatch.
type Registry struct {
	flat   map[string]RunFunc
	groups map[string]map[string]RunFunc
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		flat:   make(map[string]RunFunc),
		groups: make(map[string]map[string]RunFunc),
	}
}

// Register adds a flat command under the given name.
func (r *Registry) Register(name string, fn RunFunc) {
	r.flat[name] = fn
}

// RegisterGroup adds a set of commands under a named group prefix.
func (r *Registry) RegisterGroup(prefix string, commands map[string]RunFunc) {
	group := make(map[string]RunFunc, len(commands))
	maps.Copy(group, commands)
	r.groups[prefix] = group
}

// LookupFlat returns the RunFunc registered under name, if any.
func (r *Registry) LookupFlat(name string) (RunFunc, bool) {
	fn, ok := r.flat[name]
	return fn, ok
}

// LookupInGroups searches all groups for a command matching name.
// Returns the RunFunc, the group prefix it belongs to, and whether it was found.
func (r *Registry) LookupInGroups(name string) (RunFunc, string, bool) {
	for prefix, group := range r.groups {
		if fn, ok := group[name]; ok {
			return fn, prefix, true
		}
	}
	return nil, "", false
}

// Dispatch routes args to the appropriate registered command.
//
// Routing rules:
//   - len(args) == 0 → ErrNoCommand
//   - args[0] matches a group prefix and len(args) > 1 → call group[args[1]](args[2:])
//   - args[0] matches a group prefix and len(args) == 1 → ErrNoCommand
//   - args[0] matches a flat command → call flat[args[0]](args[1:])
//   - otherwise → ErrUnknownCommand
func (r *Registry) Dispatch(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 {
		return ErrNoCommand
	}

	cmd := args[0]

	if group, ok := r.groups[cmd]; ok {
		if len(args) == 1 {
			return ErrNoCommand
		}
		sub := args[1]
		fn, ok := group[sub]
		if !ok {
			return ErrUnknownCommand
		}
		return fn(ctx, args[2:], stdin, stdout)
	}

	if fn, ok := r.flat[cmd]; ok {
		return fn(ctx, args[1:], stdin, stdout)
	}

	return ErrUnknownCommand
}

// WrapMain adapts a zero-argument main function to the RunFunc signature.
// The wrapped function always returns nil.
func WrapMain(mainFn func()) RunFunc {
	return func(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
		mainFn()
		return nil
	}
}
