package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"git.sr.ht/~athorp96/forest-ex/expiration"
	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/forest-go/twig"

	"git.sr.ht/~whereswaldon/materials"
	"git.sr.ht/~whereswaldon/scroll"

	"git.sr.ht/~whereswaldon/sprig/anim"
	"git.sr.ht/~whereswaldon/sprig/core"
	"git.sr.ht/~whereswaldon/sprig/ds"
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigWidget "git.sr.ht/~whereswaldon/sprig/widget"
	"git.sr.ht/~whereswaldon/sprig/widget/theme"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type ReplyListView struct {
	manager ViewManager

	core.App

	CopyReplyButton widget.Clickable

	ds.AlphaReplyList

	ReplyList    layout.List
	States       *States
	Animations   *Animation
	Focused      *fields.QualifiedHash
	Ancestry     []*fields.QualifiedHash
	Descendants  []*fields.QualifiedHash
	Conversation *fields.QualifiedHash
	// Whether the Ancestry and Descendants need to be regenerated because the
	// contents of the replylist changed
	StateRefreshNeeded bool

	CreatingConversation                bool
	ReplyingTo                          ds.ReplyData
	ReplyEditor                         materials.TextField
	FilterButton                        widget.Clickable
	CancelReplyButton                   widget.Clickable
	CreateReplyButton                   widget.Clickable
	SendReplyButton                     widget.Clickable
	PasteIntoReplyButton                widget.Clickable
	CreateConversationButton            widget.Clickable
	JumpToBottomButton, JumpToTopButton widget.Clickable
	CommunityChoice                     widget.Enum
	CommunityList                       layout.List

	LoadMoreHistoryButton widget.Clickable
	// how many nodes of history does the view want
	HistoryRequestCount int

	scroll.Scrollable

	// Filtered determines whether or not the visible nodes should be
	// filtered to only those related to the selected node
	Filtered          bool
	PrefilterPosition layout.Position

	ShouldRequestKeyboardFocus bool

	// Cache the number of replies during update.
	replyCount int
	// Maximum number of visible replies encountered.
	maxRepliesVisible int
}

var _ View = &ReplyListView{}

