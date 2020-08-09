/*
Package ds implements useful data structures for sprig.
*/
package ds

import (
	"fmt"
	"log"
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
	cl.nodelist = NewNodeList(func(node forest.Node) forest.Node {
		if _, ok := node.(*forest.Community); ok {
			return node
		}
		return nil
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

// ReplyData holds the contents of a single reply and the major nodes that
// it references.
type ReplyData struct {
	*forest.Reply
	Community *forest.Community
	Author    *forest.Identity
}

// populate populates the the fields of a ReplyData object from a given node and a store.
// It can be used on an unfilled ReplyData instance in place of a constructor. It returns
// false if the node cannot be processed into ReplyData
func (r *ReplyData) Populate(reply forest.Node, store store.ExtendedStore) bool {
	asReply, ok := reply.(*forest.Reply)
	if !ok {
		return false
	}
	r.Reply = asReply
	comm, has, err := store.GetCommunity(&asReply.CommunityID)

	if err != nil || !has {
		return false
	}
	r.Community = comm.(*forest.Community)
	author, has, err := store.GetIdentity(&asReply.Author)
	if err != nil || !has {
		return false
	}
	r.Author = author.(*forest.Identity)

	// Verify twig data parses and node is not invisible
	if md, err := asReply.TwigMetadata(); err != nil {
		// Malformed metadata
		log.Printf("Error when fetching twig metadata: %v", err)
		log.Printf("Twig metadata: %v", asReply.Metadata.Blob)
		return false
	} else if md.Contains("invisible", 1) {
		// Invisible message
		log.Printf("Invisible node found. Not populating as a reply")
		return false
	}

	return true
}

// NodeList implements a generic data structure for storing ordered lists of forest nodes.
type NodeList struct {
	sync.RWMutex
	nodes    []forest.Node
	filter   NodeFilter
	sortFunc NodeSorter
}

type NodeFilter func(forest.Node) forest.Node
type NodeSorter func(a, b forest.Node) bool

// NewNodeList creates a nodelist subscribed to the provided store and initialized with the
// return value of initialize(). The nodes will be sorted using the provided sort function
// (via sort.Slice) and nodes will only be inserted into the list if the filter() function
// returns non-nil for them. The filter function may transform the data before inserting it.
// The filter function is also responsible for any deduplication.
func NewNodeList(filter NodeFilter, sort NodeSorter, initialize func() []forest.Node, s store.ExtendedStore) *NodeList {
	nl := new(NodeList)
	nl.filter = filter
	nl.sortFunc = sort
	nl.withNodesWritable(func() {
		nl.subscribeTo(s)
		nl.insert(initialize()...)
	})
	return nl
}

func (n *NodeList) Insert(nodes ...forest.Node) {
	n.withNodesWritable(func() {
		n.insert(nodes...)
	})
}

func (n *NodeList) insert(nodes ...forest.Node) {
outer:
	for _, node := range nodes {
		if filtered := n.filter(node); filtered != nil {
			for _, element := range n.nodes {
				if filtered.ID().Equals(element.ID()) {
					continue outer
				}
			}
			n.nodes = append(n.nodes, filtered)
		}
	}
	n.sort()
}

func (n *NodeList) subscribeTo(s store.ExtendedStore) {
	s.SubscribeToNewMessages(func(node forest.Node) {
		// cannot block in subscription
		go func() {
			n.Insert(node)
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
