package core

import (
	"fmt"
	"log"
	"os"

	"git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/grove"
	"git.sr.ht/~whereswaldon/forest-go/store"
	"git.sr.ht/~whereswaldon/sprig/ds"
)

// ArborService provides access to stored arbor data.
type ArborService interface {
	Store() store.ExtendedStore
	Communities() *ds.CommunityList
	Replies() *ds.ReplyList
}

type arborService struct {
	grove store.ExtendedStore
	rl    *ds.ReplyList
	cl    *ds.CommunityList
}

var _ ArborService = &arborService{}

// newArborService creates a new instance of the Arbor Service using
// the provided Settings within the app to acquire configuration.
func newArborService(app App) (ArborService, error) {
	baseStore := func() (s forest.Store) {
		defer func() {
			if s == nil {
				log.Printf("falling back to in-memory storage")
				s = store.NewMemoryStore()
			}
		}()
		var (
			err       error
			grovePath string = app.Settings().GrovePath()
		)
		if err := os.MkdirAll(grovePath, 0770); err == nil {
			log.Printf("unable to create directory for grove: %v", err)
			return
		}
		s, err = grove.New(grovePath)
		if err != nil {
			log.Printf("Failed creating grove: %v", err)
		}
		return
	}()
	a := &arborService{
		grove: store.NewArchive(baseStore),
	}
	rl, err := ds.NewReplyList(a.grove)
	if err != nil {
		return nil, fmt.Errorf("failed initializing reply list: %w", err)
	}
	a.rl = rl
	cl, err := ds.NewCommunityList(a.grove)
	if err != nil {
		return nil, fmt.Errorf("failed initializing community list: %w", err)
	}
	a.cl = cl
	return a, nil
}

func (a *arborService) Store() store.ExtendedStore {
	return a.grove
}

func (a *arborService) Communities() *ds.CommunityList {
	return a.cl
}

func (a *arborService) Replies() *ds.ReplyList {
	return a.rl
}
