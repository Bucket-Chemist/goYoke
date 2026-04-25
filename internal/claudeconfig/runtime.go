package claudeconfig

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	goyokeconfig "github.com/Bucket-Chemist/goYoke/pkg/config"
)

var stagedRoots = []string{
	"agents",
	"conventions",
	"rules",
	"schemas",
	"skills",
	"routing-schema.json",
	"CLAUDE.md",
	"settings-template.json",
}

var defaultSettingsJSON = []byte("{\n  \"use_team_pattern\": true\n}\n")

// Layout describes the isolated Claude runtime layout goYoke manages.
type Layout struct {
	ConfigDir   string
	FakeHomeDir string
}

// PrepareOptions controls how the isolated Claude runtime tree is created.
type PrepareOptions struct {
	EmbeddedFS fs.FS
	ConfigDir  string
	NativeHome string
}

// Prepare stages the embedded public Claude config into a goYoke-managed
// runtime directory and returns the config dir plus the fake HOME directory
// that Claude subprocesses should use.
func Prepare(opts PrepareOptions) (Layout, error) {
	if opts.EmbeddedFS == nil {
		return Layout{}, fmt.Errorf("claudeconfig: embedded FS is required")
	}

	configDir, err := resolveConfigDir(opts.ConfigDir)
	if err != nil {
		return Layout{}, err
	}

	fakeHomeDir := fakeHomeDirForConfig(configDir)
	if err := os.MkdirAll(fakeHomeDir, 0o700); err != nil {
		return Layout{}, fmt.Errorf("claudeconfig: create fake home dir: %w", err)
	}
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return Layout{}, fmt.Errorf("claudeconfig: create config dir: %w", err)
	}

	if err := linkFakeHome(fakeHomeDir, configDir); err != nil {
		return Layout{}, err
	}
	if err := stageEmbeddedTree(opts.EmbeddedFS, configDir); err != nil {
		return Layout{}, err
	}
	if err := ensureSettingsJSON(configDir); err != nil {
		return Layout{}, err
	}
	if err := copyAuthCredentials(configDir, opts.NativeHome); err != nil {
		return Layout{}, err
	}

	return Layout{
		ConfigDir:   configDir,
		FakeHomeDir: fakeHomeDir,
	}, nil
}

func resolveConfigDir(configDir string) (string, error) {
	if configDir == "" {
		configDir = filepath.Join(goyokeconfig.GetgoYokeDataDir(), "claude-runtime", "home", ".claude")
	}

	abs, err := filepath.Abs(configDir)
	if err != nil {
		return "", fmt.Errorf("claudeconfig: resolve config dir: %w", err)
	}
	return filepath.Clean(abs), nil
}

func fakeHomeDirForConfig(configDir string) string {
	if filepath.Base(configDir) == ".claude" {
		return filepath.Dir(configDir)
	}
	return filepath.Join(filepath.Dir(configDir), ".goyoke-claude-home")
}

func linkFakeHome(fakeHomeDir, configDir string) error {
	fakeClaudeDir := filepath.Join(fakeHomeDir, ".claude")
	if fakeClaudeDir == configDir {
		return nil
	}

	if err := os.RemoveAll(fakeClaudeDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("claudeconfig: reset fake ~/.claude link: %w", err)
	}
	if err := os.Symlink(configDir, fakeClaudeDir); err != nil {
		return fmt.Errorf("claudeconfig: create fake ~/.claude link: %w", err)
	}
	return nil
}

func stageEmbeddedTree(src fs.FS, configDir string) error {
	for _, root := range stagedRoots {
		if err := copyEmbeddedPath(src, root, configDir); err != nil {
			return err
		}
	}
	return nil
}

func copyEmbeddedPath(src fs.FS, root, configDir string) error {
	info, err := fs.Stat(src, root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("claudeconfig: stat embedded %s: %w", root, err)
	}

	targetRoot := filepath.Join(configDir, filepath.FromSlash(root))
	if !info.IsDir() {
		return writeEmbeddedFile(src, root, targetRoot, info.Mode())
	}

	return fs.WalkDir(src, root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("claudeconfig: walk %s: %w", path, walkErr)
		}

		target := filepath.Join(configDir, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("claudeconfig: stat embedded file %s: %w", path, err)
		}
		return writeEmbeddedFile(src, path, target, info.Mode())
	})
}

func writeEmbeddedFile(src fs.FS, srcPath, targetPath string, mode fs.FileMode) error {
	data, err := fs.ReadFile(src, srcPath)
	if err != nil {
		return fmt.Errorf("claudeconfig: read embedded %s: %w", srcPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("claudeconfig: create parent dir for %s: %w", targetPath, err)
	}

	perm := filePermForPath(srcPath, mode)
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("claudeconfig: reset %s: %w", targetPath, err)
	}
	if err := os.WriteFile(targetPath, data, perm); err != nil {
		return fmt.Errorf("claudeconfig: write %s: %w", targetPath, err)
	}
	return nil
}

func filePermForPath(path string, mode fs.FileMode) fs.FileMode {
	if mode.Perm()&0o111 != 0 || shouldBeExecutable(path) {
		return 0o755
	}
	return 0o644
}

func shouldBeExecutable(path string) bool {
	clean := strings.ToLower(filepath.ToSlash(path))
	if !strings.Contains(clean, "/scripts/") && !strings.Contains(clean, "/bin/") {
		return false
	}
	ext := filepath.Ext(clean)
	return ext == "" || ext == ".sh" || ext == ".py" || ext == ".rb" || ext == ".pl"
}

func ensureSettingsJSON(configDir string) error {
	settingsPath := filepath.Join(configDir, "settings.json")
	if _, err := os.Stat(settingsPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("claudeconfig: stat settings.json: %w", err)
	}

	if err := os.WriteFile(settingsPath, defaultSettingsJSON, 0o644); err != nil {
		return fmt.Errorf("claudeconfig: write settings.json: %w", err)
	}
	return nil
}

func copyAuthCredentials(configDir, nativeHome string) error {
	if nativeHome == "" {
		return nil
	}

	source := filepath.Join(nativeHome, ".claude", ".credentials.json")
	data, err := os.ReadFile(source)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("claudeconfig: read auth credentials: %w", err)
	}

	target := filepath.Join(configDir, ".credentials.json")
	if err := os.WriteFile(target, data, 0o600); err != nil {
		return fmt.Errorf("claudeconfig: write auth credentials: %w", err)
	}
	return nil
}
