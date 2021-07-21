package theme

import (
	"gioui.org/layout"
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
	MaxWidth unit.Value
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
		return layout.Inset{
			Left: interpolateInset(r.ReplyAnimationState, r.ReplyAnimationState.Progress(gtx)),
		}.Layout(gtx, func(gtx C) D {
			gtx.Constraints.Max.X -= gtx.Px(descendantInset) + gtx.Px(defaultInset)
			if mw := gtx.Px(r.MaxWidth); gtx.Constraints.Max.X > mw {
				gtx.Constraints.Max.X = mw
				gtx.Constraints.Min = gtx.Constraints.Constrain(gtx.Constraints.Min)
			}
			return layout.Stack{}.Layout(gtx,
				layout.Stacked(r.ReplyStyle.Layout),
				layout.Expanded(r.Reply.Polyclick.Layout),
			)
		})
	})
}
