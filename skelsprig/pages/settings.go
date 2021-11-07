package pages

import (
	"log"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"git.sr.ht/~gioverse/skel/router"
	"git.sr.ht/~gioverse/skel/scheduler"
	"git.sr.ht/~whereswaldon/sprig/skelsprig/settings"
	sprigWidget "git.sr.ht/~whereswaldon/sprig/widget"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

var VersionString = "git"

// Settings is a page for manipulating application settings.
type Settings struct {
	Th                                *sprigTheme.Theme
	Conn                              scheduler.Connection
	Current                           settings.Settings
	initStart, initComplete, updating bool

	widget.List
	ConnectionForm          sprigWidget.TextForm
	IdentityButton          widget.Clickable
	CommunityList           layout.List
	CommunityBoxes          []widget.Bool
	ThemeingSwitch          widget.Bool
	NotificationsSwitch     widget.Bool
	TestNotificationsButton widget.Clickable
	TestResults             string
	DarkModeSwitch          widget.Bool
	UseOrchardStoreSwitch   widget.Bool
}

// Ensure settings is a valid router.Page.
var _ router.Page = (*Settings)(nil)

// NavItem returns a navigation element for this page.
func (s *Settings) NavItem() component.NavItem {
	return component.NavItem{
		Tag:  settingsPage,
		Name: "Settings",
	}
}

// Update the settings page in response to bus events.
func (s *Settings) Update(event interface{}) bool {
	log.Printf("%T %v", event, event)
	switch event := event.(type) {
	case settings.Event:
		s.Current = event.Settings
		s.updating = false
		s.initComplete = true
	case settings.UpdateEvent:
		s.Current = event.Settings
		s.updating = false
		s.initComplete = true
	default:
		return false
	}
	s.updateForm()
	return true
}

// updateForm synchronizes the state of the form fields with the current
// Settings struct.
func (s *Settings) updateForm() {
	s.ConnectionForm.TextField.SetText(s.Current.Address)
	if s.Current.NotificationsEnabled != nil {
		s.NotificationsSwitch.Value = *s.Current.NotificationsEnabled
	}
	s.DarkModeSwitch.Value = s.Current.DarkMode
	s.UseOrchardStoreSwitch.Value = s.Current.OrchardStore
}

// Layout the settings page.
func (s *Settings) Layout(gtx C) D {
	if !s.initStart {
		// Request the current application settings over the bus.
		s.initStart = true
		s.updating = true
		s.Conn.Message(settings.Request{})

		// Perform first-time widget initialization.
		s.List.Axis = layout.Vertical
	} else if !s.initComplete {
		gtx.Queue = nil
	}
	sections := []Section{
		{
			Heading: "Identity",
			Items: []layout.Widget{
				func(gtx C) D {
					id := s.Current.ActiveIdentity
					if id != nil {
						return itemInset.Layout(gtx, sprigTheme.AuthorName(s.Th, string("placeholder"), id, true).Layout)
					}
					return itemInset.Layout(gtx, material.Button(s.Th.Theme, &s.IdentityButton, "Create new Identity").Layout)
				},
			},
		},
		{
			Heading: "Connection",
			Items: []layout.Widget{
				SimpleSectionItem{
					Theme: s.Th.Theme,
					Control: func(gtx C) D {
						return itemInset.Layout(gtx, func(gtx C) D {
							if s.ConnectionForm.Submitted() {
								address := s.ConnectionForm.TextField.Text()
								s.Conn.Message(settings.ConnectRequest{
									Address: address,
								})
							}
							form := sprigTheme.TextForm(s.Th, &s.ConnectionForm, "Connect", "HOST:PORT")
							return form.Layout(gtx)
						})
					},
					Context: "You can restart your connection to a relay by hitting the Connect button above without changing the address.",
				}.Layout,
			},
		},
		{
			Heading: "Notifications",
			Items: []layout.Widget{
				SimpleSectionItem{
					Theme: s.Th.Theme,
					Control: func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								if s.NotificationsSwitch.Changed() {
									s.Conn.Message(settings.NotificationRequest{
										Enabled: s.NotificationsSwitch.Value,
									})
								}
								return itemInset.Layout(gtx, material.Switch(s.Th.Theme, &s.NotificationsSwitch).Layout)
							}),
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Body1(s.Th.Theme, "Enable notifications").Layout)
							}),
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Button(s.Th.Theme, &s.TestNotificationsButton, "Test").Layout)
							}),
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Body2(s.Th.Theme, s.TestResults).Layout)
							}),
						)
					},
					Context: "Currently supported on Android and Linux/BSD. macOS support coming soon.",
				}.Layout,
			},
		},
		{
			Heading: "Store",
			Items: []layout.Widget{
				SimpleSectionItem{
					Theme: s.Th.Theme,
					Control: func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								if s.UseOrchardStoreSwitch.Changed() {
									s.Conn.Message(settings.OrchardRequest{
										Enabled: s.UseOrchardStoreSwitch.Value,
									})
								}
								return itemInset.Layout(gtx, material.Switch(s.Th.Theme, &s.UseOrchardStoreSwitch).Layout)
							}),
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Body1(s.Th.Theme, "Use Orchard store").Layout)
							}),
						)
					},
					Context: "Orchard is a single-file read-oriented database for storing nodes.",
				}.Layout,
			},
		},
		{
			Heading: "User Interface",
			Items: []layout.Widget{
				SimpleSectionItem{
					Theme: s.Th.Theme,
					Control: func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								if s.DarkModeSwitch.Changed() {
									s.Conn.Message(settings.DarkModeRequest{
										Enabled: s.DarkModeSwitch.Value,
									})
								}
								return itemInset.Layout(gtx, material.Switch(s.Th.Theme, &s.DarkModeSwitch).Layout)
							}),
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Body1(s.Th.Theme, "Dark Mode").Layout)
							}),
						)
					},
				}.Layout,
			},
		},
		{
			Heading: "Developer",
			Items: []layout.Widget{
				SimpleSectionItem{
					Theme: s.Th.Theme,
					Control: func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Switch(s.Th.Theme, &s.ThemeingSwitch).Layout)
							}),
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Body1(s.Th.Theme, "Display theme editor").Layout)
							}),
						)
					},
				}.Layout,
				func(gtx C) D {
					return itemInset.Layout(gtx, material.Body1(s.Th.Theme, "version: "+VersionString).Layout)
				},
			},
		},
	}
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx C) D {
			return material.List(s.Th.Theme, &s.List).Layout(gtx, len(sections), func(gtx C, index int) D {
				return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx C) D {
					return component.Surface(s.Th.Theme).Layout(gtx, func(gtx C) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return itemInset.Layout(gtx, func(gtx C) D {
							sections[index].Theme = s.Th.Theme
							return sections[index].Layout(gtx)
						})
					})
				})
			})
		}),
		layout.Expanded(func(gtx C) D {
			if !s.updating {
				return D{}
			}
			return layout.Center.Layout(gtx, material.Loader(s.Th.Theme).Layout)
		}),
	)
}

type Section struct {
	*material.Theme
	Heading string
	Items   []layout.Widget
}

var sectionItemInset = layout.UniformInset(unit.Dp(8))
var itemInset = layout.Inset{
	Left:   unit.Dp(8),
	Right:  unit.Dp(8),
	Top:    unit.Dp(2),
	Bottom: unit.Dp(2),
}

func (s Section) Layout(gtx C) D {
	items := make([]layout.FlexChild, len(s.Items)+1)
	items[0] = layout.Rigid(component.SubheadingDivider(s.Theme, s.Heading).Layout)
	for i := range s.Items {
		items[i+1] = layout.Rigid(s.Items[i])
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, items...)
}

type SimpleSectionItem struct {
	*material.Theme
	Control layout.Widget
	Context string
}

func (s SimpleSectionItem) Layout(gtx C) D {
	return layout.Inset{
		Top:    unit.Dp(4),
		Bottom: unit.Dp(4),
	}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return s.Control(gtx)
			}),
			layout.Rigid(func(gtx C) D {
				if s.Context == "" {
					return D{}
				}
				return itemInset.Layout(gtx, material.Body2(s.Theme, s.Context).Layout)
			}),
		)
	})
}
