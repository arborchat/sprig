package widget

import (
	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
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
func (r *Reply) Layout(gtx layout.Context) layout.Dimensions {
	r.Drag.Add(gtx.Ops)

	for _, e := range r.Drag.Events(gtx.Metric, gtx, gesture.Horizontal) {
		switch e.Type {
		case pointer.Press:
			r.dragStart = e.Position.X
			r.dragOffset = 0
		case pointer.Drag:
			r.dragOffset = e.Position.X - r.dragStart
		case pointer.Release:
			r.dragStart = 0
			r.dragOffset = 0
		}
	}

	if r.dragOffset < 0 {
		r.dragOffset = 0
	}
	return layout.Dimensions{}
}

func (r *Reply) DragOffset() float32 {
	return r.dragOffset
}
