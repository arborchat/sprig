//+build !ios,!android

package main

func (r *ReplyListView) requestKeyboardFocus() {
	// on desktop, actually request keyboard focus
	r.ShouldRequestKeyboardFocus = true
}

// submitShouldSend indicates whether a submit event from the reply editor
// should automatically send the message.
const submitShouldSend = true
