package main

import (
	"image/color"
	"log"

	"gioui.org/f32"
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
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigWidget "git.sr.ht/~whereswaldon/sprig/widget"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type ReplyListView struct {
	manager ViewManager

	*Settings
	*ArborState
	*sprigTheme.Theme

	DeselectButton  widget.Clickable
	CopyReplyButton widget.Clickable

	ReplyList    layout.List
	ReplyStates  []sprigWidget.Reply
	Selected     *fields.QualifiedHash
	Ancestry     []*fields.QualifiedHash
	Descendants  []*fields.QualifiedHash
	Conversation *fields.QualifiedHash
	// Whether the Ancestry and Descendants need to be regenerated because the
	// contents of the replylist changed
	StateRefreshNeeded bool

	CreatingConversation     bool
	ReplyingTo               *forest.Reply
	ReplyingToAuthor         *forest.Identity
	ReplyEditor              widget.Editor
	FilterButton             widget.Clickable
	CancelReplyButton        widget.Clickable
	CreateReplyButton        widget.Clickable
	SendReplyButton          widget.Clickable
	PasteIntoReplyButton     widget.Clickable
	CreateConversationButton widget.Clickable
	CommunityChoice          widget.Enum
	CommunityList            layout.List

	// Filtered determines whether or not the visible nodes should be
	// filtered to only those related to the selected node
	Filtered          bool
	PrefilterPosition layout.Position
}

var _ View = &ReplyListView{}

func NewReplyListView(settings *Settings, arborState *ArborState, theme *sprigTheme.Theme) View {
	c := &ReplyListView{
		Settings:   settings,
		ArborState: arborState,
		Theme:      theme,
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
	}, []materials.OverflowAction{}
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
		materials.AppBarAction{
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
	}, []materials.OverflowAction{}
}

func (c *ReplyListView) triggerReplyContextMenu(gtx layout.Context) {
	actions, overflow := c.getContextualActions()
	c.manager.RequestContextualBar(gtx, "Message Operations", actions, overflow)
}

func (c *ReplyListView) dismissReplyContextMenu(gtx layout.Context) {
	c.manager.DismissContextualBar(gtx)
}

func (c *ReplyListView) Update(gtx layout.Context) {
	overflowTag := c.manager.SelectedOverflowTag()
	for i := range c.ReplyStates {
		clickHandler := &c.ReplyStates[i]
		if clickHandler.Clicked() {
			log.Printf("clicked %s", clickHandler.Reply)
			c.triggerReplyContextMenu(gtx)
			if c.Selected == nil || !clickHandler.Reply.Equals(c.Selected) {
				c.StateRefreshNeeded = true
				c.Selected = clickHandler.Reply
				reply, _, _ := c.ArborState.SubscribableStore.Get(clickHandler.Reply)
				c.Conversation = &reply.(*forest.Reply).ConversationID
			} else {
				c.dismissReplyContextMenu(gtx)
				c.Selected = nil
				c.Filtered = false
			}
		}
	}
	if c.StateRefreshNeeded && c.Selected != nil {
		c.StateRefreshNeeded = false
		c.Ancestry, _ = c.ArborState.SubscribableStore.AncestryOf(c.Selected)
		c.Descendants, _ = c.ArborState.SubscribableStore.DescendantsOf(c.Selected)
	}
	if c.DeselectButton.Clicked() {
		c.Selected = nil
	}
	if c.FilterButton.Clicked() || overflowTag == &c.FilterButton {
		if c.Filtered {
			c.ReplyList.Position = c.PrefilterPosition
		} else {
			c.PrefilterPosition = c.ReplyList.Position
		}
		c.Filtered = !c.Filtered
		c.manager.DismissOverflow(gtx)
	}
	if c.Selected != nil && (c.CopyReplyButton.Clicked() || overflowTag == &c.CopyReplyButton) {
		reply, _, err := c.ArborState.SubscribableStore.Get(c.Selected)
		if err != nil {
			log.Printf("failed looking up selected message: %v", err)
		} else {
			c.manager.UpdateClipboard(string(reply.(*forest.Reply).Content.Blob))
		}
		c.manager.DismissOverflow(gtx)
	}
	if c.PasteIntoReplyButton.Clicked() {
		c.manager.RequestClipboardPaste()
	}
	if c.Selected != nil && (c.CreateReplyButton.Clicked() || overflowTag == &c.CreateReplyButton) {
		reply, _, err := c.ArborState.SubscribableStore.Get(c.Selected)
		if err != nil {
			log.Printf("failed looking up selected message: %v", err)
		} else {
			c.ReplyingTo = reply.(*forest.Reply)
			author, _, err := c.ArborState.SubscribableStore.GetIdentity(&c.ReplyingTo.Author)
			if err != nil {
				log.Printf("failed looking up select message author: %v", err)
			} else {
				c.ReplyingToAuthor = author.(*forest.Identity)
			}
		}
		c.manager.DismissOverflow(gtx)
	}
	if c.CreateConversationButton.Clicked() || overflowTag == &c.CreateConversationButton {
		c.CreatingConversation = true
		c.manager.DismissOverflow(gtx)
	}
	if c.CancelReplyButton.Clicked() {
		c.resetReplyState()
	}
	if c.SendReplyButton.Clicked() {
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
				reply, err := nodeBuilder.NewReply(c.ReplyingTo, c.ReplyEditor.Text(), []byte{})
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

}

func (c *ReplyListView) resetReplyState() {
	c.ReplyingTo = nil
	c.ReplyingToAuthor = nil
	c.CreatingConversation = false
	c.ReplyEditor.SetText("")
}

func (c *ReplyListView) statusOf(reply *forest.Reply) sprigTheme.ReplyStatus {
	if c.Selected == nil {
		return sprigTheme.None
	}
	if c.Selected != nil && reply.ID().Equals(c.Selected) {
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
	if c.Conversation != nil && !c.Conversation.Equals(fields.NullHash()) {
		if c.Conversation.Equals(&reply.ConversationID) {
			return sprigTheme.Sibling
		}
	}
	return sprigTheme.None
}

func (c *ReplyListView) Layout(gtx layout.Context) layout.Dimensions {
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
					if c.ReplyingTo != nil || c.CreatingConversation {
						return c.layoutEditor(gtx)
					}
					return layout.Dimensions{}
				}),
			)
		}),
	)
}