func NewReplyListView(app core.App) View {
	c := &ReplyListView{
		App: app,
		Animations: &Animation{
			Collection: make(map[*forest.Reply]*theme.ReplyAnimationState),
			Normal: anim.Normal{
				Duration: time.Millisecond * 100,
			},
		},
		HistoryRequestCount: 2048,
		States:              &States{},
	}
	c.AlphaReplyList.FilterWith(func(rd ds.ReplyData) bool {
		td, err := rd.TwigMetadata()
		if err != nil {
			return false
		}
		if _, ok := td.Values[twig.Key{Name: "invisible", Version: 1}]; ok {
			return false
		}
		if ttl, ok := td.Values[expiration.TTLKey()]; ok {
			if expiry, err := expiration.UnmarshalTTL(ttl); err != nil {
				return false
			} else {
				return time.Now().Before(expiry)
			}
		}
		return true
	})
	c.ReplyList.Axis = layout.Vertical
	// ensure that we are notified when we need to refresh the state of visible nodes
	c.Arbor().Store().SubscribeToNewMessages(func(node forest.Node) {
		c.StateRefreshNeeded = true
		go func() {
			var rd ds.ReplyData
			if rd.Populate(node, c.Arbor().Store()) {
				c.AlphaReplyList.Insert(rd)
			}
		}()
	})
	c.ReplyList.ScrollToEnd = true
	c.ReplyList.Position.BeforeEnd = false
	c.loadMoreHistory()
	return c
}

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
				Layout: func(gtx C, bg, fg color.RGBA) D {
					btn := materials.SimpleIconButton(th, &c.FilterButton, icons.FilterIcon)
					btn.Background = bg
					btn.Color = fg
					if c.Filtered {
						btn.Color, btn.Background = btn.Background, btn.Color
					}
					return btn.Layout(gtx)
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

func (c *ReplyListView) HandleClipboard(contents string) {
	c.ReplyEditor.Insert(contents)
}

func (c *ReplyListView) getContextualActions() ([]materials.AppBarAction, []materials.OverflowAction) {
	th := c.Theme().Current().Theme
	return []materials.AppBarAction{
		materials.SimpleIconAction(
			th,
			&c.CreateReplyButton,
			icons.ReplyIcon,
			materials.OverflowAction{
				Name: "Reply to selected",
				Tag:  &c.CreateReplyButton,
			},
		),
		materials.SimpleIconAction(
			th,
			&c.CopyReplyButton,
			icons.CopyIcon,
			materials.OverflowAction{
				Name: "Copy reply text",
				Tag:  &c.CopyReplyButton,
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
	currentIndex := c.AlphaReplyList.IndexForID(c.Focused)
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
			if c.shouldFilter(replies[currentIndex].Reply, status) {
				continue
			}
			c.Focused = replies[currentIndex].Reply.ID()
			c.StateRefreshNeeded = true
			c.ensureFocusedVisible(currentIndex)
			break
		}
	})
}

func (c *ReplyListView) ensureFocusedVisible(focusedIndex int) {
	currentFirst := c.ReplyList.Position.First
	notInFirstFive := currentFirst+5 > focusedIndex
	if currentFirst <= focusedIndex && notInFirstFive {
		return
	}
	c.ReplyList.Position.First = focusedIndex
	if notInFirstFive {
		//		c.ReplyList.Position.First++
	}
	c.ReplyList.Position.Offset = 0
	c.ReplyList.Position.BeforeEnd = true
}

func (c *ReplyListView) moveFocusEnd(replies []ds.ReplyData) {
	if len(replies) < 1 {
		return
	}
	c.Focused = replies[len(replies)-1].ID()
	c.StateRefreshNeeded = true
	c.requestKeyboardFocus()
	c.ReplyList.Position.BeforeEnd = false
}

func (c *ReplyListView) moveFocusStart(replies []ds.ReplyData) {
	if len(replies) < 1 {
		return
	}
	c.Focused = replies[0].ID()
	c.StateRefreshNeeded = true
	c.requestKeyboardFocus()
	c.ReplyList.Position.BeforeEnd = true
	c.ReplyList.Position.First = 0
	c.ReplyList.Position.Offset = 0
}

// reveal the reply at the given index.
func (c *ReplyListView) reveal(index int) {
	if c.replyCount < 1 || index > c.replyCount-1 {
		return
	}
	c.StateRefreshNeeded = true
	c.requestKeyboardFocus()
	c.ReplyList.Position.BeforeEnd = true
	c.ReplyList.Position.First = index
}

func (c *ReplyListView) refreshNodeStatus(gtx C) {
	if c.Focused != nil {
		c.StateRefreshNeeded = false
		c.Ancestry, _ = c.Arbor().Store().AncestryOf(c.Focused)
		c.Descendants, _ = c.Arbor().Store().DescendantsOf(c.Focused)
		c.Animations.Start(gtx.Now)
	}
}

func (c *ReplyListView) toggleFilter() {
	if c.Filtered {
		c.ReplyList.Position = c.PrefilterPosition
	} else {
		c.PrefilterPosition = c.ReplyList.Position
	}
	c.Filtered = !c.Filtered
}

func (c *ReplyListView) copyFocused() {
	reply, _, err := c.Arbor().Store().Get(c.Focused)
	if err != nil {
		log.Printf("failed looking up selected message: %v", err)
	} else {
		c.manager.UpdateClipboard(string(reply.(*forest.Reply).Content.Blob))
	}
}

func (c *ReplyListView) startReply() {
	reply, _, err := c.Arbor().Store().Get(c.Focused)
	if err != nil {
		log.Printf("failed looking up selected message: %v", err)
	} else {
		c.ReplyingTo.Reply = reply.(*forest.Reply)
		author, _, err := c.Arbor().Store().GetIdentity(&c.ReplyingTo.Reply.Author)
		if err != nil {
			log.Printf("failed looking up select message author: %v", err)
		} else {
			c.ReplyingTo.Author = author.(*forest.Identity)
		}
	}
	c.ReplyEditor.Focus()
}

func (c *ReplyListView) sendReply() {
	var newReplies []*forest.Reply
	var author *forest.Identity
	var parent forest.Node

	replyText := c.ReplyEditor.Text()
	replyText = strings.TrimSpace(replyText)

	nodeBuilder, err := c.Settings().Builder()
	if err != nil {
		log.Printf("failed acquiring node builder: %v", err)
	}
	author = nodeBuilder.User
	if c.CreatingConversation {
		if c.CommunityChoice.Value != "" {
			chosenString := c.CommunityChoice.Value
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
			cmd := exec.Command(args[0], append(args[1:], u.String())...)
			if out, err := cmd.CombinedOutput(); err != nil {
				fmt.Printf("opening link: %s %s\n", string(out), err)
			}
		}
	}
	clicked := func(c *widget.Clickable) (widget.Click, bool) {
		clicks := c.Clicks()
		if len(clicks) == 0 {
			return widget.Click{}, false
		}
		return clicks[len(clicks)-1], true
	}
	for i := range c.States.Buffer {
		handler := &c.States.Buffer[i]
		if click, ok := clicked(&handler.Clickable); ok {
			if click.Modifiers.Contain(key.ModCtrl) {
				for _, word := range strings.Fields(handler.Content) {
					tryOpenLink(word)
				}
			} else {
				c.requestKeyboardFocus()
				clickedOnFocused := handler.Hash.Equals(c.Focused)
				if !clickedOnFocused {
					c.StateRefreshNeeded = true
					c.Focused = handler.Hash
					reply, _, _ := c.Arbor().Store().Get(handler.Hash)
					c.Conversation = &reply.(*forest.Reply).ConversationID
					c.dismissReplyContextMenu(gtx)
				} else {
					c.triggerReplyContextMenu(gtx)
				}
			}
		}
	}
}

func (c *ReplyListView) startConversation() {
	c.CreatingConversation = true
	c.ReplyEditor.Focus()
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
					c.copyFocused()
				} else {
					c.startConversation()
				}
			case "V":
				if event.Modifiers.Contain(key.ModCtrl) || (runtime.GOOS == "darwin" && event.Modifiers.Contain(key.ModCommand)) {
					// TODO: move this handling code to the editor somehow, since that's where the paste needs to happen
					c.manager.RequestClipboardPaste()
				}
			case " ", "F":
				c.toggleFilter()
			}
		}
	}
	for _, event := range c.ReplyEditor.Events() {
		if _, ok := event.(widget.SubmitEvent); ok && submitShouldSend {
			c.sendReply()
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
	if c.StateRefreshNeeded {
		c.refreshNodeStatus(gtx)
	}
	if c.FilterButton.Clicked() || overflowTag == &c.FilterButton {
		c.toggleFilter()
	}
	if c.Focused != nil && (c.CopyReplyButton.Clicked() || overflowTag == &c.CopyReplyButton) {
		c.copyFocused()
	}
	if c.PasteIntoReplyButton.Clicked() {
		c.manager.RequestClipboardPaste()
	}
	if c.Focused != nil && (c.CreateReplyButton.Clicked() || overflowTag == &c.CreateReplyButton) {
		c.startReply()
	}
	if c.CreateConversationButton.Clicked() || overflowTag == &c.CreateConversationButton {
		c.startConversation()
	}
	if c.CancelReplyButton.Clicked() {
		c.resetReplyState()
	}
	if c.SendReplyButton.Clicked() {
		c.sendReply()
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
		} else {
			log.Printf("filtering out %s due to Populate() failing", rd.ID())
		}
	}
	if len(populated) < newNodeTarget {
		load()
	}
	c.AlphaReplyList.Insert(populated...)
}

