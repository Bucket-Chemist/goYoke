package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetGOgentDir_XDG_RUNTIME_DIR(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	// Set XDG_RUNTIME_DIR (highest priority)
	testDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", testDir)
	os.Setenv("XDG_CACHE_HOME", "/should/not/use/this")

	result := GetGOgentDir()
	expected := filepath.Join(testDir, "gogent")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Verify directory was created
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Error("Expected GetGOgentDir to create directory")
	}
}

func TestGetGOgentDir_XDG_CACHE_HOME(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	// Unset XDG_RUNTIME_DIR, set XDG_CACHE_HOME (second priority)
	os.Unsetenv("XDG_RUNTIME_DIR")
	testDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", testDir)

	result := GetGOgentDir()
	expected := filepath.Join(testDir, "gogent")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Verify directory was created
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Error("Expected GetGOgentDir to create directory")
	}
}

func TestGetGOgentDir_Fallback(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	// Unset both XDG vars (fallback to ~/.cache/gogent)
	os.Unsetenv("XDG_RUNTIME_DIR")
	os.Unsetenv("XDG_CACHE_HOME")

	result := GetGOgentDir()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".cache", "gogent")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGetGOgentDir_EmptyXDGVars(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	// Set to empty strings (should fallback)
	os.Setenv("XDG_RUNTIME_DIR", "")
	os.Setenv("XDG_CACHE_HOME", "")

	result := GetGOgentDir()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".cache", "gogent")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGetTierFilePath(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	}()

	testDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", testDir)

	result := GetTierFilePath()
	expected := filepath.Join(testDir, "gogent", "current-tier")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGetMaxDelegationPath(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	}()

	testDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", testDir)

	result := GetMaxDelegationPath()
	expected := filepath.Join(testDir, "gogent", "max_delegation")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestGetViolationsLogPath(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	}()

	testDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", testDir)

	result := GetViolationsLogPath()
	expected := filepath.Join(testDir, "gogent", "routing-violations.jsonl")

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Verify filename ends with .jsonl
	if !strings.HasSuffix(result, ".jsonl") {
		t.Error("Expected violations log to have .jsonl extension")
	}
}

func TestGetGOgentDir_PriorityOrder(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	}()

	runtimeDir := t.TempDir()
	cacheDir := t.TempDir()

	// Both set: XDG_RUNTIME_DIR wins
	os.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	os.Setenv("XDG_CACHE_HOME", cacheDir)

	result := GetGOgentDir()
	expected := filepath.Join(runtimeDir, "gogent")

	if result != expected {
		t.Errorf("XDG_RUNTIME_DIR should have priority. Expected %s, got %s", expected, result)
	}
}

func TestGetGOgentDir_CreatesDirectory(t *testing.T) {
	// Save original env
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer func() {
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	}()

	testDir := t.TempDir()
	gogentPath := filepath.Join(testDir, "gogent")

	// Ensure directory doesn't exist yet
	os.RemoveAll(gogentPath)

	os.Setenv("XDG_RUNTIME_DIR", testDir)

	result := GetGOgentDir()

	// Verify directory was created
	info, err := os.Stat(result)
	if os.IsNotExist(err) {
		t.Error("GetGOgentDir should create directory if it doesn't exist")
	}

	// Verify it's a directory
	if !info.IsDir() {
		t.Error("GetGOgentDir should create a directory, not a file")
	}

	// Verify permissions (0755)
	if info.Mode().Perm() != 0755 {
		t.Errorf("Expected permissions 0755, got %o", info.Mode().Perm())
	}
}
