package widget

import (
	"strings"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/x/markdown"
	"gioui.org/x/richtext"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/sprig/anim"
	"git.sr.ht/~whereswaldon/sprig/ds"
)

// MessageListEventType is a kind of message list event.
type MessageListEventType uint8

const (
	LinkOpen MessageListEventType = iota
	LinkLongPress
)

// MessageListEvent describes a user interaction with the message list.
type MessageListEvent struct {
	Type MessageListEventType
	// Data contains event-specific content:
	// - LinkOpened: the hyperlink being opened
	// - LinkLongPressed: the hyperlink that was longpressed
	Data string
}

type MessageList struct {
	widget.List
	textCache RichTextCache
	ReplyStates
	ShouldHide     func(reply ds.ReplyData) bool
	StatusOf       func(reply ds.ReplyData) ReplyStatus
	HiddenChildren func(reply ds.ReplyData) int
	UserIsActive   func(identity *fields.QualifiedHash) bool
	Animation
	events []MessageListEvent
}

// GetTextState returns state storage for a node with the given ID, as well as hint text that should
// be shown when rendering the given node (if any).
func (m *MessageList) GetTextState(id *fields.QualifiedHash) (*richtext.InteractiveText, string) {
	state := m.textCache.Get(id)
	hint := ""
	for span, events := state.Events(); span != nil; span, events = state.Events() {
		for _, event := range events {
			url := span.Get(markdown.MetadataURL)
			switch event.Type {
			case richtext.Click:
				if asStr, ok := url.(string); ok {
					m.events = append(m.events, MessageListEvent{Type: LinkOpen, Data: asStr})
				}
			case richtext.LongPress:
				if asStr, ok := url.(string); ok {
					m.events = append(m.events, MessageListEvent{Type: LinkLongPress, Data: asStr})
				}
				fallthrough
			case richtext.Hover:
				if asStr, ok := url.(string); ok {
					hint = asStr
				}
			}
		}
	}
	return state, hint
}

// Layout updates the state of the message list each frame.
func (m *MessageList) Layout(gtx layout.Context) layout.Dimensions {
	m.textCache.Frame()
	m.ReplyStates.Begin()
	m.List.Axis = layout.Vertical
	return layout.Dimensions{}
}

// Events returns user interactions with the message list that have occurred
// since the last call to Events().
func (m *MessageList) Events() []MessageListEvent {
	out := m.events
	m.events = m.events[:0]
	return out
}

type ReplyStates = States[Reply]

// States implements a buffer states such that memory
// is reused each frame, yet grows as the view expands
// to hold more values.
type States[T any] struct {
	Buffer  []T
	Current int
}

// Begin resets the buffer to the start.
func (s *States[T]) Begin() {
	s.Current = 0
}

// Next returns the next available state to use, growing the underlying
// buffer if necessary.
func (s *States[T]) Next() *T {
	defer func() { s.Current++ }()
	if s.Current > len(s.Buffer)-1 {
		s.Buffer = append(s.Buffer, *new(T))
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

func (r ReplyStatus) Contains(other ReplyStatus) bool {
	return r&other > 0
}

func (r ReplyStatus) String() string {
	var out []string
	if r.Contains(None) {
		out = append(out, "None")
	}
	if r.Contains(Sibling) {
		out = append(out, "Sibling")
	}
	if r.Contains(Selected) {
		out = append(out, "Selected")
	}
	if r.Contains(Ancestor) {
		out = append(out, "Ancestor")
	}
	if r.Contains(Descendant) {
		out = append(out, "Descendant")
	}
	if r.Contains(ConversationRoot) {
		out = append(out, "ConversationRoot")
	}
	if r.Contains(Anchor) {
		out = append(out, "Anchor")
	}
	if r.Contains(Hidden) {
		out = append(out, "Hidden")
	}
	return strings.Join(out, "|")
}

// ReplyAnimationState holds the state of an in-progress animation for a reply.
// The anim.Normal field defines how far through the animation the node is, and
// the Begin and End fields define the two states that the node is transitioning
// between.
type ReplyAnimationState struct {
	*anim.Normal
	Begin, End ReplyStatus
}

type CacheEntry struct {
	UsedSinceLastFrame bool
	richtext.InteractiveText
}

// RichTextCache holds rendered richtext state across frames, discarding any
// state that is not used during a given frame.
type RichTextCache struct {
	items map[*fields.QualifiedHash]*CacheEntry
}

func (r *RichTextCache) init() {
	r.items = make(map[*fields.QualifiedHash]*CacheEntry)
}

// Get returns richtext state for the given id if it exists, and allocates a new
// state in the cache if it doesn't.
func (r *RichTextCache) Get(id *fields.QualifiedHash) *richtext.InteractiveText {
	if r.items == nil {
		r.init()
	}
	if to, ok := r.items[id]; ok {
		r.items[id].UsedSinceLastFrame = true
		return &to.InteractiveText
	}
	r.items[id] = &CacheEntry{
		UsedSinceLastFrame: true,
	}
	return &r.items[id].InteractiveText
}

// Frame purges cache entries that haven't been used since the last frame.
func (r *RichTextCache) Frame() {
	for k, v := range r.items {
		if !v.UsedSinceLastFrame {
			delete(r.items, k)
		} else {
			v.UsedSinceLastFrame = false
		}
	}
}