func (c *ReplyListView) resetReplyState() {
	c.ReplyingTo.Reply = nil
	c.CreatingConversation = false
	c.ReplyEditor.SetText("")
}

func (c *ReplyListView) statusOf(reply *forest.Reply) sprigTheme.ReplyStatus {
	if c.Focused == nil {
		return sprigTheme.None
	}
	if c.Focused != nil && reply.ID().Equals(c.Focused) {
		return sprigTheme.Selected
	}
	for _, id := range c.Ancestry {
		if id.Equals(reply.ID()) {
			return sprigTheme.Ancestor
		}
	}
	for _, id := range c.Descendants {
		if id.Equals(reply.ID()) {
			return sprigTheme.Descendant
		}
	}
	if reply.Depth == 1 {
		return sprigTheme.ConversationRoot
	}
	if c.Conversation != nil && !c.Conversation.Equals(fields.NullHash()) {
		if c.Conversation.Equals(&reply.ConversationID) {
			return sprigTheme.Sibling
		}
	}
	return sprigTheme.None
}

func (c *ReplyListView) Layout(gtx layout.Context) layout.Dimensions {
	theme := c.Theme().Current()
	key.InputOp{Tag: c, Focus: c.ShouldRequestKeyboardFocus}.Add(gtx.Ops)
	c.ShouldRequestKeyboardFocus = false
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			sprigTheme.Rect{
				Color: theme.Background.Default,
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
					if c.ReplyingTo.Reply != nil || c.CreatingConversation {
						return c.layoutEditor(gtx)
					}
					return layout.Dimensions{}
				}),
			)
		}),
	)
}

