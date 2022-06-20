package theme

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"git.sr.ht/~whereswaldon/sprig/icons"
	"git.sr.ht/~whereswaldon/sprig/widget"
)

type TextFormStyle struct {
	State *widget.TextForm
	// internal widget separation distance
	layout.Inset
	PasteButton      material.IconButtonStyle
	SubmitButton     material.ButtonStyle
	EditorHint       string
	EditorBackground color.NRGBA
	*Theme
}

func TextForm(th *Theme, state *widget.TextForm, submitText, formHint string) TextFormStyle {
	t := TextFormStyle{
		State:            state,
		Inset:            layout.UniformInset(unit.Dp(8)),
		PasteButton:      material.IconButton(th.Theme, &state.PasteButton, icons.PasteIcon, "Paste"),
		SubmitButton:     material.Button(th.Theme, &state.SubmitButton, submitText),
		EditorHint:       formHint,
		EditorBackground: th.Background.Light.Bg,
		Theme:            th,
	}
	t.PasteButton.Inset = layout.UniformInset(unit.Dp(4))
	return t
}

func (t TextFormStyle) Layout(gtx layout.Context) layout.Dimensions {
	t.State.Layout(gtx)
	return layout.Flex{
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Inset{
				Right: t.Inset.Right,
			}.Layout(gtx, t.PasteButton.Layout)
		}),
		layout.Flexed(1, func(gtx C) D {
			return layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx C) D {
					return Rect{
						Color: t.EditorBackground,
						Size:  layout.FPt(gtx.Constraints.Min),
						Radii: float32(gtx.Dp(unit.Dp(4))),
					}.Layout(gtx)
				}),
				layout.Stacked(func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return t.Inset.Layout(gtx, func(gtx C) D {
						return t.State.TextField.Layout(gtx, t.Theme.Theme, t.EditorHint)
					})
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Inset{
				Left: t.Inset.Left,
			}.Layout(gtx, t.SubmitButton.Layout)
		}),
	)
}
