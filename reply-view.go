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

type FocusTracker struct {
	Focused      forest.Node
	Ancestry     []*fields.QualifiedHash
	Descendants  []*fields.QualifiedHash
	Conversation *fields.QualifiedHash
	// Whether the Ancestry and Descendants need to be regenerated because the
	// contents of the replylist changed
	stateRefreshNeeded bool
}

func (f *FocusTracker) SetFocus(focused forest.Node) {
	f.stateRefreshNeeded = true
	f.Focused = focused
	if reply, ok := focused.(*forest.Reply); ok {
		f.Conversation = &reply.ConversationID
	} else {
		f.Conversation = nil
	}
}
func (f *FocusTracker) Invalidate() {
	f.stateRefreshNeeded = true
}

func (f *FocusTracker) RefreshNodeStatus(s store.ExtendedStore) bool {
	if f.stateRefreshNeeded {
		f.stateRefreshNeeded = false
		if f.Focused == nil {
			f.Ancestry = nil
			f.Descendants = nil
			return true
		}
		f.Ancestry, _ = s.AncestryOf(f.Focused.ID())
		f.Descendants, _ = s.DescendantsOf(f.Focused.ID())
		return true
	}
	return false
}

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

	LoadMoreHistoryButton widget.Clickable
	// how many nodes of history does the view want
	HistoryRequestCount int

	scroll.Scrollable

	FilterState
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

func NewReplyListView(app core.App) View {
	c := &ReplyListView{
		App:                 app,
		HistoryRequestCount: 2048,
	}
	c.MessageList.Animation.Normal = anim.Normal{
		Duration: time.Millisecond * 100,
	}
	c.MessageList.ShouldHide = func(r ds.ReplyData) bool {
		return c.shouldFilter(c.statusOf(r.Reply))
	}
	c.MessageList.StatusOf = func(r ds.ReplyData) sprigWidget.ReplyStatus {
		return c.statusOf(r.Reply)
	}
	c.MessageList.UserIsActive = func(identity *fields.QualifiedHash) bool {
		return c.Status().IsActive(identity)
	}
	c.loading = true
	go func() {
		defer func() { c.loading = false }()
		c.AlphaReplyList.FilterWith(func(rd ds.ReplyData) bool {
			td, err := rd.TwigMetadata()
			if err != nil {
				return false
			}
			if _, ok := td.Values[twig.Key{Name: "invisible", Version: 1}]; ok {
				return false
			}
			if expired, err := expiration.IsExpired(rd.Reply); err != nil || expired {
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
			}()
		})
		c.MessageList.ScrollToEnd = true
		c.MessageList.Position.BeforeEnd = false
		c.loadMoreHistory()
	}()
	return c
}

func (c *ReplyListView) Filtered() bool {
	return c.FilterState != Off
}

func (c *ReplyListView) HandleIntent(intent Intent) {}

func (c *ReplyListView) BecomeVisible() {
}

func (c *ReplyListView) NavItem() *materials.NavItem {
	return &materials.NavItem{
		Name: "Messages",
		Icon: icons.ChatIcon,
	}
}