const insetUnit = 12

var (
	defaultInset    = unit.Dp(insetUnit)
	ancestorInset   = unit.Dp(2 * insetUnit)
	selectedInset   = unit.Dp(2 * insetUnit)
	descendantInset = unit.Dp(3 * insetUnit)
)

func insetForStatus(status theme.ReplyStatus) unit.Value {
	switch status {
	case sprigTheme.Selected:
		return selectedInset
	case sprigTheme.Ancestor:
		return ancestorInset
	case sprigTheme.Descendant:
		return descendantInset
	case sprigTheme.Sibling:
		return defaultInset
	default:
		return defaultInset
	}
}

func interpolateInset(anim *theme.ReplyAnimationState, progress float32) unit.Value {
	if progress == 0 {
		return insetForStatus(anim.Begin)
	}
	begin := insetForStatus(anim.Begin).V
	end := insetForStatus(anim.End).V
	return unit.Dp((end-begin)*progress + begin)
}

const buttonWidthDp = 20
const scrollSlotWidthDp = 12

func (c *ReplyListView) shouldFilter(reply *forest.Reply, status sprigTheme.ReplyStatus) bool {
	return c.Filtered && (status == sprigTheme.Sibling || status == sprigTheme.None || status == sprigTheme.ConversationRoot)
}

