package main

import (
	"context"
	"image/color"
	"log"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"git.sr.ht/~athorp96/forest-ex/expiration"
	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/forest-go/store"
	"git.sr.ht/~whereswaldon/forest-go/twig"

	materials "gioui.org/x/component"
	events "gioui.org/x/eventx"
	"gioui.org/x/scroll"

	"git.sr.ht/~whereswaldon/sprig/anim"
	"git.sr.ht/~whereswaldon/sprig/core"
	"git.sr.ht/~whereswaldon/sprig/ds"
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigWidget "git.sr.ht/~whereswaldon/sprig/widget"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type FilterState uint8

const (
	Off FilterState = iota
	Conversation
	Message
)

// FocusTracker keeps track of which message (if any) is focused and the status
// of ancestor/descendant messages relative to that message.
type FocusTracker struct {
	Focused      *ds.ReplyData
	Ancestry     []*fields.QualifiedHash
	Descendants  []*fields.QualifiedHash
	Conversation *fields.QualifiedHash
	// Whether the Ancestry and Descendants need to be regenerated because the
	// contents of the replylist changed
	stateRefreshNeeded bool
}

// SetFocus requests that the provided ReplyData become the focused message.
func (f *FocusTracker) SetFocus(focused ds.ReplyData) {
	f.stateRefreshNeeded = true
	f.Focused = &focused
	f.Conversation = f.Focused.ConversationID
}

// Invalidate notifies the FocusTracker that its Ancestry and Descendants lists
// are possibly incorrect as a result of a state change elsewhere.
func (f *FocusTracker) Invalidate() {
	f.stateRefreshNeeded = true
}

// RefreshNodeStatus updates the Ancestry and Descendants of the FocusTracker
// using the provided store to look them up. If the FocusTracker has not been
// invalidated since this method was last invoked, this method will return
// false and do nothing.
func (f *FocusTracker) RefreshNodeStatus(s store.ExtendedStore) bool {
	if f.stateRefreshNeeded {
		f.stateRefreshNeeded = false
		if f.Focused == nil {
			f.Ancestry = nil
			f.Descendants = nil
			return true
		}
		f.Ancestry, _ = s.AncestryOf(f.Focused.ID)
		f.Descendants, _ = s.DescendantsOf(f.Focused.ID)
		return true
	}
	return false
}

// ReplyListView manages the state and layout of the reply list view in
// Sprig's UI.
type ReplyListView struct {
	manager ViewManager

	core.App

	CopyReplyButton widget.Clickable

	sprigWidget.MessageList

	ds.AlphaReplyList

	FocusTracker

	sprigWidget.Composer

	FilterButton                        widget.Clickable
	CreateReplyButton                   widget.Clickable
	CreateConversationButton            widget.Clickable
	JumpToBottomButton, JumpToTopButton widget.Clickable
	HideDescendantsButton               widget.Clickable

	LoadMoreHistoryButton widget.Clickable
	// how many nodes of history does the view want
	HistoryRequestCount int

	scroll.Scrollable

	FilterState
	ds.HiddenTracker
	PrefilterPosition layout.Position

	ShouldRequestKeyboardFocus bool

	// Cache the number of replies during update.
	replyCount int
	// Maximum number of visible replies encountered.
	maxRepliesVisible int
	// Loading if replies are loading.
	loading bool
}

var _ View = &ReplyListView{}

// NewReplyListView constructs a ReplyList that relies on the provided App.
func NewReplyListView(app core.App) View {
	c := &ReplyListView{
		App:                 app,
		HistoryRequestCount: 2048,
	}
	c.MessageList.Animation.Normal = anim.Normal{
		Duration: time.Millisecond * 100,
	}
	c.MessageList.ShouldHide = func(r ds.ReplyData) bool {
		return c.HiddenTracker.IsHidden(r.ID) || c.shouldFilter(c.statusOf(r))
	}
	c.MessageList.StatusOf = func(r ds.ReplyData) sprigWidget.ReplyStatus {
		return c.statusOf(r)
	}
	c.MessageList.UserIsActive = func(identity *fields.QualifiedHash) bool {
		return c.Status().IsActive(identity)
	}
	c.MessageList.HiddenChildren = func(r ds.ReplyData) int {
		return c.HiddenTracker.NumDescendants(r.ID)
	}
	c.loading = true
	go func() {
		defer func() { c.loading = false }()
		c.AlphaReplyList.FilterWith(func(rd ds.ReplyData) bool {
			td := rd.Metadata
			if _, ok := td.Values[twig.Key{Name: "invisible", Version: 1}]; ok {
				return false
			}
			if expired, err := expiration.IsExpiredTwig(rd.Metadata); err != nil || expired {
				return false
			}
			return true
		})
		c.MessageList.Axis = layout.Vertical
		// ensure that we are notified when we need to refresh the state of visible nodes
		c.Arbor().Store().SubscribeToNewMessages(func(node forest.Node) {
			go func() {
				var rd ds.ReplyData
				if !rd.Populate(node, c.Arbor().Store()) {
					return
				}
				c.AlphaReplyList.Insert(rd)
				c.FocusTracker.Invalidate()
				c.manager.RequestInvalidate()
				c.HiddenTracker.Process(node)
			}()
		})
		c.MessageList.ScrollToEnd = true
		c.MessageList.Position.BeforeEnd = false
		c.loadMoreHistory()
	}()
	return c
}

// Filtered returns whether or not the ReplyList is currently filtering
// its contents.
func (c *ReplyListView) Filtered() bool {
	return c.FilterState != Off
}

// HandleIntent processes requests from other views in the application.
func (c *ReplyListView) HandleIntent(intent Intent) {}

// BecomeVisible handles setup for when this view becomes the visible
// view in the application.
func (c *ReplyListView) BecomeVisible() {
}

// NavItem returns the top-level navigation information for this view.
func (c *ReplyListView) NavItem() *materials.NavItem {
	return &materials.NavItem{
		Name: "Messages",
		Icon: icons.ChatIcon,
	}
}

// AppBarData returns the app bar actions for this view.
func (c *ReplyListView) AppBarData() (bool, string, []materials.AppBarAction, []materials.OverflowAction) {
	th := c.Theme().Current().Theme
	return true, "Messages", []materials.AppBarAction{
			materials.SimpleIconAction(
				&c.CreateConversationButton,
				icons.CreateConversationIcon,
				materials.OverflowAction{
					Name: "Create Conversation",
					Tag:  &c.CreateConversationButton,
				},
			),
			{
				OverflowAction: materials.OverflowAction{
					Name: "Filter by selected",
					Tag:  &c.FilterButton,
				},
				Layout: func(gtx C, bg, fg color.NRGBA) D {
					var buttonForeground, buttonBackground color.NRGBA
					var buttonText string
					btn := material.ButtonLayout(th, &c.FilterButton)
					switch c.FilterState {
					case Conversation:
						buttonForeground = bg
						buttonBackground = fg
						buttonBackground.A = 150
						buttonText = "Cvn"
					case Message:
						buttonForeground = bg
						buttonBackground = fg
						buttonText = "Msg"
					default:
						buttonForeground = fg
						buttonBackground = bg
						buttonText = "Off"
					}
					btn.Background = buttonBackground
					return btn.Layout(gtx, func(gtx C) D {
						return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx C) D {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx C) D {
									icon := icons.FilterIcon
									icon.Color = buttonForeground
									return icon.Layout(gtx, unit.Dp(24))
								}),
								layout.Rigid(func(gtx C) D {
									gtx.Constraints.Max.X = gtx.Px(unit.Dp(40))
									gtx.Constraints.Min.X = gtx.Constraints.Max.X
									label := material.Body1(th, buttonText)
									label.Color = buttonForeground
									label.MaxLines = 1
									return layout.Inset{Left: unit.Dp(6)}.Layout(gtx, label.Layout)
								}),
							)
						})
					})
				},
			},
		}, []materials.OverflowAction{
			{
				Name: "Jump to top",
				Tag:  &c.JumpToTopButton,
			},
			{
				Name: "Jump to bottom",
				Tag:  &c.JumpToBottomButton,
			},
			{
				Name: "Load more history",
				Tag:  &c.LoadMoreHistoryButton,
			},
		}
}

