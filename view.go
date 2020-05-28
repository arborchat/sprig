package main

import (
	"gioui.org/layout"
)

type View interface {
	SetManager(ViewManager)
	Update(gtx layout.Context)
	HandleClipboard(contents string)
	Layout(gtx layout.Context) layout.Dimensions
}