func (c *ReplyListView) layoutReplyList(gtx layout.Context) layout.Dimensions {
	var (
		dims                 layout.Dimensions
		th                       = c.Theme().Current()
		totalUnfilteredNodes int = 1 + len(c.Ancestry) + len(c.Descendants)
	)
	c.States.Begin()
	gtx.Constraints.Min = gtx.Constraints.Max
	c.AlphaReplyList.WithReplies(func(replies []ds.ReplyData) {
		if c.Focused == nil && len(replies) > 0 {
			c.moveFocusEnd(replies)
		}
		dims = c.ReplyList.Layout(gtx, len(replies)+1, func(gtx layout.Context, index int) layout.Dimensions {
			if index == 0 {
				return layout.Center.Layout(gtx, func(gtx C) D {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx C) D {
						return material.Button(th.Theme, &c.LoadMoreHistoryButton, "Load more history").Layout(gtx)
					})
				})
			}
			// adjust for the fact that we use the first index as a button
			index--
			var (
				reply            = replies[index]
				status           = c.statusOf(reply.Reply)
				anim             = c.Animations.Update(gtx, reply.Reply, status)
				isActive         = c.Status().IsActive(reply.AuthorID())
				collapseMetadata = func() bool {
					// This conflicts with animation feature, so we're removing the feature for now.
					// if index > 0 {
					// 	if replies[index-1].Reply.Author.Equals(&reply.Reply.Author) && replies[index-1].ID().Equals(reply.ParentID()) {
					// 		return true
					// 	}
					// }
					return false
				}()
			)
			if c.shouldFilter(reply.Reply, status) {
				return layout.Dimensions{}
			}
			// Only acquire a state after ensuring the node should be rendered. This allows
			// us to count used states in order to determine how many nodes were rendered.
			var state = c.States.Next()
			return layout.Stack{}.Layout(gtx,
				layout.Stacked(func(gtx C) D {
					var (
						extraWidth   = gtx.Px(unit.Dp(5*insetUnit + sprigTheme.DefaultIconButtonWidthDp + scrollSlotWidthDp))
						messageWidth = gtx.Constraints.Max.X - extraWidth
					)
					dims := layout.Stack{}.Layout(gtx,
						layout.Stacked(func(gtx C) D {
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							return layout.Inset{
								Top: func() unit.Value {
									if collapseMetadata {
										return unit.Dp(0)
									}
									return unit.Dp(3)
								}(),
								Bottom: unit.Dp(3),
								Left:   interpolateInset(anim, c.Animations.Progress(gtx)),
							}.Layout(gtx, func(gtx C) D {
								gtx.Constraints.Max.X = messageWidth
								return sprigTheme.
									Reply(th, anim, reply, isActive).
									HideMetadata(collapseMetadata).
									Layout(gtx)

							})
						}),
						layout.Expanded(func(gtx C) D {
							return state.
								WithHash(reply.ID()).
								WithContent(string(reply.Content.Blob)).
								Layout(gtx)
						}),
					)
					return D{
						Size: image.Point{
							X: gtx.Constraints.Max.X,
							Y: dims.Size.Y,
						},
						Baseline: dims.Baseline,
					}
				}),
				layout.Expanded(func(gtx C) D {
					return layout.E.Layout(gtx, func(gtx C) D {
						if status != sprigTheme.Selected {
							return D{}
						}
						return layout.Inset{
							Right: unit.Dp(scrollSlotWidthDp),
						}.Layout(gtx, func(gtx C) D {
							return material.IconButtonStyle{
								Background: th.Secondary.Light,
								Color:      th.Background.Dark,
								Button:     &c.CreateReplyButton,
								Icon:       icons.ReplyIcon,
								Size:       unit.Dp(sprigTheme.DefaultIconButtonWidthDp),
								Inset:      layout.UniformInset(unit.Dp(9)),
							}.Layout(gtx)
						})
					})
				}),
			)
		})
	})

	totalNodes := func() int {
		if c.Filtered {
			return totalUnfilteredNodes
		}
		return c.replyCount
	}()
	progress := float32(c.ReplyList.Position.First) / float32(c.replyCount)
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
	bar.Color = materials.AlphaMultiply(th.Color.Text, 200)
	bar.Layout(gtx)
	return dims
}

