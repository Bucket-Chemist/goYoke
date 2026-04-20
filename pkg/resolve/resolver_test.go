package resolve

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
)

func userMap(files map[string]string) fs.ReadFileFS {
	m := fstest.MapFS{}
	for k, v := range files {
		m[k] = &fstest.MapFile{Data: []byte(v)}
	}
	return m
}

// --- ReadFile ---

func TestReadFile_UserFSOnly(t *testing.T) {
	r := New(userMap(map[string]string{"a.json": "user"}), nil)
	data, err := r.ReadFile("a.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "user" {
		t.Fatalf("got %q, want %q", data, "user")
	}
}

func TestReadFile_EmbedFSOnly(t *testing.T) {
	r := New(nil, userMap(map[string]string{"a.json": "embed"}))
	data, err := r.ReadFile("a.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "embed" {
		t.Fatalf("got %q, want %q", data, "embed")
	}
}

func TestReadFile_UserFSWinsOnCollision(t *testing.T) {
	r := New(
		userMap(map[string]string{"a.json": "user"}),
		userMap(map[string]string{"a.json": "embed"}),
	)
	data, err := r.ReadFile("a.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "user" {
		t.Fatalf("got %q, want user", data)
	}
}

func TestReadFile_NotExistInEither(t *testing.T) {
	r := New(userMap(nil), userMap(nil))
	_, err := r.ReadFile("missing.json")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got %v, want fs.ErrNotExist", err)
	}
}

func TestReadFile_FallsBackToEmbed(t *testing.T) {
	r := New(
		userMap(nil),
		userMap(map[string]string{"b.json": "embed"}),
	)
	data, err := r.ReadFile("b.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "embed" {
		t.Fatalf("got %q, want embed", data)
	}
}

// --- ReadDir ---

func TestReadDir_SingleLayer(t *testing.T) {
	r := New(userMap(map[string]string{"dir/a.txt": "a"}), nil)
	entries, err := r.ReadDir("dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "a.txt" {
		t.Fatalf("unexpected entries: %v", entries)
	}
}

func TestReadDir_Union(t *testing.T) {
	r := New(
		userMap(map[string]string{"dir/a.txt": "a"}),
		userMap(map[string]string{"dir/b.txt": "b"}),
	)
	entries, err := r.ReadDir("dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name() != "a.txt" || entries[1].Name() != "b.txt" {
		t.Fatalf("unexpected order: %v, %v", entries[0].Name(), entries[1].Name())
	}
}

func TestReadDir_CollisionUserFSWins(t *testing.T) {
	r := New(
		userMap(map[string]string{"dir/a.txt": "user"}),
		userMap(map[string]string{"dir/a.txt": "embed"}),
	)
	entries, err := r.ReadDir("dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	// Verify it's the userFS entry by reading through Resolver
	data, _ := r.ReadFile("dir/a.txt")
	if string(data) != "user" {
		t.Fatalf("collision resolution: got %q, want user", data)
	}
}

func TestReadDir_NeitherLayer(t *testing.T) {
	r := New(userMap(nil), userMap(nil))
	_, err := r.ReadDir("nonexistent")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got %v, want fs.ErrNotExist", err)
	}
}

func TestReadDir_Sorted(t *testing.T) {
	r := New(
		userMap(map[string]string{"dir/c.txt": "", "dir/a.txt": ""}),
		userMap(map[string]string{"dir/b.txt": ""}),
	)
	entries, err := r.ReadDir("dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Fatalf("entries not sorted: %v", names)
		}
	}
}

// --- ReadFileAll ---

func TestReadFileAll_BothLayers(t *testing.T) {
	r := New(
		userMap(map[string]string{"a.json": "user"}),
		userMap(map[string]string{"a.json": "embed"}),
	)
	all, err := r.ReadFileAll("a.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 results, got %d", len(all))
	}
	if string(all[0]) != "user" {
		t.Fatalf("all[0] = %q, want user", all[0])
	}
	if string(all[1]) != "embed" {
		t.Fatalf("all[1] = %q, want embed", all[1])
	}
}

func TestReadFileAll_UserFSOnly(t *testing.T) {
	r := New(userMap(map[string]string{"a.json": "user"}), userMap(nil))
	all, err := r.ReadFileAll("a.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 1 || string(all[0]) != "user" {
		t.Fatalf("unexpected result: %v", all)
	}
}

func TestReadFileAll_EmbedFSOnly(t *testing.T) {
	r := New(userMap(nil), userMap(map[string]string{"a.json": "embed"}))
	all, err := r.ReadFileAll("a.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 1 || string(all[0]) != "embed" {
		t.Fatalf("unexpected result: %v", all)
	}
}

func TestReadFileAll_NeitherLayer(t *testing.T) {
	r := New(userMap(nil), userMap(nil))
	_, err := r.ReadFileAll("missing.json")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got %v, want fs.ErrNotExist", err)
	}
}

// --- HasFile ---

func TestHasFile_InUserFS(t *testing.T) {
	r := New(userMap(map[string]string{"x.json": ""}), nil)
	if !r.HasFile("x.json") {
		t.Fatal("expected true")
	}
}

func TestHasFile_InEmbedFS(t *testing.T) {
	r := New(nil, userMap(map[string]string{"x.json": ""}))
	if !r.HasFile("x.json") {
		t.Fatal("expected true")
	}
}

func TestHasFile_InBoth(t *testing.T) {
	r := New(
		userMap(map[string]string{"x.json": ""}),
		userMap(map[string]string{"x.json": ""}),
	)
	if !r.HasFile("x.json") {
		t.Fatal("expected true")
	}
}

func TestHasFile_InNeither(t *testing.T) {
	r := New(userMap(nil), userMap(nil))
	if r.HasFile("missing.json") {
		t.Fatal("expected false")
	}
}

// --- Nil layer edge cases ---

func TestNilUserFS_ReadFile(t *testing.T) {
	r := New(nil, userMap(map[string]string{"a.json": "embed"}))
	data, err := r.ReadFile("a.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "embed" {
		t.Fatalf("got %q, want embed", data)
	}
}

func TestNilEmbedFS_ReadFile(t *testing.T) {
	r := New(userMap(map[string]string{"a.json": "user"}), nil)
	data, err := r.ReadFile("a.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "user" {
		t.Fatalf("got %q, want user", data)
	}
}

func TestBothNil_ReadFile(t *testing.T) {
	r := New(nil, nil)
	_, err := r.ReadFile("any.json")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got %v, want fs.ErrNotExist", err)
	}
}

func TestBothNil_ReadDir(t *testing.T) {
	r := New(nil, nil)
	_, err := r.ReadDir("any")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got %v, want fs.ErrNotExist", err)
	}
}

func TestBothNil_ReadFileAll(t *testing.T) {
	r := New(nil, nil)
	_, err := r.ReadFileAll("any.json")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got %v, want fs.ErrNotExist", err)
	}
}

func TestBothNil_HasFile(t *testing.T) {
	r := New(nil, nil)
	if r.HasFile("any.json") {
		t.Fatal("expected false")
	}
}

func TestNilUserFS_ReadDir(t *testing.T) {
	r := New(nil, userMap(map[string]string{"dir/a.txt": ""}))
	entries, err := r.ReadDir("dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestNilEmbedFS_ReadDir(t *testing.T) {
	r := New(userMap(map[string]string{"dir/a.txt": ""}), nil)
	entries, err := r.ReadDir("dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}
