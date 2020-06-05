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
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigWidget "git.sr.ht/~whereswaldon/sprig/widget"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type ReplyListView struct {
	manager ViewManager

	*Settings
	*ArborState
	*sprigTheme.Theme

	BackButton widget.Clickable

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
	Filtered bool
}

var _ View = &ReplyListView{}

func NewReplyListView(settings *Settings, arborState *ArborState, theme *sprigTheme.Theme) View {
	c := &ReplyListView{
		Settings:   settings,
		ArborState: arborState,
		Theme:      theme,
	}
	// ensure that we are notified when we need to refresh the state of visible nodes
	c.ArborState.SubscribableStore.SubscribeToNewMessages(func(forest.Node) {
		c.StateRefreshNeeded = true
	})
	c.ReplyList.ScrollToEnd = true
	c.ReplyList.Position.BeforeEnd = false
	return c
}

func (c *ReplyListView) HandleClipboard(contents string) {
	c.ReplyEditor.Insert(contents)
}

func (c *ReplyListView) Update(gtx layout.Context) {
	for i := range c.ReplyStates {
		clickHandler := &c.ReplyStates[i]
		if clickHandler.Clicked() {
			log.Printf("clicked %s", clickHandler.Reply)
			if c.Selected == nil || !clickHandler.Reply.Equals(c.Selected) {
				c.StateRefreshNeeded = true
				c.Selected = clickHandler.Reply
				reply, _, _ := c.ArborState.SubscribableStore.Get(clickHandler.Reply)
				c.Conversation = &reply.(*forest.Reply).ConversationID
			} else {
				c.Selected = nil
			}
		}
	}
	if c.StateRefreshNeeded && c.Selected != nil {
		c.StateRefreshNeeded = false
		c.Ancestry, _ = c.ArborState.SubscribableStore.AncestryOf(c.Selected)
		c.Descendants, _ = c.ArborState.SubscribableStore.DescendantsOf(c.Selected)
	}
	if c.BackButton.Clicked() {
		c.manager.RequestViewSwitch(CommunityMenuID)
	}
	if c.DeselectButton.Clicked() {
		c.Selected = nil
	}
	if c.FilterButton.Clicked() {
		c.Filtered = !c.Filtered
	}
	if c.Selected != nil && c.CopyReplyButton.Clicked() {
		reply, _, err := c.ArborState.SubscribableStore.Get(c.Selected)
		if err != nil {
			log.Printf("failed looking up selected message: %v", err)
		} else {
			c.manager.UpdateClipboard(string(reply.(*forest.Reply).Content.Blob))
		}
	}
	if c.PasteIntoReplyButton.Clicked() {
		c.manager.RequestClipboardPaste()
	}
	if c.Selected != nil && c.CreateReplyButton.Clicked() {
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
	}
	if c.CreateConversationButton.Clicked() {
		c.CreatingConversation = true
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

type replyStatus int

const (
	none replyStatus = iota
	sibling
	selected
	ancestor
	descendant
)

func (c *ReplyListView) statusOf(reply *forest.Reply) replyStatus {
	if c.Selected == nil {
		return ancestor
	}
	if c.Selected != nil && reply.ID().Equals(c.Selected) {
		return selected
	}
	for _, id := range c.Ancestry {
		if id.Equals(reply.ID()) {
			return ancestor
		}
	}
	for _, id := range c.Descendants {
		if id.Equals(reply.ID()) {
			return descendant
		}
	}
	if c.Conversation != nil && !c.Conversation.Equals(fields.NullHash()) {
		if c.Conversation.Equals(&reply.ConversationID) {
			return sibling
		}
	}
	return none
}

func (c *ReplyListView) Layout(gtx layout.Context) layout.Dimensions {
	theme := c.Theme.Theme
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paintOp := paint.ColorOp{Color: c.Theme.Primary.Dark}
			paintOp.Add(gtx.Ops)
			paint.PaintOp{Rect: f32.Rectangle{
				Max: f32.Point{
					X: float32(gtx.Constraints.Max.X),
					Y: float32(gtx.Constraints.Max.Y),
				},
			}}.Add(gtx.Ops)
			return layout.Dimensions{}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return c.layoutReplyList(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if c.ReplyingTo != nil || c.CreatingConversation {
						return c.layoutEditor(gtx)
					}
					return layout.Dimensions{}
				}),
			)
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			buttons := []layout.FlexChild{}
			buttons = append(buttons, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(4)).Layout(gtx,
					material.IconButton(theme, &c.BackButton, icons.BackIcon).Layout,
				)
			}))
			if c.Selected != nil {
				buttons = append(buttons, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.IconButton(theme, &c.CopyReplyButton, icons.CopyIcon).Layout,
					)
				}))
			}
			if c.Selected != nil {
				buttons = append(buttons,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(4)).Layout(gtx,
							material.IconButton(theme, &c.FilterButton, icons.FilterIcon).Layout,
						)
					}))
			}
			if !c.CreatingConversation {
				buttons = append(buttons,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(4)).Layout(gtx,
							material.IconButton(theme, &c.CreateConversationButton, icons.CreateConversationIcon).Layout,
						)
					}))
			}

			return layout.Flex{Spacing: layout.SpaceBetween}.Layout(gtx, buttons...)
		}),
	)
}

