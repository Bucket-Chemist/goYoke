package util

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Constants / helpers
// ---------------------------------------------------------------------------

// maxTicksForSettling is the upper bound used in convergence tests.  A
// well-configured spring (frequency=6, damping=0.5) settles within ~120
// frames; we allow 300 as a generous upper bound for all parameter ranges.
const maxTicksForSettling = 300

// runUntilSettled advances the spring until it reports settled or the tick
// limit is reached.  It returns the final value and the number of ticks used.
func runUntilSettled(s *SpringAnimation, limit int) (finalValue float64, ticks int) {
	for i := range limit {
		val, settled := s.Tick()
		if settled {
			return val, i + 1
		}
		_ = val
	}
	return s.Value(), limit
}

// ---------------------------------------------------------------------------
// NewSpring
// ---------------------------------------------------------------------------

func TestNewSpring_StartsSettled(t *testing.T) {
	t.Parallel()

	s := NewSpring(6.0, 0.5)

	if !s.IsSettled() {
		t.Error("expected new spring to start settled")
	}
	if s.Value() != 0.0 {
		t.Errorf("expected initial value 0; got %v", s.Value())
	}
}

func TestNewSpring_SetTargetMakesUnsettled(t *testing.T) {
	t.Parallel()

	s := NewSpring(6.0, 0.5)
	s.SetTarget(100.0)

	if s.IsSettled() {
		t.Error("expected spring to be unsettled after SetTarget")
	}
}

// ---------------------------------------------------------------------------
// Value / IsSettled
// ---------------------------------------------------------------------------

func TestValue_ReturnsCurrentPosition(t *testing.T) {
	t.Parallel()

	s := NewSpring(6.0, 0.5)
	initial := s.Value()
	if initial != 0.0 {
		t.Errorf("expected 0; got %v", initial)
	}

	s.SetTarget(50.0)
	s.Tick() //nolint:errcheck // ignore return
	afterTick := s.Value()

	// After one tick toward 50, value must have moved away from 0.
	if afterTick <= 0.0 {
		t.Errorf("expected value > 0 after one tick toward 50; got %v", afterTick)
	}
}

func TestIsSettled_TrueWhenConverged(t *testing.T) {
	t.Parallel()

	s := NewSpring(6.0, 0.5)
	s.SetTarget(100.0)

	_, ticks := runUntilSettled(&s, maxTicksForSettling)

	if !s.IsSettled() {
		t.Errorf("spring did not settle within %d ticks", ticks)
	}
	if s.Value() != 100.0 {
		t.Errorf("expected settled value = 100; got %v", s.Value())
	}
}

// ---------------------------------------------------------------------------
// Tick: convergence
// ---------------------------------------------------------------------------

func TestTick_ConvergesInReasonableTicks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		freq     float64
		damping  float64
		from     float64
		target   float64
		maxTicks int
	}{
		{"0→100 critically-damped", 6.0, 1.0, 0, 100, maxTicksForSettling},
		{"0→100 under-damped", 6.0, 0.5, 0, 100, maxTicksForSettling},
		{"0→100 over-damped", 6.0, 2.0, 0, 100, maxTicksForSettling},
		{"0→1000 large transition", 6.0, 0.5, 0, 1000, maxTicksForSettling},
		{"100→0 reverse", 6.0, 0.5, 100, 0, maxTicksForSettling},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := NewSpring(tc.freq, tc.damping)
			// Start from a non-zero position by manually adjusting value.
			// We do this by targeting tc.from and running to settled first,
			// then setting the real target.
			if tc.from != 0 {
				s.SetTarget(tc.from)
				runUntilSettled(&s, maxTicksForSettling)
			}

			s.SetTarget(tc.target)
			finalVal, ticks := runUntilSettled(&s, tc.maxTicks)

			if !s.IsSettled() {
				t.Errorf("[%s] spring did not settle within %d ticks (got %d); final value %v",
					tc.name, tc.maxTicks, ticks, finalVal)
			}
			if finalVal != tc.target {
				t.Errorf("[%s] expected settled value %v; got %v", tc.name, tc.target, finalVal)
			}
		})
	}
}

func TestTick_ValueApproachesTarget(t *testing.T) {
	t.Parallel()

	s := NewSpring(6.0, 0.5)
	s.SetTarget(100.0)

	// After 10 ticks the value must be noticeably closer to 100 than 0.
	for range 10 {
		s.Tick() //nolint:errcheck
	}

	if s.Value() <= 0.0 {
		t.Errorf("expected value > 0 after 10 ticks; got %v", s.Value())
	}
	if s.Value() >= 100.0 && !s.IsSettled() {
		// Overshoot is possible for under-damped springs; that's fine as long
		// as it eventually settles — tested separately above.
		t.Logf("value %v has overshot target 100 at tick 10 (under-damped behaviour)", s.Value())
	}
}

