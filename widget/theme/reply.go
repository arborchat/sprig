package theme

import (
	"encoding/hex"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	materials "gioui.org/x/component"
	"git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/sprig/ds"
	sprigWidget "git.sr.ht/~whereswaldon/sprig/widget"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// Style applies the appropriate visual tweaks the the reply status for the
// current animation frame.
func Style(gtx C, r *sprigWidget.ReplyAnimationState, reply *ReplyStyle) {
	if r == nil {
		return
	}
	progress := r.Progress(gtx)
	if progress >= 1 {
		r.Begin = r.End
		reply.Highlight = HighlightColor(r.Begin, reply.Theme)
	}
	startColor := HighlightColor(r.Begin, reply.Theme)
	endColor := HighlightColor(r.End, reply.Theme)
	current := materials.Interpolate(startColor, endColor, progress)
	reply.Highlight = current

	startBorder := BorderColor(r.Begin, reply.Theme)
	endBorder := BorderColor(r.End, reply.Theme)
	currentBorder := materials.Interpolate(startBorder, endBorder, progress)
	reply.Border = currentBorder
}

func HighlightColor(r sprigWidget.ReplyStatus, th *Theme) color.NRGBA {
	switch r {
	case sprigWidget.Selected:
		return *th.Selected
	case sprigWidget.Ancestor:
		return *th.Ancestors
	case sprigWidget.Descendant:
		return *th.Descendants
	case sprigWidget.Sibling:
		return *th.Siblings
	default:
		return *th.Unselected
	}
}

func BorderColor(r sprigWidget.ReplyStatus, th *Theme) color.NRGBA {
	switch r {
	case sprigWidget.Selected:
		return *th.Selected
	default:
		return th.Background.Light.Bg
	}
}

type ReplyStyle struct {
	*Theme
	Highlight  color.NRGBA
	Background color.NRGBA
	TextColor  color.NRGBA
	Border     color.NRGBA
	// MaxLines limits the maximum number of lines of content text that should
	// be displayed. Values less than 1 indicate unlimited.
	MaxLines       int
	highlightWidth unit.Value

	// CollapseMetadata should be set to true if this reply can be rendered
	// without the author being displayed.
	CollapseMetadata bool

	*sprigWidget.ReplyAnimationState

	ds.ReplyData
	// Whether or not to render the user as active
	ShowActive bool
}

func Reply(th *Theme, status *sprigWidget.ReplyAnimationState, nodes ds.ReplyData, showActive bool) ReplyStyle {
	rs := ReplyStyle{
		Theme:               th,
		Background:          th.Background.Light.Bg,
		TextColor:           th.Background.Light.Fg,
		highlightWidth:      unit.Dp(10),
		ReplyData:           nodes,
		ReplyAnimationState: status,
		ShowActive:          showActive,
	}
	return rs
}

func (r ReplyStyle) Layout(gtx layout.Context) layout.Dimensions {
	radiiDp := unit.Dp(5)
	radii := float32(gtx.Px(radiiDp))
	Style(gtx, r.ReplyAnimationState, &r)
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			innerSize := gtx.Constraints.Min
			return widget.Border{
				Color:        r.Border,
				Width:        unit.Dp(2),
				CornerRadius: radiiDp,
			}.Layout(gtx, func(gtx C) D {
				return Rect{Color: r.Background, Size: layout.FPt(innerSize), Radii: radii}.Layout(gtx)
			})
		}),
		layout.Stacked(func(gtx C) D {
			return layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx C) D {
					max := layout.FPt(gtx.Constraints.Min)
					max.X = float32(gtx.Px(r.highlightWidth))
					return Rect{Color: r.Highlight, Size: max, Radii: radii}.Layout(gtx)
				}),
				layout.Stacked(func(gtx C) D {
					inset := layout.Inset{}
					inset.Left = unit.Add(gtx.Metric, r.highlightWidth, inset.Left)
					isConversationRoot := r.ReplyData.Reply.Depth == 1
					return inset.Layout(gtx, func(gtx C) D {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								return layout.UniformInset(unit.Dp(4)).Layout(gtx, r.layoutContents)
							}),
							layout.Rigid(func(gtx C) D {
								if isConversationRoot {
									badgeColors := r.Theme.Primary.Dark
									gtx.Constraints.Min.X = gtx.Constraints.Max.X
									return layout.SE.Layout(gtx, func(gtx C) D {
										return layout.Stack{}.Layout(gtx,
											layout.Expanded(func(gtx C) D {
												return Rect{Color: badgeColors.Bg, Size: layout.FPt(gtx.Constraints.Min), Radii: radii}.Layout(gtx)
											}),
											layout.Stacked(func(gtx C) D {
												th := *r.Theme.Theme
												th.Palette = ApplyAsNormal(th.Palette, badgeColors)
												return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx C) D {
													return material.Body2(&th, "Root").Layout(gtx)
												})
											}),
										)
									})
								}
								return D{}
							}),
						)
					})
				}),
			)
		}),
	)
}

