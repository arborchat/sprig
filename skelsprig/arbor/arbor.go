package arbor

import (
	"fmt"
	"log"
	"os"
	"time"

	"git.sr.ht/~athorp96/forest-ex/expiration"
	"git.sr.ht/~gioverse/skel/scheduler"
	"git.sr.ht/~whereswaldon/forest-go/orchard"
	"git.sr.ht/~whereswaldon/forest-go/store"
	"git.sr.ht/~whereswaldon/sprig/ds"
	"git.sr.ht/~whereswaldon/sprig/skelsprig/settings"
)

type Service struct {
	conn            scheduler.Connection
	nodeStore       store.ExtendedStore
	cl              *ds.CommunityList
	done            chan struct{}
	init, requested bool
	current         settings.DataDirs
}

// New creates a new instance of the Arbor Service using
// the provided Settings within the app to acquire configuration.
func New(bus scheduler.Connection) (*Service, error) {
	s := &Service{
		done: make(chan struct{}),
	}
	go s.run()
	return s, nil
}

func (s *Service) initialize() error {
	if err := os.MkdirAll(s.current.GrovePath, 0770); err != nil {
		return fmt.Errorf("preparing data directory for store: %v", err)
	}
	o, err := orchard.Open(s.current.GrovePath)
	if err != nil {
		return fmt.Errorf("opening Orchard store: %v", err)
	}
	s.nodeStore = store.NewArchive(o)
	cl, err := ds.NewCommunityList(s.nodeStore)
	if err != nil {
		return err
	}
	s.cl = cl
	expiration.ExpiredPurger{
		Logger:        log.New(log.Writer(), "purge ", log.Flags()),
		ExtendedStore: s.nodeStore,
		PurgeInterval: time.Hour,
	}.Start(s.done)
	return nil
}

type InitializeRequest struct{}

type Request struct{}

type Event struct {
	*Service
}

func (s *Service) run() {
	for event := range s.conn.Output() {
		switch event := event.(type) {
		case settings.Event:
			s.current = event.Dirs
		case InitializeRequest:
			// TODO: handle the case in which we don't yet have settings.
			if !s.init {
				s.initialize()
				if s.requested {
					s.conn.Message(Event{s})
				}
			}
		case Request:
			if !s.init {
				s.requested = true
			} else {
				s.conn.Message(Event{s})
			}
		}
	}
}

func (a *Service) Store() store.ExtendedStore {
	return a.nodeStore
}

func (a *Service) Communities() *ds.CommunityList {
	return a.cl
}

func (a *Service) StartHeartbeat() {
	// TODO
	/*
		a.Communities().WithCommunities(func(c []*forest.Community) {
			if a.SettingsService.ActiveArborIdentityID() != nil {
				builder, err := a.SettingsService.Builder()
				if err == nil {
					log.Printf("Begining active-status heartbeat")
					go status.StartActivityHeartBeat(a.Store(), c, builder, time.Minute*5)
				} else {
					log.Printf("Could not acquire builder: %v", err)
				}
			}
		})
	*/
}
