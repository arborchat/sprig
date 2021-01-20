package main

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	materials "gioui.org/x/component"
	"git.sr.ht/~whereswaldon/sprig/core"
	sprigWidget "git.sr.ht/~whereswaldon/sprig/widget"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type ConnectFormView struct {
	manager ViewManager
	Form    sprigWidget.TextForm

	core.App
}

var _ View = &ConnectFormView{}

func NewConnectFormView(app core.App) View {
	c := &ConnectFormView{
		App: app,
	}
	c.Form.TextField.SingleLine = true
	c.Form.TextField.Submit = true
	return c
}

func (c *ConnectFormView) BecomeVisible() {
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
	if c.Form.Submitted() {
		c.Settings().SetAddress(c.Form.TextField.Text())
		go c.Settings().Persist()
		c.Sprout().ConnectTo(c.Settings().Address())
		c.manager.RequestViewSwitch(IdentityFormID)
	}
	if c.Form.PasteRequested() {
		c.manager.RequestClipboardPaste()
	}
}

func (c *ConnectFormView) Layout(gtx layout.Context) layout.Dimensions {
	theme := c.Theme().Current()
	inset := layout.UniformInset(unit.Dp(8))
	return inset.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return inset.Layout(gtx,
					material.H6(theme.Theme, "Arbor Relay Address:").Layout,
				)
			}),
			layout.Rigid(func(gtx C) D {
				return inset.Layout(gtx, sprigTheme.TextForm(theme, &c.Form, "Connect", "HOST:PORT").Layout)
			}),
		)
	})
}

func (c *ConnectFormView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
