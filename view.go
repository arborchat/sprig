package main

import (
	"gioui.org/layout"
)

type View interface {
	SetManager(ViewManager)
	Update(gtx *layout.Context)
	Layout(gtx *layout.Context)
	HandleClipboard(contents string)
}
