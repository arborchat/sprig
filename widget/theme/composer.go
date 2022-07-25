package theme

import (
	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"gioui.org/x/markdown"
	"gioui.org/x/richtext"
	"git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigWidget "git.sr.ht/~whereswaldon/sprig/widget"
)

type ComposerStyle struct {
	*sprigWidget.Composer
	*Theme
	Communities []*forest.Community
}

func Composer(th *Theme, state *sprigWidget.Composer, communities []*forest.Community) ComposerStyle {
	return ComposerStyle{
		Composer:    state,
		Theme:       th,
		Communities: communities,
	}
}

func (c ComposerStyle) Layout(gtx layout.Context) layout.Dimensions {
	th := c.Theme
	c.Composer.Layout(gtx)
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			Rect{
				Color: th.Primary.Light.Bg,
				Size: f32.Point{
					X: float32(gtx.Constraints.Max.X),
					Y: float32(gtx.Constraints.Max.Y),
				},
			}.Layout(gtx)
			return layout.Dimensions{}
		}),
		layout.Stacked(func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								gtx.Constraints.Max.X = gtx.Dp(unit.Dp(30))
								gtx.Constraints.Min.X = gtx.Constraints.Max.X
								if c.ComposingConversation() {
									return material.Body1(th.Theme, "In:").Layout(gtx)
								}
								return material.Body1(th.Theme, "Re:").Layout(gtx)
							})
						}),
						layout.Flexed(1, func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								if c.ComposingConversation() {
									var dims layout.Dimensions
									dims = c.CommunityList.Layout(gtx, len(c.Communities), func(gtx layout.Context, index int) layout.Dimensions {
										community := c.Communities[index]
										if c.Community.Value == "" && index == 0 {
											c.Community.Value = community.ID().String()
										}
										radio := material.RadioButton(th.Theme, &c.Community, community.ID().String(), string(community.Name.Blob))
										radio.IconColor = th.Secondary.Default.Bg
										return radio.Layout(gtx)
									})
									return dims
								}
								content, _ := markdown.NewRenderer().Render([]byte(c.ReplyingTo.Content))
								reply := Reply(th, nil, c.ReplyingTo, richtext.Text(&c.Composer.TextState, th.Shaper, content...), false)
								reply.MaxLines = 5
								return reply.Layout(gtx)
							})
						}),
						layout.Rigid(func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								return IconButton{
									Button: &c.CancelButton,
									Icon:   icons.CancelReplyIcon,
								}.Layout(gtx, th)
							})
						}),
					)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								return IconButton{
									Button: &c.PasteButton,
									Icon:   icons.PasteIcon,
								}.Layout(gtx, th)
							})
						}),
						layout.Flexed(1, func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								return layout.Stack{}.Layout(gtx,
									layout.Expanded(func(gtx C) D {
										return Rect{
											Color: th.Background.Light.Bg,
											Size: f32.Point{
												X: float32(gtx.Constraints.Max.X),
												Y: float32(gtx.Constraints.Min.Y),
											},
											Radii: float32(gtx.Dp(unit.Dp(5))),
										}.Layout(gtx)
									}),
									layout.Stacked(func(gtx C) D {
										return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
											c.Editor.Submit = true
											return material.Editor(th.Theme, &c.Editor, c.PromptText()).Layout(gtx)
										})
									}),
								)
							})
						}),
						layout.Rigid(func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								return IconButton{
									Button: &c.SendButton,
									Icon:   icons.SendReplyIcon,
								}.Layout(gtx, th)
							})
						}),
					)
				}),
			)
		}),
	)
}
