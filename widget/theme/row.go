package theme

import (
	"image"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/x/richtext"
	chatlayout "git.sr.ht/~gioverse/chat/layout"
	"git.sr.ht/~whereswaldon/sprig/ds"
	sprigwidget "git.sr.ht/~whereswaldon/sprig/widget"
)

// ReplyRowStyle configures the presentation of a row of chat history.
type ReplyRowStyle struct {
	// VerticalMarginStyle separates the chat message vertically from
	// other messages.
	chatlayout.VerticalMarginStyle
	// MaxWidth constrains the maximum display width of a message.
	// ReplyStyle configures the presentation of the message.
	MaxWidth unit.Dp
	ReplyStyle
	*sprigwidget.Reply
}

var DefaultMaxWidth = unit.Dp(600)

// ReplyRow configures a row with sensible defaults.
func ReplyRow(th *Theme, state *sprigwidget.Reply, anim *sprigwidget.ReplyAnimationState, rd ds.ReplyData, richContent richtext.TextStyle) ReplyRowStyle {
	return ReplyRowStyle{
		VerticalMarginStyle: chatlayout.VerticalMargin(),
		ReplyStyle:          Reply(th, anim, rd, richContent, false),
		MaxWidth:            DefaultMaxWidth,
		Reply:               state,
	}
}

// Layout the row.
func (r ReplyRowStyle) Layout(gtx C) D {
	return r.VerticalMarginStyle.Layout(gtx, func(gtx C) D {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{
			Left: interpolateInset(r.ReplyAnimationState, r.ReplyAnimationState.Progress(gtx)),
		}.Layout(gtx, func(gtx C) D {
			gtx.Constraints.Max.X -= gtx.Dp(descendantInset) + gtx.Dp(defaultInset)
			if mw := gtx.Dp(r.MaxWidth); gtx.Constraints.Max.X > mw {
				gtx.Constraints.Max.X = mw
				gtx.Constraints.Min = gtx.Constraints.Constrain(gtx.Constraints.Min)
			}
			return layout.Stack{}.Layout(gtx,
				layout.Stacked(r.ReplyStyle.Layout),
				layout.Expanded(r.Reply.Polyclick.Layout),
			)
		})
		call := macro.Stop()

		defer pointer.PassOp{}.Push(gtx.Ops).Pop()
		rect := image.Rectangle{
			Max: image.Point{
				X: gtx.Constraints.Max.X,
				Y: dims.Size.Y,
			},
		}
		defer clip.Rect(rect).Push(gtx.Ops).Pop()
		r.Reply.Layout(gtx, dims.Size.X)

		offset := r.Reply.DragOffset()
		op.Offset(f32.Pt(offset, 0).Round()).Add(gtx.Ops)
		call.Add(gtx.Ops)

		return dims
	})
}
