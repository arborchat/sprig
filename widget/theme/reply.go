package theme

import (
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

func (r ReplyStyle) Layout(gtx *layout.Context, reply *forest.Reply, author *forest.Identity) {
	// higher-level state to track the height of the dynamic content. This
	// is set by the Stacked layout function, but used by the Expanded one.
	// It's counterintuitive, but it works because the stacked child is
	// evaluated first by the layout.
	var height float32
	layout.Stack{}.Layout(gtx,
		layout.Expanded(func() {
			var stack op.StackOp
			stack.Push(gtx.Ops)
			paintOp := paint.ColorOp{Color: r.Background}
			paintOp.Add(gtx.Ops)
			bounds := f32.Rectangle{
				Max: f32.Point{
					X: float32(gtx.Constraints.Width.Max),
					Y: float32(height),
				},
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
		}),
		layout.Stacked(func() {
			layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
				layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func() {
						gtx.Constraints.Width.Min = gtx.Constraints.Width.Max
						layout.NW.Layout(gtx, func() {
							name := material.Body2(r.Theme, string(author.Name.Blob))
							name.Font.Weight = text.Bold
							name.Color = r.TextColor
							name.Layout(gtx)
						})
						layout.NE.Layout(gtx, func() {
							date := material.Body2(r.Theme, reply.Created.Time().Local().Format("2006/01/02 15:04"))
							date.Color = r.TextColor
							date.Color.A = 200
							date.TextSize = unit.Dp(12)
							date.Layout(gtx)
						})
					}),
					layout.Rigid(func() {
						content := material.Body1(r.Theme, string(reply.Content.Blob))
						content.Color = r.TextColor
						content.Layout(gtx)
					}),
				)
			})
			height = float32(gtx.Dimensions.Size.Y)
		}),
	)
}
