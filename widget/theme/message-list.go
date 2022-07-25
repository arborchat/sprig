package theme

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"gioui.org/x/markdown"
	"gioui.org/x/richtext"
	"git.sr.ht/~whereswaldon/sprig/ds"
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigWidget "git.sr.ht/~whereswaldon/sprig/widget"
)

type MessageListStyle struct {
	*Theme
	State             *sprigWidget.MessageList
	Replies           []ds.ReplyData
	Prefixes          []layout.Widget
	CreateReplyButton *widget.Clickable
	material.ListStyle
}

func MessageList(th *Theme, state *sprigWidget.MessageList, replyBtn *widget.Clickable, replies []ds.ReplyData) MessageListStyle {
	mls := MessageListStyle{
		Theme:             th,
		State:             state,
		Replies:           replies,
		CreateReplyButton: replyBtn,
		ListStyle:         material.List(th.Theme, &state.List),
	}
	mls.ListStyle.Indicator.MajorMinLen = unit.Dp(12)
	return mls
}

const insetUnit = 12

var (
	defaultInset    = unit.Dp(insetUnit)
	ancestorInset   = unit.Dp(2 * insetUnit)
	selectedInset   = unit.Dp(2 * insetUnit)
	descendantInset = unit.Dp(3 * insetUnit)
)

// MaxReplyInset returns the maximum distance that a reply will be inset
// based on its position within the message tree.
func MaxReplyInset() unit.Dp {
	return descendantInset
}

func insetForStatus(status sprigWidget.ReplyStatus) unit.Dp {
	switch {
	case status&sprigWidget.Selected > 0:
		return selectedInset
	case status&sprigWidget.Ancestor > 0:
		return ancestorInset
	case status&sprigWidget.Descendant > 0:
		return descendantInset
	case status&sprigWidget.Sibling > 0:
		return defaultInset
	default:
		return defaultInset
	}
}

func interpolateInset(anim *sprigWidget.ReplyAnimationState, progress float32) unit.Dp {
	if progress == 0 {
		return insetForStatus(anim.Begin)
	}
	begin := insetForStatus(anim.Begin)
	end := insetForStatus(anim.End)
	return unit.Dp((end-begin)*unit.Dp(progress) + begin)
}

const (
	buttonWidthDp     = 20
	scrollSlotWidthDp = 12
)

func (m MessageListStyle) Layout(gtx C) D {
	m.State.Layout(gtx)
	th := m.Theme
	dims := m.ListStyle.Layout(gtx, len(m.Replies)+len(m.Prefixes), func(gtx layout.Context, index int) layout.Dimensions {
		if index < len(m.Prefixes) {
			return m.Prefixes[index](gtx)
		}
		// adjust to be a valid reply index
		index -= len(m.Prefixes)
		reply := m.Replies[index]

		// return as soon as possible if this node shouldn't be displayed
		if m.State.ShouldHide != nil && m.State.ShouldHide(reply) {
			return layout.Dimensions{}
		}
		var status sprigWidget.ReplyStatus
		if m.State.StatusOf != nil {
			status = m.State.StatusOf(reply)
		}
		var (
			anim             = m.State.Animation.Update(gtx, reply.ID, status)
			isActive         bool
			collapseMetadata = func() bool {
				// This conflicts with animation feature, so we're removing the feature for now.
				// if index > 0 {
				// 	if replies[index-1].Reply.Author.Equals(&reply.Reply.Author) && replies[index-1].ID().Equals(reply.ParentID()) {
				// 		return true
				// 	}
				// }
				return false
			}()
		)
		if m.State.UserIsActive != nil {
			isActive = m.State.UserIsActive(reply.AuthorID)
		}
		// Only acquire a state after ensuring the node should be rendered. This allows
		// us to count used states in order to determine how many nodes were rendered.
		state := m.State.ReplyStates.Next()
		return layout.Center.Layout(gtx, func(gtx C) D {
			var (
				cs         = &gtx.Constraints
				contentMax = gtx.Dp(unit.Dp(800))
			)
			if cs.Max.X > contentMax {
				cs.Max.X = contentMax
			}
			return layout.Stack{}.Layout(gtx,
				layout.Stacked(func(gtx C) D {
					var (
						extraWidth   = gtx.Dp(unit.Dp(5*insetUnit + DefaultIconButtonWidthDp + scrollSlotWidthDp))
						messageWidth = gtx.Constraints.Max.X - extraWidth
					)
					dims := layout.Stack{}.Layout(gtx,
						layout.Stacked(func(gtx C) D {
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							return layout.Inset{
								Top: func() unit.Dp {
									if collapseMetadata {
										return unit.Dp(0)
									}
									return unit.Dp(3)
								}(),
								Bottom: unit.Dp(3),
								Left:   interpolateInset(anim, m.State.Animation.Progress(gtx)),
							}.Layout(gtx, func(gtx C) D {
								gtx.Constraints.Max.X = messageWidth
								state, hint := m.State.GetTextState(reply.ID)
								content, _ := markdown.NewRenderer().Render([]byte(reply.Content))
								if hint != "" {
									macro := op.Record(gtx.Ops)
									component.Surface(th.Theme).Layout(gtx,
										func(gtx C) D {
											return layout.UniformInset(unit.Dp(4)).Layout(gtx, material.Body2(th.Theme, hint).Layout)
										})
									op.Defer(gtx.Ops, macro.Stop())
								}
								rs := Reply(th, anim, reply, richtext.Text(state, th.Shaper, content...), isActive).
									HideMetadata(collapseMetadata)
								if anim.Begin&sprigWidget.Anchor > 0 {
									rs = rs.Anchoring(th.Theme, m.State.HiddenChildren(reply))
								}

								return rs.Layout(gtx)
							})
						}),
						layout.Expanded(func(gtx C) D {
							return state.
								WithHash(reply.ID).
								WithContent(reply.Content).
								Polyclick.
								Layout(gtx)
						}),
					)
					return D{
						Size: image.Point{
							X: gtx.Constraints.Max.X,
							Y: dims.Size.Y,
						},
						Baseline: dims.Baseline,
					}
				}),
				layout.Expanded(func(gtx C) D {
					return layout.E.Layout(gtx, func(gtx C) D {
						if status != sprigWidget.Selected {
							return D{}
						}
						return layout.Inset{
							Right: unit.Dp(scrollSlotWidthDp),
						}.Layout(gtx, func(gtx C) D {
							return material.IconButtonStyle{
								Background: th.Secondary.Light.Bg,
								Color:      th.Secondary.Light.Fg,
								Button:     m.CreateReplyButton,
								Icon:       icons.ReplyIcon,
								Size:       unit.Dp(DefaultIconButtonWidthDp),
								Inset:      layout.UniformInset(unit.Dp(9)),
							}.Layout(gtx)
						})
					})
				}),
			)
		})
	})
	return dims
}
