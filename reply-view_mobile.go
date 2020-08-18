//+build ios android

package main

func (r *ReplyListView) requestKeyboardFocus() {
	// do nothing on mobile, otherwise we trigger the on-screen keyboard
}

// submitShouldSend indicates whether a submit event from the reply editor
// should automatically send the message.
const submitShouldSend = false
