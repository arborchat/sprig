//+build !android

package main

import (
	"gioui.org/io/event"
	"git.sr.ht/~whereswaldon/sprig/core"
)

func ProcessPlatformEvent(app core.App, e event.Event) bool {
	return false
}
