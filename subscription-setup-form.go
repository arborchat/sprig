package main

import (
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	materials "gioui.org/x/component"
	"git.sr.ht/~whereswaldon/sprig/core"
	"git.sr.ht/~whereswaldon/sprig/icons"
)

type SubSetupFormView struct {
	manager ViewManager

	core.App

	SubStateManager
	ConnectionList layout.List

	Refresh, Continue widget.Clickable
}

var _ View = &SubSetupFormView{}

func NewSubSetupFormView(app core.App) View {
	c := &SubSetupFormView{
		App: app,
	}
	c.SubStateManager = NewSubStateManager(app, func() {
		c.manager.RequestInvalidate()
	})
	c.ConnectionList.Axis = layout.Vertical
	return c
}

func (c *SubSetupFormView) HandleIntent(intent Intent) {}

func (c *SubSetupFormView) BecomeVisible() {
	c.SubStateManager.Refresh()
	go func() {
		time.Sleep(time.Second)
		c.SubStateManager.Refresh()
	}()
}

func (c *SubSetupFormView) NavItem() *materials.NavItem {
	return nil
}

func (c *SubSetupFormView) AppBarData() (bool, string, []materials.AppBarAction, []materials.OverflowAction) {
	return false, "", nil, nil
}

func (c *SubSetupFormView) Update(gtx layout.Context) {
	c.SubStateManager.Update()
	if c.Refresh.Clicked() {
		c.SubStateManager.Refresh()
	}
	if c.Continue.Clicked() {
		c.manager.SetView(ReplyViewID)
	}
}

func (c *SubSetupFormView) Layout(gtx layout.Context) layout.Dimensions {
	c.Update(gtx)
	sTheme := c.Theme().Current()
	theme := sTheme.Theme
	inset := layout.UniformInset(unit.Dp(12))

	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return inset.Layout(gtx, func(gtx C) D {
				return material.Body1(theme, "Subscribe to a few communities to get started:").Layout(gtx)
			})
		}),
		layout.Flexed(1.0, func(gtx C) D {
			return layout.UniformInset(unit.Dp(4)).Layout(gtx, SubscriptionList(theme, &c.ConnectionList, c.Subs).Layout)
		}),
		layout.Rigid(func(gtx C) D {
			return inset.Layout(gtx, func(gtx C) D {
				return layout.Flex{Spacing: layout.SpaceAround}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return material.IconButton(theme, &c.Refresh, icons.RefreshIcon).Layout(gtx)
					}),
					layout.Rigid(func(gtx C) D {
						return material.IconButton(theme, &c.Continue, icons.ForwardIcon).Layout(gtx)
					}),
				)
			})
		}),
	)
}

func (c *SubSetupFormView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
