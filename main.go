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
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/forest-go/grove"
	"git.sr.ht/~whereswaldon/forest-go/store"
	"git.sr.ht/~whereswaldon/wisteria/replylist"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
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
	viewManager.RegisterView(IdentityForm, NewIdentityFormView(&appState.Settings, &appState.ArborState, appState.Theme))
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
	jsonSettings, err := ioutil.ReadFile(appState.Settings.SettingsFile())
	if err != nil {
		log.Printf("failed to load settings: %v", err)
	} else {
		if err = json.Unmarshal(jsonSettings, &appState.Settings); err != nil {
			log.Printf("couldn't parse json settings: %v", err)
		}
	}
	appState.Settings.dataDir = dataDir
	return appState, nil
}

func (s Settings) SettingsFile() string {
	return filepath.Join(s.dataDir, "settings.json")
}

func (s Settings) KeysDir() string {
	return filepath.Join(s.dataDir, "keys")
}

func (s Settings) IdentitiesDir() string {
	return filepath.Join(s.dataDir, "identities")
}

type Settings struct {
	Address        string
	ActiveIdentity *fields.QualifiedHash

	dataDir string

	// state used for authoring messages
	activePrivKey *openpgp.Entity
}

func (s *Settings) CreateIdentity(name string) {
	keysDir := s.KeysDir()
	if err := os.MkdirAll(keysDir, 0770); err != nil {
		log.Printf("failed creating key storage directory: %v", err)
		return
	}
	keypair, err := openpgp.NewEntity(name, "sprig-generated arbor identity", "", &packet.Config{})
	if err != nil {
		log.Printf("failed generating new keypair: %v", err)
		return
	}
	signer, err := forest.NewNativeSigner(keypair)
	if err != nil {
		log.Printf("failed wrapping keypair into Signer: %v", err)
		return
	}
	identity, err := forest.NewIdentity(signer, name, []byte{})
	if err != nil {
		log.Printf("failed generating arbor identity from signer: %v", err)
		return
	}
	id := identity.ID()

	keyFilePath := filepath.Join(keysDir, id.String())
	keyFile, err := os.OpenFile(keyFilePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0660)
	if err != nil {
		log.Printf("failed creating key file: %v", err)
		return
	}
	defer func() {
		if err := keyFile.Close(); err != nil {
			log.Printf("failed closing key file: %v", err)
		}
	}()
	if err := keypair.SerializePrivateWithoutSigning(keyFile, nil); err != nil {
		log.Printf("failed saving private key: %v", err)
		return
	}

	idsDir := s.IdentitiesDir()
	if err := os.MkdirAll(idsDir, 0770); err != nil {
		log.Printf("failed creating identity storage directory: %v", err)
		return
	}
	idFilePath := filepath.Join(idsDir, id.String())

	idFile, err := os.OpenFile(idFilePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0660)
	if err != nil {
		log.Printf("failed creating identity file: %v", err)
		return
	}
	defer func() {
		if err := idFile.Close(); err != nil {
			log.Printf("failed closing identity file: %v", err)
		}
	}()
	binIdent, err := identity.MarshalBinary()
	if err != nil {
		log.Printf("failed serializing new identity: %v", err)
		return
	}
	if _, err := idFile.Write(binIdent); err != nil {
		log.Printf("failed writing identity: %v", err)
		return
	}

	s.ActiveIdentity = id
	s.activePrivKey = keypair
}

func (s Settings) Persist() {
	data, err := json.MarshalIndent(&s, "", "  ")
	if err != nil {
		log.Printf("couldn't marshal settings as json: %v", err)
	}
	err = ioutil.WriteFile(s.SettingsFile(), data, 0770)
	if err != nil {
		log.Printf("couldn't save settings file: %v", err)
	}
}

type ViewID int

const (
	ConnectForm ViewID = iota
	IdentityForm
	CommunityMenu
	ReplyView
)
