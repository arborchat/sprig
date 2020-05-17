package main

import "gioui.org/layout"

type ViewManager interface {
	RequestViewSwitch(ViewID)
	RegisterView(ViewID, View)
	Layout(gtx *layout.Context)
}

type viewManager struct {
	views   map[ViewID]View
	current ViewID
}

func NewViewManager() ViewManager {
	vm := &viewManager{
		views: make(map[ViewID]View),
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

func (vm *viewManager) Layout(gtx *layout.Context) {
	vm.views[vm.current].Update(gtx)
	vm.views[vm.current].Layout(gtx)
}
