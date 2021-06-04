package widget

import (
	"gioui.org/layout"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/sprig/anim"
	"git.sr.ht/~whereswaldon/sprig/ds"
)

type MessageList struct {
	layout.List
	States
	ShouldHide     func(reply ds.ReplyData) bool
	StatusOf       func(reply ds.ReplyData) ReplyStatus
	HiddenChildren func(reply ds.ReplyData) int
	UserIsActive   func(identity *fields.QualifiedHash) bool
	Animation
}

// States implements a buffer of reply states such that memory
// is reused each frame, yet grows as the view expands to hold more replies.
type States struct {
	Buffer  []Reply
	Current int
}

// Begin resets the buffer to the start.
func (s *States) Begin() {
	s.Current = 0
}

func (s *States) Next() *Reply {
	defer func() { s.Current++ }()
	if s.Current > len(s.Buffer)-1 {
		s.Buffer = append(s.Buffer, Reply{})
	}
	return &s.Buffer[s.Current]
}

// Animation maintains animation states per reply.
type Animation struct {
	anim.Normal
	animationInit bool
	Collection    map[*fields.QualifiedHash]*ReplyAnimationState
}

func (a *Animation) init() {
	a.Collection = make(map[*fields.QualifiedHash]*ReplyAnimationState)
	a.animationInit = true
}

// Lookup animation state for the given reply.
// If state doesn't exist, it will be created with using `s` as the
// beginning status.
func (a *Animation) Lookup(replyID *fields.QualifiedHash, s ReplyStatus) *ReplyAnimationState {
	if !a.animationInit {
		a.init()
	}
	_, ok := a.Collection[replyID]
	if !ok {
		a.Collection[replyID] = &ReplyAnimationState{
			Normal: &a.Normal,
			Begin:  s,
		}
	}
	return a.Collection[replyID]
}

// Update animation state for the given reply.
func (a *Animation) Update(gtx layout.Context, replyID *fields.QualifiedHash, s ReplyStatus) *ReplyAnimationState {
	anim := a.Lookup(replyID, s)
	if a.Animating(gtx) {
		anim.End = s
	} else {
		anim.Begin = s
		anim.End = s
	}
	return anim
}

type ReplyStatus int

const (
	None ReplyStatus = 1 << iota
	Sibling
	Selected
	Ancestor
	Descendant
	ConversationRoot
	// Anchor indicates that this node is visible, but its descendants have been
	// hidden.
	Anchor
	// Hidden indicates that this node is not currently visible.
	Hidden
)

// ReplyAnimationState holds the state of an in-progress animation for a reply.
// The anim.Normal field defines how far through the animation the node is, and
// the Begin and End fields define the two states that the node is transitioning
// between.
type ReplyAnimationState struct {
	*anim.Normal
	Begin, End ReplyStatus
}