// ---------------------------------------------------------------------------
// Tick: settled path
// ---------------------------------------------------------------------------

func TestTick_SettledSpringReturnsSameValue(t *testing.T) {
	t.Parallel()

	s := NewSpring(6.0, 0.5)
	// Do not call SetTarget — spring is settled at 0.

	for range 5 {
		val, settled := s.Tick()
		if !settled {
			t.Error("expected settled=true for spring with no target change")
		}
		if val != 0.0 {
			t.Errorf("expected value 0 for settled spring; got %v", val)
		}
	}
}

// ---------------------------------------------------------------------------
// SetTarget: retargeting mid-animation
// ---------------------------------------------------------------------------

func TestSetTarget_RetargetMidAnimation(t *testing.T) {
	t.Parallel()

	s := NewSpring(6.0, 0.5)
	s.SetTarget(200.0)

	// Run halfway (about 50 ticks).
	for range 50 {
		s.Tick() //nolint:errcheck
	}

	midValue := s.Value()

	// Retarget to 0 from wherever we are now.
	s.SetTarget(0.0)

	if s.IsSettled() {
		t.Error("expected unsettled after SetTarget(0) mid-animation")
	}

	finalVal, ticks := runUntilSettled(&s, maxTicksForSettling)

	if !s.IsSettled() {
		t.Errorf("spring did not re-settle within %d ticks after retarget (mid-value was %v, final %v)",
			ticks, midValue, finalVal)
	}
	if finalVal != 0.0 {
		t.Errorf("expected final value 0 after retarget to 0; got %v", finalVal)
	}
}

func TestSetTarget_MultipleRetargets(t *testing.T) {
	t.Parallel()

	targets := []float64{50.0, 20.0, 80.0, 10.0}
	s := NewSpring(6.0, 0.5)

	for _, target := range targets {
		s.SetTarget(target)
		finalVal, _ := runUntilSettled(&s, maxTicksForSettling)

		if !s.IsSettled() {
			t.Errorf("spring did not settle for target %v; final value %v", target, finalVal)
		}
		if finalVal != target {
			t.Errorf("expected %v; got %v", target, finalVal)
		}
	}
}

// ---------------------------------------------------------------------------
// SetTarget: zero-to-zero already settled
// ---------------------------------------------------------------------------

func TestSetTarget_ZeroToZeroAlreadySettled(t *testing.T) {
	t.Parallel()

	s := NewSpring(6.0, 0.5)
	// Spring starts settled at 0; setting target to 0 still marks unsettled
	// but should converge in very few ticks (or immediately if position == target).
	s.SetTarget(0.0)

	// Spring is at 0, target is 0 — first Tick should detect settled immediately.
	val, settled := s.Tick()

	if !settled {
		t.Errorf("expected settled=true when position already equals target; got val=%v", val)
	}
	if val != 0.0 {
		t.Errorf("expected value 0; got %v", val)
	}
}

// ---------------------------------------------------------------------------
// AnimateTickCmd
// ---------------------------------------------------------------------------

func TestAnimateTickCmd_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	cmd := AnimateTickCmd()
	if cmd == nil {
		t.Error("AnimateTickCmd returned nil command")
	}
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

func TestAnimationConstants(t *testing.T) {
	t.Parallel()

	if AnimationFPS <= 0 {
		t.Errorf("AnimationFPS must be positive; got %d", AnimationFPS)
	}
	if AnimationInterval <= 0 {
		t.Error("AnimationInterval must be positive")
	}
	if SettleThreshold <= 0 {
		t.Errorf("SettleThreshold must be positive; got %v", SettleThreshold)
	}

	// Verify interval matches FPS: 60fps → ~16.67ms.
	expectedNs := float64(1e9) / float64(AnimationFPS)
	gotNs := float64(AnimationInterval)
	if math.Abs(gotNs-expectedNs)/expectedNs > 0.01 {
		t.Errorf("AnimationInterval %v does not match 1s/%d", AnimationInterval, AnimationFPS)
	}
}

// ---------------------------------------------------------------------------
// absFloat helper
// ---------------------------------------------------------------------------

func TestAbsFloat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input float64
		want  float64
	}{
		{5.0, 5.0},
		{-5.0, 5.0},
		{0.0, 0.0},
		{-0.001, 0.001},
	}

	for _, tc := range tests {
		tc := tc
		t.Run("", func(t *testing.T) {
			t.Parallel()
			got := absFloat(tc.input)
			if got != tc.want {
				t.Errorf("absFloat(%v) = %v; want %v", tc.input, got, tc.want)
			}
		})
	}
}
