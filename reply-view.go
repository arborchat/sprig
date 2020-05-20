package main

import (
	"image/color"
	"log"

	"gioui.org/f32"
	"gioui.org/layout"
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

	ReplyingTo        *forest.Reply
	ReplyingToAuthor  *forest.Identity
	ReplyEditor       widget.Editor
	CancelReplyButton widget.Clickable
	CreateReplyButton widget.Clickable
	SendReplyButton   widget.Clickable

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
	c.ReplyList.ScrollToEnd = true
	c.ReplyList.Position.BeforeEnd = false
	return c
}

func (c *ReplyListView) Update(gtx *layout.Context) {
	for i := range c.ReplyStates {
		clickHandler := &c.ReplyStates[i]
		if clickHandler.Clicked(gtx) {
			log.Printf("clicked %s", clickHandler.Reply)
			if c.Selected == nil || !clickHandler.Reply.Equals(c.Selected) {
				c.Selected = clickHandler.Reply
				c.Ancestry, _ = c.ArborState.SubscribableStore.AncestryOf(clickHandler.Reply)
				c.Descendants, _ = c.ArborState.SubscribableStore.DescendantsOf(clickHandler.Reply)
				reply, _, _ := c.ArborState.SubscribableStore.Get(clickHandler.Reply)
				c.Conversation = &reply.(*forest.Reply).ConversationID
			} else {
				c.Filtered = !c.Filtered
			}
		}
	}
	if c.BackButton.Clicked(gtx) {
		c.manager.RequestViewSwitch(CommunityMenu)
	}
	if c.DeselectButton.Clicked(gtx) {
		c.Selected = nil
	}
	c.updateReplyEditState(gtx)
}

func (c *ReplyListView) updateReplyEditState(gtx *layout.Context) {
	if c.Selected != nil && c.CreateReplyButton.Clicked(gtx) {
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
	if c.CancelReplyButton.Clicked(gtx) {
		c.resetReplyState()
	}
	if c.SendReplyButton.Clicked(gtx) {
		nodeBuilder, err := c.Settings.Builder()
		if err != nil {
			log.Printf("failed acquiring node builder: %v", err)
		} else {
			reply, err := nodeBuilder.NewReply(c.ReplyingTo, c.ReplyEditor.Text(), []byte{})
			if err != nil {
				log.Printf("failed building reply: %v", err)
			}
			go func() {
				if err := c.ArborState.SubscribableStore.Add(nodeBuilder.User); err != nil {
					log.Printf("failed adding replying identity to store: %v", err)
					return
				}
				if err := c.ArborState.SubscribableStore.Add(reply); err != nil {
					log.Printf("failed adding reply to store: %v", err)
					return
				}
			}()
		}
		c.resetReplyState()
	}

}

func (c *ReplyListView) resetReplyState() {
	c.ReplyingTo = nil
	c.ReplyingToAuthor = nil
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

func (c *ReplyListView) Layout(gtx *layout.Context) {
	layout.Stack{}.Layout(gtx,
		layout.Stacked(func() {
			layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Flexed(1, func() {
					c.layoutReplyList(gtx)
				}),
				layout.Rigid(func() {
					if c.ReplyingTo != nil {
						c.layoutEditor(gtx)
					}
				}),
			)
		}),
		layout.Stacked(func() {
			gtx.Constraints.Width.Min = gtx.Constraints.Width.Max
			layout.Flex{Spacing: layout.SpaceBetween}.Layout(gtx,
				layout.Rigid(func() {
					layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
						material.IconButton(c.Theme, icons.BackIcon).Layout(gtx, &c.BackButton)
					})
				}),
				layout.Rigid(func() {
					if c.Selected != nil {
						if c.ReplyingTo == nil {
							layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
								material.IconButton(c.Theme, icons.ReplyIcon).Layout(gtx, &c.CreateReplyButton)
							})
						} else {
							layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
								material.IconButton(c.Theme, icons.CancelReplyIcon).Layout(gtx, &c.CancelReplyButton)
							})
						}
					}
				}),
				layout.Rigid(func() {
					if c.Selected != nil {
						layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
							material.IconButton(c.Theme, icons.ClearIcon).Layout(gtx, &c.DeselectButton)
						})
					}
				}),
			)
		}),
	)
}

var (
	black      = color.RGBA{A: 255}
	teal       = color.RGBA{G: 128, B: 128, A: 255}
	brightTeal = color.RGBA{G: 175, B: 175, A: 255}
	//darkGray = color.RGBA{R: 50, G: 50, B: 50, A: 255}
	//mediumGray = color.RGBA{R: 100, G: 100, B: 100, A: 255}
	white          = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	lightLightGray = color.RGBA{R: 240, G: 240, B: 240, A: 255}

//lightGray = color.RGBA{R: 230, G: 230, B: 230, A: 255}
)

