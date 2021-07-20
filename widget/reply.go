package widget

import (
	"gioui.org/x/richtext"
	"git.sr.ht/~whereswaldon/forest-go/fields"
)

// Reply holds ui state for each reply.
type Reply struct {
	Hash    *fields.QualifiedHash
	Content string
	Polyclick
	richtext.InteractiveText
	ReplyStatus
}

func (r *Reply) WithHash(h *fields.QualifiedHash) *Reply {
	r.Hash = h
	return r
}

func (r *Reply) WithContent(s string) *Reply {
	r.Content = s
	return r
}
