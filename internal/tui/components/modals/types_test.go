package modals

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ModalType String()
// ---------------------------------------------------------------------------

func TestModalTypeString(t *testing.T) {
	cases := []struct {
		mt   ModalType
		want string
	}{
		{Ask, "Ask"},
		{Confirm, "Confirm"},
		{Input, "Input"},
		{Select, "Select"},
		{Permission, "Permission"},
		{ModalType(99), "Unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.mt.String())
		})
	}
}

func TestModalTypeIota(t *testing.T) {
	// Ensure the iota ordering matches the documented contract.
	assert.Equal(t, ModalType(0), Ask)
	assert.Equal(t, ModalType(1), Confirm)
	assert.Equal(t, ModalType(2), Input)
	assert.Equal(t, ModalType(3), Select)
	assert.Equal(t, ModalType(4), Permission)
}

// ---------------------------------------------------------------------------
// ModalRequest JSON serialisation
// ---------------------------------------------------------------------------

func TestModalRequestJSONRoundTrip(t *testing.T) {
	req := ModalRequest{
		ID:        "req-001",
		Type:      Ask,
		Message:   "Which option?",
		Header:    "Choose",
		Options:   []string{"A", "B"},
		TimeoutMS: 5000,
		// ResponseCh is excluded from JSON (tagged with "-")
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var got ModalRequest
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, req.ID, got.ID)
	assert.Equal(t, req.Type, got.Type)
	assert.Equal(t, req.Message, got.Message)
	assert.Equal(t, req.Header, got.Header)
	assert.Equal(t, req.Options, got.Options)
	assert.Equal(t, req.TimeoutMS, got.TimeoutMS)
	assert.Nil(t, got.ResponseCh, "ResponseCh must not appear in JSON output")
}

func TestModalRequestResponseChExcludedFromJSON(t *testing.T) {
	ch := make(chan ModalResponse, 1)
	req := ModalRequest{ID: "x", ResponseCh: ch}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	// The JSON must not contain the channel in any form.
	assert.NotContains(t, string(data), "ResponseCh")
	assert.NotContains(t, string(data), "response_ch")
}

func TestModalRequestEmptyOptions(t *testing.T) {
	req := ModalRequest{ID: "y", Type: Input, Message: "Enter text"}
	data, err := json.Marshal(req)
	require.NoError(t, err)

	// Options with omitempty should not appear in JSON when nil.
	assert.NotContains(t, string(data), "options")
}

// ---------------------------------------------------------------------------
// ModalResponse JSON serialisation
// ---------------------------------------------------------------------------

func TestModalResponseJSONRoundTrip(t *testing.T) {
	resp := ModalResponse{
		Type:      Confirm,
		Value:     "Yes",
		Cancelled: false,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var got ModalResponse
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, resp.Type, got.Type)
	assert.Equal(t, resp.Value, got.Value)
	assert.Equal(t, resp.Cancelled, got.Cancelled)
}

func TestModalResponseCancelled(t *testing.T) {
	resp := ModalResponse{Type: Ask, Cancelled: true}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var got ModalResponse
	require.NoError(t, json.Unmarshal(data, &got))

	assert.True(t, got.Cancelled)
	assert.Empty(t, got.Value)
}

// ---------------------------------------------------------------------------
// buildOptions
// ---------------------------------------------------------------------------

func TestBuildOptionsConfirm(t *testing.T) {
	req := ModalRequest{Type: Confirm}
	opts := buildOptions(req)
	assert.Equal(t, []string{"Yes", "No"}, opts)
}

func TestBuildOptionsPermission(t *testing.T) {
	req := ModalRequest{Type: Permission}
	opts := buildOptions(req)
	assert.Equal(t, []string{"Allow", "Deny"}, opts)
}

func TestBuildOptionsAsk(t *testing.T) {
	req := ModalRequest{Type: Ask, Options: []string{"Alpha", "Beta"}}
	opts := buildOptions(req)
	// Must append "Other..." sentinel.
	assert.Equal(t, []string{"Alpha", "Beta", "Other..."}, opts)
}

func TestBuildOptionsAskNoInitialOptions(t *testing.T) {
	req := ModalRequest{Type: Ask}
	opts := buildOptions(req)
	assert.Equal(t, []string{"Other..."}, opts)
}

func TestBuildOptionsSelect(t *testing.T) {
	req := ModalRequest{Type: Select, Options: []string{"X", "Y", "Z"}}
	opts := buildOptions(req)
	assert.Equal(t, []string{"X", "Y", "Z"}, opts)
}

func TestBuildOptionsSelectEmpty(t *testing.T) {
	req := ModalRequest{Type: Select}
	opts := buildOptions(req)
	assert.Nil(t, opts)
}

func TestBuildOptionsInput(t *testing.T) {
	req := ModalRequest{Type: Input, Options: []string{"ignored"}}
	opts := buildOptions(req)
	// Input modals never display an option list.
	assert.Nil(t, opts)
}
