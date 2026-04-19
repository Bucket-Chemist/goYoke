package resolve

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// diskFS wraps fs.FS and implements fs.ReadFileFS.
// os.DirFS returns fs.FS (not fs.ReadFileFS), so this wrapper is required.
type diskFS struct {
	inner fs.FS
}

// Compile-time assertion: diskFS satisfies fs.ReadFileFS.
var _ fs.ReadFileFS = diskFS{}

func (d diskFS) Open(name string) (fs.File, error) {
	return d.inner.Open(name)
}

func (d diskFS) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(d.inner, name)
}

var (
	defaultResolver *Resolver
	defaultOnce     sync.Once
	defaultMu       sync.Mutex
	defaultSet      bool // SetDefault was explicitly called
	defaultLazied   bool // Default() lazily initialized without SetDefault
)

// SetDefault creates a Resolver with disk root as userFS and the given embedFS,
// then stores it as the package default. Must be called before Default() is first
// used. Panics if Default() already lazily initialized the singleton, or if
// SetDefault was already called.
func SetDefault(embedFS fs.ReadFileFS) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if defaultLazied {
		panic("resolve: SetDefault called after Default() was already lazily initialized")
	}
	if defaultSet {
		panic("resolve: SetDefault called more than once")
	}
	defaultSet = true
	defaultResolver = New(newDiskFS(), embedFS)
}

// Default returns the package-level Resolver. If SetDefault was called first, that
// resolver is returned. Otherwise, a disk-only resolver (no embed layer) is created
// lazily on the first call using sync.Once.
func Default() *Resolver {
	defaultOnce.Do(func() {
		defaultMu.Lock()
		defer defaultMu.Unlock()
		if !defaultSet {
			defaultLazied = true
			defaultResolver = New(newDiskFS(), nil)
		}
	})
	return defaultResolver
}

// ResetDefault resets the package-level singleton for test isolation.
// Must only be called from tests.
func ResetDefault() {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultResolver = nil
	defaultOnce = sync.Once{}
	defaultSet = false
	defaultLazied = false
}

// newDiskFS constructs an fs.ReadFileFS rooted at the resolved config directory.
// Resolution order:
//  1. $GOYOKE_PROJECT_DIR/.claude/
//  2. $CLAUDE_CONFIG_DIR
//  3. ~/.claude/
func newDiskFS() fs.ReadFileFS {
	if dir := os.Getenv("GOYOKE_PROJECT_DIR"); dir != "" {
		return diskFS{inner: os.DirFS(filepath.Join(dir, ".claude"))}
	}
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return diskFS{inner: os.DirFS(dir)}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return diskFS{inner: os.DirFS(".")}
	}
	return diskFS{inner: os.DirFS(filepath.Join(home, ".claude"))}
}
