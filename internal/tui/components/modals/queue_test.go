package modals

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestQueue() ModalQueue {
	return NewModalQueue(defaultKM())
}

func makeReq(id string, mt ModalType) ModalRequest {
	return ModalRequest{ID: id, Type: mt, Message: id}
}

// ---------------------------------------------------------------------------
// NewModalQueue
// ---------------------------------------------------------------------------

func TestNewModalQueueIsEmpty(t *testing.T) {
	q := newTestQueue()
	assert.Equal(t, 0, q.Len())
	assert.False(t, q.IsActive())
	assert.Nil(t, q.ActiveModel())
}

// ---------------------------------------------------------------------------
// Push / Pop
// ---------------------------------------------------------------------------

func TestPushIncreasesLen(t *testing.T) {
	q := newTestQueue()
	q.Push(makeReq("r1", Confirm))
	assert.Equal(t, 1, q.Len())
	q.Push(makeReq("r2", Confirm))
	assert.Equal(t, 2, q.Len())
}

func TestPopFIFOOrder(t *testing.T) {
	q := newTestQueue()
	q.Push(makeReq("first", Confirm))
	q.Push(makeReq("second", Confirm))

	req, ok := q.Pop()
	require.True(t, ok)
	assert.Equal(t, "first", req.ID)

	req, ok = q.Pop()
	require.True(t, ok)
	assert.Equal(t, "second", req.ID)
}