// getContextualActions returns the contextual app bar actions for this
// view (rather than the standard, non-contexual bar actions).
func (c *ReplyListView) getContextualActions() ([]materials.AppBarAction, []materials.OverflowAction) {
	return []materials.AppBarAction{
		materials.SimpleIconAction(
			&c.CopyReplyButton,
			icons.CopyIcon,
			materials.OverflowAction{
				Name: "Copy reply text",
				Tag:  &c.CopyReplyButton,
			},
		),
		materials.SimpleIconAction(
			&c.CreateReplyButton,
			icons.ReplyIcon,
			materials.OverflowAction{
				Name: "Reply to selected",
				Tag:  &c.CreateReplyButton,
			},
		),
		{
			OverflowAction: materials.OverflowAction{
				Name: "Hide/Show descendants",
				Tag:  &c.HideDescendantsButton,
			},
			Layout: func(gtx C, bg, fg color.NRGBA) D {
				btn := materials.SimpleIconButton(bg, fg, &c.HideDescendantsButton, icons.ExpandIcon)
				btn.Background = bg
				btn.Color = fg
				focusedID := c.FocusTracker.Focused.ID
				if c.HiddenTracker.IsAnchor(focusedID) {
					btn.Icon = icons.ExpandIcon
				} else {
					btn.Icon = icons.CollapseIcon
				}
				return btn.Layout(gtx)
			},
		},
	}, []materials.OverflowAction{}
}

