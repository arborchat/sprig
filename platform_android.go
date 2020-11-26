package main

import (
	gioapp "gioui.org/app"
	"gioui.org/io/event"
	"git.sr.ht/~whereswaldon/sprig/core"
)

func ProcessPlatformEvent(app core.App, e event.Event) bool {
	switch e := e.(type) {
	case gioapp.ViewEvent:
		app.Haptic().UpdateAndroidViewRef(e.View)
		return true
	default:
		return false
	}
}
