package resolve

import (
	"errors"
	"io/fs"
	"sort"
)

// Resolver provides layered config file resolution over two fs.ReadFileFS instances.
// userFS takes priority over embedFS in all operations. Either layer may be nil.
type Resolver struct {
	userFS  fs.ReadFileFS
	embedFS fs.ReadFileFS
}

// New creates a Resolver wrapping the given filesystems.
// Either argument may be nil to operate in single-layer mode.
func New(userFS, embedFS fs.ReadFileFS) *Resolver {
	return &Resolver{userFS: userFS, embedFS: embedFS}
}

// ReadFile returns the contents of path from the first layer that contains it.
// userFS is checked before embedFS. Returns fs.ErrNotExist if neither has the file.
func (r *Resolver) ReadFile(path string) ([]byte, error) {
	if r.userFS != nil {
		data, err := r.userFS.ReadFile(path)
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	if r.embedFS != nil {
		data, err := r.embedFS.ReadFile(path)
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	return nil, fs.ErrNotExist
}

// ReadDir returns the union of directory entries from both layers.
// On name collision, the userFS entry wins. Results are sorted by name.
// Returns fs.ErrNotExist only if neither layer contains the path.
func (r *Resolver) ReadDir(path string) ([]fs.DirEntry, error) {
	seen := make(map[string]fs.DirEntry)
	found := false

	if r.userFS != nil {
		if entries, err := fs.ReadDir(r.userFS, path); err == nil {
			found = true
			for _, e := range entries {
				seen[e.Name()] = e
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	if r.embedFS != nil {
		if entries, err := fs.ReadDir(r.embedFS, path); err == nil {
			found = true
			for _, e := range entries {
				if _, exists := seen[e.Name()]; !exists {
					seen[e.Name()] = e
				}
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	if !found {
		return nil, fs.ErrNotExist
	}

	result := make([]fs.DirEntry, 0, len(seen))
	for _, e := range seen {
		result = append(result, e)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name() < result[j].Name()
	})
	return result, nil
}

// ReadFileAll returns the contents of path from every layer that contains it,
// ordered [userFS, embedFS]. Used for merge-by-ID consumers (e.g. agents-index.json).
// Returns fs.ErrNotExist if no layer has the file.
func (r *Resolver) ReadFileAll(path string) ([][]byte, error) {
	var results [][]byte

	if r.userFS != nil {
		data, err := r.userFS.ReadFile(path)
		if err == nil {
			results = append(results, data)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	if r.embedFS != nil {
		data, err := r.embedFS.ReadFile(path)
		if err == nil {
			results = append(results, data)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	if len(results) == 0 {
		return nil, fs.ErrNotExist
	}
	return results, nil
}

// HasFile reports whether path exists in any layer.
func (r *Resolver) HasFile(path string) bool {
	if r.userFS != nil {
		if _, err := r.userFS.ReadFile(path); err == nil {
			return true
		}
	}
	if r.embedFS != nil {
		if _, err := r.embedFS.ReadFile(path); err == nil {
			return true
		}
	}
	return false
}
