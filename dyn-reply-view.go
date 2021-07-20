package main

import (
	"gioui.org/layout"
	materials "gioui.org/x/component"
	"git.sr.ht/~whereswaldon/sprig/core"
	"git.sr.ht/~whereswaldon/sprig/icons"
)

// DynamicChatView lays out a view of chat history built atop a
// dynamic list. Messages are presented chronologically, but with
// a finite number of messages loaded in memory at any given time.
// As the user scrolls forward or backward, new messages are loaded
// and messages in the opposite direction are discarded.
type DynamicChatView struct {
	manager ViewManager

	core.App
}

var _ View = &DynamicChatView{}

// NewDynamicChatView constructs a chat view.
func NewDynamicChatView(app core.App) View {
	c := &DynamicChatView{
		App: app,
	}
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
	// sTheme := c.Theme().Current()
	// theme := sTheme.Theme

	return D{}
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
