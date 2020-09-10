package main

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"git.sr.ht/~whereswaldon/materials"
	"git.sr.ht/~whereswaldon/sprig/core"
)

type ConsentView struct {
	manager     ViewManager
	AgreeButton widget.Clickable

	core.App
}

var _ View = &ConsentView{}

func NewConsentView(app core.App) View {
	c := &ConsentView{
		App: app,
	}

	return c
}

func (c *ConsentView) BecomeVisible() {
}

func (c *ConsentView) NavItem() *materials.NavItem {
	return nil
}

func (c *ConsentView) AppBarData() (bool, string, []materials.AppBarAction, []materials.OverflowAction) {
	return false, "", nil, nil
}

func (c *ConsentView) HandleClipboard(contents string) {
}

func (c *ConsentView) Update(gtx layout.Context) {
	if c.AgreeButton.Clicked() {
		c.Settings().SetAcknowledgedNoticeVersion(NoticeVersion)
		go c.Settings().Persist()
		if c.Settings().Address() == "" {
			c.manager.RequestViewSwitch(ConnectFormID)
		} else {
			c.manager.RequestViewSwitch(SettingsID)
		}
	}
}

const (
	UpdateText    = "You are seeing this message because the notice text has changed since you last accepted it."
	Notice        = "This is a chat client for the Arbor Chat Project. Before you send a message, you should know that your messages cannot be edited or deleted once sent, and that they will be publically visible to all other Arbor users."
	NoticeVersion = 1
)

func (c *ConsentView) Layout(gtx layout.Context) layout.Dimensions {
	theme := c.Theme().Current()
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.H2(theme.Theme, "Notice").Layout,
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.Body1(theme.Theme, Notice).Layout,
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if c.Settings().AcknowledgedNoticeVersion() != 0 {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(4)).Layout(gtx,
							material.Body2(theme.Theme, UpdateText).Layout,
						)
					})
				}
				return layout.Dimensions{}
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.Button(theme.Theme, &(c.AgreeButton), "I Understand And Agree").Layout,
					)
				})
			}),
		)
	})
}

func (c *ConsentView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
