package theme

import (
	"image/color"

	"gioui.org/font/gofont"
	"gioui.org/widget/material"
)

var (
	teal         = color.RGBA{R: 0x44, G: 0xa8, B: 0xad, A: 255}
	brightTeal   = color.RGBA{R: 0x79, G: 0xda, B: 0xdf, A: 255}
	darkTeal     = color.RGBA{R: 0x00, G: 0x79, B: 0x7e, A: 255}
	green        = color.RGBA{R: 0x45, G: 0xae, B: 0x7f, A: 255}
	brightGreen  = color.RGBA{R: 0x79, G: 0xe0, B: 0xae, A: 255}
	darkGreen    = color.RGBA{R: 0x00, G: 0x7e, B: 0x52, A: 255}
	gold         = color.RGBA{R: 255, G: 214, B: 79, A: 255}
	lightGold    = color.RGBA{R: 255, G: 255, B: 129, A: 255}
	darkGold     = color.RGBA{R: 200, G: 165, B: 21, A: 255}
	white        = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	lightGray    = color.RGBA{R: 225, G: 225, B: 225, A: 255}
	darkGray     = color.RGBA{R: 100, G: 100, B: 100, A: 255}
	veryDarkGray = color.RGBA{R: 50, G: 50, B: 50, A: 255}
	black        = color.RGBA{A: 255}
)

func New() *Theme {
	gioTheme := material.NewTheme(gofont.Collection())
	var t Theme
	t.Theme = gioTheme
	t.Primary = Colors{
		Default: green,
		Light:   brightGreen,
		Dark:    darkGreen,
	}
	t.Secondary = Colors{
		Default: teal,
		Light:   brightTeal,
		Dark:    darkTeal,
	}
	t.Background = Colors{
		Default: lightGray,
		Light:   white,
		Dark:    black,
	}
	t.Theme.Color.Primary = t.Primary.Default
	t.Ancestors = &t.Secondary.Default
	t.Descendants = &t.Secondary.Default
	t.Selected = &t.Secondary.Light
	t.Unselected = &t.Background.Light
	t.Siblings = t.Unselected
	return &t
}

func (t *Theme) ToDark() {
	t.Background.Dark = darkGray
	t.Background.Default = veryDarkGray
	t.Background.Light = black
	t.Color.Text = white
	t.Color.InvText = black
	t.Color.Hint = lightGray
}

type Theme struct {
	*material.Theme
	Primary    Colors
	Secondary  Colors
	Background Colors

	Ancestors, Descendants, Selected, Siblings, Unselected *color.RGBA
}

type Colors struct {
	Default, Light, Dark color.RGBA
}
