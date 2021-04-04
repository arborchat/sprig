package core

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	status "git.sr.ht/~athorp96/forest-ex/active-status"
	"git.sr.ht/~athorp96/forest-ex/expiration"
	"git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/grove"
	"git.sr.ht/~whereswaldon/forest-go/orchard"
	"git.sr.ht/~whereswaldon/forest-go/store"
	"git.sr.ht/~whereswaldon/sprig/ds"
)

// ArborService provides access to stored arbor data.
type ArborService interface {
	Store() store.ExtendedStore
	Communities() *ds.CommunityList
	StartHeartbeat()
}

type arborService struct {
	SettingsService
	grove store.ExtendedStore
	cl    *ds.CommunityList
	done  chan struct{}
}

var _ ArborService = &arborService{}

// newArborService creates a new instance of the Arbor Service using
// the provided Settings within the app to acquire configuration.
func newArborService(settings SettingsService) (ArborService, error) {
	s, err := func() (forest.Store, error) {
		path := settings.DataPath()
		if err := os.MkdirAll(path, 0770); err != nil {
			return nil, fmt.Errorf("preparing data directory for store: %v", err)
		}
		if settings.UseOrchardStore() {
			o, err := orchard.Open(filepath.Join(path, "orchard.db"))
			if err != nil {
				return nil, fmt.Errorf("opening Orchard store: %v", err)
			}
			return o, nil
		}
		g, err := grove.New(path)
		if err != nil {
			return nil, fmt.Errorf("opening Grove store: %v", err)
		}
		g.SetCorruptNodeHandler(func(id string) {
			log.Printf("Grove: corrupt node %s", id)
		})
		return g, nil
	}()
	if err != nil {
		s = store.NewMemoryStore()
	}
	log.Printf("Store: %T\n", s)
	a := &arborService{
		SettingsService: settings,
		grove:           store.NewArchive(s),
		done:            make(chan struct{}),
	}
	cl, err := ds.NewCommunityList(a.grove)
	if err != nil {
		return nil, err
	}
	a.cl = cl
	expiration.ExpiredPurger{
		Logger:        log.New(log.Writer(), "purge ", log.Flags()),
		ExtendedStore: a.grove,
		PurgeInterval: time.Hour,
	}.Start(a.done)
	return a, nil
}

func (a *arborService) Store() store.ExtendedStore {
	return a.grove
}

func (a *arborService) Communities() *ds.CommunityList {
	return a.cl
}

func (a *arborService) StartHeartbeat() {
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
}
