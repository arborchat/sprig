//+build !ios,!android

package main

func (r *ReplyListView) requestKeyboardFocus() {
	// on desktop, actually request keyboard focus
	r.ShouldRequestKeyboardFocus = true
}
