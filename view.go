package main

import (
	"gioui.org/layout"
	"git.sr.ht/~whereswaldon/materials"
)

type View interface {
	SetManager(ViewManager)
	DisplayAppBar() bool
	NavItem() *materials.NavItem
	Update(gtx layout.Context)
	HandleClipboard(contents string)
	Layout(gtx layout.Context) layout.Dimensions
}
