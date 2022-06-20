package theme

import (
	"encoding/hex"
	"fmt"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	materials "gioui.org/x/component"
	"gioui.org/x/richtext"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/sprig/ds"
	sprigWidget "git.sr.ht/~whereswaldon/sprig/widget"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

func HighlightColor(r sprigWidget.ReplyStatus, th *Theme) color.NRGBA {
	var c color.NRGBA
	switch {
	case r&sprigWidget.Selected > 0:
		c = *th.Selected
	case r&sprigWidget.Ancestor > 0:
		c = *th.Ancestors
	case r&sprigWidget.Descendant > 0:
		c = *th.Descendants
	case r&sprigWidget.Sibling > 0:
		c = *th.Siblings
	default:
		c = *th.Unselected
	}
	return c
}

func ReplyTextColor(r sprigWidget.ReplyStatus, th *Theme) color.NRGBA {
	switch {
	case r&sprigWidget.Anchor > 0:
		c := th.Theme.Fg
		c.A = 150
		return c
	case r&sprigWidget.Hidden > 0:
		c := th.Theme.Fg
		c.A = 0
		return c
	default:
		return th.Theme.Fg
	}
}

func BorderColor(r sprigWidget.ReplyStatus, th *Theme) color.NRGBA {
	var c color.NRGBA
	switch {
	case r&sprigWidget.Selected > 0:
		c = *th.Selected
	default:
		c = th.Background.Light.Bg
	}
	if r&sprigWidget.Anchor > 0 {
		c.A = 150
	}
	return c
}

func BackgroundColor(r sprigWidget.ReplyStatus, th *Theme) color.NRGBA {
	switch {
	case r&sprigWidget.Anchor > 0:
		c := th.Background.Light.Bg
		c.A = 150
		return c
	default:
		return th.Background.Light.Bg
	}
}

// ReplyStyleConfig configures aspects of the presentation of a message.
type ReplyStyleConfig struct {
	Highlight      color.NRGBA
	Background     color.NRGBA
	TextColor      color.NRGBA
	Border         color.NRGBA
	highlightWidth unit.Dp
}

// ReplyStyleConfigFor returns a configuration tailored to the given ReplyStatus
// and theme.
func ReplyStyleConfigFor(th *Theme, status sprigWidget.ReplyStatus) ReplyStyleConfig {
	return ReplyStyleConfig{
		Highlight:      HighlightColor(status, th),
		Background:     BackgroundColor(status, th),
		TextColor:      ReplyTextColor(status, th),
		Border:         BorderColor(status, th),
		highlightWidth: unit.Dp(10),
	}
}

// ReplyStyleTransition represents a transition from one ReplyStyleConfig to
// another one and provides a method for interpolating the intermediate
// results between them.
type ReplyStyleTransition struct {
	Previous, Current ReplyStyleConfig
}

// InterpolateWith returns a ReplyStyleConfig blended between the previous
// and current configurations, with 0 returning the previous configuration
// and 1 returning the current.
func (r ReplyStyleTransition) InterpolateWith(progress float32) ReplyStyleConfig {
	return ReplyStyleConfig{
		Highlight:      materials.Interpolate(r.Previous.Highlight, r.Current.Highlight, progress),
		Background:     materials.Interpolate(r.Previous.Background, r.Current.Background, progress),
		TextColor:      materials.Interpolate(r.Previous.TextColor, r.Current.TextColor, progress),
		Border:         materials.Interpolate(r.Previous.Border, r.Current.Border, progress),
		highlightWidth: r.Current.highlightWidth,
	}
}

// ReplyStyle presents a reply as a formatted chat bubble.
type ReplyStyle struct {
	// ReplyStyleTransition captures the two states that the ReplyStyle is
	// transitioning between (though it may not currently be transitioning).
	ReplyStyleTransition

	// finalConfig is the results of interpolating between the two states in
	// the ReplyStyleTransition. Its value can only be determined and used at
	// layout time.
	finalConfig ReplyStyleConfig

	// Background color for the status badge (currently only used if root node)
	BadgeColor color.NRGBA
	// Text config for the status badge
	BadgeText material.LabelStyle

	// MaxLines limits the maximum number of lines of content text that should
	// be displayed. Values less than 1 indicate unlimited.
	MaxLines int

	// CollapseMetadata should be set to true if this reply can be rendered
	// without the author being displayed.
	CollapseMetadata bool

	*sprigWidget.ReplyAnimationState

	ds.ReplyData
	// Whether or not to render the user as active
	ShowActive bool

	// Special text to overlay atop the message contents. Used for displaying
	// messages on anchor nodes with hidden children.
	AnchorText material.LabelStyle

	Content richtext.TextStyle

	AuthorNameStyle
	CommunityNameStyle ForestRefStyle
	DateStyle          material.LabelStyle

	// Padding configures the padding surrounding the entire interior content of the
	// rendered message.
	Padding layout.Inset

	// MetadataPadding configures the padding surrounding the metadata line of a node.
	MetadataPadding layout.Inset
}

// Reply configures a ReplyStyle for the provided state.
func Reply(th *Theme, status *sprigWidget.ReplyAnimationState, nodes ds.ReplyData, text richtext.TextStyle, showActive bool) ReplyStyle {
	rs := ReplyStyle{
		ReplyData:           nodes,
		ReplyAnimationState: status,
		ShowActive:          showActive,
		Content:             text,
		BadgeColor:          th.Primary.Dark.Bg,
		AuthorNameStyle:     AuthorName(th, nodes.AuthorName, nodes.AuthorID, showActive),
		CommunityNameStyle:  CommunityName(th.Theme, nodes.CommunityName, nodes.CommunityID),
		Padding:             layout.UniformInset(unit.Dp(8)),
		MetadataPadding:     layout.Inset{Bottom: unit.Dp(4)},
	}
	if status != nil {
		rs.ReplyStyleTransition = ReplyStyleTransition{
			Previous: ReplyStyleConfigFor(th, status.Begin),
			Current:  ReplyStyleConfigFor(th, status.End),
		}
	} else {
		status := sprigWidget.None
		rs.ReplyStyleTransition = ReplyStyleTransition{
			Previous: ReplyStyleConfigFor(th, status),
			Current:  ReplyStyleConfigFor(th, status),
		}
	}
	if nodes.Depth == 1 {
		theme := *th.Theme
		theme.Palette = ApplyAsNormal(th.Palette, th.Primary.Dark)
		rs.BadgeText = material.Body2(&theme, "Root")
	}
	rs.DateStyle = material.Body2(th.Theme, nodes.CreatedAt.Local().Format("2006/01/02 15:04"))
	rs.DateStyle.MaxLines = 1
	rs.DateStyle.Color.A = 200
	rs.DateStyle.TextSize = unit.Sp(12)
	return rs
}

// Anchoring modifies the ReplyStyle to indicate that it is hiding some number
// of other nodes.
func (r ReplyStyle) Anchoring(th *material.Theme, numNodes int) ReplyStyle {
	r.AnchorText = material.Body1(th, fmt.Sprintf("hidden replies: %d", numNodes))
	return r
}

// Layout renders the ReplyStyle.
func (r ReplyStyle) Layout(gtx layout.Context) layout.Dimensions {
	var progress float32
	if r.ReplyAnimationState != nil {
		progress = r.ReplyAnimationState.Progress(gtx)
	} else {
		progress = 1
	}
	r.finalConfig = r.ReplyStyleTransition.InterpolateWith(progress)
	radiiDp := unit.Dp(5)
	radii := float32(gtx.Dp(radiiDp))
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			innerSize := gtx.Constraints.Min
			return widget.Border{
				Color:        r.finalConfig.Border,
				Width:        unit.Dp(2),
				CornerRadius: radiiDp,
			}.Layout(gtx, func(gtx C) D {
				return Rect{Color: r.finalConfig.Background, Size: layout.FPt(innerSize), Radii: radii}.Layout(gtx)
			})
		}),
		layout.Stacked(func(gtx C) D {
			return layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx C) D {
					max := layout.FPt(gtx.Constraints.Min)
					max.X = float32(gtx.Dp(r.finalConfig.highlightWidth))
					return Rect{Color: r.finalConfig.Highlight, Size: max, Radii: radii}.Layout(gtx)
				}),
				layout.Stacked(func(gtx C) D {
					inset := layout.Inset{}
					inset.Left = r.finalConfig.highlightWidth + inset.Left
					isConversationRoot := r.ReplyData.Depth == 1
					return inset.Layout(gtx, func(gtx C) D {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								return r.Padding.Layout(gtx, r.layoutContents)
							}),
							layout.Rigid(func(gtx C) D {
								if isConversationRoot {
									gtx.Constraints.Min.X = gtx.Constraints.Max.X
									return layout.SE.Layout(gtx, func(gtx C) D {
										return layout.Stack{}.Layout(gtx,
											layout.Expanded(func(gtx C) D {
												return Rect{Color: r.BadgeColor, Size: layout.FPt(gtx.Constraints.Min), Radii: radii}.Layout(gtx)
											}),
											layout.Stacked(func(gtx C) D {
												return layout.UniformInset(unit.Dp(4)).Layout(gtx, r.BadgeText.Layout)
											}),
										)
									})
								}
								return D{}
							}),
						)
					})
				}),
				layout.Expanded(func(gtx C) D {
					if r.AnchorText == (material.LabelStyle{}) {
						return D{}
					}
					return layout.Center.Layout(gtx, func(gtx C) D {
						return layout.Stack{}.Layout(gtx,
							layout.Expanded(func(gtx C) D {
								max := layout.FPt(gtx.Constraints.Min)
								color := r.finalConfig.Background
								color.A = 0xff
								return Rect{Color: color, Size: max, Radii: radii}.Layout(gtx)
							}),
							layout.Stacked(func(gtx C) D {
								return layout.UniformInset(unit.Dp(4)).Layout(gtx, r.AnchorText.Layout)
							}),
						)
					})
				}),
			)
		}),
	)
}

