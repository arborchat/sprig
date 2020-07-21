package main

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"git.sr.ht/~whereswaldon/materials"
	sprigWidget "git.sr.ht/~whereswaldon/sprig/widget"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type ConnectFormView struct {
	manager ViewManager
	Form    sprigWidget.TextForm

	*Settings
	*ArborState
	*sprigTheme.Theme
}

var _ View = &ConnectFormView{}

func NewConnectFormView(settings *Settings, arborState *ArborState, theme *sprigTheme.Theme) View {
	c := &ConnectFormView{
		Settings:   settings,
		ArborState: arborState,
		Theme:      theme,
	}
	return c
}

func (c *ConnectFormView) NavItem() *materials.NavItem {
	return nil
}

func (c *ConnectFormView) AppBarData() (bool, string, []materials.AppBarAction, []materials.OverflowAction) {
	return false, "", nil, nil
}

func (c *ConnectFormView) HandleClipboard(contents string) {
	c.Form.Paste(contents)
}

func (c *ConnectFormView) Update(gtx layout.Context) {
	c.Form.SetText(c.Settings.Address)
	if c.Form.Submitted() {
		c.Settings.Address = c.Form.Text()
		go c.Settings.Persist()
		c.ArborState.RestartWorker(c.Settings.Address)
		c.manager.RequestViewSwitch(SettingsID)
	}
	if c.Form.PasteRequested() {
		c.manager.RequestClipboardPaste()
	}
}

func (c *ConnectFormView) Layout(gtx layout.Context) layout.Dimensions {
	theme := c.Theme.Theme
	inset := layout.UniformInset(unit.Dp(8))
	return inset.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return inset.Layout(gtx,
					material.H6(theme, "Arbor Relay Address:").Layout,
				)
			}),
			layout.Rigid(func(gtx C) D {
				return inset.Layout(gtx, sprigTheme.TextForm(c.Theme, &c.Form, "Connect", "HOST:PORT").Layout)
			}),
		)
	})
}

func (c *ConnectFormView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
