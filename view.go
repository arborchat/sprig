package main

import (
	"gioui.org/app"
	"gioui.org/layout"
)

type View interface {
	SetManager(ViewManager)
	Update(gtx layout.Context, window *app.Window)
	Layout(gtx layout.Context) layout.Dimensions
}
