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
	communities []*forest.Community
	nodelist    *NodeList
}

// NewCommunityList creates a CommunityList and subscribes it to the provided ExtendedStore.
// It will prepopulate the list with the contents of the store as well.
func NewCommunityList(s store.ExtendedStore) (*CommunityList, error) {
	cl := new(CommunityList)
	var err error
	var nodes []forest.Node
	cl.nodelist = NewNodeList(func(node forest.Node) bool {
		if _, ok := node.(*forest.Community); ok {
			return true
		}
		return false
	}, func(a, b forest.Node) bool {
		return a.(*forest.Community).Created < b.(*forest.Community).Created
	}, func() []forest.Node {
		nodes, err = s.Recent(fields.NodeTypeCommunity, 1024)
		return nodes
	}, s)
	if err != nil {
		return nil, fmt.Errorf("failed initializing community list: %w", err)
	}
	return cl, nil

}

// IndexForID returns the position of the node with the given `id` inside of the CommunityList,
// or -1 if it is not present.
func (c *CommunityList) IndexForID(id *fields.QualifiedHash) int {
	return c.nodelist.IndexForID(id)
}

// WithCommunities executes an arbitrary closure with access to the communities stored
// inside of the CommunitList. The closure must not modify the slice that it is
// given.
func (c *CommunityList) WithCommunities(closure func(communities []*forest.Community)) {
	c.nodelist.WithNodes(func(nodes []forest.Node) {
		c.communities = c.communities[:0]
		for _, node := range nodes {
			c.communities = append(c.communities, node.(*forest.Community))
		}
		closure(c.communities)
	})
}

// ReplyList holds a sortable list of replies that can update itself
// automatically by subscribing to a store.ExtendedStore
type ReplyList struct {
	replies  []*forest.Reply
	nodelist *NodeList
}

// NewReplyList creates a ReplyList and subscribes it to the provided ExtendedStore.
// It will prepopulate the list with the contents of the store as well.
func NewReplyList(s store.ExtendedStore) (*ReplyList, error) {
	cl := new(ReplyList)
	var err error
	var nodes []forest.Node
	cl.nodelist = NewNodeList(func(node forest.Node) bool {
		if _, ok := node.(*forest.Reply); ok {
			return true
		}
		return false
	}, func(a, b forest.Node) bool {
		return a.(*forest.Reply).Created < b.(*forest.Reply).Created
	}, func() []forest.Node {
		nodes, err = s.Recent(fields.NodeTypeReply, 1024)
		return nodes
	}, s)
	if err != nil {
		return nil, fmt.Errorf("failed initializing reply list: %w", err)
	}
	return cl, nil

}

// IndexForID returns the position of the node with the given `id` inside of the ReplyList,
// or -1 if it is not present.
func (c *ReplyList) IndexForID(id *fields.QualifiedHash) int {
	return c.nodelist.IndexForID(id)
}

// WithReplies executes an arbitrary closure with access to the replies stored
// inside of the ReplList. The closure must not modify the slice that it is
// given.
func (c *ReplyList) WithReplies(closure func(replies []*forest.Reply)) {
	c.nodelist.WithNodes(func(nodes []forest.Node) {
		c.replies = c.replies[:0]
		for _, node := range nodes {
			c.replies = append(c.replies, node.(*forest.Reply))
		}
		closure(c.replies)
	})
}

// NodeList implements a generic data structure for storing ordered lists of forest nodes.
type NodeList struct {
	sync.RWMutex
	nodes    []forest.Node
	filter   func(forest.Node) bool
	sortFunc func(a, b forest.Node) bool
}

type NodeFilter func(forest.Node) bool
type NodeSorter func(a, b forest.Node) bool

// NewNodeList creates a nodelist subscribed to the provided store and initialized with the
// return value of initialize(). The nodes will be sorted using the provided sort function
// (via sort.Slice) and nodes will only be inserted into the list if the filter() function
// returns true for them.
func NewNodeList(filter NodeFilter, sort NodeSorter, initialize func() []forest.Node, s store.ExtendedStore) *NodeList {
	nl := new(NodeList)
	nl.filter = filter
	nl.sortFunc = sort
	nl.withNodesWritable(func() {
		nl.subscribeTo(s)
		for _, node := range initialize() {
			if filter(node) {
				nl.nodes = append(nl.nodes, node)
			}
		}
		nl.sort()
	})
	return nl
}

func (n *NodeList) subscribeTo(s store.ExtendedStore) {
	s.SubscribeToNewMessages(func(node forest.Node) {
		// cannot block in subscription
		go func() {
			n.Lock()
			defer n.Unlock()
			if n.filter(node) {
				alreadyInList := false
				for _, element := range n.nodes {
					if element.Equals(node) {
						alreadyInList = true
						break
					}
				}
				if !alreadyInList {
					n.nodes = append(n.nodes, node)
					n.sort()
				}
			}
		}()
	})
}

// WithNodes executes the provided closure with readonly access to the nodes managed
// by the NodeList. This is the only way to view the nodes, and is thread-safe.
func (n *NodeList) WithNodes(closure func(nodes []forest.Node)) {
	n.RLock()
	defer n.RUnlock()
	closure(n.nodes)
}

func (n *NodeList) withNodesWritable(closure func()) {
	n.Lock()
	defer n.Unlock()
	closure()
}

func (n *NodeList) sort() {
	sort.SliceStable(n.nodes, func(i, j int) bool {
		return n.sortFunc(n.nodes[i], n.nodes[j])
	})
}

// IndexForID returns the position of the node with the given `id` inside of the CommunityList,
// or -1 if it is not present.
func (n *NodeList) IndexForID(id *fields.QualifiedHash) int {
	n.RLock()
	defer n.RUnlock()
	for i, node := range n.nodes {
		if node.ID().Equals(id) {
			return i
		}
	}
	return -1
}