func (c *ReplyListView) layoutReplyList(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min = gtx.Constraints.Max

	theme := c.Theme.Theme
	c.ReplyList.Axis = layout.Vertical
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
			var (
				background, textColor color.RGBA
				leftInset             unit.Value
			)
			status := c.statusOf(reply)
			switch status {
			case selected:
				leftInset = unit.Dp(15)
				background = c.Theme.Background.Light
				textColor = c.Theme.Theme.Color.Text
				collapseMetadata = false
				communityNode, found, err := c.ArborState.SubscribableStore.GetCommunity(&reply.CommunityID)
				if err != nil || !found {
					log.Printf("failed finding community %s for node %s", &reply.CommunityID, reply.ID())
				}
				community = communityNode.(*forest.Community)
			case ancestor:
				leftInset = unit.Dp(15)
				background = c.Theme.Background.Default
				textColor = c.Theme.Theme.Color.Text
			case descendant:
				leftInset = unit.Dp(30)
				background = c.Theme.Background.Default
				textColor = c.Theme.Theme.Color.Text
			case sibling:
				fallthrough
			default:
				if c.Filtered {
					// do not render
					return layout.Dimensions{}
				}
				leftInset = sideInset
				background = c.Theme.Primary.Light
				background.A -= 100
				textColor = c.Theme.Color.Text
			}
			stateIndex++
			var flexChildren []layout.FlexChild
			flexChildren = append(flexChildren, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				extraWidth := gtx.Px(unit.Dp(36))
				messageWidth := gtx.Constraints.Max.X - extraWidth
				return layout.Stack{}.Layout(gtx,
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
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
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Max.X = messageWidth
							replyWidget := sprigTheme.Reply(theme)
							replyWidget.Background = background
							replyWidget.TextColor = textColor
							replyWidget.CollapseMetadata = collapseMetadata
							return replyWidget.Layout(gtx, reply, author, community)
						})
					}),
					layout.Expanded(func(gtx layout.Context) layout.Dimensions {
						dims := state.Clickable.Layout(gtx)
						state.Reply = reply.ID()
						return dims
					}),
				)
			}))
			if status == selected {
				flexChildren = append(flexChildren, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						replyButton := material.IconButton(theme, &c.CreateReplyButton, icons.ReplyIcon)
						replyButton.Size = unit.Dp(20)
						replyButton.Inset = layout.UniformInset(unit.Dp(9))
						replyButton.Background = c.Theme.Secondary.Default
						replyButton.Color = c.Theme.Background.Dark
						return replyButton.Layout(gtx)
					})
				}))
			}
			return layout.Flex{}.Layout(gtx, flexChildren...)
		})
	})
	return dims
}

func (c *ReplyListView) layoutEditor(gtx layout.Context) layout.Dimensions {
	theme := c.Theme.Theme
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paintOp := paint.ColorOp{Color: c.Theme.Primary.Default}
			paintOp.Add(gtx.Ops)
			paint.PaintOp{Rect: f32.Rectangle{
				Max: f32.Point{
					X: float32(gtx.Constraints.Max.X),
					Y: float32(gtx.Constraints.Max.Y),
				},
			}}.Add(gtx.Ops)
			return layout.Dimensions{}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								if c.CreatingConversation {
									return material.Body1(theme, "New Conversation in:").Layout(gtx)
								}
								return material.Body1(theme, "Replying to:").Layout(gtx)

							})
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								if c.CreatingConversation {
									var dims layout.Dimensions
									c.ArborState.CommunityList.WithCommunities(func(comms []*forest.Community) {
										dims = c.CommunityList.Layout(gtx, len(comms), func(gtx layout.Context, index int) layout.Dimensions {
											community := comms[index]
											return material.RadioButton(theme, &c.CommunityChoice, community.ID().String(), string(community.Name.Blob)).Layout(gtx)
										})
									})
									return dims
								}
								reply := sprigTheme.Reply(theme)
								reply.Background = c.Theme.Primary.Light
								return reply.Layout(gtx, c.ReplyingTo, c.ReplyingToAuthor, nil)
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								cancelButton := material.IconButton(theme, &c.CancelReplyButton, icons.CancelReplyIcon)
								cancelButton.Size = unit.Dp(20)
								cancelButton.Inset = layout.UniformInset(unit.Dp(4))
								return cancelButton.Layout(gtx)
							})
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								pasteButton := material.IconButton(theme, &c.PasteIntoReplyButton, icons.PasteIcon)
								pasteButton.Inset = layout.UniformInset(unit.Dp(4))
								pasteButton.Size = unit.Dp(20)
								return pasteButton.Layout(gtx)
							})
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Stack{}.Layout(gtx,
									layout.Expanded(func(gtx layout.Context) layout.Dimensions {
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
										clip.Rect{
											Rect: bounds,
											NW:   radii,
											NE:   radii,
											SE:   radii,
											SW:   radii,
										}.Op(gtx.Ops).Add(gtx.Ops)
										paint.PaintOp{Rect: bounds}.Add(gtx.Ops)
										stack.Pop()
										return layout.Dimensions{}
									}),
									layout.Stacked(func(gtx layout.Context) layout.Dimensions {
										return layout.UniformInset(unit.Dp(6)).Layout(gtx,
											material.Editor(theme, &c.ReplyEditor, "type your reply here").Layout,
										)
									}),
								)
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								sendButton := material.IconButton(theme, &c.SendReplyButton, icons.SendReplyIcon)
								sendButton.Size = unit.Dp(20)
								sendButton.Inset = layout.UniformInset(unit.Dp(4))
								sendButton.Background = c.Theme.Secondary.Default
								sendButton.Color = c.Theme.Background.Dark
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
