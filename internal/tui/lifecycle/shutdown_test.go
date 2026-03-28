package lifecycle

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

// mockDriver records which lifecycle methods were called and in what order.
type mockDriver struct {
	mu              sync.Mutex
	interruptCalled bool
	shutdownCalled  bool
	callOrder       []string
	interruptDelay  time.Duration
	shutdownDelay   time.Duration
	interruptErr    error
	shutdownErr     error
}

func (d *mockDriver) Interrupt() error {
	if d.interruptDelay > 0 {
		time.Sleep(d.interruptDelay)
	}
	d.mu.Lock()
	d.interruptCalled = true
	d.callOrder = append(d.callOrder, "interrupt")
	d.mu.Unlock()
	return d.interruptErr
}

func (d *mockDriver) Shutdown() error {
	if d.shutdownDelay > 0 {
		time.Sleep(d.shutdownDelay)
	}
	d.mu.Lock()
	d.shutdownCalled = true
	d.callOrder = append(d.callOrder, "shutdown")
	d.mu.Unlock()
	return d.shutdownErr
}

// mockBridge records whether Shutdown was called.
type mockBridge struct {
	mu             sync.Mutex
	shutdownCalled bool
	callOrder      []string
}

func (b *mockBridge) Shutdown() {
	b.mu.Lock()
	b.shutdownCalled = true
	b.callOrder = append(b.callOrder, "bridge_shutdown")
	b.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Helper: build a fast-timing ShutdownManager suitable for tests.
// ---------------------------------------------------------------------------

func fastOpts(driver Shutdownable, bridge BridgeShutdownable) ShutdownOpts {
	return ShutdownOpts{
		Driver:      driver,
		Bridge:      bridge,
		TotalBudget: 2 * time.Second,
		CLIBudget:   50 * time.Millisecond,
		HookBudget:  20 * time.Millisecond,
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestShutdown_HappyPath(t *testing.T) {
	t.Parallel()

	driver := &mockDriver{}
	bridge := &mockBridge{}

	var saverCalled bool
	opts := fastOpts(driver, bridge)
	opts.SessionSaver = func() { saverCalled = true }

	sm := NewShutdownManager(opts)
	err := sm.Shutdown()

	require.NoError(t, err)
	assert.True(t, saverCalled, "sessionSaver should have been called")
	assert.True(t, driver.interruptCalled, "driver.Interrupt should have been called")
	assert.True(t, driver.shutdownCalled, "driver.Shutdown should have been called")
	assert.True(t, bridge.shutdownCalled, "bridge.Shutdown should have been called")
}

func TestShutdown_DoubleCallIsNoop(t *testing.T) {
	t.Parallel()

	var saverCount int32
	driver := &mockDriver{}
	bridge := &mockBridge{}
	opts := fastOpts(driver, bridge)
	opts.SessionSaver = func() { atomic.AddInt32(&saverCount, 1) }

	sm := NewShutdownManager(opts)

	err1 := sm.Shutdown()
	err2 := sm.Shutdown()

	require.NoError(t, err1)
	require.NoError(t, err2, "second call must return nil without executing side effects")

	// Side effects must fire exactly once despite two Shutdown() calls.
	// The first call executes Interrupt + Shutdown (2 entries); the second is a
	// no-op and adds nothing.
	assert.EqualValues(t, 1, atomic.LoadInt32(&saverCount), "sessionSaver must be called exactly once")
	assert.Equal(t, 2, len(driver.callOrder),
		"driver callOrder should have exactly 2 entries (interrupt + shutdown) from the first call only")
	assert.Equal(t, "interrupt", driver.callOrder[0], "first driver call should be interrupt")
	assert.Equal(t, "shutdown", driver.callOrder[1], "second driver call should be shutdown")
}

func TestShutdown_NilDriver(t *testing.T) {
	t.Parallel()

	bridge := &mockBridge{}
	opts := fastOpts(nil, bridge)

	sm := NewShutdownManager(opts)
	err := sm.Shutdown()

	require.NoError(t, err, "nil driver should not cause an error")
	assert.True(t, bridge.shutdownCalled, "bridge.Shutdown should still be called with nil driver")
}

func TestShutdown_NilBridge(t *testing.T) {
	t.Parallel()

	driver := &mockDriver{}
	opts := fastOpts(driver, nil)

	sm := NewShutdownManager(opts)
	err := sm.Shutdown()

	require.NoError(t, err, "nil bridge should not cause an error")
	assert.True(t, driver.interruptCalled, "driver.Interrupt should still be called with nil bridge")
	assert.True(t, driver.shutdownCalled, "driver.Shutdown should still be called with nil bridge")
}

func TestShutdown_NilSessionSaver(t *testing.T) {
	t.Parallel()

	driver := &mockDriver{}
	bridge := &mockBridge{}
	opts := fastOpts(driver, bridge)
	// SessionSaver intentionally left nil.

	sm := NewShutdownManager(opts)
	err := sm.Shutdown()

	require.NoError(t, err, "nil sessionSaver should not cause an error")
	assert.True(t, driver.interruptCalled, "driver.Interrupt should be called even with nil sessionSaver")
}

func TestShutdown_InterruptError(t *testing.T) {
	t.Parallel()

	driver := &mockDriver{
		interruptErr: errors.New("process not running"),
	}
	bridge := &mockBridge{}
	opts := fastOpts(driver, bridge)

	sm := NewShutdownManager(opts)
	err := sm.Shutdown()

	// Interrupt error must be swallowed; Shutdown must still proceed.
	require.NoError(t, err)
	assert.True(t, driver.interruptCalled)
	assert.True(t, driver.shutdownCalled, "Shutdown must proceed even when Interrupt errors")
	assert.True(t, bridge.shutdownCalled)
}

func TestShutdown_StatusCallbacks(t *testing.T) {
	t.Parallel()

	driver := &mockDriver{}
	bridge := &mockBridge{}
	opts := fastOpts(driver, bridge)

	var statuses []string
	var mu sync.Mutex
	opts.OnStatus = func(s string) {
		mu.Lock()
		statuses = append(statuses, s)
		mu.Unlock()
	}

	sm := NewShutdownManager(opts)
	require.NoError(t, sm.Shutdown())

	mu.Lock()
	got := make([]string, len(statuses))
	copy(got, statuses)
	mu.Unlock()

	expected := []string{
		"Saving session...",
		"Stopping CLI...",
		"Closing bridge...",
		"Waiting for hooks...",
		"Shutdown complete",
	}
	assert.Equal(t, expected, got, "status callbacks must be called in the correct order")
}

func TestShutdown_TimingBudget(t *testing.T) {
	t.Parallel()

	driver := &mockDriver{}
	bridge := &mockBridge{}

	opts := ShutdownOpts{
		Driver:      driver,
		Bridge:      bridge,
		TotalBudget: 500 * time.Millisecond,
		CLIBudget:   100 * time.Millisecond,
		HookBudget:  50 * time.Millisecond,
	}

	sm := NewShutdownManager(opts)

	start := time.Now()
	err := sm.Shutdown()
	elapsed := time.Since(start)

	require.NoError(t, err)
	// Should complete well within the 500 ms total budget.
	assert.Less(t, elapsed, 500*time.Millisecond,
		"shutdown must complete within the total budget (%s elapsed)", elapsed)
}

func TestShutdown_PhaseOrdering(t *testing.T) {
	t.Parallel()

	// Record wall-clock timestamps for each phase to verify ordering.
	type event struct {
		name string
		at   time.Time
	}
	var mu sync.Mutex
	var events []event

	record := func(name string) {
		mu.Lock()
		events = append(events, event{name: name, at: time.Now()})
		mu.Unlock()
	}

	driver := &mockDriver{}
	bridge := &mockBridge{}

	var saverAt, interruptAt, shutdownAt, bridgeAt time.Time

	opts := ShutdownOpts{
		Driver: &recordingDriver{
			onInterrupt: func() error {
				interruptAt = time.Now()
				record("interrupt")
				return nil
			},
			onShutdown: func() error {
				shutdownAt = time.Now()
				record("shutdown")
				return nil
			},
		},
		Bridge: &recordingBridge{
			onShutdown: func() {
				bridgeAt = time.Now()
				record("bridge")
			},
		},
		SessionSaver: func() {
			saverAt = time.Now()
			record("saver")
		},
		TotalBudget: 2 * time.Second,
		CLIBudget:   30 * time.Millisecond,
		HookBudget:  10 * time.Millisecond,
	}

	sm := NewShutdownManager(opts)
	require.NoError(t, sm.Shutdown())

	// Ordering assertions.
	assert.False(t, saverAt.IsZero(), "sessionSaver must be called")
	assert.False(t, interruptAt.IsZero(), "Interrupt must be called")
	assert.False(t, shutdownAt.IsZero(), "Shutdown must be called")
	assert.False(t, bridgeAt.IsZero(), "bridge.Shutdown must be called")

	assert.True(t, saverAt.Before(interruptAt) || saverAt.Equal(interruptAt),
		"sessionSaver must complete before Interrupt")
	assert.True(t, interruptAt.Before(shutdownAt) || interruptAt.Equal(shutdownAt),
		"Interrupt must be called before driver.Shutdown")
	assert.True(t, shutdownAt.Before(bridgeAt) || shutdownAt.Equal(bridgeAt),
		"driver.Shutdown must be called before bridge.Shutdown")

	_ = driver
	_ = bridge
	_ = events
}

func TestShutdown_IsDone(t *testing.T) {
	t.Parallel()

	sm := NewShutdownManager(fastOpts(nil, nil))

	assert.False(t, sm.IsDone(), "IsDone must return false before Shutdown")
	require.NoError(t, sm.Shutdown())
	assert.True(t, sm.IsDone(), "IsDone must return true after Shutdown")
}

func TestShutdown_ConcurrentSafe(t *testing.T) {
	t.Parallel()

	var saverCount int32
	opts := fastOpts(nil, nil)
	opts.SessionSaver = func() { atomic.AddInt32(&saverCount, 1) }

	sm := NewShutdownManager(opts)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = sm.Shutdown()
		}()
	}

	wg.Wait()

	assert.EqualValues(t, 1, atomic.LoadInt32(&saverCount),
		"sessionSaver must be executed exactly once across all concurrent calls")
	assert.True(t, sm.IsDone())
}

// ---------------------------------------------------------------------------
// Helpers for TestShutdown_PhaseOrdering
// ---------------------------------------------------------------------------

// recordingDriver is an instrumented Shutdownable that calls user-supplied
// callbacks so the test can record timestamps.
type recordingDriver struct {
	onInterrupt func() error
	onShutdown  func() error
}

func (r *recordingDriver) Interrupt() error {
	if r.onInterrupt != nil {
		return r.onInterrupt()
	}
	return nil
}

func (r *recordingDriver) Shutdown() error {
	if r.onShutdown != nil {
		return r.onShutdown()
	}
	return nil
}

// recordingBridge is an instrumented BridgeShutdownable.
type recordingBridge struct {
	onShutdown func()
}

func (r *recordingBridge) Shutdown() {
	if r.onShutdown != nil {
		r.onShutdown()
	}
}
