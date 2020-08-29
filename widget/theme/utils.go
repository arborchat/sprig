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
	// Color of the scroll indicator.
	Color color.RGBA
	// Progress overrides the internal progress of the scrollable.
	// This lets external systems hint to where the indicator should render.
	Progress float32
	// Anchor is the content-relative position to anchor to.
	// Defaults to `End`, which is usually what you want.
	Anchor Anchor
	// Axis along which the scrollbar is oriented.
	Axis Axis
	// Axis independent size.
	Thickness unit.Value
	Length    unit.Value
}

// Anchor specifies where to anchor to.
// On the horizontal axis this becomes left-right.
// On the vertical axis this becomes top-bottom.
// Default to `End`.
type Anchor bool

const (
	End          = false
	Start Anchor = true
)

// Axis specifies the scroll bar orientation.
// Default to `Vertical`.
type Axis bool

const (
	Vertical   = false
	Horizontal = true
)

// Layout renders the ScrollBar into the provided context.
func (sb ScrollBar) Layout(gtx C) D {
	sb.Update()
	if sb.Scrolled() {
		op.InvalidateOp{}.Add(gtx.Ops)
	}
	return sb.Anchor.Layout(gtx, sb.Axis, func(gtx C) D {
		if sb.Length == (unit.Value{}) {
			sb.Length = unit.Dp(16)
		}
		if sb.Thickness == (unit.Value{}) {
			sb.Thickness = unit.Dp(8)
		}
		var (
			total float32
			size  f32.Point
			top   = unit.Dp(2)
			left  = unit.Dp(2)
		)
		switch sb.Axis {
		case Horizontal:
			sb.length = gtx.Constraints.Max.X
			size = f32.Point{
				X: float32(gtx.Px(sb.Length)),
				Y: float32(gtx.Px(sb.Thickness)),
			}
			total = float32(gtx.Constraints.Max.X) / gtx.Metric.PxPerDp
			left = unit.Dp(total * sb.Progress)
			if left.V+sb.Length.V > total {
				left = unit.Dp(total - sb.Length.V)
			}
		case Vertical:
			sb.length = gtx.Constraints.Max.Y
			size = f32.Point{
				X: float32(gtx.Px(sb.Thickness)),
				Y: float32(gtx.Px(sb.Length)),
			}
			total = float32(gtx.Constraints.Max.Y) / gtx.Metric.PxPerDp
			top = unit.Dp(total * sb.Progress)
			if top.V+sb.Length.V > total {
				top = unit.Dp(total - sb.Length.V)
			}
		}
		return ClickBox(gtx, &sb.Clickable, func(gtx C) D {
			return Dimensions(layout.Inset{
				Top:    top,
				Right:  unit.Dp(2),
				Left:   left,
				Bottom: unit.Dp(2),
			}.Layout(gtx, func(gtx C) D {
				return Rect{
					Color: sb.Color,
					Size:  size,
					Radii: float32(gtx.Px(unit.Dp(4))),
				}.Layout(gtx)
			})).Where(func(d *Dimensions) {
				switch sb.Axis {
				case Vertical:
					d.Size.Y = gtx.Constraints.Max.Y
				case Horizontal:
					d.Size.X = gtx.Constraints.Max.X
				}
			}).Into()
		})
	})
}

func (an Anchor) Layout(gtx C, axis Axis, widget layout.Widget) D {
	if axis == Vertical && an == Start {
		return layout.NW.Layout(gtx, widget)
	}
	if axis == Vertical && an == End {
		return layout.NE.Layout(gtx, widget)
	}
	if axis == Horizontal && an == Start {
		return layout.NW.Layout(gtx, widget)
	}
	if axis == Horizontal && an == End {
		return layout.SW.Layout(gtx, widget)
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

type Dimensions layout.Dimensions

// Where transforms Dimensions by applying a series of operations.
// Allows for a declarative calling style.
func (d Dimensions) Where(ops ...func(*Dimensions)) Dimensions {
	for _, op := range ops {
		op(&d)
	}
	return d
}

func (d Dimensions) Into() layout.Dimensions {
	return layout.Dimensions(d)
}
