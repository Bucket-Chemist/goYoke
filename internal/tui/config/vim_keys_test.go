package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// VimMode.String
// ---------------------------------------------------------------------------

func TestVimMode_String(t *testing.T) {
	tests := []struct {
		mode VimMode
		want string
	}{
		{VimNormal, "NORMAL"},
		{VimInsert, "INSERT"},
		{VimMode(99), "UNKNOWN"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.mode.String())
		})
	}
}

// ---------------------------------------------------------------------------
// DefaultVimKeys — all bindings populated
// ---------------------------------------------------------------------------

func TestDefaultVimKeys_AllBindingsNonEmpty(t *testing.T) {
	vk := DefaultVimKeys()

	bindings := []struct {
		name    string
		keys    []string
		helpKey string
		helpDesc string
	}{
		{"Up", vk.Up.Keys(), vk.Up.Help().Key, vk.Up.Help().Desc},
		{"Down", vk.Down.Keys(), vk.Down.Help().Key, vk.Down.Help().Desc},
		{"Left", vk.Left.Keys(), vk.Left.Help().Key, vk.Left.Help().Desc},
		{"Right", vk.Right.Keys(), vk.Right.Help().Key, vk.Right.Help().Desc},
		{"Top", vk.Top.Keys(), vk.Top.Help().Key, vk.Top.Help().Desc},
		{"Bottom", vk.Bottom.Keys(), vk.Bottom.Help().Key, vk.Bottom.Help().Desc},
		{"Insert", vk.Insert.Keys(), vk.Insert.Help().Key, vk.Insert.Help().Desc},
		{"Normal", vk.Normal.Keys(), vk.Normal.Help().Key, vk.Normal.Help().Desc},
	}

	for _, b := range bindings {
		t.Run(b.name, func(t *testing.T) {
			require.NotEmpty(t, b.keys, "keys must not be empty")
			assert.NotEmpty(t, b.helpKey, "Help().Key must not be empty")
			assert.NotEmpty(t, b.helpDesc, "Help().Desc must not be empty")
		})
	}
}

func TestDefaultVimKeys_KeyAssignments(t *testing.T) {
	vk := DefaultVimKeys()

	assert.Equal(t, []string{"k"}, vk.Up.Keys(), "Up must be bound to k")
	assert.Equal(t, []string{"j"}, vk.Down.Keys(), "Down must be bound to j")
	assert.Equal(t, []string{"h"}, vk.Left.Keys(), "Left must be bound to h")
	assert.Equal(t, []string{"l"}, vk.Right.Keys(), "Right must be bound to l")
	assert.Equal(t, []string{"g"}, vk.Top.Keys(), "Top must be bound to g")
	assert.Equal(t, []string{"G"}, vk.Bottom.Keys(), "Bottom must be bound to G")
	assert.Equal(t, []string{"i"}, vk.Insert.Keys(), "Insert must be bound to i")
	assert.Equal(t, []string{"esc"}, vk.Normal.Keys(), "Normal must be bound to esc")
}

// ---------------------------------------------------------------------------
// DefaultKeyMap — VimEnabled off by default, Vim bindings populated
// ---------------------------------------------------------------------------

func TestDefaultKeyMap_VimEnabledFalseByDefault(t *testing.T) {
	km := DefaultKeyMap()
	assert.False(t, km.VimEnabled, "VimEnabled must be false by default")
}

func TestDefaultKeyMap_VimModeNormalByDefault(t *testing.T) {
	km := DefaultKeyMap()
	assert.Equal(t, VimNormal, km.VimMode, "VimMode must be VimNormal by default")
}

func TestDefaultKeyMap_VimBindingsPopulated(t *testing.T) {
	km := DefaultKeyMap()
	// Spot-check: Vim struct must be the default, not a zero-value struct.
	assert.NotEmpty(t, km.Vim.Up.Keys(), "KeyMap.Vim.Up must be populated")
	assert.NotEmpty(t, km.Vim.Down.Keys(), "KeyMap.Vim.Down must be populated")
}

// ---------------------------------------------------------------------------
// VimEnabled toggle (field mutation)
// ---------------------------------------------------------------------------

func TestKeyMap_VimEnabledToggle(t *testing.T) {
	km := DefaultKeyMap()

	assert.False(t, km.VimEnabled)
	km.VimEnabled = true
	assert.True(t, km.VimEnabled)
	km.VimEnabled = false
	assert.False(t, km.VimEnabled)
}

func TestKeyMap_VimModeTransitions(t *testing.T) {
	km := DefaultKeyMap()
	km.VimEnabled = true

	// Start in normal.
	assert.Equal(t, VimNormal, km.VimMode)
	assert.Equal(t, "NORMAL", km.VimMode.String())

	// Transition to insert.
	km.VimMode = VimInsert
	assert.Equal(t, VimInsert, km.VimMode)
	assert.Equal(t, "INSERT", km.VimMode.String())

	// Transition back to normal.
	km.VimMode = VimNormal
	assert.Equal(t, VimNormal, km.VimMode)
}

// ---------------------------------------------------------------------------
// DefaultVimKeys is idempotent
// ---------------------------------------------------------------------------

func TestDefaultVimKeys_IsIdempotent(t *testing.T) {
	vk1 := DefaultVimKeys()
	vk2 := DefaultVimKeys()

	assert.Equal(t, vk1.Up.Keys(), vk2.Up.Keys())
	assert.Equal(t, vk1.Down.Keys(), vk2.Down.Keys())
	assert.Equal(t, vk1.Left.Keys(), vk2.Left.Keys())
	assert.Equal(t, vk1.Right.Keys(), vk2.Right.Keys())
	assert.Equal(t, vk1.Top.Keys(), vk2.Top.Keys())
	assert.Equal(t, vk1.Bottom.Keys(), vk2.Bottom.Keys())
	assert.Equal(t, vk1.Insert.Keys(), vk2.Insert.Keys())
	assert.Equal(t, vk1.Normal.Keys(), vk2.Normal.Keys())
}
