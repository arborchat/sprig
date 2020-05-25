/*
Package ds implements useful data structures for sprig.
*/
package ds

import (
	"fmt"
	"sort"
	"sync"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/forest-go/store"
)

// CommunityList holds a sortable list of communities that can update itself
// automatically by subscribing to a store.ExtendedStore
type CommunityList struct {
	sync.RWMutex
	communities []*forest.Community
}

// New creates a CommunityList and subscribes it to the provided ExtendedStore.
// It will prepopulate the list with the contents of the store as well.
func New(s store.ExtendedStore) (*CommunityList, error) {
	rl := new(CommunityList)
	err := rl.SubscribeTo(s)
	if err != nil {
		return nil, err
	}
	return rl, nil

}

// SubscribeTo makes this CommunityList watch a particular ExtendedStore. You
// shouldn't need to do this often, as the New() function does this for
// you if you construct the CommunityList that way.
func (r *CommunityList) SubscribeTo(s store.ExtendedStore) error {
	s.SubscribeToNewMessages(func(node forest.Node) {
		// cannot block in subscription
		go func() {
			r.Lock()
			defer r.Unlock()
			if community, ok := node.(*forest.Community); ok {
				alreadyInList := false
				for _, element := range r.communities {
					if element.Equals(community) {
						alreadyInList = true
						break
					}
				}
				if !alreadyInList {
					r.communities = append(r.communities, community)
				}
			}
		}()
	})
	const defaultArchiveCommunityListLen = 1024

	// prepopulate the CommunityList
	nodes, err := s.Recent(fields.NodeTypeCommunity, defaultArchiveCommunityListLen)
	if err != nil {
		return fmt.Errorf("Failed loading most recent messages: %w", err)
	}
	for _, n := range nodes {
		if community, ok := n.(*forest.Community); ok {
			r.communities = append(r.communities, community)
		}
	}
	r.Sort()
	return nil
}

func (r *CommunityList) Sort() {
	r.Lock()
	defer r.Unlock()
	sort.SliceStable(r.communities, func(i, j int) bool {
		return r.communities[i].Created < r.communities[j].Created
	})
}

// IndexForID returns the position of the node with the given `id` inside of the CommunityList,
// or -1 if it is not present.
func (r *CommunityList) IndexForID(id *fields.QualifiedHash) int {
	r.RLock()
	defer r.RUnlock()
	for i, n := range r.communities {
		if n.ID().Equals(id) {
			return i
		}
	}
	return -1
}

// WithCommunities executes an arbitrary closure with access to the communities stored
// inside of the CommunitList. The closure must not modify the slice that it is
// given.
func (r *CommunityList) WithCommunities(closure func(communities []*forest.Community)) {
	r.RLock()
	defer r.RUnlock()
	closure(r.communities)
}