// triggerReplyContextMenu changes the app bar to contextual mode and
// populates its actions with contextual options.
func (c *ReplyListView) triggerReplyContextMenu(gtx layout.Context) {
	actions, overflow := c.getContextualActions()
	c.manager.RequestContextualBar(gtx, "Message Operations", actions, overflow)
}

// dismissReplyContextMenu returns the app bar to non-contextual mode.
func (c *ReplyListView) dismissReplyContextMenu(gtx layout.Context) {
	c.manager.DismissContextualBar(gtx)
}

// moveFocusUp shifts the focused message up by one, if possible.
func (c *ReplyListView) moveFocusUp() {
	c.moveFocus(-1)
}

// moveFocusDown shifts the focused message down by one, if possible.
func (c *ReplyListView) moveFocusDown() {
	c.moveFocus(1)
}

// moveFocus shifts the focused message by the provided amount, if possible.
func (c *ReplyListView) moveFocus(indexIncrement int) {
	if c.Focused == nil {
		return
	}
	currentIndex := c.AlphaReplyList.IndexForID(c.Focused.ID)
	if currentIndex < 0 {
		return
	}
	c.AlphaReplyList.WithReplies(func(replies []ds.ReplyData) {
		for {
			currentIndex += indexIncrement
			if currentIndex >= len(replies) || currentIndex < 0 {
				break
			}
			status := c.statusOf(replies[currentIndex])
			if c.shouldFilter(status) {
				continue
			}
			c.FocusTracker.SetFocus(replies[currentIndex])
			c.ensureFocusedVisible(currentIndex)
			break
		}
	})
}

// ensureFocusedVisible attempts to ensure that the message at the
// provided index in the list being displayed is currently visible.
func (c *ReplyListView) ensureFocusedVisible(focusedIndex int) {
	currentFirst := c.MessageList.Position.First
	notInFirstFive := currentFirst+5 > focusedIndex
	if currentFirst <= focusedIndex && notInFirstFive {
		return
	}
	c.MessageList.Position.First = focusedIndex
	c.MessageList.Position.Offset = 0
	c.MessageList.Position.BeforeEnd = true
}

// moveFocusEnd shifts the focused message to the end of the list of
// replies.
func (c *ReplyListView) moveFocusEnd(replies []ds.ReplyData) {
	if len(replies) < 1 {
		return
	}
	c.SetFocus(replies[len(replies)-1])
	c.requestKeyboardFocus()
	c.MessageList.Position.BeforeEnd = false
}

// moveFocusStart shifts the focused message to the beginning of the
// list of replies.
func (c *ReplyListView) moveFocusStart(replies []ds.ReplyData) {
	if len(replies) < 1 {
		return
	}
	c.SetFocus(replies[0])
	c.requestKeyboardFocus()
	c.MessageList.Position.BeforeEnd = true
	c.MessageList.Position.First = 0
	c.MessageList.Position.Offset = 0
}

