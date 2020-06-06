package main

import (
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/unit"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"

	"git.sr.ht/~whereswaldon/colorpicker"
)

type ThemeEditorView struct {
	manager ViewManager

	PrimaryDefault colorpicker.State
	PrimaryDark    colorpicker.State
	PrimaryLight   colorpicker.State

	SecondaryDefault colorpicker.State
	SecondaryDark    colorpicker.State
	SecondaryLight   colorpicker.State

	BackgroundDefault colorpicker.State
	BackgroundDark    colorpicker.State
	BackgroundLight   colorpicker.State

	TextColor         colorpicker.State
	HintColor         colorpicker.State
	InvertedTextColor colorpicker.State

	ColorsList layout.List
	listElems  []colorListElement

	*sprigTheme.Theme
}

type colorListElement struct {
	*colorpicker.State
	Label        string
	TargetColors []*color.RGBA
}

var _ View = &ThemeEditorView{}

func NewThemeEditorView(theme *sprigTheme.Theme) View {
	c := &ThemeEditorView{
		Theme: theme,
	}
	c.PrimaryDefault.SetColor(c.Theme.Primary.Default)
	c.PrimaryDark.SetColor(c.Theme.Primary.Dark)
	c.PrimaryLight.SetColor(c.Theme.Primary.Light)
	c.SecondaryDefault.SetColor(c.Theme.Secondary.Default)
	c.SecondaryDark.SetColor(c.Theme.Secondary.Dark)
	c.SecondaryLight.SetColor(c.Theme.Secondary.Light)
	c.BackgroundDefault.SetColor(c.Theme.Background.Default)
	c.BackgroundDark.SetColor(c.Theme.Background.Dark)
	c.BackgroundLight.SetColor(c.Theme.Background.Light)

	c.ColorsList.Axis = layout.Vertical
	c.listElems = []colorListElement{
		{
			Label: "Primary",
			TargetColors: []*color.RGBA{
				&c.Theme.Primary.Default,
				&c.Theme.Theme.Color.Primary,
			},
			State: &c.PrimaryDefault,
		},
		{
			Label: "Primary Light",
			TargetColors: []*color.RGBA{
				&c.Theme.Primary.Light,
			},
			State: &c.PrimaryLight,
		},
		{
			Label: "Primary Dark",
			TargetColors: []*color.RGBA{
				&c.Theme.Primary.Dark,
			},
			State: &c.PrimaryDark,
		},
		{
			Label: "Secondary",
			TargetColors: []*color.RGBA{
				&c.Theme.Secondary.Default,
			},
			State: &c.SecondaryDefault,
		},
		{
			Label: "Secondary Light",
			TargetColors: []*color.RGBA{
				&c.Theme.Secondary.Light,
			},
			State: &c.SecondaryLight,
		},
		{
			Label: "Secondary Dark",
			TargetColors: []*color.RGBA{
				&c.Theme.Secondary.Dark,
			},
			State: &c.SecondaryDark,
		},
		{
			Label: "Background",
			TargetColors: []*color.RGBA{
				&c.Theme.Background.Default,
			},
			State: &c.BackgroundDefault,
		},
		{
			Label: "Background Light",
			TargetColors: []*color.RGBA{
				&c.Theme.Background.Light,
			},
			State: &c.BackgroundLight,
		},
		{
			Label: "Background Dark",
			TargetColors: []*color.RGBA{
				&c.Theme.Background.Dark,
			},
			State: &c.BackgroundDark,
		},
		{
			Label: "Text",
			TargetColors: []*color.RGBA{
				&c.Theme.Theme.Color.Text,
			},
			State: &c.TextColor,
		},
		{
			Label: "Hint",
			TargetColors: []*color.RGBA{
				&c.Theme.Theme.Color.Hint,
			},
			State: &c.HintColor,
		},
		{
			Label: "Inverted Text",
			TargetColors: []*color.RGBA{
				&c.Theme.Theme.Color.InvText,
			},
			State: &c.InvertedTextColor,
		},
	}

	for _, elem := range c.listElems {
		if len(elem.TargetColors) < 1 || elem.TargetColors[0] == nil {
			continue
		}
		elem.SetColor(*elem.TargetColors[0])
	}

	return c
}

func (c *ThemeEditorView) HandleClipboard(contents string) {
}

func (c *ThemeEditorView) Update(gtx layout.Context) {
	for _, elem := range c.listElems {
		if elem.Changed() {
			for _, target := range elem.TargetColors {
				*target = elem.Color()
			}
		}
	}
}

func (c *ThemeEditorView) Layout(gtx layout.Context) layout.Dimensions {
	return c.ColorsList.Layout(gtx, len(c.listElems), func(gtx C, index int) D {
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx C) D {
				return sprigTheme.DrawRect(gtx, color.RGBA{A: 255}, f32.Point{
					X: float32(gtx.Constraints.Min.X),
					Y: float32(gtx.Constraints.Min.Y),
				}, 0)
			}),
			layout.Stacked(func(gtx C) D {
				return layout.UniformInset(unit.Dp(3)).Layout(gtx, func(gtx C) D {
					return layout.Stack{}.Layout(gtx,
						layout.Expanded(func(gtx C) D {
							return sprigTheme.DrawRect(gtx, color.RGBA{R: 255, G: 255, B: 255, A: 255}, f32.Point{
								X: float32(gtx.Constraints.Min.X),
								Y: float32(gtx.Constraints.Min.Y),
							}, 0)
						}),
						layout.Stacked(func(gtx C) D {
							elem := c.listElems[index]
							return colorpicker.Picker(c.Theme.Theme, elem.State, elem.Label).Layout(gtx)
						}),
					)
				})
			}),
		)
	})
}

func (c *ThemeEditorView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
