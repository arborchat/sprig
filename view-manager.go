package main

import (
	"gioui.org/app"
	"gioui.org/layout"
)

type ViewManager interface {
	RequestViewSwitch(ViewID)
	RegisterView(ViewID, View)
	RequestClipboardPaste()
	HandleClipboard(contents string)
	UpdateClipboard(string)
	Layout(gtx layout.Context) layout.Dimensions
}

type viewManager struct {
	views   map[ViewID]View
	current ViewID
	window  *app.Window
}

func NewViewManager(window *app.Window) ViewManager {
	vm := &viewManager{
		views:  make(map[ViewID]View),
		window: window,
	}
	return vm
}

func (vm *viewManager) RegisterView(id ViewID, view View) {
	vm.views[id] = view
	view.SetManager(vm)
}

func (vm *viewManager) RequestViewSwitch(id ViewID) {
	vm.current = id
}

func (vm *viewManager) RequestClipboardPaste() {
	vm.window.ReadClipboard()
}

func (vm *viewManager) UpdateClipboard(contents string) {
	vm.window.WriteClipboard(contents)
}

func (vm *viewManager) HandleClipboard(contents string) {
	vm.views[vm.current].HandleClipboard(contents)
}

func (vm *viewManager) Layout(gtx layout.Context) layout.Dimensions {
	vm.views[vm.current].Update(gtx)
	return vm.views[vm.current].Layout(gtx)
}
