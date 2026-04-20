package resolve

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestValidateEmbeddedFS_AllPresent(t *testing.T) {
	fs := fstest.MapFS{
		"agents/agents-index.json":    {Data: []byte("{}")},
		"routing-schema.json":         {Data: []byte("{}")},
		"CLAUDE.md":                   {Data: []byte("# Config")},
		"settings-template.json":      {Data: []byte("{}")},
		"agents/go-pro/go-pro.md":     {Data: []byte("")},
		"conventions/go.md":           {Data: []byte("")},
		"rules/agent-guidelines.md":   {Data: []byte("")},
	}
	if err := ValidateEmbeddedFS(fs); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateEmbeddedFS_MissingFiles(t *testing.T) {
	fs := fstest.MapFS{
		"agents/go-pro/go-pro.md":   {Data: []byte("")},
		"conventions/go.md":         {Data: []byte("")},
		"rules/agent-guidelines.md": {Data: []byte("")},
	}
	err := ValidateEmbeddedFS(fs)
	if err == nil {
		t.Fatal("expected error for missing critical files")
	}
	for _, f := range []string{"agents-index.json", "routing-schema.json", "CLAUDE.md", "settings-template.json"} {
		if !strings.Contains(err.Error(), f) {
			t.Errorf("error should mention %s: %v", f, err)
		}
	}
}

func TestValidateEmbeddedFS_EmptyDirs(t *testing.T) {
	fs := fstest.MapFS{
		"agents/agents-index.json": {Data: []byte("{}")},
		"routing-schema.json":      {Data: []byte("{}")},
		"CLAUDE.md":                {Data: []byte("")},
		"settings-template.json":   {Data: []byte("{}")},
	}
	err := ValidateEmbeddedFS(fs)
	if err == nil {
		t.Fatal("expected error for empty directories")
	}
	for _, d := range []string{"conventions", "rules"} {
		if !strings.Contains(err.Error(), d) {
			t.Errorf("error should mention %s: %v", d, err)
		}
	}
}

func TestValidateEmbeddedFS_Nil(t *testing.T) {
	err := ValidateEmbeddedFS(nil)
	if err == nil {
		t.Fatal("expected error for nil FS")
	}
}
