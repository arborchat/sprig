package main

import (
	"log"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	materials "gioui.org/x/component"
	"gioui.org/x/markdown"
	"gioui.org/x/richtext"
	"git.sr.ht/~gioverse/chat/list"
	"git.sr.ht/~whereswaldon/forest-go/fields"
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

	core.App
}

var _ View = &DynamicChatView{}

// NewDynamicChatView constructs a chat view.
func NewDynamicChatView(app core.App) View {
	c := &DynamicChatView{
		App: app,
	}
	c.chatList.Axis = layout.Vertical
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
	return c
}

// DynamicChatViewName defines the user-presented name for this view.
const DynamicChatViewName = "Chat"

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

// loadMessages loads chat messages in a given direction relative to a given
// other chat message.
func (c *DynamicChatView) loadMessages(dir list.Direction, relativeTo list.Serial) []list.Element {
	log.Println(dir, relativeTo)
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
		md, err := reply.TwigMetadata()
		if err != nil {
			continue
		}
		if md.Contains("invisible", 1) {
			continue
		}
		var rd ds.ReplyData
		rd.Populate(reply, c.Arbor().Store())
		elements = append(elements, rd)
	}
	return elements
}

// layoutReply returns a widget that will render the provided reply using the
// provided state.
func (c *DynamicChatView) layoutReply(replyData list.Element, state interface{}) layout.Widget {
	sTheme := c.Theme().Current()
	theme := sTheme.Theme
	return func(gtx C) D {
		state := state.(*sprigwidget.Reply)
		rd := replyData.(ds.ReplyData)
		content, _ := markdown.NewRenderer().Render(theme, []byte(rd.Content))
		richContent := richtext.Text(&state.InteractiveText, theme.Shaper, content...)
		animState := &sprigwidget.ReplyAnimationState{
			Normal: &c.FocusAnimation,
			Begin:  state.ReplyStatus,
		}
		return sprigtheme.Reply(sTheme, animState, rd, richContent, false).Layout(gtx)
	}
}
