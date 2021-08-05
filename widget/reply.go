package widget

import (
	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/x/richtext"
	"git.sr.ht/~whereswaldon/forest-go/fields"
)

// Reply holds ui state for each reply.
type Reply struct {
	Hash    *fields.QualifiedHash
	Content string
	Polyclick
	richtext.InteractiveText
	ReplyStatus
	gesture.Drag
	dragStart, dragOffset float32
	dragFinished          bool
	events                []ReplyEvent
}

func (r *Reply) WithHash(h *fields.QualifiedHash) *Reply {
	r.Hash = h
	return r
}

func (r *Reply) WithContent(s string) *Reply {
	r.Content = s
	return r
}

// Layout adds the drag operation (using the most recently laid out
// pointer hit area) and processes drag status.
func (r *Reply) Layout(gtx layout.Context, replyWidth int) layout.Dimensions {
	r.Drag.Add(gtx.Ops)

	for _, e := range r.Drag.Events(gtx.Metric, gtx, gesture.Horizontal) {
		switch e.Type {
		case pointer.Press:
			r.dragStart = e.Position.X
			r.dragOffset = 0
			r.dragFinished = false
		case pointer.Drag:
			r.dragOffset = e.Position.X - r.dragStart
		case pointer.Release, pointer.Cancel:
			r.dragStart = 0
			r.dragOffset = 0
			r.dragFinished = false
		}
	}

	if r.Dragging() {
		op.InvalidateOp{}.Add(gtx.Ops)
	}

	if r.dragOffset < 0 {
		r.dragOffset = 0
	}
	if replyWidth+int(r.dragOffset) >= gtx.Constraints.Max.X {
		r.dragOffset = float32(gtx.Constraints.Max.X - replyWidth)
		if !r.dragFinished {
			r.events = append(r.events, ReplyEvent{Type: SwipedRight})
			r.dragFinished = true
		}
	}
	return layout.Dimensions{}
}

// DragOffset returns the X-axis offset for this reply as a result of a user
// dragging it.
func (r *Reply) DragOffset() float32 {
	return r.dragOffset
}

// Events returns reply events that have occurred since the last call to Events.
func (r *Reply) Events() []ReplyEvent {
	events := r.events
	r.events = r.events[:0]
	return events
}

// ReplyEvent models a change or interaction with a reply.
type ReplyEvent struct {
	Type ReplyEventType
}

// ReplyEventType encodes a kind of event.
type ReplyEventType uint8

const (
	// SwipedRight indicates that a given reply was swiped to the right margin
	// by a user.
	SwipedRight ReplyEventType = iota
)
