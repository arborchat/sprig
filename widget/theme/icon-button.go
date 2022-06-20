package theme

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// IconButton applies defaults before rendering a `material.IconButtonStyle` to reduce noise.
// The main paramaters for each button are the state and icon.
// Color, size and inset are often the same.
// This wrapper reduces noise by defaulting those things.
type IconButton struct {
	Button *widget.Clickable
	Icon   *widget.Icon
	Size   unit.Dp
	Inset  layout.Inset
}

const DefaultIconButtonWidthDp = 20

func (btn IconButton) Layout(gtx C, th *Theme) D {
	if btn.Size == 0 {
		btn.Size = unit.Dp(DefaultIconButtonWidthDp)
	}
	if btn.Inset == (layout.Inset{}) {
		btn.Inset = layout.UniformInset(unit.Dp(4))
	}

	return material.IconButtonStyle{
		Background: th.Palette.ContrastBg,
		Color:      th.Palette.ContrastFg,
		Icon:       btn.Icon,
		Size:       btn.Size,
		Inset:      btn.Inset,
		Button:     btn.Button,
	}.Layout(gtx)
}
