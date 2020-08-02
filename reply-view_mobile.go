//+build ios android

package main

func (r *ReplyListView) requestKeyboardFocus() {
	// do nothing on mobile, otherwise we trigger the on-screen keyboard
}
