package widget

import (
	"gioui.org/io/clipboard"
	"gioui.org/layout"
	"gioui.org/widget"
	materials "gioui.org/x/component"
)

// TextForm holds the theme-independent state of a simple form that
// allows a user to provide a single text value and supports pasting.
// It can be submitted with either the submit button or pressing enter
// on the keyboard.
type TextForm struct {
	submitted    bool
	TextField    materials.TextField
	SubmitButton widget.Clickable
	PasteButton  widget.Clickable
}

func (c *TextForm) Layout(gtx layout.Context) layout.Dimensions {
	c.submitted = false
	for _, e := range c.TextField.Events() {
		if _, ok := e.(widget.SubmitEvent); ok {
			c.submitted = true
		}
	}
	if c.SubmitButton.Clicked() {
		c.submitted = true
	}
	if c.PasteButton.Clicked() {
		clipboard.ReadOp{Tag: c}.Add(gtx.Ops)
	}
	for _, e := range gtx.Events(c) {
		switch e := e.(type) {
		case clipboard.Event:
			c.TextField.Editor.Insert(e.Text)
		}
	}
	return layout.Dimensions{}
}

func (c *TextForm) Submitted() bool {
	return c.submitted
}