func (c *ReplyListView) layoutEditor(gtx layout.Context) layout.Dimensions {
	var (
		th = c.Theme().Current()
	)
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			sprigTheme.Rect{
				Color: th.Primary.Light,
				Size: f32.Point{
					X: float32(gtx.Constraints.Max.X),
					Y: float32(gtx.Constraints.Max.Y),
				},
			}.Layout(gtx)
			return layout.Dimensions{}
		}),
		layout.Stacked(func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								gtx.Constraints.Max.X = gtx.Px(unit.Dp(30))
								gtx.Constraints.Min.X = gtx.Constraints.Max.X
								if c.CreatingConversation {
									return material.Body1(th.Theme, "In:").Layout(gtx)
								}
								return material.Body1(th.Theme, "Re:").Layout(gtx)

							})
						}),
						layout.Flexed(1, func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								if c.CreatingConversation {
									var dims layout.Dimensions
									c.Arbor().Communities().WithCommunities(func(comms []*forest.Community) {
										dims = c.CommunityList.Layout(gtx, len(comms), func(gtx layout.Context, index int) layout.Dimensions {
											community := comms[index]
											if c.CommunityChoice.Value == "" && index == 0 {
												c.CommunityChoice.Value = community.ID().String()
											}
											radio := material.RadioButton(th.Theme, &c.CommunityChoice, community.ID().String(), string(community.Name.Blob))
											radio.IconColor = th.Secondary.Default
											return radio.Layout(gtx)
										})
									})
									return dims
								}
								reply := sprigTheme.Reply(th, &theme.ReplyAnimationState{
									Normal: &c.Animations.Normal,
								}, c.ReplyingTo, true)
								reply.Highlight = th.Primary.Default
								reply.MaxLines = 5
								return reply.Layout(gtx)
							})
						}),
						layout.Rigid(func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								return sprigTheme.IconButton{
									Theme:  th,
									Button: &c.CancelReplyButton,
									Icon:   icons.CancelReplyIcon,
								}.Layout(gtx)
							})
						}),
					)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								return sprigTheme.IconButton{
									Theme:  th,
									Button: &c.PasteIntoReplyButton,
									Icon:   icons.PasteIcon,
								}.Layout(gtx)
							})
						}),
						layout.Flexed(1, func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								return layout.Stack{}.Layout(gtx,
									layout.Expanded(func(gtx C) D {
										return sprigTheme.Rect{
											Color: th.Background.Light,
											Size: f32.Point{
												X: float32(gtx.Constraints.Max.X),
												Y: float32(gtx.Constraints.Min.Y),
											},
											Radii: float32(gtx.Px(unit.Dp(5))),
										}.Layout(gtx)

									}),
									layout.Stacked(func(gtx C) D {
										return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
											c.ReplyEditor.Submit = true
											return c.ReplyEditor.Layout(gtx, th.Theme, "Compose your reply")
										})
									}),
								)
							})
						}),
						layout.Rigid(func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								return sprigTheme.IconButton{
									Theme:  th,
									Button: &c.SendReplyButton,
									Icon:   icons.SendReplyIcon,
								}.Layout(gtx)
							})
						}),
					)
				}),
			)
		}),
	)
}

func (c *ReplyListView) SetManager(mgr ViewManager) {
	c.manager = mgr
}

// States implements a buffer of reply states such that memory
// is reused each frame, yet grows as the view expands to hold more replies.
type States struct {
	Buffer  []sprigWidget.Reply
	Current int
}

// Begin resets the buffer to the start.
func (s *States) Begin() {
	s.Current = 0
}

func (s *States) Next() *sprigWidget.Reply {
	defer func() { s.Current++ }()
	if s.Current > len(s.Buffer)-1 {
		s.Buffer = append(s.Buffer, sprigWidget.Reply{})
	}
	return &s.Buffer[s.Current]
}

// Animation maintains animation states per reply.
type Animation struct {
	anim.Normal
	Collection map[*forest.Reply]*theme.ReplyAnimationState
}

// Lookup animation state for the given reply.
// If state doesn't exist, it will be created with using `s` as the
// beginning status.
func (a *Animation) Lookup(r *forest.Reply, s sprigTheme.ReplyStatus) *theme.ReplyAnimationState {
	_, ok := a.Collection[r]
	if !ok {
		a.Collection[r] = &theme.ReplyAnimationState{
			Normal: &a.Normal,
			Begin:  s,
		}
	}
	return a.Collection[r]
}

// Update animation state for the given reply.
func (a *Animation) Update(gtx layout.Context, r *forest.Reply, s sprigTheme.ReplyStatus) *theme.ReplyAnimationState {
	anim := a.Lookup(r, s)
	if a.Animating(gtx) {
		anim.End = s
	} else {
		anim.Begin = s
		anim.End = s
	}
	return anim
}
