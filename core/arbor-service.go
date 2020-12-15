package core

import (
	"log"
	"os"
	"time"

	"git.sr.ht/~athorp96/forest-ex/expiration"
	"git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/grove"
	"git.sr.ht/~whereswaldon/forest-go/store"
	"git.sr.ht/~whereswaldon/sprig/ds"
)

// ArborService provides access to stored arbor data.
type ArborService interface {
	Store() store.ExtendedStore
	Communities() *ds.CommunityList
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
	baseStore := func() (s forest.Store) {
		defer func() {
			if s == nil {
				log.Printf("falling back to in-memory storage")
				s = store.NewMemoryStore()
			}
		}()
		var (
			err       error
			grovePath string = settings.GrovePath()
		)
		if err := os.MkdirAll(grovePath, 0770); err != nil {
			log.Printf("unable to create directory for grove: %v", err)
			return
		}
		g, err := grove.New(grovePath)
		if err != nil {
			log.Printf("Failed creating grove: %v", err)
		}
		g.SetCorruptNodeHandler(func(id string) {
			log.Printf("Grove reported corrupt node %s", id)
		})
		return g
	}()
	a := &arborService{
		SettingsService: settings,
		grove:           store.NewArchive(baseStore),
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
