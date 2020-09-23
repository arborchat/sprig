package widget

import (
	"gioui.org/widget"
	"git.sr.ht/~whereswaldon/forest-go/fields"
)

// Reply holds ui state for each reply.
type Reply struct {
	widget.Clickable
	Hash    *fields.QualifiedHash
	Content string
}

func (r *Reply) WithHash(h *fields.QualifiedHash) *Reply {
	r.Hash = h
	return r
}

func (r *Reply) WithContent(s string) *Reply {
	r.Content = s
	return r
}
