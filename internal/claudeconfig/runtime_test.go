package claudeconfig

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestPrepareStagesEmbeddedSkillsAndCredentials(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	nativeHome := t.TempDir()
	sourceCreds := filepath.Join(nativeHome, ".claude", ".credentials.json")
	if err := os.MkdirAll(filepath.Dir(sourceCreds), 0o755); err != nil {
		t.Fatalf("create native claude dir: %v", err)
	}
	if err := os.WriteFile(sourceCreds, []byte(`{"token":"secret"}`), 0o600); err != nil {
		t.Fatalf("write native credentials: %v", err)
	}

	src := fstest.MapFS{
		"agents/agents-index.json":                  {Data: []byte(`{"agents":[]}`)},
		"conventions/go.md":                         {Data: []byte("go")},
		"rules/router-guidelines.md":                {Data: []byte("router")},
		"schemas/teams/team-config.json":            {Data: []byte(`{"type":"object"}`)},
		"skills/ticket/SKILL.md":                    {Data: []byte("# Ticket")},
		"skills/ticket/scripts/find-next-ticket.sh": {Data: []byte("#!/usr/bin/env bash\nexit 0\n"), Mode: 0o755},
		"skills/ticket/templates/summary.md.tmpl":   {Data: []byte("summary")},
		"routing-schema.json":                       {Data: []byte(`{}`)},
		"CLAUDE.md":                                 {Data: []byte("# Claude")},
		"settings-template.json":                    {Data: []byte(`{"hooks":{"SessionStart":[]}}`)},
	}

	layout, err := Prepare(PrepareOptions{
		EmbeddedFS: src,
		NativeHome: nativeHome,
	})
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(layout.ConfigDir, "skills", "ticket", "scripts", "find-next-ticket.sh")); err != nil {
		t.Fatalf("expected staged ticket script: %v", err)
	}

	info, err := os.Stat(filepath.Join(layout.ConfigDir, "skills", "ticket", "scripts", "find-next-ticket.sh"))
	if err != nil {
		t.Fatalf("stat staged ticket script: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("ticket script permissions = %o, want 0755", info.Mode().Perm())
	}

	indexInfo, err := os.Stat(filepath.Join(layout.ConfigDir, "agents", "agents-index.json"))
	if err != nil {
		t.Fatalf("stat staged agents-index.json: %v", err)
	}
	if indexInfo.Mode().Perm() != 0o644 {
		t.Fatalf("agents-index.json permissions = %o, want 0644", indexInfo.Mode().Perm())
	}

	settingsData, err := os.ReadFile(filepath.Join(layout.ConfigDir, "settings.json"))
	if err != nil {
		t.Fatalf("read staged settings.json: %v", err)
	}
	if string(settingsData) != string(defaultSettingsJSON) {
		t.Fatalf("settings.json = %q, want %q", string(settingsData), string(defaultSettingsJSON))
	}

	credsData, err := os.ReadFile(filepath.Join(layout.ConfigDir, ".credentials.json"))
	if err != nil {
		t.Fatalf("read staged credentials: %v", err)
	}
	if string(credsData) != `{"token":"secret"}` {
		t.Fatalf("staged credentials = %q", string(credsData))
	}

	if layout.ConfigDir != filepath.Join(layout.FakeHomeDir, ".claude") {
		t.Fatalf("default config dir should live under fake HOME, got config=%s fakeHome=%s", layout.ConfigDir, layout.FakeHomeDir)
	}
}

func TestPrepareCanRestageExistingReadonlyFiles(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	src := fstest.MapFS{
		"agents/agents-index.json": {Data: []byte(`{"agents":[]}`)},
		"routing-schema.json":      {Data: []byte(`{}`)},
		"CLAUDE.md":                {Data: []byte("# Claude")},
		"settings-template.json":   {Data: []byte(`{}`)},
	}

	layout, err := Prepare(PrepareOptions{EmbeddedFS: src})
	if err != nil {
		t.Fatalf("first Prepare returned error: %v", err)
	}

	target := filepath.Join(layout.ConfigDir, "agents", "agents-index.json")
	if err := os.Chmod(target, 0o444); err != nil {
		t.Fatalf("chmod agents-index.json readonly: %v", err)
	}

	if _, err := Prepare(PrepareOptions{
		EmbeddedFS: src,
		ConfigDir:  layout.ConfigDir,
	}); err != nil {
		t.Fatalf("second Prepare returned error: %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat restaged agents-index.json: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("restaged agents-index.json permissions = %o, want 0644", info.Mode().Perm())
	}
}

func TestPrepareCreatesFakeHomeSymlinkForExplicitConfigDir(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "runtime-config")
	src := fstest.MapFS{
		"agents/agents-index.json": {Data: []byte(`{"agents":[]}`)},
		"routing-schema.json":      {Data: []byte(`{}`)},
		"CLAUDE.md":                {Data: []byte("# Claude")},
		"settings-template.json":   {Data: []byte(`{}`)},
	}

	layout, err := Prepare(PrepareOptions{
		EmbeddedFS: src,
		ConfigDir:  configDir,
	})
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}

	if layout.ConfigDir != configDir {
		t.Fatalf("ConfigDir = %s, want %s", layout.ConfigDir, configDir)
	}

	fakeClaude := filepath.Join(layout.FakeHomeDir, ".claude")
	info, err := os.Lstat(fakeClaude)
	if err != nil {
		t.Fatalf("lstat fake ~/.claude: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to be a symlink", fakeClaude)
	}

	target, err := os.Readlink(fakeClaude)
	if err != nil {
		t.Fatalf("read fake ~/.claude symlink: %v", err)
	}
	if target != configDir {
		t.Fatalf("fake ~/.claude link target = %s, want %s", target, configDir)
	}
}
