package main

import (
	"log"
	"sort"
	"time"

	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	materials "gioui.org/x/component"
	"gioui.org/x/markdown"
	"gioui.org/x/richtext"
	"git.sr.ht/~gioverse/chat/list"
	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/forest-go/store"
	"git.sr.ht/~whereswaldon/sprig/anim"
	"git.sr.ht/~whereswaldon/sprig/core"
	"git.sr.ht/~whereswaldon/sprig/ds"
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigwidget "git.sr.ht/~whereswaldon/sprig/widget"
	sprigtheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

// DynamicChatView lays out a view of chat history built atop a
// dynamic list. Messages are presented chronologically, but with
// a finite number of messages loaded in memory at any given time.
// As the user scrolls forward or backward, new messages are loaded
// and messages in the opposite direction are discarded.
type DynamicChatView struct {
	manager ViewManager

	// chatList holds the list and scrollbar state.
	chatList widget.List
	// chatManager controls which list elements are loaded into memory.
	chatManager *list.Manager

	// FocusAnimation is the shared animation state for all messages.
	FocusAnimation anim.Normal

	FocusTracker

	core.App
}

var _ View = &DynamicChatView{}

// NewDynamicChatView constructs a chat view.
func NewDynamicChatView(app core.App) View {
	c := &DynamicChatView{
		App: app,
	}
	c.chatList.Axis = layout.Vertical
	c.chatList.List.ScrollToEnd = true
	c.FocusAnimation.Duration = time.Millisecond * 100
	c.chatManager = list.NewManager(100, list.Hooks{
		Invalidator: func() { c.manager.RequestInvalidate() },
		Allocator: func(elem list.Element) interface{} {
			return &sprigwidget.Reply{}
		},
		Comparator: func(a, b list.Element) bool {
			aRD := a.(ds.ReplyData)
			bRD := b.(ds.ReplyData)
			return aRD.CreatedAt.Before(bRD.CreatedAt)
		},
		Synthesizer: func(prev, current list.Element) []list.Element {
			return []list.Element{current}
		},
		Presenter: c.layoutReply,
		Loader:    c.loadMessages,
	})

	c.Arbor().Store().SubscribeToNewMessages(c.handleNewNode)
	return c
}

// DynamicChatViewName defines the user-presented name for this view.
const DynamicChatViewName = "Chat"

// handleNewNode processes nodes that have been recieved new after the
// view was instantiated.
func (c *DynamicChatView) handleNewNode(node forest.Node) {
	go func() {
		switch node := node.(type) {
		case *forest.Reply:
			if !replyIsVisible(node) {
				return
			}
			var rd ds.ReplyData
			if !rd.Populate(node, c.Arbor().Store()) {
				return
			}
			c.chatManager.Modify([]list.Element{rd}, nil, nil)
			c.FocusTracker.Invalidate()
			c.manager.RequestInvalidate()
		default:
			// Discard, we only display replies in this view.
		}
	}()
}

// AppBarData returns the configuration of the app bar for this view.
func (c *DynamicChatView) AppBarData() (bool, string, []materials.AppBarAction, []materials.OverflowAction) {
	return true, DynamicChatViewName, []materials.AppBarAction{}, []materials.OverflowAction{}
}

// NavItem returns the configuration of the navigation drawer item for
// this view.
func (c *DynamicChatView) NavItem() *materials.NavItem {
	return &materials.NavItem{
		Tag:  c,
		Name: DynamicChatViewName,
		Icon: icons.SubscriptionIcon,
	}
}

// Update the state of the view in response to events.
func (c *DynamicChatView) Update(gtx layout.Context) {
	if c.FocusTracker.RefreshNodeStatus(c.Arbor().Store()) {
		c.FocusAnimation.Start(gtx.Now)
	}
}

// Layout the view in the provided context.
func (c *DynamicChatView) Layout(gtx layout.Context) layout.Dimensions {
	c.Update(gtx)
	sTheme := c.Theme().Current()
	theme := sTheme.Theme

	return material.List(theme, &c.chatList).Layout(gtx, c.chatManager.UpdatedLen(&c.chatList.List), c.chatManager.Layout)
}

// Set the view manager for this view.
func (c *DynamicChatView) SetManager(mgr ViewManager) {
	c.manager = mgr
}

// Handle a request from another view.
func (c *DynamicChatView) HandleIntent(intent Intent) {}

// BecomeVisible prepares the chat to be displayed to the user.
func (c *DynamicChatView) BecomeVisible() {
}

// pageableStore defines the interface of a type that can answer
// time-based, paginated queries about forest nodes.
type pageableStore interface {
	RecentReplies(ts fields.Timestamp, q int) (replies []forest.Reply, err error)
	RepliesAfter(ts fields.Timestamp, q int) (replies []forest.Reply, err error)
	forest.Store
}

func serialToID(s list.Serial) *fields.QualifiedHash {
	asString := string(s)
	var qh fields.QualifiedHash
	err := qh.UnmarshalText([]byte(asString))
	if err != nil {
		return fields.NullHash()
	}
	return &qh
}

func replyIsVisible(reply *forest.Reply) bool {
	md, err := reply.TwigMetadata()
	if err != nil {
		return false
	}
	if md.Contains("invisible", 1) {
		return false
	}
	return true
}

func replyToElement(store store.ExtendedStore, reply *forest.Reply) list.Element {
	if !replyIsVisible(reply) {
		return nil
	}
	var rd ds.ReplyData
	rd.Populate(reply, store)
	return rd
}

func replyNodesToElements(store store.ExtendedStore, replies ...forest.Node) []list.Element {
	elements := make([]list.Element, 0, len(replies))
	for _, reply := range replies {
		if reply, ok := reply.(*forest.Reply); ok {
			element := replyToElement(store, reply)
			if element != nil {
				elements = append(elements, element)
			}
		}
	}
	return elements
}
func repliesToElements(store store.ExtendedStore, replies ...forest.Reply) []list.Element {
	elements := make([]list.Element, 0, len(replies))
	for _, reply := range replies {
		element := replyToElement(store, &reply)
		if element != nil {
			elements = append(elements, element)
		}
	}
	return elements
}

func (c *DynamicChatView) loadMessagesPaged(store pageableStore, dir list.Direction, relativeTo list.Serial) []list.Element {
	const batchSize = 10
	batch := make([]list.Element, 0, batchSize)
	if relativeTo == list.NoSerial {
		// We are loading the very first messages, so look relative to
		// right now (plus a little because clock skew).
		startTime := time.Now().Add(time.Hour)
		// Since some messages aren't meant to be displayed, we may have
		// to query multiple times. Keep trying until we get enough messages
		// or there are no more messages.
		for len(batch) < batchSize {
			replies, err := store.RecentReplies(fields.TimestampFrom(startTime), batchSize)
			if err != nil || len(replies) == 0 {
				return batch
			}
			batch = append(batch, repliesToElements(c.Arbor().Store(), replies...)...)
			sort.Slice(replies, func(i, j int) bool {
				return replies[i].CreatedAt().Before(replies[j].CreatedAt())
			})
			startTime = replies[0].CreatedAt()
		}
		return batch
	}
	relID := serialToID(relativeTo)
	if relID.Equals(fields.NullHash()) {
		return nil
	}
	node, has, err := store.Get(relID)
	if err != nil || !has {
		return nil
	}
	createdAt := node.CreatedAt()
	if dir == list.Before {
		for len(batch) < batchSize {
			replies, err := store.RecentReplies(fields.TimestampFrom(createdAt), batchSize)
			if err != nil || len(replies) == 0 {
				return batch
			}
			batch = append(batch, repliesToElements(c.Arbor().Store(), replies...)...)
			sort.Slice(replies, func(i, j int) bool {
				return replies[i].CreatedAt().Before(replies[j].CreatedAt())
			})
			createdAt = replies[0].CreatedAt()
		}
		return batch
	}
	for len(batch) < batchSize {
		replies, err := store.RepliesAfter(fields.TimestampFrom(createdAt), batchSize)
		if err != nil || len(replies) == 0 {
			return batch
		}
		batch = append(batch, repliesToElements(c.Arbor().Store(), replies...)...)
		sort.Slice(replies, func(i, j int) bool {
			return replies[i].CreatedAt().Before(replies[j].CreatedAt())
		})
		createdAt = replies[len(replies)-1].CreatedAt()
	}
	return batch
}

// loadMessages loads chat messages in a given direction relative to a given
// other chat message.
func (c *DynamicChatView) loadMessages(dir list.Direction, relativeTo list.Serial) []list.Element {
	if archive, ok := c.Arbor().Store().(*store.Archive); ok {
		if pageable, ok := archive.UnderlyingStore().(pageableStore); ok {
			return c.loadMessagesPaged(pageable, dir, relativeTo)
		}
	}
	if relativeTo != list.NoSerial || dir == 0 {
		return nil
	}
	replies, err := c.Arbor().Store().Recent(fields.NodeTypeReply, 100)
	if err != nil {
		log.Printf("failed loading replies: %v", err)
		return nil
	}
	elements := make([]list.Element, 0, len(replies))
	for _, reply := range replies {
		if reply, ok := reply.(*forest.Reply); !ok {
			continue
		} else if !replyIsVisible(reply) {
			continue
		}
		var rd ds.ReplyData
		rd.Populate(reply, c.Arbor().Store())
		elements = append(elements, rd)
	}
	return elements
}

// replyState returns the display status of a given message within the view.
// This varies based on what is selected, filtered, and hidden.
func (c *DynamicChatView) replyState(reply ds.ReplyData) (status sprigwidget.ReplyStatus) {
	if reply.Depth == 1 {
		status |= sprigwidget.ConversationRoot
	}
	status |= c.FocusTracker.StatusFor(reply)
	return
}

// layoutReply returns a widget that will render the provided reply using the
// provided state.
func (c *DynamicChatView) layoutReply(replyData list.Element, state interface{}) layout.Widget {
	sTheme := c.Theme().Current()
	theme := sTheme.Theme
	return func(gtx C) D {
		// Expose the concrete types of the parameters.
		state := state.(*sprigwidget.Reply)
		rd := replyData.(ds.ReplyData)
		// Render the markdown content of the reply.
		content, _ := markdown.NewRenderer().Render(theme, []byte(rd.Content))
		richContent := richtext.Text(&state.InteractiveText, theme.Shaper, content...)
		// Construct an animation state using the shared animation progress
		// but use discrete begin and end states for this reply.
		animState := &sprigwidget.ReplyAnimationState{
			Normal: &c.FocusAnimation,
			Begin:  state.ReplyStatus,
			End:    c.replyState(rd),
		}
		// At the end of an animation, update the persistent state of the
		// reply to reflect its new state.
		if !c.FocusAnimation.Animating(gtx) {
			state.ReplyStatus = animState.End
		}
		// Process any clicks on the reply.
		for _, e := range state.Polyclick.Events(gtx) {
			switch e.Type {
			case gesture.TypeClick:
				if e.NumClicks == 1 {
					c.FocusTracker.SetFocusDeferred(rd)
				}
			}
		}
		// Layout the reply.
		return sprigtheme.ReplyRow(sTheme, state, animState, rd, richContent).Layout(gtx)
	}
}
