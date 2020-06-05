package theme

import (
	"image/color"

	"gioui.org/widget/material"
)

var (
	teal       = color.RGBA{G: 128, B: 128, A: 255}
	brightTeal = color.RGBA{R: 78, G: 186, B: 170, A: 255}
	darkTeal   = color.RGBA{G: 91, B: 79, A: 255}
	gold       = color.RGBA{R: 255, G: 214, B: 79, A: 255}
	lightGold  = color.RGBA{R: 255, G: 255, B: 129, A: 255}
	darkGold   = color.RGBA{R: 200, G: 165, B: 21, A: 255}
	white      = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	lightGray  = color.RGBA{R: 225, G: 225, B: 225, A: 255}
	black      = color.RGBA{A: 255}
)

func New() *Theme {
	gioTheme := material.NewTheme()
	var t Theme
	t.Theme = gioTheme
	t.Primary = Colors{
		Default: teal,
		Light:   brightTeal,
		Dark:    darkTeal,
	}
	t.Secondary = Colors{
		Default: gold,
		Light:   lightGold,
		Dark:    darkGold,
	}
	t.Background = Colors{
		Default: lightGray,
		Light:   white,
		Dark:    black,
	}
	t.Theme.Color.Primary = t.Primary.Default
	return &t
}

type Theme struct {
	*material.Theme
	Primary    Colors
	Secondary  Colors
	Background Colors
}

type Colors struct {
	Default, Light, Dark color.RGBA
}
