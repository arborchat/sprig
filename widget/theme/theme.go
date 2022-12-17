package theme

import (
	_ "embed"
	"image/color"

	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/text"
	"gioui.org/widget/material"
)

// PairFor wraps the provided theme color in a Color type with an automatically
// populated Text color. The Text field value is chosen based on the luminance
// of the provided color.
func PairFor(bg color.NRGBA) ContrastPair {
	col := ContrastPair{
		Bg: bg,
	}
	lum := grayscaleLuminance(bg)
	if lum < 150 {
		col.Fg = white
	} else {
		col.Fg = black
	}
	return col
}

func grayscaleLuminance(c color.NRGBA) uint8 {
	return uint8(float32(c.R)*.3 + float32(c.G)*.59 + float32(c.B)*.11)
}

var (
	teal         = color.NRGBA{R: 0x44, G: 0xa8, B: 0xad, A: 255}
	brightTeal   = color.NRGBA{R: 0x79, G: 0xda, B: 0xdf, A: 255}
	darkTeal     = color.NRGBA{R: 0x00, G: 0x79, B: 0x7e, A: 255}
	green        = color.NRGBA{R: 0x45, G: 0xae, B: 0x7f, A: 255}
	brightGreen  = color.NRGBA{R: 0x79, G: 0xe0, B: 0xae, A: 255}
	darkGreen    = color.NRGBA{R: 0x00, G: 0x7e, B: 0x52, A: 255}
	gold         = color.NRGBA{R: 255, G: 214, B: 79, A: 255}
	lightGold    = color.NRGBA{R: 255, G: 255, B: 129, A: 255}
	darkGold     = color.NRGBA{R: 200, G: 165, B: 21, A: 255}
	white        = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	lightGray    = color.NRGBA{R: 225, G: 225, B: 225, A: 255}
	gray         = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
	darkGray     = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
	veryDarkGray = color.NRGBA{R: 50, G: 50, B: 50, A: 255}
	black        = color.NRGBA{A: 255}

	purple1           = color.NRGBA{R: 69, G: 56, B: 127, A: 255}
	lightPurple1      = color.NRGBA{R: 121, G: 121, B: 174, A: 255}
	darkPurple1       = color.NRGBA{R: 99, G: 41, B: 115, A: 255}
	purple2           = color.NRGBA{R: 127, G: 96, B: 183, A: 255}
	lightPurple2      = color.NRGBA{R: 121, G: 150, B: 223, A: 255}
	darkPurple2       = color.NRGBA{R: 101, G: 89, B: 223, A: 255}
	dmBackground      = color.NRGBA{R: 12, G: 12, B: 15, A: 255}
	dmDarkBackground  = black
	dmLightBackground = color.NRGBA{R: 27, G: 22, B: 33, A: 255}
	dmText            = color.NRGBA{R: 194, G: 196, B: 199, A: 255}
)

//go:embed fonts/static/NotoEmoji-Regular.ttf
var emojiTTF []byte

var emoji text.FontFace = func() text.FontFace {
	face, _ := opentype.Parse(emojiTTF)
	return text.FontFace{
		Font: text.Font{Typeface: "emoji"},
		Face: face,
	}
}()

func New() *Theme {
	collection := gofont.Collection()
	collection = append(collection, emoji)
	gioTheme := material.NewTheme(collection)
	var t Theme
	t.Theme = gioTheme
	t.Primary = Swatch{
		Default: PairFor(green),
		Light:   PairFor(brightGreen),
		Dark:    PairFor(darkGreen),
	}
	t.Secondary = Swatch{
		Default: PairFor(teal),
		Light:   PairFor(brightTeal),
		Dark:    PairFor(darkTeal),
	}
	t.Background = Swatch{
		Default: PairFor(lightGray),
		Light:   PairFor(white),
		Dark:    PairFor(gray),
	}
	t.Theme.Palette.ContrastBg = t.Primary.Default.Bg
	t.Theme.Palette.ContrastFg = t.Primary.Default.Fg
	t.Ancestors = &t.Secondary.Default.Bg
	t.Descendants = &t.Secondary.Default.Bg
	t.Selected = &t.Secondary.Light.Bg
	t.Unselected = &t.Background.Light.Bg
	t.Siblings = t.Unselected

	t.FadeAlpha = 128

	return &t
}

func (t *Theme) ToDark() {
	t.Background.Dark = PairFor(darkGray)
	t.Background.Default = PairFor(veryDarkGray)
	t.Background.Light = PairFor(black)
	t.Primary.Default = PairFor(purple1)
	t.Primary.Light = PairFor(lightPurple1)
	t.Primary.Dark = PairFor(darkPurple1)
	t.Secondary.Default = PairFor(purple2)
	t.Secondary.Light = PairFor(lightPurple2)
	t.Secondary.Dark = PairFor(darkPurple2)

	t.Background.Default = PairFor(dmBackground)
	t.Background.Light = PairFor(dmLightBackground)
	t.Background.Dark = PairFor(dmDarkBackground)

	// apply to theme
	t.Theme.Palette.Fg, t.Theme.Palette.Bg = t.Theme.Palette.Bg, t.Theme.Palette.Fg
	t.Theme.Palette = ApplyAsContrast(t.Theme.Palette, t.Primary.Default)
}

type ContrastPair struct {
	Fg, Bg color.NRGBA
}

func ApplyAsContrast(p material.Palette, pair ContrastPair) material.Palette {
	p.ContrastBg = pair.Bg
	p.ContrastFg = pair.Fg
	return p
}

func ApplyAsNormal(p material.Palette, pair ContrastPair) material.Palette {
	p.Bg = pair.Bg
	p.Fg = pair.Fg
	return p
}

type Swatch struct {
	Light, Dark, Default ContrastPair
}

type Theme struct {
	*material.Theme
	Primary    Swatch
	Secondary  Swatch
	Background Swatch

	FadeAlpha uint8

	Ancestors, Descendants, Selected, Siblings, Unselected *color.NRGBA
}

func (t *Theme) ApplyAlpha(c color.NRGBA) color.NRGBA {
	c.A = t.FadeAlpha
	return c
}