func (c *ReplyListView) layoutReplyList(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min = gtx.Constraints.Max

	theme := c.Theme.Theme
	stateIndex := 0
	var dims layout.Dimensions
	c.ArborState.ReplyList.WithReplies(func(replies []*forest.Reply) {
		dims = c.ReplyList.Layout(gtx, len(replies), func(gtx layout.Context, index int) layout.Dimensions {
			if stateIndex >= len(c.ReplyStates) {
				c.ReplyStates = append(c.ReplyStates, sprigWidget.Reply{})
			}
			state := &c.ReplyStates[stateIndex]
			reply := replies[index]
			collapseMetadata := false
			if index > 0 {
				if replies[index-1].Author.Equals(&reply.Author) && replies[index-1].ID().Equals(reply.ParentID()) {
					collapseMetadata = true
				}
			}
			authorNode, found, err := c.ArborState.SubscribableStore.GetIdentity(&reply.Author)
			if err != nil || !found {
				log.Printf("failed finding author %s for node %s", &reply.Author, reply.ID())
			}
			var community *forest.Community
			author := authorNode.(*forest.Identity)
			sideInset := unit.Dp(3)
			var leftInset unit.Value

			status := c.statusOf(reply)
			switch status {
			case sprigTheme.Selected:
				leftInset = unit.Dp(15)
				collapseMetadata = false
				communityNode, found, err := c.ArborState.SubscribableStore.GetCommunity(&reply.CommunityID)
				if err != nil || !found {
					log.Printf("failed finding community %s for node %s", &reply.CommunityID, reply.ID())
				}
				community = communityNode.(*forest.Community)
			case sprigTheme.Ancestor:
				leftInset = unit.Dp(15)
			case sprigTheme.Descendant:
				leftInset = unit.Dp(30)
			case sprigTheme.Sibling:
				leftInset = sideInset
			default:
				leftInset = sideInset
			}
			if c.Filtered && (status == sprigTheme.Sibling || status == sprigTheme.None) {
				// do not render
				return layout.Dimensions{}
			}
			stateIndex++
			return layout.Flex{}.Layout(gtx,
				layout.Flexed(1, func(gtx C) D {
					extraWidth := gtx.Px(unit.Dp(36))
					messageWidth := gtx.Constraints.Max.X - extraWidth
					return layout.Stack{}.Layout(gtx,
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
								Right:  sideInset,
							}.Layout(gtx, func(gtx C) D {
								gtx.Constraints.Max.X = messageWidth
								replyWidget := sprigTheme.Reply(c.Theme, status)
								replyWidget.CollapseMetadata = collapseMetadata
								return replyWidget.Layout(gtx, reply, author, community)
							})
						}),
						layout.Expanded(func(gtx C) D {
							dims := state.Clickable.Layout(gtx)
							state.Reply = reply.ID()
							return dims
						}),
					)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
						if status != sprigTheme.Selected {
							return D{}
						}
						replyButton := material.IconButton(theme, &c.CreateReplyButton, icons.ReplyIcon)
						replyButton.Size = unit.Dp(20)
						replyButton.Inset = layout.UniformInset(unit.Dp(9))
						replyButton.Background = c.Theme.Secondary.Light
						replyButton.Color = c.Theme.Background.Dark
						return replyButton.Layout(gtx)
					})
				}),
			)
		})
	})
	return dims
}

func (c *ReplyListView) layoutEditor(gtx layout.Context) layout.Dimensions {
	theme := c.Theme.Theme
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
									return material.Body1(theme, "New Conversation in:").Layout(gtx)
								}
								return material.Body1(theme, "Replying to:").Layout(gtx)

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
											radio := material.RadioButton(theme, &c.CommunityChoice, community.ID().String(), string(community.Name.Blob))
											radio.IconColor = c.Theme.Secondary.Default
											return radio.Layout(gtx)
										})
									})
									return dims
								}
								reply := sprigTheme.Reply(c.Theme, sprigTheme.None)
								reply.Highlight = c.Theme.Primary.Default
								reply.MaxLines = 5
								return reply.Layout(gtx, c.ReplyingTo, c.ReplyingToAuthor, nil)
							})
						}),
						layout.Rigid(func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								cancelButton := material.IconButton(theme, &c.CancelReplyButton, icons.CancelReplyIcon)
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
								pasteButton := material.IconButton(theme, &c.PasteIntoReplyButton, icons.PasteIcon)
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
										return layout.UniformInset(unit.Dp(6)).Layout(gtx,
											material.Editor(theme, &c.ReplyEditor, "type your reply here").Layout,
										)
									}),
								)
							})
						}),
						layout.Rigid(func(gtx C) D {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
								sendButton := material.IconButton(theme, &c.SendReplyButton, icons.SendReplyIcon)
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
