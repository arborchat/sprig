package main

import (
	"flag"
	"fmt"
	"log"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/widget/material"
	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/store"
	"git.sr.ht/~whereswaldon/wisteria/replylist"
)

func main() {
	gofont.Register()
	go func() {
		w := app.NewWindow()
		if err := eventLoop(w); err != nil {
			log.Println(err)
			return
		}
	}()
	app.Main()
}

func eventLoop(w *app.Window) error {
	address := flag.String("address", "", "arbor relay address to connect to")
	flag.Parse()
	appState, err := NewAppState()
	if err != nil {
		return err
	}
	appState.Settings.Address = *address

	viewManager := NewViewManager()
	viewManager.RegisterView(ConnectForm, NewConnectFormView(&appState.Settings, &appState.ArborState, appState.Theme))
	viewManager.RegisterView(CommunityMenu, NewCommunityMenuView(&appState.Settings, &appState.ArborState, appState.Theme))
	viewManager.RegisterView(ReplyView, NewReplyListView(&appState.Settings, &appState.ArborState, appState.Theme))
	viewManager.RequestViewSwitch(ConnectForm)

	appState.SubscribableStore.SubscribeToNewMessages(func(n forest.Node) {
		w.Invalidate()
	})
	gtx := new(layout.Context)
	for {
		switch event := (<-w.Events()).(type) {
		case system.DestroyEvent:
			return event.Err
		case system.FrameEvent:
			gtx.Reset(event.Queue, event.Config, event.Size)
			viewManager.Layout(gtx)
			event.Frame(gtx.Ops)
		}
	}
}

type AppState struct {
	Settings
	ArborState
	*material.Theme
}

func NewAppState() (*AppState, error) {
	archive := store.NewArchive(store.NewMemoryStore())
	rl, err := replylist.New(archive)
	if err != nil {
		return nil, fmt.Errorf("failed to construct replylist: %w", err)
	}
	return &AppState{
		ArborState: ArborState{
			SubscribableStore: archive,
			ReplyList:         rl,
		},
		Theme: material.NewTheme(),
	}, nil
}

type Settings struct {
	Address string
}

type ViewID int

const (
	ConnectForm ViewID = iota
	CommunityMenu
	ReplyView
)