// reveal the reply at the given index.
func (c *ReplyListView) reveal(index int) {
	if c.replyCount < 1 || index > c.replyCount-1 {
		return
	}
	c.FocusTracker.Invalidate()
	c.requestKeyboardFocus()
	c.MessageList.Position.BeforeEnd = true
	c.MessageList.Position.First = index
}

// refreshNodeStatus triggers a check for changes to status updates and
// triggers animations if statuses have changed.
func (c *ReplyListView) refreshNodeStatus(gtx C) {
	if c.FocusTracker.RefreshNodeStatus(c.Arbor().Store()) {
		c.MessageList.Animation.Start(gtx.Now)
	}
}

// toggleFilter cycles between filter states.
func (c *ReplyListView) toggleFilter() {
	switch c.FilterState {
	case Conversation:
		c.FilterState = Message
	case Message:
		c.MessageList.Position = c.PrefilterPosition
		c.FilterState = Off
	default:
		c.PrefilterPosition = c.MessageList.Position
		c.FilterState = Conversation
	}
}

// copyFocused writes the contents of the focused message into the
// clipboard.
func (c *ReplyListView) copyFocused(gtx layout.Context) {
	reply := c.Focused
	clipboard.WriteOp{
		Text: reply.Content,
	}.Add(gtx.Ops)
}

// startReply begins replying to the focused message.
func (c *ReplyListView) startReply() {
	data := c.Focused
	c.Composer.StartReply(*data)
}

// sendReply sends the reply with the current contents of the editor.
func (c *ReplyListView) sendReply() {
	replyText := c.Composer.Text()
	if replyText == "" {
		return
	}
	var (
		newReplies []*forest.Reply
		author     *forest.Identity
		parent     forest.Node
		has        bool
	)

	replyText = strings.TrimSpace(replyText)

	nodeBuilder, err := c.Settings().Builder()
	if err != nil {
		log.Printf("failed acquiring node builder: %v", err)
	}
	author = nodeBuilder.User
	if c.Composer.ComposingConversation() {
		if c.Community.Value != "" {
			chosenString := c.Community.Value
			c.Arbor().Communities().WithCommunities(func(communities []*forest.Community) {
				for _, community := range communities {
					if community.ID().String() == chosenString {
						parent = community
						break
					}
				}
			})
		}
	} else {
		parent, has, err = c.Arbor().Store().Get(c.ReplyingTo.ID)
		if err != nil {
			log.Println("failed finding parent node %v in store: %v", c.ReplyingTo.ID, err)
			return
		} else if !has {
			log.Println("parent node %v is not in store: %v", c.ReplyingTo.ID, err)
			return
		}
	}

	for _, paragraph := range strings.Split(replyText, "\n\n") {
		if paragraph != "" {
			reply, err := nodeBuilder.NewReply(parent, paragraph, []byte{})
			if err != nil {
				log.Printf("failed creating new conversation: %v", err)
			} else {
				newReplies = append(newReplies, reply)
			}
			parent = reply
		}
	}

	c.postReplies(author, newReplies)
	c.resetReplyState()
}

// postReplies actually adds the replies to the store of history (and
// causes them to be sent because the sprout working is watching the
// store for updates).
func (c *ReplyListView) postReplies(author *forest.Identity, replies []*forest.Reply) {
	go func() {
		for _, reply := range replies {
			if err := c.Arbor().Store().Add(author); err != nil {
				log.Printf("failed adding replying identity to store: %v", err)
				return
			}
			if err := c.Arbor().Store().Add(reply); err != nil {
				log.Printf("failed adding reply to store: %v", err)
				return
			}
		}
	}()
}

