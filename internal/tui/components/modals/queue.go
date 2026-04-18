package modals

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
)

// ---------------------------------------------------------------------------
// ModalQueue
//
// ModalQueue ensures that at most one modal is displayed at a time.
// Additional requests are held in a FIFO slice until the active modal is
// resolved.  This prevents concurrent permission dialogs from overlapping and
// guarantees the user always sees a clean, single-modal UI.
// ---------------------------------------------------------------------------

// ModalQueue serialises modal requests into a FIFO queue.
//
// The zero value is not usable; use NewModalQueue instead.
type ModalQueue struct {
	// items holds queued requests that are waiting for the active modal to
	// be resolved before they are displayed.
	items []ModalRequest

	// active is the ModalModel currently shown to the user, or nil when no
	// modal is displayed.
	active *ModalModel

	// km is the application keybinding registry injected at construction time.
	km config.KeyMap
}

// NewModalQueue returns an empty ModalQueue ready for use.
func NewModalQueue(km config.KeyMap) ModalQueue {
	return ModalQueue{km: km}
}

// Push appends a ModalRequest to the back of the queue.  If no modal is
// currently active the caller should immediately call Activate to display it.
func (q *ModalQueue) Push(req ModalRequest) {
	q.items = append(q.items, req)
}

// Pop removes and returns the front-most ModalRequest.  The second return
// value is false when the queue is empty.
func (q *ModalQueue) Pop() (ModalRequest, bool) {
	if len(q.items) == 0 {
		return ModalRequest{}, false
	}
	req := q.items[0]
	q.items = q.items[1:]
	return req, true
}

// IsActive reports whether a modal overlay is currently being shown.
func (q *ModalQueue) IsActive() bool {
	return q.active != nil
}

// ActiveModel returns a pointer to the currently displayed ModalModel, or nil
// when no modal is active.  Callers that need to forward tea.Msg events to
// the active modal should use this method.
func (q *ModalQueue) ActiveModel() *ModalModel {
	return q.active
}

// Activate pops the next request from the queue and creates a ModalModel for
// it.  It returns true when a new modal was activated, or false when the queue
// was empty.  Callers should not call Activate when IsActive returns true.
func (q *ModalQueue) Activate() bool {
	req, ok := q.Pop()
	if !ok {
		q.active = nil
		return false
	}
	m := newModalModel(req, q.km)
	q.active = &m
	return true
}

// Resolve closes the active modal with the supplied response and attempts to
// activate the next queued request.  It returns a tea.Cmd that should be
// returned from AppModel.Update so Bubbletea delivers any pending messages.
//
// Calling Resolve when IsActive returns false is a no-op that returns nil.
func (q *ModalQueue) Resolve(resp ModalResponse) tea.Cmd {
	if q.active == nil {
		return nil
	}

	// Deliver to any blocking goroutine via the channel.
	if q.active.request.ResponseCh != nil {
		select {
		case q.active.request.ResponseCh <- resp:
		default:
		}
	}

	q.active = nil

	// Try to show the next queued modal.
	q.Activate()

	return nil
}

// Len returns the number of requests still waiting in the queue (not counting
// any currently active modal).
func (q *ModalQueue) Len() int {
	return len(q.items)
}

// UpdateActive forwards a tea.Msg to the active ModalModel and returns the
// updated model together with any commands it produced.  It is a no-op when
// no modal is active.
//
// AppModel.Update should call this for every message when IsActive is true.
func (q *ModalQueue) UpdateActive(msg tea.Msg) tea.Cmd {
	if q.active == nil {
		return nil
	}
	updated, cmd := q.active.Update(msg)
	m := updated.(ModalModel)
	q.active = &m
	return cmd
}

// View returns the rendered string of the active modal, or an empty string
// when no modal is active.
func (q *ModalQueue) View() string {
	if q.active == nil {
		return ""
	}
	return q.active.View()
}

// SetTermSize propagates the current terminal dimensions to the active
// ModalModel so it can centre itself correctly.  Call this from
// AppModel.Update on every tea.WindowSizeMsg.
func (q *ModalQueue) SetTermSize(w, h int) {
	if q.active != nil {
		q.active.SetTermSize(w, h)
	}
}
