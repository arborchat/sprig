package arbor

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"git.sr.ht/~athorp96/forest-ex/expiration"
	"git.sr.ht/~gioverse/skel/scheduler"
	"git.sr.ht/~whereswaldon/forest-go/orchard"
	"git.sr.ht/~whereswaldon/forest-go/store"
	"git.sr.ht/~whereswaldon/sprig/ds"
	"git.sr.ht/~whereswaldon/sprig/skelsprig/settings"
)

// Service manages the storage for arbor chat history.
type Service struct {
	conn      scheduler.Connection
	nodeStore store.ExtendedStore
	cl        *ds.CommunityList
	done      chan struct{}
	current   settings.DataDirs
}

// New creates a new instance of the Arbor Service using
// the provided Settings within the app to acquire configuration.
func New(bus scheduler.Connection) (*Service, error) {
	s := &Service{
		conn: bus,
		done: make(chan struct{}),
	}
	go s.run()
	return s, nil
}

// initialize loads the chat data from disk.
func (s *Service) initialize() error {
	if err := os.MkdirAll(filepath.Dir(s.current.OrchardPath), 0770); err != nil {
		return fmt.Errorf("preparing data directory for store: %v", err)
	}
	o, err := orchard.Open(s.current.OrchardPath)
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
	log.Println("Arbor service initialized")
	return nil
}

// InitializeRequest asks the arbor service to initialize. This allows for
// delayed initialization when a storage migration must occur before the
// service starts.
type InitializeRequest struct{}

// Request can be sent over the bus to ask for an Event to be sent over the bus.
type Request struct{}

// Event provides a handle to the service over the bus. This allows other
// parts of the application to invoke methods on the service.
type Event struct {
	*Service
}

func (s *Service) run() {
	var (
		canInit, initRequested, requested, initialized bool
	)
	s.conn.Message(settings.Request{})
	for event := range s.conn.Output() {
		switch event := event.(type) {
		case settings.Event:
			s.current = event.Dirs
			canInit = true
		case InitializeRequest:
			initRequested = true
		case Request:
			requested = true
		}
		if canInit && initRequested && !initialized {
			if err := s.initialize(); err != nil {
				log.Printf("failed initializing arbor service: %v", err)
			} else {
				initialized = true
			}
		}
		if requested && initialized {
			requested = false
			s.conn.Message(Event{s})
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
