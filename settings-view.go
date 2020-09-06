package main

import (
	"log"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"git.sr.ht/~whereswaldon/materials"
	"git.sr.ht/~whereswaldon/sprig/core"
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigWidget "git.sr.ht/~whereswaldon/sprig/widget"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type SettingsView struct {
	manager ViewManager

	core.App
	*sprigTheme.Theme

	layout.List
	ConnectionForm          sprigWidget.TextForm
	IdentityButton          widget.Clickable
	CommunityList           layout.List
	CommunityBoxes          []widget.Bool
	ProfilingSwitch         widget.Bool
	ThemeingSwitch          widget.Bool
	NotificationsSwitch     widget.Bool
	TestNotificationsButton widget.Clickable
	TestResults             string
	BottomBarSwitch         widget.Bool
}

var _ View = &SettingsView{}

func NewCommunityMenuView(app core.App, theme *sprigTheme.Theme) View {
	c := &SettingsView{
		App:   app,
		Theme: theme,
	}
	c.List.Axis = layout.Vertical
	c.ConnectionForm.SetText(c.Settings().Address())
	return c
}

func (c *SettingsView) AppBarData() (bool, string, []materials.AppBarAction, []materials.OverflowAction) {
	return true, "Settings", []materials.AppBarAction{}, []materials.OverflowAction{}
}

func (c *SettingsView) NavItem() *materials.NavItem {
	return &materials.NavItem{
		Name: "Settings",
		Icon: icons.SettingsIcon,
	}
}

func (c *SettingsView) HandleClipboard(contents string) {
	c.ConnectionForm.Paste(contents)
}

func (c *SettingsView) Update(gtx layout.Context) {
	settingsChanged := false
	for i := range c.CommunityBoxes {
		box := &c.CommunityBoxes[i]
		if box.Changed() {
			log.Println("updated")
		}
	}
	if c.IdentityButton.Clicked() {
		c.manager.RequestViewSwitch(IdentityFormID)
	}
	if c.ProfilingSwitch.Changed() {
		c.manager.SetProfiling(c.ProfilingSwitch.Value)
	}
	if c.ThemeingSwitch.Changed() {
		c.manager.SetThemeing(c.ThemeingSwitch.Value)
	}
	if c.ConnectionForm.Submitted() {
		c.Settings().SetAddress(c.ConnectionForm.Text())
		settingsChanged = true
		c.Sprout().ConnectTo(c.Settings().Address())
	}
	if c.ConnectionForm.PasteRequested() {
		c.manager.RequestClipboardPaste()
	}
	if c.NotificationsSwitch.Changed() {
		c.Settings().SetNotificationsGloballyAllowed(c.NotificationsSwitch.Value)
		settingsChanged = true
	}
	if c.TestNotificationsButton.Clicked() {
		err := c.Notifications().Notify("Testing!", "This is a test notification from sprig.")
		if err == nil {
			c.TestResults = "Sent without errors"
		} else {
			c.TestResults = "Failed: " + err.Error()
		}
	}
	if c.BottomBarSwitch.Changed() {
		c.Settings().SetBottomAppBar(c.BottomBarSwitch.Value)
		settingsChanged = true
	}
	if settingsChanged {
		c.manager.ApplySettings(c.Settings())
		go c.Settings().Persist()
	}
}

func (c *SettingsView) BecomeVisible() {
	c.ConnectionForm.SetText(c.Settings().Address())
	c.NotificationsSwitch.Value = c.Settings().NotificationsGloballyAllowed()
	c.BottomBarSwitch.Value = c.Settings().BottomAppBar()
}

func (c *SettingsView) Layout(gtx layout.Context) layout.Dimensions {
	theme := c.Theme.Theme
	itemInset := layout.UniformInset(unit.Dp(8))
	areas := []func(C) D{
		func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return itemInset.Layout(gtx, material.H6(theme, "Identity").Layout)
				}),
				layout.Rigid(func(gtx C) D {
					if c.Settings().ActiveArborIdentityID() != nil {
						id, _ := c.Settings().Identity()
						return itemInset.Layout(gtx, sprigTheme.AuthorName(theme, id).Layout)
					}
					return itemInset.Layout(gtx, material.Button(theme, &c.IdentityButton, "Create new Identity").Layout)
				}),
			)
		},
		func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return itemInset.Layout(gtx, material.H6(theme, "Connection").Layout)
				}),
				layout.Rigid(func(gtx C) D {
					return itemInset.Layout(gtx, func(gtx C) D {
						form := sprigTheme.TextForm(c.Theme, &c.ConnectionForm, "Connect", "HOST:PORT")
						return form.Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx C) D {
					return itemInset.Layout(gtx, material.Body2(theme, "You can restart your connection to a relay by hitting the Connect button above without changing the address.").Layout)
				}),
			)
		},
		func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return itemInset.Layout(gtx, material.H6(theme, "Notifications").Layout)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return itemInset.Layout(gtx, material.Switch(theme, &c.NotificationsSwitch).Layout)
						}),
						layout.Rigid(func(gtx C) D {
							return itemInset.Layout(gtx, material.Body1(theme, "Enable notifications").Layout)
						}),
						layout.Rigid(func(gtx C) D {
							return itemInset.Layout(gtx, material.Button(theme, &c.TestNotificationsButton, "Test").Layout)
						}),
						layout.Rigid(func(gtx C) D {
							return itemInset.Layout(gtx, material.Body2(theme, c.TestResults).Layout)
						}),
					)
				}),
				layout.Rigid(func(gtx C) D {
					return itemInset.Layout(gtx, material.Body2(theme, "Currently supported on Android and Linux/BSD. macOS support coming soon.").Layout)
				}),
			)
		},
		func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return itemInset.Layout(gtx, material.H6(theme, "User Interface").Layout)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return itemInset.Layout(gtx, material.Switch(theme, &c.BottomBarSwitch).Layout)
						}),
						layout.Rigid(func(gtx C) D {
							return itemInset.Layout(gtx, material.Body1(theme, "Use bottom app bar").Layout)
						}),
					)
				}),
				layout.Rigid(func(gtx C) D {
					return itemInset.Layout(gtx, material.Body2(theme, "Only recommended on mobile devices.").Layout)
				}),
			)
		},
		func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return itemInset.Layout(gtx, material.H6(theme, "Developer").Layout)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return itemInset.Layout(gtx, material.Switch(theme, &c.ProfilingSwitch).Layout)
						}),
						layout.Rigid(func(gtx C) D {
							return itemInset.Layout(gtx, material.Body1(theme, "Display graphics profiling").Layout)
						}),
					)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return itemInset.Layout(gtx, material.Switch(theme, &c.ThemeingSwitch).Layout)
						}),
						layout.Rigid(func(gtx C) D {
							return itemInset.Layout(gtx, material.Body1(theme, "Display theme editor").Layout)
						}),
					)
				}),
			)
		},
	}
	return c.List.Layout(gtx, len(areas), func(gtx C, index int) D {
		return itemInset.Layout(gtx, func(gtx C) D {
			return areas[index](gtx)
		})
	})
}

func (c *SettingsView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