func (c *ReplyListView) AppBarData() (bool, string, []materials.AppBarAction, []materials.OverflowAction) {
	th := c.Theme().Current().Theme
	return true, "Messages", []materials.AppBarAction{
			materials.SimpleIconAction(
				th,
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

func (c *ReplyListView) getContextualActions() ([]materials.AppBarAction, []materials.OverflowAction) {
	th := c.Theme().Current().Theme
	return []materials.AppBarAction{
		materials.SimpleIconAction(
			th,
			&c.CopyReplyButton,
			icons.CopyIcon,
			materials.OverflowAction{
				Name: "Copy reply text",
				Tag:  &c.CopyReplyButton,
			},
		),
		materials.SimpleIconAction(
			th,
			&c.CreateReplyButton,
			icons.ReplyIcon,
			materials.OverflowAction{
				Name: "Reply to selected",
				Tag:  &c.CreateReplyButton,
			},
		),
	}, []materials.OverflowAction{}
}

func (c *ReplyListView) triggerReplyContextMenu(gtx layout.Context) {
	actions, overflow := c.getContextualActions()
	c.manager.RequestContextualBar(gtx, "Message Operations", actions, overflow)
}

func (c *ReplyListView) dismissReplyContextMenu(gtx layout.Context) {
	c.manager.DismissContextualBar(gtx)
}

func (c *ReplyListView) moveFocusUp() {
	c.moveFocus(-1)
}

func (c *ReplyListView) moveFocusDown() {
	c.moveFocus(1)
}

func (c *ReplyListView) moveFocus(indexIncrement int) {
	if c.Focused == nil {
		return
	}
	currentIndex := c.AlphaReplyList.IndexForID(c.Focused.ID())
	if currentIndex < 0 {
		return
	}
	c.AlphaReplyList.WithReplies(func(replies []ds.ReplyData) {
		for {
			currentIndex += indexIncrement
			if currentIndex >= len(replies) || currentIndex < 0 {
				break
			}
			status := c.statusOf(replies[currentIndex].Reply)
			if c.shouldFilter(status) {
				continue
			}
			c.FocusTracker.SetFocus(replies[currentIndex].Reply)
			c.ensureFocusedVisible(currentIndex)
			break
		}
	})
}

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

func (c *ReplyListView) moveFocusEnd(replies []ds.ReplyData) {
	if len(replies) < 1 {
		return
	}
	c.SetFocus(replies[len(replies)-1])
	c.requestKeyboardFocus()
	c.MessageList.Position.BeforeEnd = false
}

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

func (c *ReplyListView) refreshNodeStatus(gtx C) {
	if c.FocusTracker.RefreshNodeStatus(c.Arbor().Store()) {
		c.MessageList.Animation.Start(gtx.Now)
	}
}

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

func (c *ReplyListView) copyFocused(gtx layout.Context) {
	reply := c.Focused
	clipboard.WriteOp{
		Text: string(reply.(*forest.Reply).Content.Blob),
	}.Add(gtx.Ops)
}

func (c *ReplyListView) startReply() {
	reply := c.Focused
	var data ds.ReplyData
	data.Reply = reply.(*forest.Reply)
	author, _, err := c.Arbor().Store().GetIdentity(&data.Reply.Author)
	if err != nil {
		log.Printf("failed looking up select message author: %v", err)
	} else {
		data.Author = author.(*forest.Identity)
	}
	c.Composer.StartReply(data)
}

func (c *ReplyListView) sendReply() {
	replyText := c.Composer.Text()
	if replyText == "" {
		return
	}
	var newReplies []*forest.Reply
	var author *forest.Identity
	var parent forest.Node

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
		parent = c.ReplyingTo.Reply
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
		c.SetFocus(reply)
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
				clickedOnFocused := handler.Hash.Equals(c.Focused.ID())
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

func (c *ReplyListView) startConversation() {
	c.Composer.StartConversation()
}

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
				case " ", "F":
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
}

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

func (c *ReplyListView) resetReplyState() {
	c.Composer.Reset()
}

func (c *ReplyListView) statusOf(reply *forest.Reply) sprigWidget.ReplyStatus {
	if c.Focused == nil {
		return sprigWidget.None
	}
	if c.Focused != nil && reply.ID().Equals(c.Focused.ID()) {
		return sprigWidget.Selected
	}
	for _, id := range c.Ancestry {
		if id.Equals(reply.ID()) {
			return sprigWidget.Ancestor
		}
	}
	for _, id := range c.Descendants {
		if id.Equals(reply.ID()) {
			return sprigWidget.Descendant
		}
	}
	if reply.Depth == 1 {
		return sprigWidget.ConversationRoot
	}
	if c.Conversation != nil && !c.Conversation.Equals(fields.NullHash()) {
		if c.Conversation.Equals(&reply.ConversationID) {
			return sprigWidget.Sibling
		}
	}
	return sprigWidget.None
}

func (c *ReplyListView) shouldDisplayEditor() bool {
	return c.Composer.Composing()
}

func (c *ReplyListView) hideEditor() {
	c.Composer.Reset()
	c.requestKeyboardFocus()
}

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

func (c *ReplyListView) shouldFilter(status sprigWidget.ReplyStatus) bool {
	switch c.FilterState {
	case Conversation:
		return status == sprigWidget.None || status == sprigWidget.ConversationRoot
	case Message:
		return status == sprigWidget.Sibling || status == sprigWidget.None || status == sprigWidget.ConversationRoot
	default:
		return false
	}
}

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
		ml.Prefixes = []layout.Widget{
			func(gtx C) D {
				return layout.Center.Layout(gtx, func(gtx C) D {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx C) D {
						return material.Button(th.Theme, &c.LoadMoreHistoryButton, "Load more history").Layout(gtx)
					})
				})
			},
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

func (c *ReplyListView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
