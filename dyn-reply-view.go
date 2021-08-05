package main

import (
	"image"
	"log"
	"runtime"
	"sort"
	"time"

	"gioui.org/gesture"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
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
	"github.com/inkeliz/giohyperlink"
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
	// updatedListLen holds the most recent value of the chatManager's UpdatedLen()
	// This view calls UpdatedLen early so that it can traverse the resulting list
	// state, and this field allows passing the resulting length to the layout
	// call site.
	updatedListLen int

	// FocusAnimation is the shared animation state for all messages.
	FocusAnimation anim.Normal

	FocusTracker

	BackgroundClick gesture.Click

	core.App

	Hint string
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
		Synthesizer: func(prev, current, next list.Element) []list.Element {
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

func (c *DynamicChatView) Update(gtx layout.Context) {
	c.updatedListLen = c.chatManager.UpdatedLen(&c.chatList.List)
	key.InputOp{Tag: c}.Add(gtx.Ops)
	key.FocusOp{Tag: c}.Add(gtx.Ops)
	for _, event := range gtx.Events(c) {
		switch event := event.(type) {
		case key.Event:
			if event.State == key.Press {
				switch event.Name {
				case "D", key.NameDeleteBackward:
					if event.Modifiers.Contain(key.ModShift) {
						c.toggleConversationHidden()
					} else {
						c.toggleDescendantsHidden()
					}
				case "K", key.NameUpArrow:
					c.moveFocusUp(gtx)
				case "J", key.NameDownArrow:
					c.moveFocusDown(gtx)
				case key.NameHome:
					c.moveFocusStart(gtx)
				case "G":
					if !event.Modifiers.Contain(key.ModShift) {
						c.moveFocusStart(gtx)
					} else {
						c.moveFocusEnd(gtx)
					}
				case key.NameEnd:
					c.moveFocusEnd(gtx)
				case key.NameReturn, key.NameEnter:
					c.startReply()
				case "C":
					if event.Modifiers.Contain(key.ModCtrl) || (runtime.GOOS == "darwin" && event.Modifiers.Contain(key.ModCommand)) {
						c.copyFocused(gtx)
					} else {
						c.startConversation()
					}
				case key.NameSpace, "F":
					c.toggleFilter()
				}
			}
		}
	}

	elements := c.chatManager.ManagedElements(gtx)
	states := c.chatManager.ManagedState(gtx)

	c.Hint = ""

	for _, e := range elements {
		element, ok := e.(ds.ReplyData)
		if !ok {
			continue
		}
		state, ok := states[e.Serial()]
		if !ok {
			continue
		}
		switch state := state.(type) {
		case *sprigwidget.Reply:
			c.processReplyStateUpdates(gtx, element, state)
		}
	}

	if c.FocusTracker.RefreshNodeStatus(c.Arbor().Store()) {
		c.FocusAnimation.Start(gtx.Now)
	}
	for _, e := range c.BackgroundClick.Events(gtx) {
		switch e.Type {
		case gesture.TypeClick:
			c.FocusTracker.SetFocus(nil)
		}
	}
}

// TODO
func (c *DynamicChatView) toggleConversationHidden() {
}

// TODO
func (c *DynamicChatView) toggleDescendantsHidden() {
}

// moveFocusUp shifts the focused message upwards by one message.
// If there is no focused message, it automatically selects the
// final message.
func (c *DynamicChatView) moveFocusUp(gtx layout.Context) {
	if c.Focused == nil {
		c.moveFocusEnd(gtx)
		return
	}
	defer c.makeFocusedVisible(gtx)
	elements := c.chatManager.ManagedElements(gtx)
	var lastElement *ds.ReplyData
searchLoop:
	for _, e := range elements {
		switch e := e.(type) {
		case ds.ReplyData:
			if c.Focused.ID.Equals(e.ID) {
				break searchLoop
			} else {
				lastElement = &e
			}
		}
	}
	if lastElement == nil {
		return
	}
	c.SetFocus(lastElement)
}

// moveFocusDown shifts the focused message downward by one message.
// If there is no focused message, it automatically selects the
// final message.
func (c *DynamicChatView) moveFocusDown(gtx layout.Context) {
	if c.Focused == nil {
		c.moveFocusEnd(gtx)
		return
	}
	defer c.makeFocusedVisible(gtx)
	elements := c.chatManager.ManagedElements(gtx)
	var foundFocused bool
	for _, e := range elements {
		switch e := e.(type) {
		case ds.ReplyData:
			if c.Focused.ID.Equals(e.ID) {
				foundFocused = true
			} else if foundFocused {
				c.SetFocus(&e)
				return
			}
		}
	}
}

// moveFocusStart shifts focus to the first loaded element of history.
func (c *DynamicChatView) moveFocusStart(gtx layout.Context) {
	defer c.makeFocusedVisible(gtx)
	elements := c.chatManager.ManagedElements(gtx)
	for _, e := range elements {
		switch e := e.(type) {
		case ds.ReplyData:
			c.SetFocus(&e)
			return
		}
	}
}

// moveFocusEnd shifts focus to the final loaded element of history.
func (c *DynamicChatView) moveFocusEnd(gtx layout.Context) {
	defer c.makeFocusedVisible(gtx)
	elements := c.chatManager.ManagedElements(gtx)
	for i := len(elements) - 1; i >= 0; i-- {
		e := elements[i]
		switch e := e.(type) {
		case ds.ReplyData:
			c.SetFocus(&e)
			return
		}
	}
}

// makeFocusedVisible ensures that the focused message (if any) is visible
// in the UI by manipulating the scroll position.
func (c *DynamicChatView) makeFocusedVisible(gtx layout.Context) {
	if c.Focused == nil {
		return
	}
	elements := c.chatManager.ManagedElements(gtx)
	index := -1
searchLoop:
	for i, e := range elements {
		switch e := e.(type) {
		case ds.ReplyData:
			if e.ID.Equals(c.Focused.ID) {
				index = i
				break searchLoop
			}
		}
	}
	if index == -1 {
		return
	}
	visibleStart := c.chatList.Position.First
	visibleEnd := visibleStart + c.chatList.Position.Count - 1

	// If the focused element is before the start of the current viewport,
	// move the viewport to begin with it.
	if visibleStart >= index {
		c.chatList.Position.First = index
		c.chatList.Position.Offset = 0
		c.chatList.Position.BeforeEnd = true
		return
	}
	if visibleEnd == index && c.chatList.Position.OffsetLast != 0 {
		c.chatList.Position.Offset -= c.chatList.Position.OffsetLast
		c.chatList.Position.OffsetLast = 0
		return
	}
	if visibleEnd < index {
		c.chatList.Position.First = index
		c.chatList.Position.Offset = 0
		c.chatList.Position.BeforeEnd = true
		return
	}
}

// TODO
func (c *DynamicChatView) copyFocused(gtx layout.Context) {
}

// TODO
func (c *DynamicChatView) startConversation() {
}

// TODO
func (c *DynamicChatView) startReply() {
}

// TODO
func (c *DynamicChatView) toggleFilter() {
}

// Layout the view in the provided context.
func (c *DynamicChatView) Layout(gtx layout.Context) layout.Dimensions {
	sTheme := c.Theme().Current()
	theme := sTheme.Theme

	// Show text, if any.
	if c.Hint != "" {
		macro := op.Record(gtx.Ops)
		layout.SW.Layout(gtx, func(gtx C) D {
			return component.Surface(theme).Layout(gtx,
				func(gtx C) D {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx, material.Body2(theme, c.Hint).Layout)
				})
		})
		op.Defer(gtx.Ops, macro.Stop())
	}

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			defer op.Save(gtx.Ops).Load()
			pointer.Rect(image.Rectangle{
				Max: gtx.Constraints.Max,
			}).Add(gtx.Ops)
			c.BackgroundClick.Add(gtx.Ops)
			return D{Size: gtx.Constraints.Max}
		}),
		layout.Stacked(func(gtx C) D {
			gtx.Constraints.Min = gtx.Constraints.Max
			return material.List(theme, &c.chatList).Layout(gtx, c.updatedListLen, c.chatManager.Layout)
		}),
	)
}

func (c *DynamicChatView) processReplyStateUpdates(gtx layout.Context, element ds.ReplyData, state *sprigwidget.Reply) {
	// Process any clicks on the reply.
	for _, e := range state.Polyclick.Events(gtx) {
		switch e.Type {
		case gesture.TypeClick:
			if e.NumClicks == 1 {
				c.FocusTracker.SetFocus(&element)
			}
		}
	}
	for span, events := state.InteractiveText.Events(); len(events) > 0; span, events = state.InteractiveText.Events() {
		for _, event := range events {
			url := span.Get(markdown.MetadataURL)
			switch event.Type {
			case richtext.Click:
				giohyperlink.Open(url)
			case richtext.LongPress:
				c.Haptic().Buzz()
				fallthrough
			case richtext.Hover:
				c.Hint = url
			}
		}
	}
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
		// Layout the reply.
		return sprigtheme.ReplyRow(sTheme, state, animState, rd, richContent).Layout(gtx)
	}
}
