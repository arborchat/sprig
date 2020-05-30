package main

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type ConsentView struct {
	manager     ViewManager
	AgreeButton widget.Clickable

	*Settings
	*ArborState
	*material.Theme
}

var _ View = &ConsentView{}

func NewConsentView(settings *Settings, arborState *ArborState, theme *material.Theme) View {
	c := &ConsentView{
		Settings:   settings,
		ArborState: arborState,
		Theme:      theme,
	}

	return c
}

func (c *ConsentView) HandleClipboard(contents string) {
}

func (c *ConsentView) Update(gtx layout.Context) {
	if c.AgreeButton.Clicked() {
		c.Settings.AcknowledgedNoticeVersion = NoticeVersion
		go c.Settings.Persist()
		c.manager.RequestViewSwitch(CommunityMenuID)
	}
}

const (
	UpdateText    = "You are seeing this message because the notice text has changed since you last accepted it."
	Notice        = "This is a chat client for the Arbor Chat Project. Before you send a message, you should know that your messages cannot be edited or deleted once sent, and that they will be publically visible to all other Arbor users."
	NoticeVersion = 1
)

func (c *ConsentView) Layout(gtx layout.Context) layout.Dimensions {
	theme := c.Theme
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.H2(theme, "Notice").Layout,
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.Body1(theme, Notice).Layout,
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if c.Settings.AcknowledgedNoticeVersion != 0 {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(4)).Layout(gtx,
							material.Body2(theme, UpdateText).Layout,
						)
					})
				}
				return layout.Dimensions{}
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.Button(theme, &(c.AgreeButton), "I Understand And Agree").Layout,
					)
				})
			}),
		)
	})
}

func (c *ConsentView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
