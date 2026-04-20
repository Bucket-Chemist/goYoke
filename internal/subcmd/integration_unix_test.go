//go:build integration && !windows

package subcmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestArgv0Dispatch(t *testing.T) {
	symlink := filepath.Join(filepath.Dir(binary), "goyoke-validate")
	if err := os.Symlink(binary, symlink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}
	defer os.Remove(symlink)

	cmd := exec.Command(symlink)
	cmd.Stdin = strings.NewReader(minimalHookEvent)
	out, _ := cmd.CombinedOutput()

	if strings.Contains(string(out), "panic:") {
		t.Fatalf("argv0 dispatch via goyoke-validate panicked:\n%s", out)
	}
}
