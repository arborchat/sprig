package theme

import (
	"encoding/hex"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"git.sr.ht/~whereswaldon/forest-go"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

type ReplyStyle struct {
	*material.Theme
	Background color.RGBA
	TextColor  color.RGBA

	// CollapseMetadata should be set to true if this reply can be rendered
	// without the author being displayed.
	CollapseMetadata bool
}

func Reply(th *material.Theme) ReplyStyle {
	defaultBackground := color.RGBA{R: 250, G: 250, B: 250, A: 255}
	defaultTextColor := color.RGBA{A: 255}
	return ReplyStyle{
		Theme:      th,
		Background: defaultBackground,
		TextColor:  defaultTextColor,
	}
}

func (r ReplyStyle) Layout(gtx layout.Context, reply *forest.Reply, author *forest.Identity, community *forest.Community) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			max := f32.Point{
				X: float32(gtx.Constraints.Min.X),
				Y: float32(gtx.Constraints.Min.Y),
			}
			radii := float32(gtx.Px(unit.Dp(5)))
			return DrawRect(gtx, r.Background, max, radii)
		}),
		layout.Stacked(func(gtx C) D {
			return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx C) D {
				return r.layoutContents(gtx, reply, author, community)
			})
		}),
	)
}

func max(is ...int) int {
	max := is[0]
	for i := range is {
		if i > max {
			max = i
		}
	}
	return max
}

func (r ReplyStyle) layoutContents(gtx layout.Context, reply *forest.Reply, author *forest.Identity, community *forest.Community) layout.Dimensions {
	if !r.CollapseMetadata {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						nameMacro := op.Record(gtx.Ops)
						nameDim := AuthorName(r.Theme, author).Layout(gtx)
						nameWidget := nameMacro.Stop()

						communityMacro := op.Record(gtx.Ops)
						communityDim := CommunityName(r.Theme, community).Layout(gtx)
						communityWidget := communityMacro.Stop()

						dateMacro := op.Record(gtx.Ops)
						dateDim := r.layoutDate(gtx, reply)
						dateWidget := dateMacro.Stop()

						gtx.Constraints.Min.Y = max(nameDim.Size.Y, communityDim.Size.Y, dateDim.Size.Y)
						gtx.Constraints.Min.X = gtx.Constraints.Max.X

						shouldDisplayDate := gtx.Constraints.Max.X-nameDim.Size.X > dateDim.Size.X
						shouldDisplayCommunity := shouldDisplayDate && gtx.Constraints.Max.X-(nameDim.Size.X+dateDim.Size.X) > communityDim.Size.X

						flexChildren := []layout.FlexChild{
							layout.Rigid(func(gtx C) D {
								return layout.S.Layout(gtx, func(gtx C) D {
									nameWidget.Add(gtx.Ops)
									return nameDim
								})
							}),
						}
						if shouldDisplayCommunity {
							flexChildren = append(flexChildren,
								layout.Rigid(func(gtx C) D {
									return layout.S.Layout(gtx, func(gtx C) D {
										communityWidget.Add(gtx.Ops)
										return communityDim
									})
								}),
							)
						}
						if shouldDisplayDate {
							flexChildren = append(flexChildren,
								layout.Rigid(func(gtx C) D {
									return layout.S.Layout(gtx, func(gtx C) D {
										dateWidget.Add(gtx.Ops)
										return dateDim
									})
								}),
							)
						}

						return layout.Flex{Spacing: layout.SpaceBetween}.Layout(gtx, flexChildren...)
					})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.layoutContent(gtx, reply)
			}),
		)
	}
	return layout.Flex{Spacing: layout.SpaceBetween}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return r.layoutContent(gtx, reply)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return r.layoutDate(gtx, reply)
		}),
	)
}

func (r ReplyStyle) layoutDate(gtx layout.Context, reply *forest.Reply) layout.Dimensions {
	date := material.Body2(r.Theme, reply.Created.Time().Local().Format("2006/01/02 15:04"))
	date.MaxLines = 1
	date.Color = r.TextColor
	date.Color.A = 200
	date.TextSize = unit.Dp(12)
	return date.Layout(gtx)
}

func (r ReplyStyle) layoutContent(gtx layout.Context, reply *forest.Reply) layout.Dimensions {
	content := material.Body1(r.Theme, string(reply.Content.Blob))
	content.Color = r.TextColor
	return content.Layout(gtx)
}

type AuthorNameStyle struct {
	*forest.Identity
	*material.Theme
}

func AuthorName(theme *material.Theme, identity *forest.Identity) AuthorNameStyle {
	return AuthorNameStyle{
		Identity: identity,
		Theme:    theme,
	}
}

func (a AuthorNameStyle) Layout(gtx layout.Context) layout.Dimensions {
	if a.Identity == nil {
		return layout.Dimensions{}
	}
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			name := material.Body2(a.Theme, string(a.Identity.Name.Blob))
			name.Font.Weight = text.Bold
			name.MaxLines = 1
			return name.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			suffix := a.Identity.ID().Blob
			suffix = suffix[len(suffix)-2:]
			suffixLabel := material.Body2(a.Theme, "#"+hex.EncodeToString(suffix))
			suffixLabel.Color.A = 150
			suffixLabel.MaxLines = 1
			return suffixLabel.Layout(gtx)
		}),
	)
}

type CommunityNameStyle struct {
	*forest.Community
	Prefix string
	*material.Theme
}

func CommunityName(theme *material.Theme, community *forest.Community) CommunityNameStyle {
	return CommunityNameStyle{
		Community: community,
		Theme:     theme,
	}
}

func (a CommunityNameStyle) Layout(gtx layout.Context) layout.Dimensions {
	if a.Community == nil {
		return layout.Dimensions{}
	}
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			prefixLabel := material.Body2(a.Theme, a.Prefix)
			prefixLabel.Color.A = 150
			prefixLabel.MaxLines = 1
			return prefixLabel.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			name := material.Body2(a.Theme, string(a.Community.Name.Blob))
			name.Font.Weight = text.Bold
			name.MaxLines = 1
			return name.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			suffix := a.Community.ID().Blob
			suffix = suffix[len(suffix)-2:]
			suffixLabel := material.Body2(a.Theme, "#"+hex.EncodeToString(suffix))
			suffixLabel.Color.A = 150
			suffixLabel.MaxLines = 1
			return suffixLabel.Layout(gtx)
		}),
	)
}
