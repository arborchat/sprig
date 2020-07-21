package widget

import (
	"log"

	"gioui.org/layout"
	"gioui.org/widget"
)

// TextForm holds the theme-independent state of a simple form that
// allows a user to provide a single text value and supports pasting.
// It can be submitted with either the submit button or pressing enter
// on the keyboard.
type TextForm struct {
	submitted      bool
	pasteRequested bool
	widget.Editor
	SubmitButton widget.Clickable
	PasteButton  widget.Clickable
}

func (c *TextForm) Layout(gtx layout.Context) layout.Dimensions {
	c.submitted = false
	c.pasteRequested = false
	for _, e := range c.Editor.Events() {
		if _, ok := e.(widget.SubmitEvent); ok {
			log.Printf("submit event")
			c.submitted = true
		}
	}
	if c.SubmitButton.Clicked() {
		log.Printf("submit event")
		c.submitted = true
	}
	if c.PasteButton.Clicked() {
		c.pasteRequested = true
	}
	return layout.Dimensions{}
}

func (c *TextForm) Submitted() bool {
	return c.submitted
}

func (c *TextForm) PasteRequested() bool {
	return c.pasteRequested
}

func (c *TextForm) Paste(data string) {
	c.Editor.Insert(data)
}
