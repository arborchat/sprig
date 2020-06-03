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

func (c *ConnectFormView) HandleClipboard(contents string) {
}

func (c *ConnectFormView) Update(gtx layout.Context) {
	if c.ConnectButton.Clicked() {
		c.Settings.Address = c.Editor.Text()
		go c.Settings.Persist()
		c.ArborState.RestartWorker(c.Settings.Address)
		c.manager.RequestViewSwitch(CommunityMenuID)
	}
}

func (c *ConnectFormView) Layout(gtx layout.Context) layout.Dimensions {
	theme := c.Theme
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.Body1(theme, "Arbor Relay Address:").Layout,
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.Editor(theme, &(c.Editor), "HOST:PORT").Layout,
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.Button(theme, &(c.ConnectButton), "Connect").Layout,
					)
				})
			}),
		)
	})
}

func (c *ConnectFormView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
