package main

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type IdentityFormView struct {
	manager ViewManager
	widget.Editor
	CreateButton widget.Clickable

	*Settings
	*ArborState
	*material.Theme
}

var _ View = &IdentityFormView{}

func NewIdentityFormView(settings *Settings, arborState *ArborState, theme *material.Theme) View {
	c := &IdentityFormView{
		Settings:   settings,
		ArborState: arborState,
		Theme:      theme,
	}

	return c
}

func (c *IdentityFormView) HandleClipboard(contents string) {
}

func (c *IdentityFormView) Update(gtx *layout.Context) {
	if c.CreateButton.Clicked(gtx) {
		c.Settings.CreateIdentity(c.Editor.Text())
		c.manager.RequestViewSwitch(CommunityMenu)
	}
}

func (c *IdentityFormView) Layout(gtx *layout.Context) {
	theme := c.Theme
	layout.Center.Layout(gtx, func() {
		layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func() {
				layout.Center.Layout(gtx, func() {
					layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
						material.Body1(theme, "Your Arbor Username:").Layout(gtx)
					})
				})
			}),
			layout.Rigid(func() {
				layout.Center.Layout(gtx, func() {
					layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
						material.Editor(theme, "username").Layout(gtx, &(c.Editor))
					})
				})
			}),
			layout.Rigid(func() {
				layout.Center.Layout(gtx, func() {
					layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
						material.Body2(theme, "Your username is public, and cannot currently be changed once it is chosen.").Layout(gtx)
					})
				})
			}),
			layout.Rigid(func() {
				layout.Center.Layout(gtx, func() {
					layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
						material.Button(theme, "Create").Layout(gtx, &(c.CreateButton))
					})
				})
			}),
		)
	})
}

func (c *IdentityFormView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
