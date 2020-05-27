package main

import (
	"image"
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
	*material.Theme

	BackButton widget.Clickable

	DeselectButton widget.Clickable

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
	CancelReplyButton        widget.Clickable
	CreateReplyButton        widget.Clickable
	SendReplyButton          widget.Clickable
	CreateConversationButton widget.Clickable
	CommunityChoice          widget.Enum
	CommunitList             layout.List

	// Filtered determines whether or not the visible nodes should be
	// filtered to only those related to the selected node
	Filtered bool
}

var _ View = &ReplyListView{}

func NewReplyListView(settings *Settings, arborState *ArborState, theme *material.Theme) View {
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
				c.Filtered = !c.Filtered
			}
		}
	}
	if c.StateRefreshNeeded && c.Selected != nil {
		c.StateRefreshNeeded = false
		c.Ancestry, _ = c.ArborState.SubscribableStore.AncestryOf(c.Selected)
		c.Descendants, _ = c.ArborState.SubscribableStore.DescendantsOf(c.Selected)
	}
	if c.BackButton.Clicked() {
		c.manager.RequestViewSwitch(CommunityMenu)
	}
	if c.DeselectButton.Clicked() {
		c.Selected = nil
	}
	c.updateReplyEditState(gtx)
}

func (c *ReplyListView) updateReplyEditState(gtx layout.Context) {
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
	return layout.Stack{}.Layout(gtx,
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
					material.IconButton(c.Theme, &c.BackButton, icons.BackIcon).Layout,
				)
			}))
			if c.Selected != nil && c.ReplyingTo == nil {
				buttons = append(buttons, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.IconButton(c.Theme, &c.CreateReplyButton, icons.ReplyIcon).Layout,
					)
				}))
			}
			if c.ReplyingTo != nil || c.CreatingConversation {
				buttons = append(buttons, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx,
						material.IconButton(c.Theme, &c.CancelReplyButton, icons.CancelReplyIcon).Layout,
					)
				}))
			}
			if !c.CreatingConversation {
				buttons = append(buttons,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(4)).Layout(gtx,
							material.IconButton(c.Theme, &c.CreateConversationButton, icons.CreateConversationIcon).Layout,
						)
					}))
			}
			if c.Selected != nil {
				buttons = append(buttons,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(4)).Layout(gtx,
							material.IconButton(c.Theme, &c.DeselectButton, icons.ClearIcon).Layout,
						)
					}))
			}

			return layout.Flex{Spacing: layout.SpaceBetween}.Layout(gtx, buttons...)
		}),
	)
}

var (
	black      = color.RGBA{A: 255}
	teal       = color.RGBA{G: 128, B: 128, A: 255}
	brightTeal = color.RGBA{G: 200, B: 200, A: 255}
	//darkGray = color.RGBA{R: 50, G: 50, B: 50, A: 255}
	//mediumGray = color.RGBA{R: 100, G: 100, B: 100, A: 255}
	white          = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	lightLightGray = color.RGBA{R: 240, G: 240, B: 240, A: 255}

//lightGray = color.RGBA{R: 230, G: 230, B: 230, A: 255}
)

func (c *ReplyListView) layoutReplyList(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Min = gtx.Constraints.Max

	theme := c.Theme
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
			author := authorNode.(*forest.Identity)
			sideInset := unit.Dp(3)
			var (
				background, textColor color.RGBA
				leftInset             unit.Value
			)
			switch c.statusOf(reply) {
			case selected:
				leftInset = unit.Dp(15)
				background = white
				textColor = black
				collapseMetadata = false
			case ancestor:
				leftInset = unit.Dp(15)
				background = lightLightGray
				textColor = black
			case descendant:
				leftInset = unit.Dp(30)
				background = lightLightGray
				textColor = black
			case sibling:
				fallthrough
			default:
				if c.Filtered {
					// do not render
					return layout.Dimensions{}
				}
				leftInset = sideInset
				background = teal
				background.A = 0
				background.G += 10
				background.B += 10
				textColor = black
			}
			messageWidth := gtx.Constraints.Max.X - gtx.Px(unit.Dp(36))
			stateIndex++
			return layout.Stack{}.Layout(gtx,
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Stack{}.Layout(gtx,
						layout.Expanded(func(gtx layout.Context) layout.Dimensions {
							paintOp := paint.ColorOp{Color: color.RGBA{G: 128, B: 128, A: 255}}
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
							margin := unit.Dp(6)
							if collapseMetadata {
								margin = unit.Dp(3)
							}
							return layout.Inset{Left: leftInset, Top: margin, Right: sideInset}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								gtx.Constraints.Max.X = messageWidth
								replyWidget := sprigTheme.Reply(theme)
								replyWidget.Background = background
								replyWidget.TextColor = textColor
								replyWidget.CollapseMetadata = collapseMetadata
								return replyWidget.Layout(gtx, reply, author)
							})
						}),
					)
				}),
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					dims := state.Clickable.Layout(gtx)
					state.Reply = reply.ID()
					return dims
				}),
			)
		})
	})
	return dims
}

func (c *ReplyListView) layoutEditor(gtx layout.Context) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paintOp := paint.ColorOp{Color: brightTeal}
			paintOp.Add(gtx.Ops)
			paint.PaintOp{Rect: f32.Rectangle{
				Max: f32.Point{
					X: float32(gtx.Constraints.Max.X),
					Y: float32(gtx.Constraints.Max.Y),
				},
			}}.Add(gtx.Ops)
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								if c.CreatingConversation {
									return material.Body1(c.Theme, "New Conversation in:").Layout(gtx)
								}
								return material.Body1(c.Theme, "Replying to:").Layout(gtx)

							})
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								if c.CreatingConversation {
									var dims layout.Dimensions
									c.ArborState.CommunityList.WithCommunities(func(comms []*forest.Community) {
										c.CommunitList.Axis = layout.Vertical
										dims = c.CommunitList.Layout(gtx, len(comms), func(gtx layout.Context, index int) layout.Dimensions {
											community := comms[index]
											return material.RadioButton(c.Theme, &c.CommunityChoice, community.ID().String(), string(community.Name.Blob)).Layout(gtx)
										})
									})
									return dims
								}
								return sprigTheme.Reply(c.Theme).Layout(gtx, c.ReplyingTo, c.ReplyingToAuthor)
							})
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Stack{}.Layout(gtx,
									layout.Expanded(func(gtx layout.Context) layout.Dimensions {
										var stack op.StackOp
										stack.Push(gtx.Ops)
										paintOp := paint.ColorOp{Color: white}
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
										return layout.Dimensions{Size: image.Point{X: gtx.Constraints.Max.X, Y: gtx.Constraints.Min.Y}}
									}),
									layout.Stacked(func(gtx layout.Context) layout.Dimensions {
										return layout.UniformInset(unit.Dp(4)).Layout(gtx,
											material.Editor(c.Theme, &c.ReplyEditor, "type your reply here").Layout,
										)
									}),
								)
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								sendButton := material.IconButton(c.Theme, &c.SendReplyButton, icons.SendReplyIcon)
								sendButton.Size = unit.Dp(40)
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
