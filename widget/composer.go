package widget

import (
	"gioui.org/layout"
	"gioui.org/widget"
	"git.sr.ht/~whereswaldon/forest-go"
)

type ComposerMode uint

const (
	CreatingConversation ComposerMode = iota
	CreatingReply
)

type Composer struct {
	SendButton, CancelButton, CopyButton, PasteButton widget.Clickable
	widget.Editor

	Mode ComposerMode

	Communities layout.List
	Community   widget.Enum

	ReplyingTo       *forest.Reply
	ReplyingToAuthor *forest.Identity
}