func (c *ReplyListView) layoutReplyList(gtx *layout.Context) {
	gtx.Constraints.Height.Min = gtx.Constraints.Height.Max
	gtx.Constraints.Width.Min = gtx.Constraints.Width.Max

	theme := c.Theme
	c.ReplyList.Axis = layout.Vertical
	stateIndex := 0
	c.ArborState.ReplyList.WithReplies(func(replies []*forest.Reply) {
		c.ReplyList.Layout(gtx, len(replies), func(index int) {
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
					return
				}
				leftInset = sideInset
				background = teal
				background.A = 0
				background.G += 10
				background.B += 10
				textColor = black
			}
			messageWidth := gtx.Constraints.Width.Max - gtx.Px(unit.Dp(36))
			layout.Stack{}.Layout(gtx,
				layout.Stacked(func() {
					gtx.Constraints.Width.Min = gtx.Constraints.Width.Max
					layout.Stack{}.Layout(gtx,
						layout.Expanded(func() {
							paintOp := paint.ColorOp{Color: color.RGBA{G: 128, B: 128, A: 255}}
							paintOp.Add(gtx.Ops)
							paint.PaintOp{Rect: f32.Rectangle{
								Max: f32.Point{
									X: float32(gtx.Constraints.Width.Max),
									Y: float32(gtx.Constraints.Height.Max),
								},
							}}.Add(gtx.Ops)
						}),
						layout.Stacked(func() {
							margin := unit.Dp(6)
							if collapseMetadata {
								margin = unit.Dp(3)
							}
							layout.Inset{Left: leftInset, Top: margin, Right: sideInset}.Layout(gtx, func() {
								gtx.Constraints.Width.Max = messageWidth
								replyWidget := sprigTheme.Reply(theme)
								replyWidget.Background = background
								replyWidget.TextColor = textColor
								replyWidget.CollapseMetadata = collapseMetadata
								replyWidget.Layout(gtx, reply, author)
							})
						}),
					)
				}),
				layout.Expanded(func() {
					state.Clickable.Layout(gtx)
					state.Reply = reply.ID()
				}),
			)
			stateIndex++
		})
	})
}

func (c *ReplyListView) layoutEditor(gtx *layout.Context) {
	layout.Stack{}.Layout(gtx,
		layout.Expanded(func() {
			paintOp := paint.ColorOp{Color: brightTeal}
			paintOp.Add(gtx.Ops)
			paint.PaintOp{Rect: f32.Rectangle{
				Max: f32.Point{
					X: float32(gtx.Constraints.Width.Max),
					Y: float32(gtx.Constraints.Height.Max),
				},
			}}.Add(gtx.Ops)
		}),
		layout.Stacked(func() {
			layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func() {
					layout.Flex{}.Layout(gtx,
						layout.Rigid(func() {
							layout.UniformInset(unit.Dp(6)).Layout(gtx, func() {
								material.Body1(c.Theme, "Replying to:").Layout(gtx)
							})
						}),
						layout.Flexed(1, func() {
							layout.UniformInset(unit.Dp(6)).Layout(gtx, func() {
								sprigTheme.Reply(c.Theme).Layout(gtx, c.ReplyingTo, c.ReplyingToAuthor)
							})
						}),
					)
				}),
				layout.Rigid(func() {
					layout.Flex{}.Layout(gtx,
						layout.Flexed(1, func() {
							layout.UniformInset(unit.Dp(6)).Layout(gtx, func() {
								layout.Stack{}.Layout(gtx,
									layout.Expanded(func() {
										paintOp := paint.ColorOp{Color: white}
										paintOp.Add(gtx.Ops)
										paint.PaintOp{Rect: f32.Rectangle{
											Max: f32.Point{
												X: float32(gtx.Constraints.Width.Max),
												Y: float32(gtx.Constraints.Height.Max),
											},
										}}.Add(gtx.Ops)
									}),
									layout.Stacked(func() {
										material.Editor(c.Theme, "type your reply here").Layout(gtx, &c.ReplyEditor)
									}),
								)
							})
						}),
						layout.Rigid(func() {
							layout.UniformInset(unit.Dp(6)).Layout(gtx, func() {
								sendButton := material.IconButton(c.Theme, icons.SendReplyIcon)
								sendButton.Size = unit.Dp(40)
								sendButton.Padding = unit.Dp(10)
								sendButton.Layout(gtx, &c.SendReplyButton)
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
