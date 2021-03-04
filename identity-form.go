package main

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	materials "gioui.org/x/component"
	"git.sr.ht/~whereswaldon/sprig/core"
	sprigWidget "git.sr.ht/~whereswaldon/sprig/widget"
)

type IdentityFormView struct {
	manager ViewManager
	sprigWidget.TextForm
	CreateButton widget.Clickable

	core.App
}

var _ View = &IdentityFormView{}

func NewIdentityFormView(app core.App) View {
	c := &IdentityFormView{
		App: app,
	}
	c.TextForm.TextField.Editor.SingleLine = true

	return c
}

func (c *IdentityFormView) BecomeVisible() {
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
		c.Settings().CreateIdentity(c.TextField.Text())
		c.manager.RequestViewSwitch(SubscriptionSetupFormViewID)
	}
}

func (c *IdentityFormView) Layout(gtx layout.Context) layout.Dimensions {
	theme := c.Theme().Current().Theme
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
					return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx C) D {
						return c.TextField.Layout(gtx, theme, "Username")
					})
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
