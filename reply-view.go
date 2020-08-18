package main

import (
	"image"
	"image/color"
	"log"
	"runtime"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/materials"
	"git.sr.ht/~whereswaldon/sprig/anim"
	"git.sr.ht/~whereswaldon/sprig/ds"
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigWidget "git.sr.ht/~whereswaldon/sprig/widget"
	"git.sr.ht/~whereswaldon/sprig/widget/theme"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type ReplyListView struct {
	manager ViewManager

	*Settings
	*ArborState
	*sprigTheme.Theme

	CopyReplyButton widget.Clickable

	ReplyList       layout.List
	ReplyStates     []sprigWidget.Reply
	ReplyAnimations map[*forest.Reply]*theme.ReplyAnimationState
	ReplyAnim       anim.Normal
	Focused         *fields.QualifiedHash
	Ancestry        []*fields.QualifiedHash
	Descendants     []*fields.QualifiedHash
	Conversation    *fields.QualifiedHash
	// Whether the Ancestry and Descendants need to be regenerated because the
	// contents of the replylist changed
	StateRefreshNeeded bool

	CreatingConversation                bool
	ReplyingTo                          ds.ReplyData
	ReplyEditor                         widget.Editor
	FilterButton                        widget.Clickable
	CancelReplyButton                   widget.Clickable
	CreateReplyButton                   widget.Clickable
	SendReplyButton                     widget.Clickable
	PasteIntoReplyButton                widget.Clickable
	CreateConversationButton            widget.Clickable
	JumpToBottomButton, JumpToTopButton widget.Clickable
	CommunityChoice                     widget.Enum
	CommunityList                       layout.List

	// Filtered determines whether or not the visible nodes should be
	// filtered to only those related to the selected node
	Filtered          bool
	PrefilterPosition layout.Position

	ShouldRequestKeyboardFocus bool
}

var _ View = &ReplyListView{}

func NewReplyListView(settings *Settings, arborState *ArborState, th *sprigTheme.Theme) View {
	c := &ReplyListView{
		Settings:   settings,
		ArborState: arborState,
		Theme:      th,
		ReplyAnim: anim.Normal{
			Duration: time.Millisecond * 100,
		},
		ReplyAnimations: make(map[*forest.Reply]*theme.ReplyAnimationState),
	}
	c.ReplyList.Axis = layout.Vertical
	// ensure that we are notified when we need to refresh the state of visible nodes
	c.ArborState.SubscribableStore.SubscribeToNewMessages(func(forest.Node) {
		c.StateRefreshNeeded = true
	})
	c.ReplyList.ScrollToEnd = true
	c.ReplyList.Position.BeforeEnd = false
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
	th := c.Theme.Theme
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
		}
}

func (c *ReplyListView) HandleClipboard(contents string) {
	c.ReplyEditor.Insert(contents)
}

func (c *ReplyListView) getContextualActions() ([]materials.AppBarAction, []materials.OverflowAction) {
	th := c.Theme.Theme
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
	if c.Focused == nil {
		return
	}
	currentIndex := c.ArborState.ReplyList.IndexForID(c.Focused)
	if currentIndex < 0 {
		return
	}
	c.ArborState.ReplyList.WithReplies(func(replies []ds.ReplyData) {
		if currentIndex <= 0 {
			return
		}
		nextIndex := currentIndex - 1
		c.Focused = replies[nextIndex].Reply.ID()
		c.StateRefreshNeeded = true
		c.ensureFocusedVisible(nextIndex)
	})
}