// HideMetadata configures the node metadata line to not be displayed in
// the reply.
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
	author := r.AuthorNameStyle
	author.NameStyle.Color = r.finalConfig.TextColor
	author.SuffixStyle.Color = r.finalConfig.TextColor
	author.ActivityIndicatorStyle.Color.A = r.finalConfig.TextColor.A
	nameDim := inset.Layout(gtx, author.Layout)
	nameWidget := nameMacro.Stop()

	communityMacro := op.Record(gtx.Ops)
	comm := r.CommunityNameStyle
	comm.NameStyle.Color = r.finalConfig.TextColor
	comm.SuffixStyle.Color = r.finalConfig.TextColor
	communityDim := inset.Layout(gtx, comm.Layout)
	communityWidget := communityMacro.Stop()

	dateMacro := op.Record(gtx.Ops)
	dateDim := r.DateStyle.Layout(gtx)
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
				return r.MetadataPadding.Layout(gtx, r.layoutMetadata)
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
		layout.Rigid(r.DateStyle.Layout),
	)
}

func (r ReplyStyle) layoutContent(gtx layout.Context) layout.Dimensions {
	for _, c := range r.Content.Styles {
		c.Color.A = r.finalConfig.TextColor.A
	}
	return r.Content.Layout(gtx)
}

