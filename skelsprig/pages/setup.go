package pages

import (
	"log"
	"os"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"git.sr.ht/~gioverse/skel/router"
	"git.sr.ht/~gioverse/skel/scheduler"
	"git.sr.ht/~whereswaldon/forest-go/grove"
	"git.sr.ht/~whereswaldon/forest-go/orchard"
	"git.sr.ht/~whereswaldon/sprig/skelsprig/arbor"
	"git.sr.ht/~whereswaldon/sprig/skelsprig/settings"
	sprigWidget "git.sr.ht/~whereswaldon/sprig/widget"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

// Setup is a page for manipulating application settings.
type Setup struct {
	Th                                                *sprigTheme.Theme
	Conn                                              scheduler.Connection
	Phase                                             SetupPhase
	Current                                           settings.Settings
	AgreeButton                                       widget.Clickable
	requestedSettings, gotSettings                    bool
	startedStorageMigration, finishedStorageMigration bool
	notifiedComplete                                  bool
	dataDirs                                          settings.DataDirs
	username                                          component.TextField
	createIdButton                                    widget.Clickable
	connectForm                                       sprigWidget.TextForm
	connectionList                                    widget.List
}

type SetupPhase uint8

const (
	Starting SetupPhase = iota
	DisplayingNotice
	UpdatingStorage
	CreatingIdentity
	GettingRelayAddress
	SubscribingToCommunity
	Complete
)

// Ensure setup is a valid router.Page.
var _ router.Page = (*Setup)(nil)

// SetupCompleteEvent indicates that the setup page is finished
// and the router should switch to displaying another page.
type SetupCompleteEvent struct{}

// StandalonePage implements the StandalonePage interface, and
// ensures that this page is presented without an app bar or
// navigation drawer.
func (s *Setup) StandalonePage() {}

// Update the settings page in response to bus events.
func (s *Setup) Update(event interface{}) bool {
	switch event := event.(type) {
	case settings.UpdateEvent:
		s.Current = event.Settings
		s.dataDirs = event.Dirs
		s.gotSettings = true
	case settings.Event:
		s.Current = event.Settings
		s.dataDirs = event.Dirs
		s.gotSettings = true
	case StorageMigrationCompleteEvent:
		s.finishedStorageMigration = true
		s.Conn.Message(arbor.InitializeRequest{})
	default:
		return false
	}
	return true
}

func (s *Setup) loader(gtx C) D {
	return layout.Center.Layout(gtx, material.Loader(s.Th.Theme).Layout)
}

type StorageMigrationCompleteEvent struct {
	Err error
}

// migrateStorage dumps the contents of the local grove into the
// local orchard, then renames the grove. If the grove has been
// renamed away from the default name, it does nothing.
func migrateStorage(paths settings.DataDirs) interface{} {
	_, err := os.Stat(paths.GrovePath)
	if err != nil {
		// There is no grove, so no need to migrate.
		return StorageMigrationCompleteEvent{}
	}
	log.Println("Opening grove")
	g, err := grove.New(paths.GrovePath)
	if err != nil {
		// Failed constructing the grove. Leave it alone.
		log.Printf("Failed constructing grove: %v", err)
		return StorageMigrationCompleteEvent{err}
	}
	log.Println("Opening orchard")
	o, err := orchard.Open(paths.OrchardPath)
	if err != nil {
		log.Printf("Failed constructing orchard: %v", err)
		return StorageMigrationCompleteEvent{err}
	}

	log.Println("Copying data from grove into orchard")
	if err = g.CopyInto(o); err != nil {
		log.Printf("Failed migrating grove data to orchard: %v", err)
		// Do not return here. We still need to close the orchard.
	}
	log.Println("Finished copying data from grove into orchard")

	if err = o.Close(); err != nil {
		log.Printf("Failed closing orchard: %v", err)
		return StorageMigrationCompleteEvent{err}
	}
	log.Println("Closed orchard")

	if err = os.Rename(paths.GrovePath, paths.GrovePath+"-old"); err != nil {
		log.Printf("Failed renaming grove: %v", err)
		return StorageMigrationCompleteEvent{err}
	}
	log.Println("Renamed grove")

	return StorageMigrationCompleteEvent{}
}

// Layout the setup page.
func (s *Setup) Layout(gtx C) D {
	for {
		switch s.Phase {
		case Starting:
			if !s.requestedSettings {
				s.Conn.Message(settings.Request{})
				s.requestedSettings = true
				return s.loader(gtx)
			} else if !s.gotSettings {
				return s.loader(gtx)
			} else {
				s.Phase = DisplayingNotice
			}
		case DisplayingNotice:
			if s.Current.AcknowledgedNoticeVersion < NoticeVersion {
				return s.displayNotice(gtx)
			} else {
				s.Phase = UpdatingStorage
			}
		case UpdatingStorage:
			if !s.startedStorageMigration {
				s.startedStorageMigration = true
				// Copy the data dirs into a local variable so that
				// the scheduled work doesn't race against updates
				// to the page's copy.
				dataDirEvent := s.dataDirs
				s.Conn.Schedule(func() interface{} {
					return migrateStorage(dataDirEvent)
				})
				return s.loader(gtx)
			} else if !s.finishedStorageMigration {
				return s.loader(gtx)
			} else {
				s.Phase = CreatingIdentity
			}
		case CreatingIdentity:
			if s.Current.ActiveIdentity == nil {
				return s.layoutCreateIdentityForm(gtx)
			} else {
				s.Phase = GettingRelayAddress
			}
		case GettingRelayAddress:
			if s.Current.Address == "" {
				return s.layoutConnectForm(gtx)
			} else {
				s.Phase = SubscribingToCommunity
			}
		case SubscribingToCommunity:
			if len(s.Current.Subscriptions) == 0 {
			} else {
				s.Phase = Complete
			}
		case Complete:
			if !s.notifiedComplete {
				s.Conn.MessageLocal(SetupCompleteEvent{})
				s.notifiedComplete = true
			}
			return s.loader(gtx)
		}
	}
}

const (
	UpdateText    = "You are seeing this message because the notice text has changed since you last accepted it."
	Notice        = "This is a chat client for the Arbor Chat Project. Before you send a message, you should know that your messages cannot be edited or deleted once sent, and that they will be publically visible to all other Arbor users."
	NoticeVersion = 1
)

func (s *Setup) displayNotice(gtx C) D {
	if s.AgreeButton.Clicked() {
		s.Conn.Message(settings.AcknowledgedNoticeRequest{
			Version: NoticeVersion,
		})
	}
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.H2(s.Th.Theme, "Notice").Layout,
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.Body1(s.Th.Theme, Notice).Layout,
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if s.Current.AcknowledgedNoticeVersion != 0 {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(4)).Layout(gtx,
							material.Body2(s.Th.Theme, UpdateText).Layout,
						)
					})
				}
				return layout.Dimensions{}
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.Button(s.Th.Theme, &s.AgreeButton, "I Understand And Agree").Layout,
					)
				})
			}),
		)
	})
}

