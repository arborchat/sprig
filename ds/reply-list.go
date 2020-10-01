package ds

import (
	"sort"
	"sync"

	"git.sr.ht/~whereswaldon/forest-go/fields"
)

// sortable is a slice of reply data that conforms to the sort.Interface
// and also tracks the index for each element in a separate map
type sortable struct {
	indexForID map[string]int
	data       []ReplyData
}

var _ sort.Interface = sortable{}

func (s sortable) Len() int {
	return len(s.data)
}

func (s sortable) ensureIndexed(i int) {
	s.indexForID[s.data[i].Reply.ID().String()] = i
}

func (s sortable) Swap(i, j int) {
	s.data[i], s.data[j] = s.data[j], s.data[i]
	s.ensureIndexed(i)
	s.ensureIndexed(j)
}

func (s sortable) Less(i, j int) bool {
	s.ensureIndexed(i)
	s.ensureIndexed(j)
	return s.data[i].Reply.Created < s.data[j].Reply.Created
}

// replyList creates a thread-safe list of ReplyData that maintains its
// internal sort order and supports looking up the index of specific nodes.
type replyList struct {
	sync.RWMutex
	sortable
}

func (r *replyList) asWritable(f func()) {
	r.Lock()
	defer r.Unlock()
	f()
}

func (r *replyList) asReadable(f func()) {
	r.RLock()
	defer r.RUnlock()
	f()
}

func (r *replyList) sort() {
	sort.Sort(r.sortable)
}

func (r *replyList) insert(nodes ...ReplyData) {
	r.data = append(r.data, nodes...)
	r.sort()
}

// Insert adds the ReplyData to the list and updates the list sort order
func (r *replyList) Insert(nodes ...ReplyData) {
	r.asWritable(func() {
		r.insert(nodes...)
	})
}

// IndexForID returns the index at which the given ID's data is stored.
// It is safe (and recommended) to call this function from within the function
// passed to WithReplies(), as otherwise the node may by moved by another
// goroutine between looking up its index and using it.
func (r *replyList) IndexForID(id *fields.QualifiedHash) int {
	var out int
	r.asReadable(func() {
		var ok bool
		if out, ok = r.indexForID[id.String()]; !ok {
			out = -1
		}
	})
	return out
}

// Contains returns whether the list currently contains the node with the given
// ID.
func (r *replyList) Contains(id *fields.QualifiedHash) bool {
	return r.IndexForID(id) != -1
}

// WithReplies accepts a closure that it will run with access to the stored list
// of replies. It is invalid to access the replies list stored by a replyList
// except from within this closure. References to the slice are not valid after
// the closure returns, and using them will cause confusing bugs.
func (r *replyList) WithReplies(f func(replies []ReplyData)) {
	r.asReadable(func() {
		f(r.data)
	})
}