func TestPopEmptyQueue(t *testing.T) {
	q := newTestQueue()
	_, ok := q.Pop()
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// Activate
// ---------------------------------------------------------------------------

func TestActivateEmptyQueueReturnsFalse(t *testing.T) {
	q := newTestQueue()
	ok := q.Activate()
	assert.False(t, ok)
	assert.False(t, q.IsActive())
}

func TestActivatePopsFrontItem(t *testing.T) {
	q := newTestQueue()
	q.Push(makeReq("m1", Confirm))
	q.Push(makeReq("m2", Ask))

	ok := q.Activate()
	require.True(t, ok)
	assert.True(t, q.IsActive())
	// The active modal should be the first request.
	active := q.ActiveModel()
	require.NotNil(t, active)
	assert.Equal(t, "m1", active.request.ID)
	// The second item must still be in the queue.
	assert.Equal(t, 1, q.Len())
}

func TestActivateWhenAlreadyActiveReplacesActive(t *testing.T) {
	// Although callers should not call Activate when IsActive is true, the
	// queue must behave deterministically if they do.
	q := newTestQueue()
	q.Push(makeReq("x", Confirm))
	q.Push(makeReq("y", Select))
	q.Activate()
	q.Activate() // activates "y"
	assert.Equal(t, "y", q.ActiveModel().request.ID)
}

// ---------------------------------------------------------------------------
// Resolve
// ---------------------------------------------------------------------------

func TestResolveNoActiveIsNoop(t *testing.T) {
	q := newTestQueue()
	cmd := q.Resolve(ModalResponse{Type: Confirm, Value: "Yes"})
	assert.Nil(t, cmd, "Resolve on empty queue must return nil")
	assert.False(t, q.IsActive())
}

func TestResolveClosesActiveModal(t *testing.T) {
	q := newTestQueue()
	q.Push(makeReq("a", Confirm))
	q.Activate()
	require.True(t, q.IsActive())

	q.Resolve(ModalResponse{Type: Confirm, Value: "Yes"})
	assert.False(t, q.IsActive())
}

func TestResolveActivatesNextInQueue(t *testing.T) {
	q := newTestQueue()
	q.Push(makeReq("first", Confirm))
	q.Push(makeReq("second", Permission))
	q.Activate()

	// Resolve the first modal.
	q.Resolve(ModalResponse{Type: Confirm, Value: "Yes"})

	// The second request should now be active.
	assert.True(t, q.IsActive())
	require.NotNil(t, q.ActiveModel())
	assert.Equal(t, "second", q.ActiveModel().request.ID)
	assert.Equal(t, 0, q.Len())
}

func TestResolveDeliversViaChannel(t *testing.T) {
	ch := make(chan ModalResponse, 1)
	q := newTestQueue()
	q.Push(ModalRequest{ID: "ch", Type: Confirm, ResponseCh: ch})
	q.Activate()

	q.Resolve(ModalResponse{Type: Confirm, Value: "Yes"})

	select {
	case resp := <-ch:
		assert.Equal(t, "Yes", resp.Value)
	default:
		t.Fatal("Resolve must deliver to ResponseCh")
	}
}

func TestResolveQueueDrainsCompletely(t *testing.T) {
	q := newTestQueue()
	for i := range 5 {
		_ = i
		q.Push(makeReq("item", Confirm))
	}
	q.Activate()

	for range 5 {
		require.True(t, q.IsActive())
		q.Resolve(ModalResponse{Type: Confirm, Value: "Yes"})
	}

	assert.False(t, q.IsActive())
	assert.Equal(t, 0, q.Len())
}

// ---------------------------------------------------------------------------
// UpdateActive
// ---------------------------------------------------------------------------

func TestUpdateActiveNoModal(t *testing.T) {
	q := newTestQueue()
	cmd := q.UpdateActive(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
}

func TestUpdateActiveForwardsMessage(t *testing.T) {
	q := newTestQueue()
	q.Push(makeReq("u1", Confirm))
	q.Activate()

	// Navigate down then up — selectedIdx should reflect navigation.
	q.UpdateActive(tea.KeyMsg{Type: tea.KeyDown})
	q.UpdateActive(tea.KeyMsg{Type: tea.KeyUp})

	assert.Equal(t, 0, q.ActiveModel().selectedIdx)
}

func TestUpdateActiveEscapeReturnsCmd(t *testing.T) {
	q := newTestQueue()
	q.Push(makeReq("u2", Select, ))
	q.Activate()

	cmd := q.UpdateActive(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)

	msg := cmd()
	resp, ok := msg.(ModalResponseMsg)
	require.True(t, ok)
	assert.True(t, resp.Response.Cancelled)
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func TestViewNoActiveReturnsEmpty(t *testing.T) {
	q := newTestQueue()
	assert.Empty(t, q.View())
}

func TestViewWithActiveReturnsContent(t *testing.T) {
	q := newTestQueue()
	q.Push(ModalRequest{Type: Confirm, Header: "Test Header", Message: "body"})
	q.Activate()
	q.SetTermSize(120, 40)

	view := q.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Test Header")
}

// ---------------------------------------------------------------------------
// SetTermSize
// ---------------------------------------------------------------------------

func TestSetTermSizeNoActiveIsNoop(t *testing.T) {
	q := newTestQueue()
	// Must not panic when no modal is active.
	q.SetTermSize(100, 30)
}

func TestSetTermSizePropagatedToActive(t *testing.T) {
	q := newTestQueue()
	q.Push(makeReq("sz", Confirm))
	q.Activate()

	q.SetTermSize(200, 50)
	m := q.ActiveModel()
	require.NotNil(t, m)
	assert.Equal(t, 200, m.termWidth)
	assert.Equal(t, 50, m.termHeight)
}

// ---------------------------------------------------------------------------
// Sequential processing guarantee
// ---------------------------------------------------------------------------

func TestSequentialProcessingNoConcurrentModals(t *testing.T) {
	// Two simultaneous requests must not both be active.
	q := newTestQueue()
	q.Push(makeReq("seq1", Confirm))
	q.Push(makeReq("seq2", Confirm))

	q.Activate()
	// Only seq1 should be active; seq2 remains queued.
	assert.Equal(t, "seq1", q.ActiveModel().request.ID)
	assert.Equal(t, 1, q.Len())

	// Resolve seq1; seq2 must become active automatically.
	q.Resolve(ModalResponse{Type: Confirm, Value: "Yes"})
	assert.Equal(t, "seq2", q.ActiveModel().request.ID)
	assert.Equal(t, 0, q.Len())
}
