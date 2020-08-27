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
	// Color of the scroll bar.
	Color color.RGBA
	// Progress is how far down we are as a fraction between 0 and 1.
	Progress float32
	// Anchor is the direction to anchor the bar.
	Anchor layout.Direction
	// Size of the bar.
	Size image.Point
}

// Layout renders the ScrollBar into the provided context.
func (sb ScrollBar) Layout(gtx C) D {
	return sb.Anchor.Layout(gtx, func(gtx C) D {
		if sb.Size.X == 0 || sb.Size.Y == 0 {
			// Note(jfm): Default size themeable?
			sb.Size = image.Point{
				X: 8,
				Y: 16,
			}
		}
		var (
			width       = unit.Dp(float32(sb.Size.X))
			height      = unit.Dp(float32(sb.Size.Y))
			totalHeight = float32(gtx.Constraints.Max.Y) / gtx.Metric.PxPerDp
			top         = unit.Dp(totalHeight * sb.Progress)
		)
		if top.V+height.V > totalHeight {
			top = unit.Dp(totalHeight - height.V)
		}
		return layout.Inset{
			Top:    top,
			Right:  unit.Dp(2),
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
	})
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
