package main

import (
	"image/color"
	"log"

	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"git.sr.ht/~whereswaldon/materials"
	"git.sr.ht/~whereswaldon/sprig/core"
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"

	"git.sr.ht/~whereswaldon/colorpicker"
)

type ThemeEditorView struct {
	manager ViewManager
	core.App

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

	AncestorMux    colorpicker.MuxState
	DescendantMux  colorpicker.MuxState
	SelectedMux    colorpicker.MuxState
	SiblingMux     colorpicker.MuxState
	NonselectedMux colorpicker.MuxState

	MuxList      layout.List
	muxListElems []muxListElement

	*sprigTheme.Theme
	widgetTheme *material.Theme
}

type colorListElement struct {
	*colorpicker.State
	Label        string
	TargetColors []*color.NRGBA
}

type muxListElement struct {
	*colorpicker.MuxState
	Label       string
	TargetColor **color.NRGBA
}

var _ View = &ThemeEditorView{}

func NewThemeEditorView(app core.App) View {
	c := &ThemeEditorView{
		App:         app,
		widgetTheme: material.NewTheme(gofont.Collection()),
	}

	c.ConfigurePickersFor(app.Theme().Current())
	return c
}

func (c *ThemeEditorView) ConfigurePickersFor(th *sprigTheme.Theme) {
	c.PrimaryDefault.SetColor(th.Primary.Default.Bg)
	c.PrimaryDark.SetColor(th.Primary.Dark.Bg)
	c.PrimaryLight.SetColor(th.Primary.Light.Bg)
	c.SecondaryDefault.SetColor(th.Secondary.Default.Bg)
	c.SecondaryDark.SetColor(th.Secondary.Dark.Bg)
	c.SecondaryLight.SetColor(th.Secondary.Light.Bg)
	c.BackgroundDefault.SetColor(th.Background.Default.Bg)
	c.BackgroundDark.SetColor(th.Background.Dark.Bg)
	c.BackgroundLight.SetColor(th.Background.Light.Bg)

	c.ColorsList.Axis = layout.Vertical
	c.listElems = []colorListElement{
		{
			Label: "Primary",
			TargetColors: []*color.NRGBA{
				&th.Primary.Default.Bg,
				&th.Theme.Palette.Bg,
			},
			State: &c.PrimaryDefault,
		},
		{
			Label: "Primary Light",
			TargetColors: []*color.NRGBA{
				&th.Primary.Light.Bg,
			},
			State: &c.PrimaryLight,
		},
		{
			Label: "Primary Dark",
			TargetColors: []*color.NRGBA{
				&th.Primary.Dark.Bg,
			},
			State: &c.PrimaryDark,
		},
		{
			Label: "Secondary",
			TargetColors: []*color.NRGBA{
				&th.Secondary.Default.Bg,
			},
			State: &c.SecondaryDefault,
		},
		{
			Label: "Secondary Light",
			TargetColors: []*color.NRGBA{
				&th.Secondary.Light.Bg,
			},
			State: &c.SecondaryLight,
		},
		{
			Label: "Secondary Dark",
			TargetColors: []*color.NRGBA{
				&th.Secondary.Dark.Bg,
			},
			State: &c.SecondaryDark,
		},
		{
			Label: "Background",
			TargetColors: []*color.NRGBA{
				&th.Background.Default.Bg,
			},
			State: &c.BackgroundDefault,
		},
		{
			Label: "Background Light",
			TargetColors: []*color.NRGBA{
				&th.Background.Light.Bg,
			},
			State: &c.BackgroundLight,
		},
		{
			Label: "Background Dark",
			TargetColors: []*color.NRGBA{
				&th.Background.Dark.Bg,
			},
			State: &c.BackgroundDark,
		},
	}

	muxOptions := []colorpicker.MuxOption{}
	for _, elem := range c.listElems {
		if len(elem.TargetColors) < 1 || elem.TargetColors[0] == nil {
			continue
		}
		elem.SetColor(*elem.TargetColors[0])
		muxOptions = append(muxOptions, colorpicker.MuxOption{
			Label: elem.Label,
			Value: elem.TargetColors[0],
		})
	}
	c.muxListElems = []muxListElement{
		{
			Label:       "Ancestors",
			MuxState:    &c.AncestorMux,
			TargetColor: &th.Ancestors,
		},
		{
			Label:       "Descendants",
			MuxState:    &c.DescendantMux,
			TargetColor: &th.Descendants,
		},
		{
			Label:       "Selected",
			MuxState:    &c.SelectedMux,
			TargetColor: &th.Selected,
		},
		{
			Label:       "Siblings",
			MuxState:    &c.SiblingMux,
			TargetColor: &th.Siblings,
		},
		{
			Label:       "Unselected",
			MuxState:    &c.NonselectedMux,
			TargetColor: &th.Unselected,
		},
	}
	for _, mux := range c.muxListElems {
		*mux.MuxState = colorpicker.NewMuxState(muxOptions...)
	}
}