func (r ReplyStyle) HideMetadata(b bool) ReplyStyle {
	r.CollapseMetadata = b
	return r
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

func (r ReplyStyle) layoutMetadata(gtx layout.Context) layout.Dimensions {
	inset := layout.Inset{Right: unit.Dp(4)}
	nameMacro := op.Record(gtx.Ops)
	nameDim := inset.Layout(gtx, AuthorName(r.Theme, r.ReplyData.Author, r.ShowActive).Layout)
	nameWidget := nameMacro.Stop()

	communityMacro := op.Record(gtx.Ops)
	communityDim := inset.Layout(gtx, CommunityName(r.Theme.Theme, r.ReplyData.Community).Layout)
	communityWidget := communityMacro.Stop()

	dateMacro := op.Record(gtx.Ops)
	dateDim := r.layoutDate(gtx)
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
}

func (r ReplyStyle) layoutContents(gtx layout.Context) layout.Dimensions {
	if !r.CollapseMetadata {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, r.layoutMetadata)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.layoutContent(gtx)
			}),
		)
	}
	return layout.Flex{Spacing: layout.SpaceBetween}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return r.layoutContent(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return r.layoutDate(gtx)
		}),
	)
}

func (r ReplyStyle) layoutDate(gtx layout.Context) layout.Dimensions {
	date := material.Body2(r.Theme.Theme, r.ReplyData.Reply.Created.Time().Local().Format("2006/01/02 15:04"))
	date.MaxLines = 1
	date.Color = r.TextColor
	date.Color.A = 200
	date.TextSize = unit.Dp(12)
	return date.Layout(gtx)
}

func (r ReplyStyle) layoutContent(gtx layout.Context) layout.Dimensions {
	reply := r.ReplyData.Reply
	content := material.Body1(r.Theme.Theme, string(reply.Content.Blob))
	content.MaxLines = r.MaxLines
	content.Color = r.TextColor
	return content.Layout(gtx)
}

type AuthorNameStyle struct {
	*forest.Identity
	*Theme
	Active bool
}

func AuthorName(theme *Theme, identity *forest.Identity, active bool) AuthorNameStyle {
	return AuthorNameStyle{
		Identity: identity,
		Theme:    theme,
		Active:   active,
	}
}

func (a AuthorNameStyle) Layout(gtx layout.Context) layout.Dimensions {
	if a.Identity == nil {
		return layout.Dimensions{}
	}

	return layout.Flex{}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			name := material.Body2(a.Theme.Theme, string(a.Identity.Name.Blob))
			name.Font.Weight = text.Bold
			name.MaxLines = 1
			return name.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			suffix := a.Identity.ID().Blob
			suffix = suffix[len(suffix)-2:]
			suffixLabel := material.Body2(a.Theme.Theme, "#"+hex.EncodeToString(suffix))
			suffixLabel.Color.A = 150
			suffixLabel.MaxLines = 1
			return suffixLabel.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			if !a.Active {
				return D{}
			}
			name := material.Body2(a.Theme.Theme, "●")
			name.Color = a.Theme.Primary.Light.Bg
			name.Font.Weight = text.Bold
			return name.Layout(gtx)
		}),
	)
}

type CommunityNameStyle struct {
	*forest.Community
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
