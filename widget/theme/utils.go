package theme

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
)

// Rect creates a rectangle of the provided background color with
// Dimensions specified by size and a corner radius (on all corners)
// specified by radii.
type Rect struct {
	Color color.RGBA
	Size  f32.Point
	Radii float32
}

// Layout renders the Rect into the provided context
func (r Rect) Layout(gtx C) D {
	return DrawRect(gtx, r.Color, r.Size, r.Radii)
}

// DrawRect creates a rectangle of the provided background color with
// Dimensions specified by size and a corner radius (on all corners)
// specified by radii.
func DrawRect(gtx C, background color.RGBA, size f32.Point, radii float32) D {
	stack := op.Push(gtx.Ops)
	paint.ColorOp{Color: background}.Add(gtx.Ops)
	bounds := f32.Rectangle{Max: size}
	if radii != 0 {
		clip.RRect{
			Rect: bounds,
			NW:   radii,
			NE:   radii,
			SE:   radii,
			SW:   radii,
		}.Add(gtx.Ops)
	}
	paint.PaintOp{Rect: bounds}.Add(gtx.Ops)
	stack.Pop()
	return layout.Dimensions{Size: image.Pt(int(size.X), int(size.Y))}
}

// ScrollBar renders a scroll bar anchored to a side.
type ScrollBar struct {
	// Track clicks.
	*widget.Clickable
	// Length of scrollbar after layouting has been computed.
	Length *int
	// Color of the scroll bar.
	Color color.RGBA
	// Progress is how far down we are as a fraction between 0 and 1.
	Progress float32
	// Anchor is the content-relative position to anchor to.
	// Defaults to `End`, which is usually what you want.
	Anchor Anchor
	// Size of the bar.
	Size f32.Point
}

// Anchor specifies where to anchor to.
// On the horizontal axis this becomes left-right.
// On the vertical axis this becomes top-bottom.
type Anchor bool

const (
	Start Anchor = true
	End          = false
)

// Layout renders the ScrollBar into the provided context.
func (sb ScrollBar) Layout(gtx C) D {
	return sb.Anchor.Layout(gtx, layout.Vertical, func(gtx C) D {
		if sb.Size.X == 0 {
			sb.Size.X = 8
		}
		if sb.Size.Y == 0 {
			sb.Size.Y = 16
		}
		var (
			width       = unit.Dp(sb.Size.X)
			height      = unit.Dp(sb.Size.Y)
			totalHeight = float32(gtx.Constraints.Max.Y) / gtx.Metric.PxPerDp
			top         = unit.Dp(totalHeight * sb.Progress)
		)
		if top.V+height.V > totalHeight {
			top = unit.Dp(totalHeight - height.V)
		}
		return ClickBox(gtx, sb.Clickable, func(gtx C) D {
			if sb.Length != nil {
				*sb.Length = gtx.Constraints.Max.Y
			}
			return layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx C) D {
					return Rect{Size: f32.Point{
						X: float32(gtx.Constraints.Min.X),
						Y: float32(gtx.Constraints.Max.Y),
					}}.Layout(gtx)
				}),
				layout.Stacked(func(gtx C) D {
					return layout.Inset{
						Top:    top,
						Right:  unit.Dp(2),
						Left:   unit.Dp(2),
						Bottom: unit.Dp(2),
					}.Layout(gtx, func(gtx C) D {
						return Rect{
							Color: sb.Color,
							Size: f32.Point{
								X: float32(gtx.Px(width)),
								Y: float32(gtx.Px(height)),
							},
							Radii: float32(gtx.Px(unit.Dp(4))),
						}.Layout(gtx)
					})
				}),
			)
		})
	})
}

func (an Anchor) Layout(gtx C, axis layout.Axis, widget layout.Widget) D {
	switch an {
	case Start:
		if axis == layout.Vertical {
			return layout.W.Layout(gtx, widget)
		}
		if axis == layout.Horizontal {
			return layout.N.Layout(gtx, widget)
		}
	case End:
		if axis == layout.Vertical {
			return layout.E.Layout(gtx, widget)
		}
		if axis == layout.Horizontal {
			return layout.S.Layout(gtx, widget)
		}
	}
	return layout.Dimensions{}
}

// WithAlpha returns the color with a modified alpha.
func WithAlpha(c color.RGBA, alpha uint8) color.RGBA {
	return color.RGBA{
		R: c.R,
		G: c.G,
		B: c.B,
		A: alpha,
	}
}

// ClickBox lays out a rectangular clickable widget without further
// decoration. No Inking.
func ClickBox(gtx layout.Context, button *widget.Clickable, w layout.Widget) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(button.Layout),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			clip.RRect{
				Rect: f32.Rectangle{Max: f32.Point{
					X: float32(gtx.Constraints.Min.X),
					Y: float32(gtx.Constraints.Min.Y),
				}},
			}.Add(gtx.Ops)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Stacked(w),
	)
}
