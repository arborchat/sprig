package ds

import (
	"fmt"
	"sync"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/forest-go/store"
)

// HiddenTracker tracks which nodes have been manually hidden by a user.
// This is modeled as a set of "anchor" nodes, the desendants of which
// are not visible. Anchors themselves are visible, and can be used to
// reveal their descendants. HiddenTracker is safe for concurrent use.
type HiddenTracker struct {
	sync.RWMutex
	anchors map[string][]*fields.QualifiedHash
	hidden  IDSet
}

// init initializes the underlying data structures.
func (h *HiddenTracker) init() {
	if h.anchors == nil {
		h.anchors = make(map[string][]*fields.QualifiedHash)
	}
}

// IsHidden returns whether the provided node should be hidden.
func (h *HiddenTracker) IsHidden(id *fields.QualifiedHash) bool {
	h.RLock()
	defer h.RUnlock()
	return h.isHidden(id)
}

func (h *HiddenTracker) isHidden(id *fields.QualifiedHash) bool {
	return h.hidden.Contains(id)
}

// IsAnchor returns whether the provided node is serving as an anchor
// that hides its descendants.
func (h *HiddenTracker) IsAnchor(id *fields.QualifiedHash) bool {
	h.RLock()
	defer h.RUnlock()
	return h.isAnchor(id)
}

func (h *HiddenTracker) isAnchor(id *fields.QualifiedHash) bool {
	_, ok := h.anchors[id.String()]
	return ok
}

// NumDescendants returns the number of hidden descendants for the given anchor
// node.
func (h *HiddenTracker) NumDescendants(id *fields.QualifiedHash) int {
	h.RLock()
	defer h.RUnlock()
	return h.numDescendants(id)
}

func (h *HiddenTracker) numDescendants(id *fields.QualifiedHash) int {
	return len(h.anchors[id.String()])
}

// ToggleAnchor switches the anchor state of the given ID.
func (h *HiddenTracker) ToggleAnchor(id *fields.QualifiedHash, s store.ExtendedStore) error {
	h.Lock()
	defer h.Unlock()
	return h.toggleAnchor(id, s)
}

func (h *HiddenTracker) toggleAnchor(id *fields.QualifiedHash, s store.ExtendedStore) error {
	if h.isAnchor(id) {
		h.reveal(id)
		return nil
	}
	return h.hide(id, s)
}

// Hide makes the given ID into an anchor and hides its descendants.
func (h *HiddenTracker) Hide(id *fields.QualifiedHash, s store.ExtendedStore) error {
	h.Lock()
	defer h.Unlock()
	return h.hide(id, s)
}

func (h *HiddenTracker) hide(id *fields.QualifiedHash, s store.ExtendedStore) error {
	h.init()
	descendants, err := s.DescendantsOf(id)
	if err != nil {
		return fmt.Errorf("failed looking up descendants of %s: %w", id.String(), err)
	}
	// ensure that any descendants that were previously hidden are subsumed by
	// hiding their ancestor.
	for _, d := range descendants {
		if _, ok := h.anchors[d.String()]; ok {
			delete(h.anchors, d.String())
		}
	}
	h.anchors[id.String()] = descendants
	h.hidden.Add(descendants...)
	return nil
}

// Process ensures that the internal state of the HiddenTracker accounts
// for the provided node. This is primarily useful for nodes that were inserted
// into the store *after* their ancestor was made into an anchor. Each time
// a new node is received, it should be Process()ed.
func (h *HiddenTracker) Process(node forest.Node) {
	h.Lock()
	defer h.Unlock()
	h.process(node)
}

func (h *HiddenTracker) process(node forest.Node) {
	if h.isHidden(node.ParentID()) || h.isAnchor(node.ParentID()) {
		h.hidden.Add(node.ID())
	}
}

// Reveal makes the given node no longer an anchor, thereby un-hiding all
// of its children.
func (h *HiddenTracker) Reveal(id *fields.QualifiedHash) {
	h.Lock()
	defer h.Unlock()
	h.Reveal(id)
}

func (h *HiddenTracker) reveal(id *fields.QualifiedHash) {
	h.init()
	descendants, ok := h.anchors[id.String()]
	if !ok {
		return
	}
	h.hidden.Remove(descendants...)
	delete(h.anchors, id.String())
}

// IDSet implements basic set operations on node IDs. It is not safe for
// concurrent use.
type IDSet struct {
	contents map[string]struct{}
}

// init allocates the underlying map type.
func (h *IDSet) init() {
	h.contents = make(map[string]struct{})
}

// Add inserts the list of IDs into the set.
func (h *IDSet) Add(ids ...*fields.QualifiedHash) {
	if h.contents == nil {
		h.init()
	}
	for _, id := range ids {
		h.contents[id.String()] = struct{}{}
	}
}

// Contains returns whether the given ID is in the set.
func (h *IDSet) Contains(id *fields.QualifiedHash) bool {
	if h.contents == nil {
		h.init()
	}
	_, contains := h.contents[id.String()]
	return contains
}

// Remove deletes the provided IDs from the set.
func (h *IDSet) Remove(ids ...*fields.QualifiedHash) {
	if h.contents == nil {
		h.init()
	}
	for _, id := range ids {
		if h.Contains(id) {
			delete(h.contents, id.String())
		}
	}
}