func (c *ThemeEditorView) BecomeVisible() {
	c.ConfigurePickersFor(c.App.Theme().Current())
}

func (c *ThemeEditorView) NavItem() *materials.NavItem {
	return &materials.NavItem{
		Name: "Theme",
		Icon: icons.CancelReplyIcon,
	}
}

func (c *ThemeEditorView) AppBarData() (bool, string, []materials.AppBarAction, []materials.OverflowAction) {
	return true, "Theme", []materials.AppBarAction{}, []materials.OverflowAction{}
}

func (c *ThemeEditorView) HandleClipboard(contents string) {
}

func (c *ThemeEditorView) Update(gtx layout.Context) {
	for i, elem := range c.listElems {
		if elem.Changed() {
			for _, target := range elem.TargetColors {
				*target = elem.Color()
			}
			op.InvalidateOp{}.Add(gtx.Ops)
			log.Printf("picker %d changed", i)
		}
	}
	for _, elem := range c.muxListElems {
		if elem.Changed() {
			*elem.TargetColor = elem.Color()
			op.InvalidateOp{}.Add(gtx.Ops)
			log.Printf("mux changed")
		}
	}
}

func (c *ThemeEditorView) Layout(gtx layout.Context) layout.Dimensions {
	return c.layoutPickers(gtx)
}

func (c *ThemeEditorView) layoutPickers(gtx layout.Context) layout.Dimensions {
	return c.ColorsList.Layout(gtx, len(c.listElems)+1, func(gtx C, index int) D {
		if index == len(c.listElems) {
			return c.layoutMuxes(gtx)
		}
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx C) D {
				return sprigTheme.Rect{
					Color: color.NRGBA{A: 255},
					Size: f32.Point{
						X: float32(gtx.Constraints.Min.X),
						Y: float32(gtx.Constraints.Min.Y),
					},
				}.Layout(gtx)
			}),
			layout.Stacked(func(gtx C) D {
				return layout.UniformInset(unit.Dp(3)).Layout(gtx, func(gtx C) D {
					return layout.Stack{}.Layout(gtx,
						layout.Expanded(func(gtx C) D {
							return sprigTheme.Rect{
								Color: color.NRGBA{R: 255, G: 255, B: 255, A: 255},
								Size: f32.Point{
									X: float32(gtx.Constraints.Min.X),
									Y: float32(gtx.Constraints.Min.Y),
								},
							}.Layout(gtx)
						}),
						layout.Stacked(func(gtx C) D {
							elem := c.listElems[index]
							dims := colorpicker.Picker(c.widgetTheme, elem.State, elem.Label).Layout(gtx)
							return dims
						}),
					)
				})
			}),
		)
	})
}

func (c *ThemeEditorView) layoutMuxes(gtx layout.Context) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			return sprigTheme.Rect{
				Color: color.NRGBA{R: 255, G: 255, B: 255, A: 255},
				Size: f32.Point{
					X: float32(gtx.Constraints.Min.X),
					Y: float32(gtx.Constraints.Min.Y),
				},
			}.Layout(gtx)
		}),
		layout.Stacked(func(gtx C) D {
			return c.MuxList.Layout(gtx, len(c.muxListElems), func(gtx C, index int) D {
				element := c.muxListElems[index]
				return colorpicker.Mux(c.widgetTheme, element.MuxState, element.Label).Layout(gtx)
			})
		}),
	)
}

func (c *ThemeEditorView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
