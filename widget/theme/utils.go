package theme

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// Rect creates a rectangle of the provided background color with
// Dimensions specified by size and a corner radius (on all corners)
// specified by radii.
type Rect struct {
	Color color.NRGBA
	Size  f32.Point
	Radii float32
}

// Layout renders the Rect into the provided context
func (r Rect) Layout(gtx C) D {
	paint.FillShape(gtx.Ops, r.Color, clip.UniformRRect(f32.Rectangle{Max: r.Size}, r.Radii).Op(gtx.Ops))
	return layout.Dimensions{Size: image.Pt(int(r.Size.X), int(r.Size.Y))}
}
