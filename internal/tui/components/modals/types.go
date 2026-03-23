// Package modals implements the modal dialog system for the GOgent-Fortress TUI.
// It provides five modal types (Ask, Confirm, Input, Select, Permission) with
// sequential queue management — only one modal is ever shown at a time.
package modals

// ModalType identifies which interaction style a modal uses.
type ModalType int

const (
	// Ask presents a list of labelled options plus an optional free-text "Other" entry.
	Ask ModalType = iota

	// Confirm presents a binary yes/no choice.
	Confirm

	// Input presents a free-text input field.
	Input

	// Select presents a list of labelled options (no free-text fallback).
	Select

	// Permission presents a tool-use permission request with allow/deny options.
	Permission
)

// String returns the human-readable name of the ModalType.
func (t ModalType) String() string {
	switch t {
	case Ask:
		return "Ask"
	case Confirm:
		return "Confirm"
	case Input:
		return "Input"
	case Select:
		return "Select"
	case Permission:
		return "Permission"
	default:
		return "Unknown"
	}
}

// ModalRequest describes a dialog that should be presented to the user.
// Callers that need to block on the response should create a buffered
// ResponseCh and wait on it after enqueuing the request.
type ModalRequest struct {
	// ID uniquely identifies this request. Used to correlate responses in the
	// bridge layer (e.g. IPC request round-trips).
	ID string `json:"id"`

	// Type determines the interaction style rendered by ModalModel.
	Type ModalType `json:"type"`

	// Message is the body text of the modal, displayed below the header.
	Message string `json:"message"`

	// Header is the title text rendered at the top of the modal box.
	// When empty the modal type name is used as a fallback.
	Header string `json:"header"`

	// Options lists the selectable button or list labels.
	// For Ask modals the list is shown above the free-text "Other" entry.
	// For Confirm modals Options is ignored; "Yes" and "No" are rendered.
	// For Input modals Options is ignored; a text field is rendered instead.
	Options []string `json:"options,omitempty"`

	// TimeoutMS, when positive, causes the modal to auto-cancel after the
	// given number of milliseconds.  Zero means no automatic timeout.
	TimeoutMS int `json:"timeout_ms,omitempty"`

	// ResponseCh receives exactly one ModalResponse when the user confirms or
	// cancels.  Callers that do not need to observe the response may leave
	// this nil; the queue will still advance correctly.
	ResponseCh chan ModalResponse `json:"-"`
}

// ModalResponse carries the user's answer after a modal is resolved.
type ModalResponse struct {
	// Type mirrors the ModalType of the originating ModalRequest.
	Type ModalType `json:"type"`

	// Value holds the selected option label (for Ask/Select/Confirm) or the
	// typed text (for Input modals).
	Value string `json:"value"`

	// Cancelled is true when the user dismissed the modal without making a
	// selection (e.g. pressed Escape).
	Cancelled bool `json:"cancelled"`
}
