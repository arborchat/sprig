package main

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"git.sr.ht/~whereswaldon/materials"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type IdentityFormView struct {
	manager ViewManager
	widget.Editor
	CreateButton widget.Clickable

	*Settings
	*ArborState
	*sprigTheme.Theme
}

var _ View = &IdentityFormView{}

func NewIdentityFormView(settings *Settings, arborState *ArborState, theme *sprigTheme.Theme) View {
	c := &IdentityFormView{
		Settings:   settings,
		ArborState: arborState,
		Theme:      theme,
	}

	return c
}

func (c *IdentityFormView) NavItem() *materials.NavItem {
	return nil
}

func (c *IdentityFormView) AppBarData() (bool, string, []materials.AppBarAction, []materials.OverflowAction) {
	return false, "", nil, nil
}

func (c *IdentityFormView) HandleClipboard(contents string) {
}

func (c *IdentityFormView) Update(gtx layout.Context) {
	if c.CreateButton.Clicked() {
		c.Settings.CreateIdentity(c.Editor.Text())
		c.manager.RequestViewSwitch(SettingsID)
	}
}

func (c *IdentityFormView) Layout(gtx layout.Context) layout.Dimensions {
	theme := c.Theme.Theme
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.Body1(theme, "Your Arbor Username:").Layout,
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.Editor(theme, &(c.Editor), "username").Layout,
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.Body2(theme, "Your username is public, and cannot currently be changed once it is chosen.").Layout,
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.Button(theme, &(c.CreateButton), "Create").Layout,
					)
				})
			}),
		)
	})
}

func (c *IdentityFormView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