// ForestRefStyle configures the presentation of a reference to a forest
// node that has both a name and an ID.
type ForestRefStyle struct {
	NameStyle, SuffixStyle, ActivityIndicatorStyle material.LabelStyle
}

// ForestRef constructs a ForestRefStyle for the node with the provided info.
func ForestRef(theme *material.Theme, name string, id *fields.QualifiedHash) ForestRefStyle {
	suffix := id.Blob
	suffix = suffix[len(suffix)-2:]
	a := ForestRefStyle{
		NameStyle:   material.Body2(theme, name),
		SuffixStyle: material.Body2(theme, "#"+hex.EncodeToString(suffix)),
	}
	a.NameStyle.Font.Weight = text.Bold
	a.NameStyle.MaxLines = 1
	a.SuffixStyle.Color.A = 150
	a.SuffixStyle.MaxLines = 1
	return a
}

// CommunityName constructs a ForestRefStyle for the provided community.
func CommunityName(theme *material.Theme, communityName string, communityID *fields.QualifiedHash) ForestRefStyle {
	return ForestRef(theme, communityName, communityID)
}

// Layout renders the ForestRefStyle.
func (f ForestRefStyle) Layout(gtx C) D {
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return f.NameStyle.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			return f.SuffixStyle.Layout(gtx)
		}),
	)
}

// AuthorNameStyle configures the presentation of an Author name that can be presented with an activity indicator.
type AuthorNameStyle struct {
	Active bool
	ForestRefStyle
	ActivityIndicatorStyle material.LabelStyle
}

// AuthorName constructs an AuthorNameStyle for the user with the provided info.
func AuthorName(theme *Theme, authorName string, authorID *fields.QualifiedHash, active bool) AuthorNameStyle {
	a := AuthorNameStyle{
		Active:                 active,
		ForestRefStyle:         ForestRef(theme.Theme, authorName, authorID),
		ActivityIndicatorStyle: material.Body2(theme.Theme, "●"),
	}
	a.ActivityIndicatorStyle.Color = theme.Primary.Light.Bg
	a.ActivityIndicatorStyle.Font.Weight = text.Bold
	return a
}

// Layout renders the AuthorNameStyle.
func (a AuthorNameStyle) Layout(gtx layout.Context) layout.Dimensions {
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return a.ForestRefStyle.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			if !a.Active {
				return D{}
			}
			return a.ActivityIndicatorStyle.Layout(gtx)
		}),
	)
}
