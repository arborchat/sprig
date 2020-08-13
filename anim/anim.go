/*
Package anim provides simple animation primitives
*/
package anim

import (
	"time"

	"gioui.org/layout"
	"gioui.org/op"
)

// Normal holds state for an animation between two states that
// is not invertible.
type Normal struct {
	time.Duration
	StartTime time.Time
}

// Progress returns the current progress through the animation
// as a value in the range [0,1]
func (n *Normal) Progress(gtx layout.Context) float32 {
	if n.Duration == time.Duration(0) {
		return 0
	}
	progressDur := gtx.Now.Sub(n.StartTime)
	if progressDur > n.Duration {
		return 1
	}
	op.InvalidateOp{}.Add(gtx.Ops)
	progress := float32(progressDur.Milliseconds()) / float32(n.Duration.Milliseconds())
	return progress
}

func (n *Normal) Start(now time.Time) {
	n.StartTime = now
}

func (n *Normal) SetDuration(d time.Duration) {
	n.Duration = d
}

func (n *Normal) Animating(gtx layout.Context) bool {
	if n.Duration == 0 {
		return false
	}
	if gtx.Now.After(n.StartTime.Add(n.Duration)) {
		return false
	}
	return true
}
