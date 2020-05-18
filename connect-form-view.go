package main

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type ConnectFormView struct {
	manager ViewManager
	widget.Editor
	ConnectButton widget.Clickable

	*Settings
	*ArborState
	*material.Theme

	// initialized tracks whether or not the view has been updated at least once.
	// this is used to determine whether or not the Update method should immediately
	// switch to the CommunityMenu view if an address has already been provided.
	initialized bool
}

var _ View = &ConnectFormView{}

func NewConnectFormView(settings *Settings, arborState *ArborState, theme *material.Theme) View {
	c := &ConnectFormView{
		Settings:   settings,
		ArborState: arborState,
		Theme:      theme,
	}

	c.Editor.SetText(settings.Address)
	return c
}

func (c *ConnectFormView) Update(gtx *layout.Context) {
	switch {
	case c.ConnectButton.Clicked(gtx):
		c.Settings.Address = c.Editor.Text()
		go c.Settings.Persist()
		fallthrough
	case !c.initialized && c.Settings.Address != "":
		c.ArborState.RestartWorker(c.Settings.Address)
		c.manager.RequestViewSwitch(CommunityMenu)
	}
	c.initialized = true
}

func (c *ConnectFormView) Layout(gtx *layout.Context) {
	theme := c.Theme
	layout.Center.Layout(gtx, func() {
		layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func() {
				layout.Center.Layout(gtx, func() {
					layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
						material.Body1(theme, "Arbor Relay Address:").Layout(gtx)
					})
				})
			}),
			layout.Rigid(func() {
				layout.Center.Layout(gtx, func() {
					layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
						material.Editor(theme, "HOST:PORT").Layout(gtx, &(c.Editor))
					})
				})
			}),
			layout.Rigid(func() {
				layout.Center.Layout(gtx, func() {
					layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
						material.Button(theme, "Connect").Layout(gtx, &(c.ConnectButton))
					})
				})
			}),
		)
	})
}

func (c *ConnectFormView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
