package widget

import (
	"gioui.org/io/clipboard"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/x/richtext"

	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/sprig/ds"
	"git.sr.ht/~whereswaldon/sprig/platform"
)

// ComposerEvent represents a change in the Composer's state
type ComposerEvent uint

type MessageType int32

const (
	MessageTypeNone MessageType = iota
	MessageTypeConversation
	MessageTypeReply
)

const (
	ComposerSubmitted ComposerEvent = iota
	ComposerCancelled
)

// Editor prompts
const (
	replyPrompt        = "Compose your reply"
	conversationPrompt = "Start a new conversation"
)

// Composer holds the state for a widget that creates new arbor nodes.
type Composer struct {
	CommunityList layout.List
	Community     widget.Enum

	SendButton, CancelButton, PasteButton widget.Clickable
	widget.Editor

	TextState richtext.InteractiveText

	ReplyingTo ds.ReplyData

	events      []ComposerEvent
	composing   bool
	messageType MessageType
}

// update handles all state processing.
func (c *Composer) update(gtx layout.Context) {
	for _, e := range c.Editor.Events() {
		if _, ok := e.(widget.SubmitEvent); ok && !platform.Mobile {
			c.events = append(c.events, ComposerSubmitted)
		}
	}
	if c.PasteButton.Clicked() {
		clipboard.ReadOp{Tag: &c.composing}.Add(gtx.Ops)
	}
	for _, e := range gtx.Events(&c.composing) {
		switch e := e.(type) {
		case clipboard.Event:
			c.Editor.Insert(e.Text)
		}
	}
	if c.CancelButton.Clicked() {
		c.events = append(c.events, ComposerCancelled)
	}
	if c.SendButton.Clicked() {
		c.events = append(c.events, ComposerSubmitted)
	}
}

// Layout updates the state of the composer
func (c *Composer) Layout(gtx layout.Context) layout.Dimensions {
	c.update(gtx)
	return layout.Dimensions{}
}

// StartReply configures the composer to write a reply to the provided
// ReplyData.
func (c *Composer) StartReply(to ds.ReplyData) {
	c.Reset()
	c.composing = true
	c.ReplyingTo = to
	c.Editor.Focus()
}

// StartConversation configures the composer to write a new conversation.
func (c *Composer) StartConversation() {
	c.Reset()
	c.messageType = MessageTypeConversation
	c.composing = true
	c.Editor.Focus()
}

// Reset clears the internal state of the composer.
func (c *Composer) Reset() {
	c.messageType = MessageTypeNone
	c.ReplyingTo = ds.ReplyData{}
	c.Editor.SetText("")
	c.composing = false
}

// ComposingConversation returns whether the composer is currently creating
// a conversation (rather than a new reply within an existing conversation)
func (c *Composer) ComposingConversation() bool {
	return (c.ReplyingTo.ID == nil || c.ReplyingTo.ID.Equals(fields.NullHash())) && c.Composing()
}

// Composing indicates whether the composer is composing a message of any
// kind.
func (c Composer) Composing() bool {
	return c.composing
}

// PromptText returns the text prompt for the composer, based off of the message type
func (c Composer) PromptText() string {
	if c.messageType == MessageTypeConversation {
		return conversationPrompt
	} else {
		return replyPrompt
	}
}

func (c Composer) MessageType() MessageType {
	return c.messageType
}

// Events returns state change events for the composer since the last call
// to events.
func (c *Composer) Events() (out []ComposerEvent) {
	out, c.events = c.events, c.events[:0]
	return
}
