package theme

import (
	"encoding/hex"
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"git.sr.ht/~whereswaldon/forest-go"
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
	// higher-level state to track the height of the dynamic content. This
	// is set by the Stacked layout function, but used by the Expanded one.
	// It's counterintuitive, but it works because the stacked child is
	// evaluated first by the layout.
	var height float32
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			var stack op.StackOp
			stack.Push(gtx.Ops)
			paintOp := paint.ColorOp{Color: r.Background}
			paintOp.Add(gtx.Ops)
			max := f32.Point{
				X: float32(gtx.Constraints.Max.X),
				Y: float32(height),
			}
			bounds := f32.Rectangle{
				Max: max,
			}
			radii := float32(gtx.Px(unit.Dp(5)))
			clip.Rect{
				Rect: bounds,
				NW:   radii,
				NE:   radii,
				SE:   radii,
				SW:   radii,
			}.Op(gtx.Ops).Add(gtx.Ops)
			paint.PaintOp{
				Rect: bounds,
			}.Add(gtx.Ops)
			stack.Pop()
			return layout.Dimensions{Size: image.Pt(int(max.X), int(max.Y))}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			dim := layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				if !r.CollapseMetadata {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							var dim layout.Dimensions
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							dim.Size.X = gtx.Constraints.Max.X
							textDim := layout.NW.Layout(gtx, AuthorName(r.Theme, author).Layout)
							if community != nil {
								layout.N.Layout(gtx, CommunityName(r.Theme, community).Layout)
							}
							dim.Size.Y = textDim.Size.Y
							textDim = layout.NE.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return r.layoutDate(gtx, reply)
							})
							if textDim.Size.Y > dim.Size.Y {
								dim.Size.Y = textDim.Size.Y
							}
							return dim
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
			})
			height = float32(dim.Size.Y)
			return dim
		}),
	)
}

func (r ReplyStyle) layoutDate(gtx layout.Context, reply *forest.Reply) layout.Dimensions {
	date := material.Body2(r.Theme, reply.Created.Time().Local().Format("2006/01/02 15:04"))
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
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			name := material.Body2(a.Theme, string(a.Identity.Name.Blob))
			name.Font.Weight = text.Bold
			return name.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			suffix := a.Identity.ID().Blob
			suffix = suffix[len(suffix)-2:]
			suffixLabel := material.Body2(a.Theme, "#"+hex.EncodeToString(suffix))
			suffixLabel.Color.A = 150
			return suffixLabel.Layout(gtx)
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
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			prefixLabel := material.Body2(a.Theme, "in:")
			prefixLabel.Color.A = 150
			return prefixLabel.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			name := material.Body2(a.Theme, string(a.Community.Name.Blob))
			name.Font.Weight = text.Bold
			return name.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			suffix := a.Community.ID().Blob
			suffix = suffix[len(suffix)-2:]
			suffixLabel := material.Body2(a.Theme, "#"+hex.EncodeToString(suffix))
			suffixLabel.Color.A = 150
			return suffixLabel.Layout(gtx)
		}),
	)
}