// processMessagePointerEvents checks for specific pointer interactions
// with messages in the list and handles them.
func (c *ReplyListView) processMessagePointerEvents(gtx C) {
	tryOpenLink := func(word string) {
		if !strings.HasPrefix(word, "http") {
			return
		}
		if u, err := url.ParseRequestURI(word); err == nil {
			var args []string
			switch runtime.GOOS {
			case "darwin":
				args = []string{"open"}
			case "windows":
				args = []string{"cmd", "/c", "start"}
			default:
				args = []string{"xdg-open"}
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			cmd := exec.CommandContext(ctx, args[0], append(args[1:], u.String())...)
			if out, err := cmd.CombinedOutput(); err != nil {
				log.Printf("failed opening link: %s %s\n", string(out), err)
			}
		}
	}
	clicked := func(c *sprigWidget.Polyclick) (widget.Click, bool) {
		clicks := c.Clicks()
		if len(clicks) == 0 {
			return widget.Click{}, false
		}
		return clicks[len(clicks)-1], true
	}
	focus := func(handler *sprigWidget.Reply) {
		reply, _, _ := c.Arbor().Store().Get(handler.Hash)
		var data ds.ReplyData
		data.Populate(reply, c.Arbor().Store())
		c.SetFocus(data)
	}
	for i := range c.States.Buffer {
		handler := &c.States.Buffer[i]
		if click, ok := clicked(&handler.Polyclick); ok {
			if click.Modifiers.Contain(key.ModCtrl) {
				for _, word := range strings.Fields(handler.Content) {
					go tryOpenLink(word)
				}
			} else {
				c.requestKeyboardFocus()
				clickedOnFocused := handler.Hash.Equals(c.Focused.ID)
				if !clickedOnFocused {
					focus(handler)
					c.dismissReplyContextMenu(gtx)
				}
			}
			if click.NumClicks > 1 {
				c.toggleFilter()
			}
		}
		if handler.Polyclick.LongPressed() {
			focus(handler)
			c.Haptic().Buzz()
			c.triggerReplyContextMenu(gtx)
		}
	}
}

// startConversation triggers composition of a new conversation message.
func (c *ReplyListView) startConversation() {
	c.Composer.StartConversation()
}

// Update updates the state of the view in response to user input events.
func (c *ReplyListView) Update(gtx layout.Context) {
	c.replyCount = func() (count int) {
		c.AlphaReplyList.WithReplies(func(replies []ds.ReplyData) {
			count = len(replies)
		})
		return count
	}()
	jumpStart := func() {
		c.AlphaReplyList.WithReplies(func(replies []ds.ReplyData) {
			c.moveFocusStart(replies)
		})
	}
	jumpEnd := func() {
		c.AlphaReplyList.WithReplies(func(replies []ds.ReplyData) {
			c.moveFocusEnd(replies)
		})
	}
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
					c.moveFocusUp()
				case "J", key.NameDownArrow:
					c.moveFocusDown()
				case key.NameHome:
					jumpStart()
				case "G":
					if !event.Modifiers.Contain(key.ModShift) {
						jumpStart()
						break
					}
					fallthrough
				case key.NameEnd:
					jumpEnd()
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

	for _, e := range c.Composer.Events() {
		switch e {
		case sprigWidget.ComposerSubmitted:
			c.sendReply()
		case sprigWidget.ComposerCancelled:
			c.resetReplyState()
		}
	}
	overflowTag := c.manager.SelectedOverflowTag()
	if overflowTag == &c.JumpToBottomButton || c.JumpToBottomButton.Clicked() {
		jumpEnd()
	}
	if overflowTag == &c.JumpToTopButton || c.JumpToTopButton.Clicked() {
		jumpStart()
	}
	if overflowTag == &c.HideDescendantsButton || c.HideDescendantsButton.Clicked() {
		c.toggleDescendantsHidden()
	}
	c.processMessagePointerEvents(gtx)
	c.refreshNodeStatus(gtx)
	if c.FilterButton.Clicked() || overflowTag == &c.FilterButton {
		c.toggleFilter()
	}
	if c.Focused != nil && (c.CopyReplyButton.Clicked() || overflowTag == &c.CopyReplyButton) {
		c.copyFocused(gtx)
	}

	if c.Focused != nil && (c.CreateReplyButton.Clicked() || overflowTag == &c.CreateReplyButton) {
		c.startReply()
	}
	if c.CreateConversationButton.Clicked() || overflowTag == &c.CreateConversationButton {
		c.startConversation()
	}
	if did, progress := c.Scrollable.Scrolled(); did {
		c.reveal(int(float32(c.replyCount) * progress))
	}
	if c.LoadMoreHistoryButton.Clicked() || overflowTag == &c.LoadMoreHistoryButton {
		go c.loadMoreHistory()
	}
	for _, event := range c.MessageList.Events() {
		switch event.Type {
		case sprigWidget.LinkLongPress:
			c.Haptic().Buzz()
		case sprigWidget.LinkOpen:
			log.Println("Opening %s", event.Data)
		}
	}
}

// toggleDescendantsHidden makes the descendants of the current message
// hidden (or reverses it).
func (c *ReplyListView) toggleDescendantsHidden() {
	focusedID := c.FocusTracker.Focused.ID
	if err := c.HiddenTracker.ToggleAnchor(focusedID, c.Arbor().Store()); err != nil {
		log.Printf("Failed hiding descendants of selected: %v", err)
	}
}

// toggleConversationHidden makes the descendants of the current message's
// conversation hidden (or reverses it).
func (c *ReplyListView) toggleConversationHidden() {
	focusedID := c.FocusTracker.Focused.ConversationID
	if focusedID.Equals(fields.NullHash()) {
		// if the focused message is the root of a conversation, use its own ID
		focusedID = c.FocusTracker.Focused.ID
	}
	c.WithReplies(func(replies []ds.ReplyData) {
		for _, rd := range replies {
			if rd.ID.Equals(focusedID) {
				c.SetFocus(rd)
				if err := c.HiddenTracker.ToggleAnchor(rd.ID, c.Arbor().Store()); err != nil {
					log.Printf("Failed hiding descendants of selected: %v", err)
				}
				return
			}
		}
	})
}

// loadMoreHistory attempts to fetch more history from disk.
func (c *ReplyListView) loadMoreHistory() {
	const newNodeTarget = 1024
	var (
		nodes []forest.Node
		err   error
	)
	load := func() {
		nodes, err = c.Arbor().Store().Recent(fields.NodeTypeReply, c.HistoryRequestCount)
		c.HistoryRequestCount += newNodeTarget
		if err != nil {
			log.Printf("failed loading extra history: %v", err)
			return
		}
	}
	load()
	var populated []ds.ReplyData
	for i := range nodes {
		var rd ds.ReplyData
		if rd.Populate(nodes[i], c.Arbor().Store()) {
			populated = append(populated, rd)
		}
	}
	if len(populated) < newNodeTarget {
		load()
	}
	c.AlphaReplyList.Insert(populated...)
}

// resetReplyState erases the current contents of the message composer.
func (c *ReplyListView) resetReplyState() {
	c.Composer.Reset()
}

// statusOf returns the current UI status of a reply.
func (c *ReplyListView) statusOf(reply ds.ReplyData) (status sprigWidget.ReplyStatus) {
	if c.HiddenTracker.IsAnchor(reply.ID) {
		status |= sprigWidget.Anchor
	}
	if c.HiddenTracker.IsHidden(reply.ID) {
		status |= sprigWidget.Hidden
	}
	if c.Focused == nil {
		status |= sprigWidget.None
		return
	}
	if c.Focused != nil && reply.ID.Equals(c.Focused.ID) {
		status |= sprigWidget.Selected
		return
	}
	for _, id := range c.Ancestry {
		if id.Equals(reply.ID) {
			status |= sprigWidget.Ancestor
			return
		}
	}
	for _, id := range c.Descendants {
		if id.Equals(reply.ID) {
			status |= sprigWidget.Descendant
			return
		}
	}
	if reply.Depth == 1 {
		status |= sprigWidget.ConversationRoot
		return
	}
	if c.Conversation != nil && !c.Conversation.Equals(fields.NullHash()) {
		if c.Conversation.Equals(reply.ConversationID) {
			status |= sprigWidget.Sibling
			return
		}
	}
	status |= sprigWidget.None
	return
}

// shouldDisplayEditor returns whether the composer should be visible.
func (c *ReplyListView) shouldDisplayEditor() bool {
	return c.Composer.Composing()
}

// hideEditor makes the editor invisible.
func (c *ReplyListView) hideEditor() {
	c.Composer.Reset()
	c.requestKeyboardFocus()
}

// Layout renders the whole view into the provided context.
func (c *ReplyListView) Layout(gtx layout.Context) layout.Dimensions {
	theme := c.Theme().Current()
	c.ShouldRequestKeyboardFocus = false
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			sprigTheme.Rect{
				Color: theme.Background.Default.Bg,
				Size:  layout.FPt(gtx.Constraints.Max),
			}.Layout(gtx)
			return layout.Dimensions{}
		}),
		layout.Stacked(func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Flexed(1, func(gtx C) D {
					return c.layoutReplyList(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					if c.shouldDisplayEditor() {
						return c.layoutEditor(gtx)
					} else {
						key.InputOp{Tag: c}.Add(gtx.Ops)
						key.FocusOp{Tag: c}.Add(gtx.Ops)
					}
					return layout.Dimensions{}
				}),
			)
		}),
	)
}

