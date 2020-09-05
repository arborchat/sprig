package main

import (
	"flag"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/sprig/core"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

func main() {
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
	profile := flag.Bool("profile", false, "log profiling data")
	flag.Parse()
	dataDir, err := app.DataDir()
	if err != nil {
		log.Printf("failed finding application data dir: %v", err)
	}

	app, err := core.NewApp(dataDir)
	if err != nil {
		log.Fatalf("Failed initializing application: %v", err)
	}
	app.Notifications().Register(app.Arbor().Store())

	theme := sprigTheme.New()

	viewManager := NewViewManager(w, theme, *profile)
	viewManager.ApplySettings(app.Settings())
	viewManager.RegisterView(ReplyViewID, NewReplyListView(app, theme))
	viewManager.RegisterView(ConnectFormID, NewConnectFormView(app, theme))
	viewManager.RegisterView(SettingsID, NewCommunityMenuView(app, theme))
	viewManager.RegisterView(IdentityFormID, NewIdentityFormView(app, theme))
	viewManager.RegisterView(ConsentViewID, NewConsentView(app, theme))
	if app.Settings().AcknowledgedNoticeVersion() < NoticeVersion {
		viewManager.RequestViewSwitch(ConsentViewID)
	} else if app.Settings().Address() == "" {
		viewManager.RequestViewSwitch(ConnectFormID)
	} else if app.Settings().ActiveArborIdentityID() == nil {
		viewManager.RequestViewSwitch(IdentityFormID)
	} else {
		viewManager.RequestViewSwitch(ReplyViewID)
	}

	app.Arbor().Store().SubscribeToNewMessages(func(n forest.Node) {
		w.Invalidate()
	})
	var ops op.Ops
	for {
		switch event := (<-w.Events()).(type) {
		case system.DestroyEvent:
			return event.Err
		case system.ClipboardEvent:
			viewManager.HandleClipboard(event.Text)
		case *system.CommandEvent:
			if event.Type == system.CommandBack {
				viewManager.HandleBackNavigation(event)
			}
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, event)
			layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx C) D {
					return sprigTheme.DrawRect(gtx, theme.Background.Dark, f32.Pt(float32(gtx.Constraints.Max.X), float32(gtx.Constraints.Max.Y)), 0)
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
								return sprigTheme.DrawRect(gtx, theme.Background.Default, f32.Pt(float32(gtx.Constraints.Max.X), float32(gtx.Constraints.Max.Y)), 0)
							}),
							layout.Stacked(viewManager.Layout),
						)
					})
				}),
			)
			event.Frame(gtx.Ops)
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
