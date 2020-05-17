package widget

import (
	"gioui.org/widget"
	"git.sr.ht/~whereswaldon/forest-go/fields"
)

type Reply struct {
	widget.Clickable
	Reply *fields.QualifiedHash
}
