package main

import (
	"log"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"git.sr.ht/~whereswaldon/materials"
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigWidget "git.sr.ht/~whereswaldon/sprig/widget"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type SettingsView struct {
	manager ViewManager

	*Settings
	*ArborState
	*sprigTheme.Theme

	layout.List
	ConnectionForm      sprigWidget.TextForm
	IdentityButton      widget.Clickable
	CommunityList       layout.List
	CommunityBoxes      []widget.Bool
	ProfilingSwitch     widget.Bool
	ThemeingSwitch      widget.Bool
	NotificationsSwitch widget.Bool
}

var _ View = &SettingsView{}

func NewCommunityMenuView(settings *Settings, arborState *ArborState, theme *sprigTheme.Theme) View {
	c := &SettingsView{
		Settings:   settings,
		ArborState: arborState,
		Theme:      theme,
	}
	c.List.Axis = layout.Vertical
	c.ConnectionForm.SetText(c.Settings.Address)
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
		c.Settings.Address = c.ConnectionForm.Text()
		settingsChanged = true
		c.ArborState.RestartWorker(c.Settings.Address)
	}
	if c.ConnectionForm.PasteRequested() {
		c.manager.RequestClipboardPaste()
	}
	if c.NotificationsSwitch.Changed() {
		c.NotificationsEnabled = &c.NotificationsSwitch.Value
		settingsChanged = true
	}
	if settingsChanged {
		go c.Settings.Persist()
	}
}

func (c *SettingsView) BecomeVisible() {
	c.ConnectionForm.SetText(c.Settings.Address)
	c.NotificationsSwitch.Value = c.Settings.NotificationsGloballyAllowed()
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
					if c.Settings.ActiveIdentity != nil {
						id, _ := c.Settings.Identity()
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
	/*
		c.CommunityList.Axis = layout.Vertical
		width := gtx.Constraints.Constrain(image.Point{X: gtx.Px(unit.Dp(200))}).X
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			inset := layout.UniformInset(unit.Dp(4))
			gtx.Constraints.Max.X = width
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return inset.Layout(gtx, func(gtx C) D {
						if c.Settings.ActiveIdentity != nil {
							id, _ := c.Settings.Identity()
							return layout.Flex{Alignment: layout.Baseline}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return material.Body1(theme, "Identity:").Layout(gtx)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Left: unit.Dp(12)}.Layout(gtx,
										sprigTheme.AuthorName(theme, id).Layout)
								}),
							)
						} else {
							return material.Button(theme, &c.IdentityButton, "Create new Identity").Layout(gtx)
						}
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = width
					return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Body1(theme, "Known communities:").Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					var dims layout.Dimensions
					c.ArborState.CommunityList.WithCommunities(func(communities []*forest.Community) {
						gtx.Constraints.Max.X = width
						newCommunities := len(communities) - len(c.CommunityBoxes)
						for ; newCommunities > 0; newCommunities-- {
							c.CommunityBoxes = append(c.CommunityBoxes, widget.Bool{})
						}
						dims = c.CommunityList.Layout(gtx, len(communities), func(gtx layout.Context, index int) layout.Dimensions {
							gtx.Constraints.Max.X = width
							community := communities[index]
							// checkbox := &c.CommunityBoxes[index]
							// return layout.Flex{}.Layout(gtx,
							// 	layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							// 		return layout.UniformInset(unit.Dp(8)).Layout(gtx,
							// 			material.CheckBox(theme, checkbox, "").Layout,
							// 		)
							// 	}),
							// layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return inset.Layout(gtx,
								sprigTheme.CommunityName(theme, community).Layout,
							)
							// 	}),
							// )
						})
					})
					return dims
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if runtime.GOOS == "linux" || runtime.GOOS == "android" {
						gtx.Constraints.Max.X = width
						in := layout.UniformInset(unit.Dp(8))
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Max.X = width
							return layout.Flex{}.Layout(gtx,
								layout.Rigid(func(gtx C) D {
									return in.Layout(gtx, material.Body1(theme, "Graphics performance stats:").Layout)
								}),
								layout.Rigid(func(gtx C) D {
									return in.Layout(gtx, material.Switch(theme, &c.ProfilingSwitch).Layout)
								}),
							)
						})
					}
					return D{}
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = width
					in := layout.UniformInset(unit.Dp(8))
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Max.X = width
						return layout.Flex{}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								return in.Layout(gtx, material.Body1(theme, "Edit Theme:").Layout)
							}),
							layout.Rigid(func(gtx C) D {
								return in.Layout(gtx, material.Switch(theme, &c.ThemeingSwitch).Layout)
							}),
						)
					})
				}),
			)
		})
	*/
}

func (c *SettingsView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
