package toast

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// sendToast sends a model.ToastMsg to m and returns the updated model + cmd.
func sendToast(m ToastModel, text string, level model.ToastLevel) (ToastModel, tea.Cmd) {
	return m.Update(model.ToastMsg{Text: text, Level: level})
}

// sendTick sends a tickMsg to m and returns the updated model + cmd.
func sendTick(m ToastModel) (ToastModel, tea.Cmd) {
	return m.Update(tickMsg(time.Now()))
}

// makeExpiredItem returns a ToastItem that is already past its default
// expiry window (CreatedAt 6 seconds ago).
func makeExpiredItem(msg string, level model.ToastLevel) ToastItem {
	return ToastItem{
		Message:   msg,
		Level:     level,
		CreatedAt: time.Now().Add(-6 * time.Second),
		Duration:  defaultDuration,
	}
}

// makeYoungItem returns a ToastItem that was just created (not yet expired).
func makeYoungItem(msg string, level model.ToastLevel) ToastItem {
	return ToastItem{
		Message:   msg,
		Level:     level,
		CreatedAt: time.Now(),
		Duration:  defaultDuration,
	}
}

// ---------------------------------------------------------------------------
// TestNewToastModel_Defaults
// ---------------------------------------------------------------------------

func TestNewToastModel_Defaults(t *testing.T) {
	m := NewToastModel()

	assert.True(t, m.IsEmpty(), "new model should have no items")
	assert.Equal(t, 0, m.Count(), "count should be 0")
	assert.Equal(t, defaultMaxItems, m.maxItems, "maxItems should equal defaultMaxItems")
}

// ---------------------------------------------------------------------------
// TestUpdate_ToastMsg_AddsItem
// ---------------------------------------------------------------------------

func TestUpdate_ToastMsg_AddsItem(t *testing.T) {
	m := NewToastModel()

	m2, cmd := sendToast(m, "hello", "info")

	assert.Equal(t, 1, m2.Count(), "one item should be present after adding")
	assert.Equal(t, "hello", m2.items[0].Message)
	assert.Equal(t, model.ToastLevelInfo, m2.items[0].Level)
	require.NotNil(t, cmd, "a tick command should be returned when the first toast is added")
}

// ---------------------------------------------------------------------------
// TestUpdate_ToastMsg_MaxItems
// ---------------------------------------------------------------------------

func TestUpdate_ToastMsg_MaxItems(t *testing.T) {
	m := NewToastModel()

	m, _ = sendToast(m, "first", "info")
	m, _ = sendToast(m, "second", "info")
	m, _ = sendToast(m, "third", "info")
	// Adding a 4th should evict the oldest ("first").
	m, _ = sendToast(m, "fourth", "info")

	assert.Equal(t, 3, m.Count(), "count must not exceed maxItems")
	assert.Equal(t, "second", m.items[0].Message, "oldest should have been evicted")
	assert.Equal(t, "fourth", m.items[2].Message, "newest should be last")
}

// ---------------------------------------------------------------------------
// TestUpdate_TickMsg_ExpiresOld
// ---------------------------------------------------------------------------

func TestUpdate_TickMsg_ExpiresOld(t *testing.T) {
	m := NewToastModel()
	// Inject a pre-expired item directly into the slice.
	m.items = []ToastItem{makeExpiredItem("stale", "error")}

	m2, cmd := sendTick(m)

	assert.Equal(t, 0, m2.Count(), "expired item should have been removed")
	assert.Nil(t, cmd, "no tick should be scheduled when queue is empty")
}

// ---------------------------------------------------------------------------
// TestUpdate_TickMsg_KeepsYoung
// ---------------------------------------------------------------------------

func TestUpdate_TickMsg_KeepsYoung(t *testing.T) {
	m := NewToastModel()
	m.items = []ToastItem{makeYoungItem("fresh", "success")}

	m2, _ := sendTick(m)

	assert.Equal(t, 1, m2.Count(), "non-expired item must be kept")
	assert.Equal(t, "fresh", m2.items[0].Message)
}

// ---------------------------------------------------------------------------
// TestUpdate_TickMsg_ReturnsTick_WhenItemsRemain
// ---------------------------------------------------------------------------

func TestUpdate_TickMsg_ReturnsTick_WhenItemsRemain(t *testing.T) {
	m := NewToastModel()
	m.items = []ToastItem{makeYoungItem("alive", "warning")}

	_, cmd := sendTick(m)

	require.NotNil(t, cmd, "tick should be rescheduled when items remain")
}

// ---------------------------------------------------------------------------
// TestUpdate_TickMsg_NoTick_WhenEmpty
// ---------------------------------------------------------------------------

func TestUpdate_TickMsg_NoTick_WhenEmpty(t *testing.T) {
	m := NewToastModel()
	// All items are expired.
	m.items = []ToastItem{
		makeExpiredItem("old1", "info"),
		makeExpiredItem("old2", "error"),
	}

	m2, cmd := sendTick(m)

	assert.Equal(t, 0, m2.Count(), "all expired items should be removed")
	assert.Nil(t, cmd, "no tick should be returned when queue drains to empty")
}