func (c *ReplyListView) moveFocusDown() {
	if c.Focused == nil {
		return
	}
	currentIndex := c.ArborState.ReplyList.IndexForID(c.Focused)
	if currentIndex < 0 {
		return
	}
	c.ArborState.ReplyList.WithReplies(func(replies []ds.ReplyData) {
		if currentIndex >= len(replies)-1 {
			return
		}
		nextIndex := currentIndex + 1
		c.Focused = replies[nextIndex].Reply.ID()
		c.StateRefreshNeeded = true
		c.ensureFocusedVisible(nextIndex)
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

func (c *ReplyListView) refreshNodeStatus(gtx C) {
	if c.Focused != nil {
		c.StateRefreshNeeded = false
		c.Ancestry, _ = c.ArborState.SubscribableStore.AncestryOf(c.Focused)
		c.Descendants, _ = c.ArborState.SubscribableStore.DescendantsOf(c.Focused)
		c.ReplyAnim.Start(gtx.Now)
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
	reply, _, err := c.ArborState.SubscribableStore.Get(c.Focused)
	if err != nil {
		log.Printf("failed looking up selected message: %v", err)
	} else {
		c.manager.UpdateClipboard(string(reply.(*forest.Reply).Content.Blob))
	}
}

func (c *ReplyListView) startReply() {
	reply, _, err := c.ArborState.SubscribableStore.Get(c.Focused)
	if err != nil {
		log.Printf("failed looking up selected message: %v", err)
	} else {
		c.ReplyingTo.Reply = reply.(*forest.Reply)
		author, _, err := c.ArborState.SubscribableStore.GetIdentity(&c.ReplyingTo.Reply.Author)
		if err != nil {
			log.Printf("failed looking up select message author: %v", err)
		} else {
			c.ReplyingTo.Author = author.(*forest.Identity)
		}
	}
	c.ReplyEditor.Focus()
}

func (c *ReplyListView) sendReply() {
	var newReply *forest.Reply
	var author *forest.Identity
	if c.CreatingConversation {
		if c.CommunityChoice.Value != "" {
			var chosen *forest.Community
			chosenString := c.CommunityChoice.Value
			c.ArborState.CommunityList.WithCommunities(func(communities []*forest.Community) {
				for _, community := range communities {
					if community.ID().String() == chosenString {
						chosen = community
						break
					}
				}
			})
			nodeBuilder, err := c.Settings.Builder()
			if err != nil {
				log.Printf("failed acquiring node builder: %v", err)
			} else {
				author = nodeBuilder.User
				convo, err := nodeBuilder.NewReply(chosen, c.ReplyEditor.Text(), []byte{})
				if err != nil {
					log.Printf("failed creating new conversation: %v", err)
				} else {
					newReply = convo
				}
			}
		}
	} else {
		nodeBuilder, err := c.Settings.Builder()
		if err != nil {
			log.Printf("failed acquiring node builder: %v", err)
		} else {
			author = nodeBuilder.User
			reply, err := nodeBuilder.NewReply(c.ReplyingTo.Reply, c.ReplyEditor.Text(), []byte{})
			if err != nil {
				log.Printf("failed building reply: %v", err)
			} else {
				newReply = reply
			}
		}
	}
	if newReply != nil {
		go func() {
			if err := c.ArborState.SubscribableStore.Add(author); err != nil {
				log.Printf("failed adding replying identity to store: %v", err)
				return
			}
			if err := c.ArborState.SubscribableStore.Add(newReply); err != nil {
				log.Printf("failed adding reply to store: %v", err)
				return
			}
		}()
		c.resetReplyState()
	}
}

func (c *ReplyListView) processMessagePointerEvents(gtx C) {
	for i := range c.ReplyStates {
		clickHandler := &c.ReplyStates[i]
		if clickHandler.Clicked() {
			c.requestKeyboardFocus()
			clickedOnFocused := clickHandler.Reply.Equals(c.Focused)
			if !clickedOnFocused {
				c.StateRefreshNeeded = true
				c.Focused = clickHandler.Reply
				reply, _, _ := c.ArborState.SubscribableStore.Get(clickHandler.Reply)
				c.Conversation = &reply.(*forest.Reply).ConversationID
				c.dismissReplyContextMenu(gtx)
			} else {
				c.triggerReplyContextMenu(gtx)
			}
		}
	}
}

func (c *ReplyListView) startConversation() {
	c.CreatingConversation = true
	c.ReplyEditor.Focus()
}

func (c *ReplyListView) Update(gtx layout.Context) {
	const jumpNone, jumpStart, jumpEnd = 0, 1, 2
	jump := jumpNone
	for _, event := range gtx.Events(c) {
		switch event := event.(type) {
		case key.Event:
			switch event.Name {
			case "K", key.NameUpArrow:
				c.moveFocusUp()
			case "J", key.NameDownArrow:
				c.moveFocusDown()
			case key.NameHome:
				jump = jumpStart
			case "G":
				if !event.Modifiers.Contain(key.ModShift) {
					jump = jumpStart
					break
				}
				fallthrough
			case key.NameEnd:
				jump = jumpEnd
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
		jump = jumpEnd
	}
	if overflowTag == &c.JumpToTopButton || c.JumpToTopButton.Clicked() {
		jump = jumpStart
	}
	if jump != jumpNone {
		c.ArborState.WithReplies(func(replies []ds.ReplyData) {
			switch jump {
			case jumpStart:
				c.moveFocusStart(replies)
			case jumpEnd:
				c.moveFocusEnd(replies)
			}
		})
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
	key.InputOp{Tag: c, Focus: c.ShouldRequestKeyboardFocus}.Add(gtx.Ops)
	c.ShouldRequestKeyboardFocus = false
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			paintOp := paint.ColorOp{Color: c.Theme.Background.Default}
			paintOp.Add(gtx.Ops)
			paint.PaintOp{Rect: f32.Rectangle{
				Max: layout.FPt(gtx.Constraints.Max),
			}}.Add(gtx.Ops)
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

func (c *ReplyListView) layoutReplyList(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min = gtx.Constraints.Max

	th := c.Theme.Theme
	stateIndex := 0
	var dims layout.Dimensions
	var replyListLen int
	c.ArborState.ReplyList.WithReplies(func(replies []ds.ReplyData) {
		replyListLen = len(replies)
		if c.Focused == nil && len(replies) > 0 {
			c.moveFocusEnd(replies)
		}
		dims = c.ReplyList.Layout(gtx, len(replies), func(gtx layout.Context, index int) layout.Dimensions {
			if stateIndex >= len(c.ReplyStates) {
				c.ReplyStates = append(c.ReplyStates, sprigWidget.Reply{})
			}
			state := &c.ReplyStates[stateIndex]
			reply := replies[index]
			collapseMetadata := false
			// if index > 0 {
			// 	if replies[index-1].Reply.Author.Equals(&reply.Reply.Author) && replies[index-1].ID().Equals(reply.ParentID()) {
			// 		collapseMetadata = true
			// 	}
			// }

			status := c.statusOf(reply.Reply)
			if c.Filtered && (status == sprigTheme.Sibling || status == sprigTheme.None || status == sprigTheme.ConversationRoot) {
				// do not render
				return layout.Dimensions{}
			}
			anim, ok := c.ReplyAnimations[reply.Reply]
			if !ok {
				anim = &theme.ReplyAnimationState{
					Normal: &c.ReplyAnim,
					Begin:  status,
				}
				c.ReplyAnimations[reply.Reply] = anim
			}
			if c.ReplyAnim.Animating(gtx) {
				anim.End = status
			} else {
				anim.Begin = status
				anim.End = status
			}
			leftInset := interpolateInset(anim, c.ReplyAnim.Progress(gtx))
			stateIndex++
			return layout.Stack{}.Layout(gtx,
				layout.Stacked(func(gtx C) D {
					extraWidth := gtx.Px(unit.Dp(5 * insetUnit))
					messageWidth := gtx.Constraints.Max.X - extraWidth
					dims := layout.Stack{}.Layout(gtx,
						layout.Stacked(func(gtx C) D {
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							margin := unit.Dp(3)
							if collapseMetadata {
								margin = unit.Dp(0)
							}
							return layout.Inset{
								Top:    margin,
								Bottom: unit.Dp(3),
								Left:   leftInset,
							}.Layout(gtx, func(gtx C) D {
								gtx.Constraints.Max.X = messageWidth
								replyWidget := sprigTheme.Reply(c.Theme, anim, reply)
								replyWidget.CollapseMetadata = collapseMetadata
								return replyWidget.Layout(gtx)
							})
						}),
						layout.Expanded(func(gtx C) D {
							dims := state.Clickable.Layout(gtx)
							state.Reply = reply.ID()
							return dims
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
							Left:  unit.Dp(8),
							Right: unit.Dp(12),
						}.Layout(gtx, func(gtx C) D {
							replyButton := material.IconButton(th, &c.CreateReplyButton, icons.ReplyIcon)
							replyButton.Size = unit.Dp(20)
							replyButton.Inset = layout.UniformInset(unit.Dp(9))
							replyButton.Background = c.Theme.Secondary.Light
							replyButton.Color = c.Theme.Background.Dark
							return replyButton.Layout(gtx)
						})
					})
				}),
			)
		})
	})
	progress := float32(c.ReplyList.Position.First) / float32(replyListLen)
	layout.NE.Layout(gtx, func(gtx C) D {
		indicatorHeightDp := unit.Dp(16)
		indicatorHeightPx := gtx.Px(indicatorHeightDp)
		heightDp := float32(gtx.Constraints.Max.Y) / gtx.Metric.PxPerDp
		width := gtx.Px(unit.Dp(8))
		top := unit.Dp(heightDp * progress)
		if top.V+indicatorHeightDp.V > heightDp {
			top = unit.Dp(heightDp - indicatorHeightDp.V)
		}
		rr := float32(gtx.Px(unit.Dp(4)))
		return layout.Inset{
			Top:    top,
			Right:  unit.Dp(2),
			Bottom: unit.Dp(2),
		}.Layout(gtx, func(gtx C) D {
			bg := c.Theme.Background.Dark
			bg.A = 200
			return sprigTheme.DrawRect(gtx, bg, f32.Point{X: float32(width), Y: float32(indicatorHeightPx)}, rr)
		})
	})

	return dims
}

func (c *ReplyListView) layoutEditor(gtx layout.Context) layout.Dimensions {
	th := c.Theme.Theme
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			paintOp := paint.ColorOp{Color: c.Theme.Primary.Light}
			paintOp.Add(gtx.Ops)
			paint.PaintOp{Rect: f32.Rectangle{
				Max: f32.Point{
					X: float32(gtx.Constraints.Max.X),
					Y: float32(gtx.Constraints.Max.Y),
				},
			}}.Add(gtx.Ops)
			return layout.Dimensions{}
		}),
		layout.Stacked(func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								if c.CreatingConversation {
									return material.Body1(th, "New Conversation in:").Layout(gtx)
								}
								return material.Body1(th, "Replying to:").Layout(gtx)

							})
						}),
						layout.Flexed(1, func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								if c.CreatingConversation {
									var dims layout.Dimensions
									c.ArborState.CommunityList.WithCommunities(func(comms []*forest.Community) {
										dims = c.CommunityList.Layout(gtx, len(comms), func(gtx layout.Context, index int) layout.Dimensions {
											community := comms[index]
											if c.CommunityChoice.Value == "" && index == 0 {
												c.CommunityChoice.Value = community.ID().String()
											}
											radio := material.RadioButton(th, &c.CommunityChoice, community.ID().String(), string(community.Name.Blob))
											radio.IconColor = c.Theme.Secondary.Default
											return radio.Layout(gtx)
										})
									})
									return dims
								}
								reply := sprigTheme.Reply(c.Theme, &theme.ReplyAnimationState{
									Normal: &c.ReplyAnim,
								}, c.ReplyingTo)
								reply.Highlight = c.Theme.Primary.Default
								reply.MaxLines = 5
								return reply.Layout(gtx)
							})
						}),
						layout.Rigid(func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								cancelButton := material.IconButton(th, &c.CancelReplyButton, icons.CancelReplyIcon)
								cancelButton.Size = unit.Dp(20)
								cancelButton.Inset = layout.UniformInset(unit.Dp(4))
								return cancelButton.Layout(gtx)
							})
						}),
					)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								pasteButton := material.IconButton(th, &c.PasteIntoReplyButton, icons.PasteIcon)
								pasteButton.Inset = layout.UniformInset(unit.Dp(4))
								pasteButton.Size = unit.Dp(20)
								return pasteButton.Layout(gtx)
							})
						}),
						layout.Flexed(1, func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								return layout.Stack{}.Layout(gtx,
									layout.Expanded(func(gtx C) D {
										stack := op.Push(gtx.Ops)
										paintOp := paint.ColorOp{Color: c.Theme.Background.Light}
										paintOp.Add(gtx.Ops)
										bounds := f32.Rectangle{
											Max: f32.Point{
												X: float32(gtx.Constraints.Max.X),
												Y: float32(gtx.Constraints.Min.Y),
											},
										}
										radii := float32(gtx.Px(unit.Dp(5)))
										clip.RRect{
											Rect: bounds,
											NW:   radii,
											NE:   radii,
											SE:   radii,
											SW:   radii,
										}.Add(gtx.Ops)
										paint.PaintOp{Rect: bounds}.Add(gtx.Ops)
										stack.Pop()
										return layout.Dimensions{}
									}),
									layout.Stacked(func(gtx C) D {
										return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
											editor := material.Editor(th, &c.ReplyEditor, "type your reply here")
											editor.Editor.Submit = true
											return editor.Layout(gtx)
										})
									}),
								)
							})
						}),
						layout.Rigid(func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								sendButton := material.IconButton(th, &c.SendReplyButton, icons.SendReplyIcon)
								sendButton.Size = unit.Dp(20)
								sendButton.Inset = layout.UniformInset(unit.Dp(4))
								sendButton.Background = c.Theme.Primary.Default
								sendButton.Color = c.Theme.Background.Light
								return sendButton.Layout(gtx)
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
