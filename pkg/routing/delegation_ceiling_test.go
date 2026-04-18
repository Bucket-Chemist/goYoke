package routing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDelegationCeiling_FileExists(t *testing.T) {
	tmpDir := t.TempDir()
	ceilingDir := filepath.Join(tmpDir, ".goyoke", "tmp")
	os.MkdirAll(ceilingDir, 0755)

	ceilingPath := filepath.Join(ceilingDir, "max_delegation")
	os.WriteFile(ceilingPath, []byte("haiku"), 0644)

	ceiling, err := LoadDelegationCeiling(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load ceiling: %v", err)
	}

	if ceiling.MaxTier != "haiku" {
		t.Errorf("Expected max tier 'haiku', got: %s", ceiling.MaxTier)
	}
}

func TestLoadDelegationCeiling_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	ceiling, err := LoadDelegationCeiling(tmpDir)
	if err != nil {
		t.Fatalf("Expected no error when file missing, got: %v", err)
	}

	if ceiling.MaxTier != "sonnet" {
		t.Errorf("Expected default 'sonnet', got: %s", ceiling.MaxTier)
	}
}

func TestCheckDelegationCeiling_WithinCeiling(t *testing.T) {
	schema := &Schema{
		TierLevels: TierLevels{
			Haiku:  10,
			Sonnet: 20,
			Opus:   30,
		},
	}

	ceiling := &DelegationCeilingRuntime{MaxTier: "sonnet"}

	// Request haiku (below ceiling)
	allowed, msg := CheckDelegationCeiling(schema, ceiling, "haiku")
	if !allowed {
		t.Errorf("haiku should be allowed under sonnet ceiling: %s", msg)
	}

	// Request sonnet (at ceiling)
	allowed, msg = CheckDelegationCeiling(schema, ceiling, "sonnet")
	if !allowed {
		t.Errorf("sonnet should be allowed at sonnet ceiling: %s", msg)
	}
}

func TestCheckDelegationCeiling_ExceedsCeiling(t *testing.T) {
	schema := &Schema{
		TierLevels: TierLevels{
			Haiku:  10,
			Sonnet: 20,
			Opus:   30,
		},
	}

	ceiling := &DelegationCeilingRuntime{MaxTier: "haiku"}

	// Request sonnet (above haiku ceiling)
	allowed, msg := CheckDelegationCeiling(schema, ceiling, "sonnet")
	if allowed {
		t.Error("sonnet should not be allowed under haiku ceiling")
	}

	if msg == "" {
		t.Error("Expected error message for ceiling violation")
	}

	if !contains(msg, "haiku") || !contains(msg, "sonnet") {
		t.Errorf("Message should mention both tiers: %s", msg)
	}

	if !contains(msg, "--force-delegation=") {
		t.Error("Message should suggest override flag")
	}
}

func TestCheckDelegationCeiling_NoTierLevels(t *testing.T) {
	schema := &Schema{
		TierLevels: TierLevels{},
	}

	ceiling := &DelegationCeilingRuntime{MaxTier: "haiku"}

	// Should allow when no tier levels defined
	allowed, _ := CheckDelegationCeiling(schema, ceiling, "opus")
	if !allowed {
		t.Error("Should allow all when tier levels not defined")
	}
}