const buttonWidthDp = 20
const scrollSlotWidthDp = 12

// shouldFilter returns whether the provided status should be filtered based
// on the current filter state.
func (c *ReplyListView) shouldFilter(status sprigWidget.ReplyStatus) bool {
	if status&sprigWidget.Hidden > 0 {
		return true
	}
	switch c.FilterState {
	case Conversation:
		return status&sprigWidget.None > 0 || status&sprigWidget.ConversationRoot > 0
	case Message:
		return status&sprigWidget.Sibling > 0 || status&sprigWidget.None > 0 || status&sprigWidget.ConversationRoot > 0
	default:
		return false
	}
}

// layoutReplyList renders the list of replies into the provided graphics context.
func (c *ReplyListView) layoutReplyList(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min = gtx.Constraints.Max
	var (
		dims                 layout.Dimensions
		th                       = c.Theme().Current()
		totalUnfilteredNodes int = 1 + len(c.Ancestry) + len(c.Descendants)
	)
	if c.loading {
		return layout.Center.Layout(gtx, func(gtx C) D {
			return material.Loader(th.Theme).Layout(gtx)
		})
	}
	c.AlphaReplyList.WithReplies(func(replies []ds.ReplyData) {
		if c.Focused == nil && len(replies) > 0 {
			c.moveFocusEnd(replies)
		}
		ml := sprigTheme.MessageList(th, &c.MessageList, &c.CreateReplyButton, replies)
		if !c.Filtered() {
			ml.Prefixes = []layout.Widget{
				func(gtx C) D {
					return layout.Center.Layout(gtx, func(gtx C) D {
						return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx C) D {
							return material.Button(th.Theme, &c.LoadMoreHistoryButton, "Load more history").Layout(gtx)
						})
					})
				},
			}
		}
		dims = ml.Layout(gtx)
	})

	totalNodes := func() int {
		if c.Filtered() {
			return totalUnfilteredNodes
		}
		return c.replyCount
	}()
	progress := float32(c.MessageList.Position.First) / float32(c.replyCount)
	visibleFraction := float32(0)
	if c.replyCount > 0 {
		if c.States.Current > c.maxRepliesVisible {
			c.maxRepliesVisible = c.States.Current
		}
		visibleFraction = float32(c.maxRepliesVisible) / float32(totalNodes)
		if visibleFraction > 1 {
			visibleFraction = 1
		}
	}
	bar := scroll.DefaultBar(&c.Scrollable, progress, visibleFraction)
	bar.Color = materials.WithAlpha(th.Background.Default.Fg, 200)
	layout.Inset{
		Top:    unit.Dp(2),
		Bottom: unit.Dp(2),
	}.Layout(gtx, bar.Layout)

	return dims
}

// layoutEditor renders the message composition editor into the provided graphics
// context.
func (c *ReplyListView) layoutEditor(gtx layout.Context) layout.Dimensions {
	var (
		th  = c.Theme().Current()
		spy *events.Spy
	)
	spy, gtx = events.Enspy(gtx)
	var dims layout.Dimensions
	c.Arbor().Communities().WithCommunities(func(comms []*forest.Community) {
		dims = sprigTheme.Composer(th, &c.Composer, comms).Layout(gtx)
	})

	for _, group := range spy.AllEvents() {
		for _, e := range group.Items {
			switch ev := e.(type) {
			case key.Event:
				if ev.State == key.Press {
					switch {
					case ev.Name == key.NameEscape || (ev.Name == "[" && ev.Modifiers.Contain(key.ModCtrl)):
						c.hideEditor()
					}
				}
			}
		}
	}
	return dims
}

// SetManager configures the view manager for this view.
func (c *ReplyListView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
