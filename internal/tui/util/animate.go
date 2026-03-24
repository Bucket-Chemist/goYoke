// Package util provides shared utility functions for the GOgent-Fortress TUI.
package util

import (
	"time"

	"github.com/charmbracelet/harmonica"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	// AnimationFPS is the target frame rate for spring animations.
	AnimationFPS = 60
	// AnimationInterval is the tick interval for AnimationFPS frames per second.
	AnimationInterval = time.Second / AnimationFPS
	// SettleThreshold is the minimum delta (in both position and velocity) at
	// which a spring is considered settled. Smaller values give more precise
	// convergence at the cost of additional frames.
	SettleThreshold = 0.1
)

// SpringAnimation wraps harmonica.Spring for use in Bubbletea models.
//
// SpringAnimation is a value type; copy it when embedding in a model struct.
// The zero value is settled at position 0 with no target.
//
// Typical usage inside a Bubbletea Update loop:
//
//	case util.AnimateTickMsg:
//	    val, settled := m.spring.Tick()
//	    m.currentPos = val
//	    if !settled {
//	        return m, util.AnimateTickCmd()
//	    }
//	    return m, nil
type SpringAnimation struct {
	spring   harmonica.Spring
	value    float64
	velocity float64
	target   float64
	settled  bool
}

// NewSpring creates a SpringAnimation with the given angular frequency and
// damping ratio, pre-computed for AnimationFPS frames per second.
//
// angularFrequency controls the speed of motion — higher values make the
// spring react faster (typical range: 3–12).
//
// dampingRatio controls oscillation behavior:
//   - dampingRatio > 1: over-damped (slow, no overshoot)
//   - dampingRatio = 1: critically-damped (fastest without overshoot)
//   - dampingRatio < 1: under-damped (fast, overshoots and oscillates)
//
// Typical values: angularFrequency=6.0, dampingRatio=0.5.
// The returned SpringAnimation starts settled at position 0.
func NewSpring(angularFrequency, dampingRatio float64) SpringAnimation {
	return SpringAnimation{
		spring:  harmonica.NewSpring(harmonica.FPS(AnimationFPS), angularFrequency, dampingRatio),
		settled: true,
	}
}

// SetTarget sets the position the spring will animate toward and marks the
// spring as unsettled so Tick calls will advance the animation.
//
// Calling SetTarget while the spring is in motion resumes from the current
// position and velocity toward the new target, enabling smooth retargeting
// mid-animation.
func (s *SpringAnimation) SetTarget(target float64) {
	s.target = target
	s.settled = false
}

// Value returns the current animated position.
func (s *SpringAnimation) Value() float64 {
	return s.value
}

// IsSettled reports whether the spring has reached its target. A settled
// spring does not need further Tick calls until SetTarget is called again.
func (s *SpringAnimation) IsSettled() bool {
	return s.settled
}

// Tick advances the spring by one animation frame.
//
// It returns the updated position and whether the spring has settled.
// When settled is true the position is snapped exactly to the target and
// velocity is reset to zero.
//
// Callers should chain AnimateTickCmd() when settled is false:
//
//	val, settled := m.spring.Tick()
//	if !settled {
//	    cmds = append(cmds, util.AnimateTickCmd())
//	}
func (s *SpringAnimation) Tick() (value float64, settled bool) {
	if s.settled {
		return s.value, true
	}

	s.value, s.velocity = s.spring.Update(s.value, s.velocity, s.target)

	if absFloat(s.value-s.target) < SettleThreshold && absFloat(s.velocity) < SettleThreshold {
		s.value = s.target
		s.velocity = 0
		s.settled = true
	}

	return s.value, s.settled
}

// absFloat returns the absolute value of x.
func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// AnimateTickMsg is a Bubbletea message that triggers one animation frame.
// Components that drive spring animations should match on this message type
// in their Update functions and call Tick on any active SpringAnimations.
type AnimateTickMsg struct{}

// AnimateTickCmd returns a tea.Cmd that delivers AnimateTickMsg after one
// frame interval (1/AnimationFPS seconds). Components should return this
// command from Update whenever a spring is unsettled.
func AnimateTickCmd() tea.Cmd {
	return tea.Tick(AnimationInterval, func(_ time.Time) tea.Msg {
		return AnimateTickMsg{}
	})
}