// ---------------------------------------------------------------------------
// TestView_EmptyReturnsEmpty
// ---------------------------------------------------------------------------

func TestView_EmptyReturnsEmpty(t *testing.T) {
	m := NewToastModel()
	assert.Equal(t, "", m.View(), "empty toast model should return empty string")
}

// ---------------------------------------------------------------------------
// TestView_SingleToast_ContainsMessage
// ---------------------------------------------------------------------------

func TestView_SingleToast_ContainsMessage(t *testing.T) {
	m := NewToastModel()
	m.items = []ToastItem{makeYoungItem("deployment complete", "success")}

	view := m.View()

	assert.NotEmpty(t, view, "view should not be empty when items exist")
	assert.Contains(t, view, "deployment complete", "message text must appear in view")
}

// ---------------------------------------------------------------------------
// TestView_LevelColors
// ---------------------------------------------------------------------------

func TestView_LevelColors(t *testing.T) {
	levels := []model.ToastLevel{"info", "success", "warning", "error"}
	views := make(map[model.ToastLevel]string, len(levels))

	for _, level := range levels {
		m := NewToastModel()
		m.items = []ToastItem{makeYoungItem("msg", level)}
		views[level] = m.View()
	}

	// Each level must produce a non-empty view containing the message.
	for _, level := range levels {
		assert.NotEmpty(t, views[level], "view for level %q must not be empty", level)
		assert.Contains(t, views[level], "msg", "message must appear for level %q", level)
	}

	// Different levels should produce visually distinct output (different
	// border/icon ANSI escape sequences).
	assert.NotEqual(t, views["info"], views["error"],
		"info and error views should differ visually")
	assert.NotEqual(t, views["success"], views["warning"],
		"success and warning views should differ visually")
}

// ---------------------------------------------------------------------------
// TestCount_TracksItems
// ---------------------------------------------------------------------------

func TestCount_TracksItems(t *testing.T) {
	m := NewToastModel()
	assert.Equal(t, 0, m.Count())

	m, _ = sendToast(m, "a", "info")
	assert.Equal(t, 1, m.Count())

	m, _ = sendToast(m, "b", "info")
	assert.Equal(t, 2, m.Count())

	m, _ = sendToast(m, "c", "info")
	assert.Equal(t, 3, m.Count())
}

// ---------------------------------------------------------------------------
// TestSetSize_UpdatesDimensions
// ---------------------------------------------------------------------------

func TestSetSize_UpdatesDimensions(t *testing.T) {
	m := NewToastModel()
	assert.Equal(t, 0, m.width)
	assert.Equal(t, 0, m.height)

	m.SetSize(120, 40)

	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

// ---------------------------------------------------------------------------
// Additional: unknown level falls back to info styling
// ---------------------------------------------------------------------------

func TestView_UnknownLevelFallsBackToInfo(t *testing.T) {
	m := NewToastModel()
	m.items = []ToastItem{makeYoungItem("fallback test", "unknown-level")}

	view := m.View()
	assert.Contains(t, view, "fallback test", "message must appear for unknown level")
	assert.NotEmpty(t, view)
}

// ---------------------------------------------------------------------------
// Additional: second toast added while tick is already running
// ---------------------------------------------------------------------------

func TestUpdate_ToastMsg_NoExtraTickWhenAlreadyTicking(t *testing.T) {
	m := NewToastModel()

	// First toast starts the tick.
	m, cmd1 := sendToast(m, "first", "info")
	require.NotNil(t, cmd1, "first toast should start tick")

	// Second toast: queue is non-empty so no new tick cmd is returned.
	_, cmd2 := sendToast(m, "second", "info")
	assert.Nil(t, cmd2, "second toast should not restart tick (already running)")
}

// ---------------------------------------------------------------------------
// HandleMsg pointer-receiver
// ---------------------------------------------------------------------------

func TestHandleMsg_PointerReceiverMutates(t *testing.T) {
	m := NewToastModel()
	m.HandleMsg(model.ToastMsg{Text: "test notification", Level: "info"})
	assert.Equal(t, 1, m.Count(), "HandleMsg should add toast via pointer mutation")
}

// ---------------------------------------------------------------------------
// Additional: ToastItem.expired uses custom duration
// ---------------------------------------------------------------------------

func TestToastItem_Expired_CustomDuration(t *testing.T) {
	t.Run("zero duration uses default", func(t *testing.T) {
		item := ToastItem{
			Message:   "test",
			CreatedAt: time.Now().Add(-6 * time.Second),
			Duration:  0,
		}
		assert.True(t, item.expired(), "zero-duration item older than 5s should expire")
	})

	t.Run("custom short duration expires quickly", func(t *testing.T) {
		item := ToastItem{
			Message:   "test",
			CreatedAt: time.Now().Add(-2 * time.Second),
			Duration:  time.Second,
		}
		assert.True(t, item.expired(), "item past its custom duration should expire")
	})

	t.Run("fresh item not expired", func(t *testing.T) {
		item := ToastItem{
			Message:   "test",
			CreatedAt: time.Now(),
			Duration:  defaultDuration,
		}
		assert.False(t, item.expired(), "just-created item should not be expired")
	})
}
