package ui

import (
	"math"
	"time"
)

type AnimationStatus int

const (
	AnimationIdle AnimationStatus = iota
	AnimationForward
	AnimationCompleted
)

type Curve func(float64) float64

func Linear(t float64) float64 {
	return clamp01(t)
}

func EaseInOut(t float64) float64 {
	t = clamp01(t)
	return t * t * (3 - 2*t)
}

type FloatTween struct {
	Begin float64
	End   float64
}

func (t FloatTween) At(value float64) float64 {
	return t.Begin + (t.End-t.Begin)*value
}

type AnimationOptions struct {
	Duration time.Duration
	Curve    Curve
}

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

func (c *AnimationController) Forward() {
	c.ForwardAt(time.Now())
}

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

func (c *AnimationController) Reset() {
	c.checkNotDisposed()
	c.status = AnimationIdle
	c.value = 0
	c.dirty = true
	c.unregister()
	c.requestBuild()
}

func (c *AnimationController) Value() float64 {
	return c.curve(c.value)
}

func (c *AnimationController) RawValue() float64 {
	return c.value
}

func (c *AnimationController) Status() AnimationStatus {
	return c.status
}

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
