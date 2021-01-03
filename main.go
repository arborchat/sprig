package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	status "git.sr.ht/~athorp96/forest-ex/active-status"
	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/sprig/core"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	go func() {
		w := app.NewWindow(app.Title("Sprig"))
		if err := eventLoop(w); err != nil {
			log.Fatalf("exiting due to error: %v", err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func eventLoop(w *app.Window) error {
	dataDir, err := app.DataDir()
	if err != nil {
		log.Printf("failed finding application data dir: %v", err)
	}
	dataDir = filepath.Join(dataDir, "sprig")
	profile := flag.Bool("profile", false, "log profiling data")
	invalidate := flag.Bool("invalidate", false, "invalidate every single frame, only useful for profiling")
	flag.StringVar(&dataDir, "data-dir", dataDir, "application state directory")
	flag.Parse()

	app, err := core.NewApp(w, dataDir)
	if err != nil {
		log.Fatalf("Failed initializing application: %v", err)
	}

	go func() {
		// Start active-status heartbeat
		app.Arbor().Communities().WithCommunities(func(c []*forest.Community) {
			if app.Settings().ActiveArborIdentityID() != nil {
				builder, err := app.Settings().Builder()
				if err == nil {
					log.Printf("Begining active-status heartbeat")
					go status.StartActivityHeartBeat(app.Arbor().Store(), c, builder, time.Minute*5)
				} else {
					log.Printf("Could not acquire builder: %v", err)
				}
			}
		})
		app.Arbor().Store().SubscribeToNewMessages(func(n forest.Node) {
			w.Invalidate()
		})
	}()

	// handle ctrl+c to shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	// this function should perform any and all cleanup work, and it must block
	// for the necessary duration of that work.
	shutdown := func() {
		log.Printf("cleaning up")
		connections := app.Sprout().Connections()
		for _, conn := range connections {
			if worker := app.Sprout().WorkerFor(conn); worker != nil {
				var nodes []forest.Node

				app.Arbor().Communities().WithCommunities(func(coms []*forest.Community) {
					if app.Settings().ActiveArborIdentityID() != nil {
						builder, err := app.Settings().Builder()
						if err == nil {
							log.Printf("Killing active-status heartbeat")
							for _, c := range coms {
								n, err := status.NewActivityNode(c, builder, status.Inactive, time.Minute*5)
								if err != nil {
									log.Printf("Error creating inactive node: %v", err)
									continue
								}
								log.Printf("Sending offline node to community %s", c.ID())
								nodes = append(nodes, n)
							}
						} else {
							log.Printf("Could not acquire builder: %v", err)
						}
					}
				})

				if err := worker.SendAnnounce(nodes, time.NewTicker(time.Second*5).C); err != nil {
					log.Printf("error sending shutdown messages: %v", err)
				}
			}
		}
		log.Printf("shutting down")
	}

	viewManager := NewViewManager(w, app, *profile)
	viewManager.ApplySettings(app.Settings())
	viewManager.RegisterView(ReplyViewID, NewReplyListView(app))
	viewManager.RegisterView(ConnectFormID, NewConnectFormView(app))
	viewManager.RegisterView(SettingsID, NewCommunityMenuView(app))
	viewManager.RegisterView(IdentityFormID, NewIdentityFormView(app))
	viewManager.RegisterView(ConsentViewID, NewConsentView(app))

	if app.Settings().AcknowledgedNoticeVersion() < NoticeVersion {
		viewManager.RequestViewSwitch(ConsentViewID)
	} else if app.Settings().Address() == "" {
		viewManager.RequestViewSwitch(ConnectFormID)
	} else if app.Settings().ActiveArborIdentityID() == nil {
		viewManager.RequestViewSwitch(IdentityFormID)
	} else {
		viewManager.RequestViewSwitch(ReplyViewID)
	}

	var ops op.Ops
	for {
		select {
		case <-sigs:
			shutdown()
			return nil
		case event := (<-w.Events()):
			switch event := event.(type) {
			case system.DestroyEvent:
				shutdown()
				return event.Err
			case *system.CommandEvent:
				if event.Type == system.CommandBack {
					viewManager.HandleBackNavigation(event)
				}
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, event)
				if *invalidate {
					op.InvalidateOp{}.Add(gtx.Ops)
				}
				th := app.Theme().Current()
				layout.Stack{}.Layout(gtx,
					layout.Expanded(func(gtx C) D {
						return sprigTheme.Rect{
							Color: th.Background.Dark.Bg,
							Size: f32.Point{
								X: float32(gtx.Constraints.Max.X),
								Y: float32(gtx.Constraints.Max.Y),
							},
						}.Layout(gtx)
					}),
					layout.Stacked(func(gtx C) D {
						return layout.Inset{
							Bottom: event.Insets.Bottom,
							Left:   event.Insets.Left,
							Right:  event.Insets.Right,
							Top:    event.Insets.Top,
						}.Layout(gtx, func(gtx C) D {
							return layout.Stack{}.Layout(gtx,
								layout.Expanded(func(gtx C) D {
									return sprigTheme.Rect{
										Color: th.Background.Default.Bg,
										Size: f32.Point{
											X: float32(gtx.Constraints.Max.X),
											Y: float32(gtx.Constraints.Max.Y),
										},
									}.Layout(gtx)
								}),
								layout.Stacked(viewManager.Layout),
							)
						})
					}),
				)
				event.Frame(gtx.Ops)
			default:
				ProcessPlatformEvent(app, event)
			}
		}
	}
}

type ViewID int

const (
	ConnectFormID ViewID = iota
	IdentityFormID
	SettingsID
	ReplyViewID
	ConsentViewID
)
