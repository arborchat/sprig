package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/widget/material"
	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/grove"
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
	dataDir, err := app.DataDir()
	if err != nil {
		log.Printf("failed finding application data dir: %v", err)
	}

	appState, err := NewAppState(dataDir)
	if err != nil {
		return err
	}
	if *address != "" {
		appState.Settings.Address = *address
	}

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
			layout.Inset{
				Bottom: event.Insets.Bottom,
				Top:    event.Insets.Top,
			}.Layout(gtx, func() {
				viewManager.Layout(gtx)
			})
			event.Frame(gtx.Ops)
		}
	}
}

type AppState struct {
	Settings
	ArborState
	*material.Theme
	DataDir string
}

func NewAppState(dataDir string) (*AppState, error) {
	dataDir = filepath.Join(dataDir, "sprig")
	var baseStore forest.Store
	var err error
	if err = os.MkdirAll(dataDir, 0770); err != nil {
		log.Printf("couldn't create app data dir: %v", err)
	}
	if dataDir != "" {
		grovePath := filepath.Join(dataDir, "grove")
		if err := os.MkdirAll(grovePath, 0770); err != nil {
			log.Printf("unable to create directory for grove: %v", err)
		}
		baseStore, err = grove.New(grovePath)
		if err != nil {
			log.Printf("unable to create grove (falling back to in-memory): %v", err)
		}
	}
	if baseStore == nil {
		baseStore = store.NewMemoryStore()
	}
	archive := store.NewArchive(baseStore)
	rl, err := replylist.New(archive)
	if err != nil {
		return nil, fmt.Errorf("failed to construct replylist: %w", err)
	}
	appState := &AppState{
		ArborState: ArborState{
			SubscribableStore: archive,
			ReplyList:         rl,
		},
		DataDir: dataDir,
		Theme:   material.NewTheme(),
	}
	jsonSettings, err := ioutil.ReadFile(SettingsFile(dataDir))
	if err != nil {
		log.Printf("failed to load settings: %v", err)
	} else {
		if err = json.Unmarshal(jsonSettings, &appState.Settings); err != nil {
			log.Printf("couldn't parse json settings: %v", err)
		}
	}
	appState.Settings.savePath = SettingsFile(dataDir)
	return appState, nil
}

func SettingsFile(dataDir string) string {
	return filepath.Join(dataDir, "settings.json")
}

type Settings struct {
	Address string

	savePath string
}

func (s Settings) Persist() {
	data, err := json.MarshalIndent(&s, "", "  ")
	if err != nil {
		log.Printf("couldn't marshal settings as json: %v", err)
	}
	err = ioutil.WriteFile(s.savePath, data, 0770)
	if err != nil {
		log.Printf("couldn't save settings file: %v", err)
	}
}

type ViewID int

const (
	ConnectForm ViewID = iota
	CommunityMenu
	ReplyView
)
