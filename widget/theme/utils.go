package theme

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// DrawRect creates a rectangle of the provided background color with
// Dimensions specified by size and a corner radius (on all corners)
// specified by radii.
func DrawRect(gtx C, background color.RGBA, size f32.Point, radii float32) D {
	stack := op.Push(gtx.Ops)
	paintOp := paint.ColorOp{Color: background}
	paintOp.Add(gtx.Ops)
	bounds := f32.Rectangle{
		Max: size,
	}
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
	return layout.Dimensions{Size: image.Pt(int(size.X), int(size.Y))}
}