func (s *Setup) layoutCreateIdentityForm(gtx C) D {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.Body1(s.Th.Theme, "Your Arbor Username:").Layout,
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx C) D {
						return s.username.Layout(gtx, s.Th.Theme, "Username")
					})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.Body2(s.Th.Theme, "Your username is public, and cannot currently be changed once it is chosen.").Layout,
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.Button(s.Th.Theme, &s.createIdButton, "Create").Layout,
					)
				})
			}),
		)
	})
}

func (s *Setup) layoutConnectForm(gtx C) D {
	inset := layout.UniformInset(unit.Dp(8))
	return inset.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return inset.Layout(gtx,
					material.H6(s.Th.Theme, "Arbor Relay Address:").Layout,
				)
			}),
			layout.Rigid(func(gtx C) D {
				return inset.Layout(gtx, sprigTheme.TextForm(s.Th, &s.connectForm, "Connect", "HOST:PORT").Layout)
			}),
		)
	})
}

func (s *Setup) layoutSubscriptionForm(gtx C) D {
	inset := layout.UniformInset(unit.Dp(12))

	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return inset.Layout(gtx, func(gtx C) D {
				return material.Body1(s.Th.Theme, "Subscribe to a few communities to get started:").Layout(gtx)
			})
		}),
		layout.Flexed(1.0, func(gtx C) D {
			return material.List(s.Th.Theme, &s.connectionList).Layout(gtx, 1, func(gtx C, index int) D {
				return D{}
			})
		}),
	)
}