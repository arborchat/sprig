package ds

import (
	"sort"
	"sync"

	"git.sr.ht/~whereswaldon/forest-go/fields"
)

// sortable is a slice of reply data that conforms to the sort.Interface
// and also tracks the index for each element in a separate map
type sortable struct {
	initialized bool
	indexForID  map[string]int
	data        []ReplyData
	allow       func(ReplyData) bool
}

var _ sort.Interface = &sortable{}

func (s *sortable) initialize() {
	if s.initialized {
		return
	}
	s.initialized = true
	s.indexForID = make(map[string]int)
}

func (s *sortable) Len() int {
	return len(s.data)
}

func (s *sortable) ensureIndexed(i int) {
	s.indexForID[s.data[i].ID.String()] = i
}

func (s *sortable) Swap(i, j int) {
	s.initialize()
	s.data[i], s.data[j] = s.data[j], s.data[i]
	s.ensureIndexed(i)
	s.ensureIndexed(j)
}

func (s *sortable) Less(i, j int) bool {
	s.initialize()
	s.ensureIndexed(i)
	s.ensureIndexed(j)
	return s.data[i].CreatedAt.Before(s.data[j].CreatedAt)
}

func (s *sortable) IndexForID(id *fields.QualifiedHash) int {
	s.initialize()
	if out, ok := s.indexForID[id.String()]; !ok {
		return -1
	} else {
		return out
	}
}

func (s *sortable) Sort() {
	s.initialize()
	sort.Sort(s)
}

func (s *sortable) Contains(id *fields.QualifiedHash) bool {
	s.initialize()
	return s.IndexForID(id) != -1
}

func (s *sortable) shouldAllow(rd ReplyData) bool {
	if s.allow != nil {
		return s.allow(rd)
	}
	return true
}

func (s *sortable) Insert(nodes ...ReplyData) {
	s.initialize()
	var newNodes []ReplyData
	for _, node := range nodes {
		if s.shouldAllow(node) && !s.Contains(node.ID) {
			newNodes = append(newNodes, node)
		}
	}
	s.data = append(s.data, newNodes...)
	s.Sort()
}

// AlphaReplyList creates a thread-safe list of ReplyData that maintains its
// internal sort order and supports looking up the index of specific nodes.
// It enforces uniqueness on the nodes it contains
type AlphaReplyList struct {
	sync.RWMutex
	sortable
}

func (r *AlphaReplyList) asWritable(f func()) {
	r.Lock()
	defer r.Unlock()
	f()
}

func (r *AlphaReplyList) asReadable(f func()) {
	r.RLock()
	defer r.RUnlock()
	f()
}

func (r *AlphaReplyList) FilterWith(f func(ReplyData) bool) {
	r.asWritable(func() {
		r.sortable.allow = f
	})
}

// Insert adds the ReplyData to the list and updates the list sort order
func (r *AlphaReplyList) Insert(nodes ...ReplyData) {
	r.asWritable(func() {
		r.sortable.Insert(nodes...)
	})
}

// IndexForID returns the index at which the given ID's data is stored.
// It is safe (and recommended) to call this function from within the function
// passed to WithReplies(), as otherwise the node may by moved by another
// goroutine between looking up its index and using it.
func (r *AlphaReplyList) IndexForID(id *fields.QualifiedHash) (index int) {
	r.asReadable(func() {
		index = r.sortable.IndexForID(id)
	})
	return
}

// Contains returns whether the list currently contains the node with the given
// ID.
func (r *AlphaReplyList) Contains(id *fields.QualifiedHash) (isContained bool) {
	r.asReadable(func() {
		isContained = r.sortable.Contains(id)
	})
	return
}

// WithReplies accepts a closure that it will run with access to the stored list
// of replies. It is invalid to access the replies list stored by a replyList
// except from within this closure. References to the slice are not valid after
// the closure returns, and using them will cause confusing bugs.
func (r *AlphaReplyList) WithReplies(f func(replies []ReplyData)) {
	r.asReadable(func() {
		f(r.data)
	})
}
