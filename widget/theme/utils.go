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

// Scrollable holds the stateful part of a scrolling. The Progress property
// can be used to check how far the bar has been scrolled, and the Scrolled()
// method can be used to determine if the scroll position changed within the
// last frame.
type Scrollable struct {
	// Track clicks.
	widget.Clickable
	// Has the bar scrolled since the previous frame?
	scrolled bool
	// Cached length of scroll region after layout has been computed. This can be
	// off if the screen is being resized, but we have no better way to acquire
	// this data.
	length int
	// Progress is how far along we are as a fraction between 0 and 1.
	Progress float32
}

// Update the internal state of the scrollbar.
func (sb *Scrollable) Update() {
	sb.scrolled = false
	if sb.Clicked() {
		// Resolve the click position to a fraction of the bar height
		// Assuming vertical axis.
		// Looping over each press is required for consistent snapping.
		for _, press := range sb.History() {
			sb.Progress = float32(press.Position.Y) / float32(sb.length)
			sb.scrolled = true
		}
	}
}

// Scrolled returns true if the scroll position changed within the last frame.
func (sb Scrollable) Scrolled() bool {
	return sb.scrolled
}

// ScrollBar renders a scroll bar anchored to a side.
type ScrollBar struct {
	*Scrollable
	// Color of the scroll bar.
	Color color.RGBA
	// Anchor is the content-relative position to anchor to.
	// Defaults to `End`, which is usually what you want.
	Anchor Anchor
	// Size of the bar.
	Size f32.Point
	// Progress overrides the internal progress of the scrollable.
	// This lets external systems hint to where the indicator should render.
	Progress float32
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
	sb.Update()
	if sb.Scrolled() {
		op.InvalidateOp{}.Add(gtx.Ops)
	}
	// assuming vertical
	sb.length = gtx.Constraints.Max.Y
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
		return ClickBox(gtx, &sb.Clickable, func(gtx C) D {
			barAreaDims := layout.Inset{
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
			barAreaDims.Size.Y = gtx.Constraints.Max.Y
			return barAreaDims
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
