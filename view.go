package main

import (
	"gioui.org/layout"
	materials "gioui.org/x/component"
)

type View interface {
	SetManager(ViewManager)
	AppBarData() (bool, string, []materials.AppBarAction, []materials.OverflowAction)
	NavItem() *materials.NavItem
	BecomeVisible()
	Update(gtx layout.Context)
	Layout(gtx layout.Context) layout.Dimensions
}
