package theme

import (
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"git.sr.ht/~whereswaldon/forest-go"
)

type ReplyStyle struct {
	*material.Theme
	Background color.RGBA
}

func Reply(th *material.Theme) ReplyStyle {
	defaultBackground := color.RGBA{R: 230, G: 230, B: 230, A: 255}
	return ReplyStyle{
		Theme:      th,
		Background: defaultBackground,
	}
}

func (r ReplyStyle) Layout(gtx *layout.Context, reply *forest.Reply, author *forest.Identity) {
	layout.Stack{}.Layout(gtx,
		layout.Expanded(func() {
			paintOp := paint.ColorOp{Color: r.Background}
			paintOp.Add(gtx.Ops)
			paint.PaintOp{Rect: f32.Rectangle{
				Max: f32.Point{
					X: float32(gtx.Constraints.Width.Max),
					Y: float32(gtx.Constraints.Height.Max),
				},
			}}.Add(gtx.Ops)
		}),
		layout.Stacked(func() {
			layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
				layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func() {
						gtx.Constraints.Width.Min = gtx.Constraints.Width.Max
						layout.NW.Layout(gtx, func() {
							name := material.Body2(r.Theme, string(author.Name.Blob))
							name.Font.Weight = text.Bold
							name.Layout(gtx)
						})
						layout.NE.Layout(gtx, func() {
							date := material.Body2(r.Theme, reply.Created.Time().Local().Format("2006/01/02 15:04"))
							date.Color.A = 200
							date.TextSize = unit.Dp(12)
							date.Layout(gtx)
						})
					}),
					layout.Rigid(func() {
						material.Body1(r.Theme, string(reply.Content.Blob)).Layout(gtx)
					}),
				)
			})
		}),
	)
}
