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

	Colors layout.List

	*sprigTheme.Theme
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

	c.Colors.Axis = layout.Vertical

	return c
}

func (c *ThemeEditorView) HandleClipboard(contents string) {
}

func (c *ThemeEditorView) Update(gtx layout.Context) {
	if c.PrimaryDefault.Changed() {
		c.Theme.Primary.Default = c.PrimaryDefault.Color()
		c.Theme.Theme.Color.Primary = c.Theme.Primary.Default
	}
	if c.PrimaryDark.Changed() {
		c.Theme.Primary.Dark = c.PrimaryDark.Color()
	}
	if c.PrimaryLight.Changed() {
		c.Theme.Primary.Light = c.PrimaryLight.Color()
	}
	if c.SecondaryDefault.Changed() {
		c.Theme.Secondary.Default = c.SecondaryDefault.Color()
	}
	if c.SecondaryDark.Changed() {
		c.Theme.Secondary.Dark = c.SecondaryDark.Color()
	}
	if c.SecondaryLight.Changed() {
		c.Theme.Secondary.Light = c.SecondaryLight.Color()
	}
	if c.BackgroundDefault.Changed() {
		c.Theme.Background.Default = c.BackgroundDefault.Color()
	}
	if c.BackgroundDark.Changed() {
		c.Theme.Background.Dark = c.BackgroundDark.Color()
	}
	if c.BackgroundLight.Changed() {
		c.Theme.Background.Light = c.BackgroundLight.Color()
	}
}

func (c *ThemeEditorView) Layout(gtx layout.Context) layout.Dimensions {
	return c.Colors.Layout(gtx, 9, func(gtx C, index int) D {
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
							gtx.Constraints.Max.Y = gtx.Px(unit.Dp(150))
							var colorState *colorpicker.State
							var label string
							switch index {
							case 0:
								colorState = &c.PrimaryDefault
								label = "Primary"
							case 1:
								colorState = &c.PrimaryDark
								label = "Primary Dark"
							case 2:
								colorState = &c.PrimaryLight
								label = "Primary Light"
							case 3:
								colorState = &c.SecondaryDefault
								label = "Secondary"
							case 4:
								colorState = &c.SecondaryDark
								label = "Secondary Dark"
							case 5:
								colorState = &c.SecondaryLight
								label = "Secondary Light"
							case 6:
								colorState = &c.BackgroundDefault
								label = "Background"
							case 7:
								colorState = &c.BackgroundDark
								label = "Background Dark"
							case 8:
								colorState = &c.BackgroundLight
								label = "BackgroundLight"
							}
							return colorpicker.Picker(c.Theme.Theme, colorState, label).Layout(gtx)
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
