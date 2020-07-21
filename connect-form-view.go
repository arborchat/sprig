package main

import (
	"log"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"git.sr.ht/~whereswaldon/materials"
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type ConnectFormView struct {
	manager ViewManager
	widget.Editor
	ConnectButton widget.Clickable
	PasteButton   widget.Clickable

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
	c.Editor.SingleLine = true
	c.Editor.Submit = true

	c.Editor.SetText(settings.Address)
	return c
}

func (c *ConnectFormView) NavItem() *materials.NavItem {
	return &materials.NavItem{
		Name: "Connection",
		Icon: icons.ServerIcon,
	}
}

func (c *ConnectFormView) AppBarData() (bool, string, []materials.AppBarAction, []materials.OverflowAction) {
	return true, "Connection", []materials.AppBarAction{}, []materials.OverflowAction{}
}

func (c *ConnectFormView) HandleClipboard(contents string) {
	c.Editor.Insert(contents)
}

func (c *ConnectFormView) Update(gtx layout.Context) {
	submit := false
	for _, e := range c.Editor.Events() {
		if _, ok := e.(widget.SubmitEvent); ok {
			submit = true
		}
		log.Printf("editor event: %v %T", e, e)
	}
	if c.ConnectButton.Clicked() {
		submit = true
	}
	if submit {
		c.Settings.Address = c.Editor.Text()
		go c.Settings.Persist()
		c.ArborState.RestartWorker(c.Settings.Address)
		c.manager.RequestViewSwitch(CommunityMenuID)
	}
	if c.PasteButton.Clicked() {
		c.manager.RequestClipboardPaste()
	}
}

func (c *ConnectFormView) Layout(gtx layout.Context) layout.Dimensions {
	theme := c.Theme.Theme
	inset := layout.UniformInset(unit.Dp(4))
	return layout.Center.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Center.Layout(gtx, func(gtx C) D {
					return inset.Layout(gtx,
						material.Body1(theme, "Arbor Relay Address:").Layout,
					)
				})
			}),
			layout.Rigid(func(gtx C) D {
				return layout.Center.Layout(gtx, func(gtx C) D {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return inset.Layout(gtx, func(gtx C) D {
								pasteButton := material.IconButton(theme, &c.PasteButton, icons.PasteIcon)
								pasteButton.Inset = layout.UniformInset(unit.Dp(4))
								pasteButton.Size = unit.Dp(20)
								return pasteButton.Layout(gtx)
							})
						}),
						layout.Rigid(func(gtx C) D {
							return inset.Layout(gtx,
								material.Editor(theme, &(c.Editor), "HOST:PORT").Layout,
							)
						}),
					)
				})
			}),
			layout.Rigid(func(gtx C) D {
				return layout.Center.Layout(gtx, func(gtx C) D {
					return inset.Layout(gtx,
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
