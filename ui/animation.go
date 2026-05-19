package ui

import (
	"math"
	"time"
)

// AnimationStatus describes the lifecycle state of an animation controller.
type AnimationStatus int

const (
	// AnimationIdle indicates that the controller is stopped at its current value.
	AnimationIdle AnimationStatus = iota
	// AnimationForward indicates that the controller is advancing toward 1.
	AnimationForward
	// AnimationCompleted indicates that the controller reached the end value.
	AnimationCompleted
)

// Curve maps a linear animation progress value in [0, 1] to an eased value.
type Curve func(float64) float64

// Linear returns t clamped to the animation progress range.
func Linear(t float64) float64 {
	return clamp01(t)
}

// EaseInOut returns a smoothstep curve for t clamped to the animation progress range.
func EaseInOut(t float64) float64 {
	t = clamp01(t)
	return t * t * (3 - 2*t)
}

// FloatTween interpolates between two float64 values.
type FloatTween struct {
	// Begin is the value returned at progress 0.
	Begin float64
	// End is the value returned at progress 1.
	End float64
}

// At returns the interpolated value at progress value.
func (t FloatTween) At(value float64) float64 {
	return t.Begin + (t.End-t.Begin)*value
}

// AnimationOptions configures a state-owned animation controller.
type AnimationOptions struct {
	// Duration is the time from progress 0 to progress 1.
	Duration time.Duration
	// Curve maps raw progress to the value returned by AnimationController.Value.
	Curve Curve
}

// AnimationController drives frame-scheduled animation progress for a StateBase.
type AnimationController struct {
	owner    *StateBase
	duration time.Duration
	curve    Curve
	status   AnimationStatus
	start    time.Time
	value    float64
	dirty    bool
	disposed bool
}

// Forward starts the animation using the current wall-clock time.
func (c *AnimationController) Forward() {
	c.ForwardAt(time.Now())
}

// ForwardAt starts the animation using now as its start time.
func (c *AnimationController) ForwardAt(now time.Time) {
	c.checkNotDisposed()
	c.start = now
	c.value = 0
	c.dirty = true
	if c.duration <= 0 {
		c.status = AnimationCompleted
		c.value = 1
		c.requestBuild()
		return
	}
	c.status = AnimationForward
	c.register()
	c.requestBuild()
}

// Stop pauses a running animation at its current value.
func (c *AnimationController) Stop() {
	c.checkNotDisposed()
	if c.status != AnimationForward {
		return
	}
	c.status = AnimationIdle
	c.dirty = true
	c.unregister()
	c.requestBuild()
}

// Reset stops the animation and returns it to 0.
func (c *AnimationController) Reset() {
	c.checkNotDisposed()
	c.status = AnimationIdle
	c.value = 0
	c.dirty = true
	c.unregister()
	c.requestBuild()
}

// Value returns the curved animation progress.
func (c *AnimationController) Value() float64 {
	return c.curve(c.value)
}

// RawValue returns the uncurved animation progress.
func (c *AnimationController) RawValue() float64 {
	return c.value
}

// Status returns the controller's current lifecycle state.
func (c *AnimationController) Status() AnimationStatus {
	return c.status
}

// Running reports whether the controller is currently advancing.
func (c *AnimationController) Running() bool {
	return c.status == AnimationForward
}

func (c *AnimationController) tick(now time.Time) bool {
	if c.disposed {
		return false
	}
	wasDirty := c.dirty
	c.dirty = false
	if c.status != AnimationForward {
		return wasDirty
	}
	old := c.value
	c.value = clamp01(float64(now.Sub(c.start)) / float64(c.duration))
	if c.value >= 1 {
		c.status = AnimationCompleted
		c.unregister()
	}
	return wasDirty || c.value != old
}

func (c *AnimationController) dispose() {
	if c.disposed {
		return
	}
	c.unregister()
	c.owner = nil
	c.disposed = true
}

func (c *AnimationController) register() {
	if c.owner == nil || c.owner.element == nil || c.owner.element.owner == nil {
		return
	}
	c.owner.element.owner.app.registerAnimation(c)
}

func (c *AnimationController) unregister() {
	if c.owner == nil || c.owner.element == nil || c.owner.element.owner == nil {
		return
	}
	c.owner.element.owner.app.unregisterAnimation(c)
}

func (c *AnimationController) requestBuild() {
	if c.owner == nil || c.owner.element == nil || c.owner.element.owner == nil {
		return
	}
	if c.owner.element.owner.building {
		c.owner.element.owner.app.RequestFrame()
		return
	}
	c.owner.MarkNeedsBuild()
}

func (c *AnimationController) checkNotDisposed() {
	if c.disposed {
		panic("ui: AnimationController used after Dispose")
	}
}

func clamp01(value float64) float64 {
	if math.IsNaN(value) || value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
