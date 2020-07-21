package main

import (
	"fmt"
	"runtime"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/profile"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"git.sr.ht/~whereswaldon/materials"
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type ViewManager interface {
	RequestViewSwitch(ViewID)
	RegisterView(id ViewID, view View)
	RequestClipboardPaste()
	HandleClipboard(contents string)
	UpdateClipboard(string)
	HandleBackNavigation(*system.CommandEvent)
	Layout(gtx layout.Context) layout.Dimensions
	SetProfiling(bool)
	SetThemeing(bool)
}

type viewManager struct {
	views   map[ViewID]View
	current ViewID
	window  *app.Window
	Theme   *sprigTheme.Theme

	*materials.ModalNavDrawer
	*materials.AppBar

	// tracking the handling of "back" events
	viewStack []ViewID

	// runtime profiling data
	profiling   bool
	profile     profile.Event
	lastMallocs uint64

	// runtime themeing state
	themeing  bool
	themeView View
}

func NewViewManager(window *app.Window, theme *sprigTheme.Theme, profile bool) ViewManager {
	vm := &viewManager{
		views:          make(map[ViewID]View),
		window:         window,
		profiling:      profile,
		Theme:          theme,
		themeView:      NewThemeEditorView(theme),
		ModalNavDrawer: materials.NewModalNav(theme.Theme, "Sprig", "Arbor chat client"),
		AppBar:         materials.NewAppBar(theme.Theme),
	}
	vm.AppBar.NavigationIcon = icons.MenuIcon
	return vm
}

func (vm *viewManager) RegisterView(id ViewID, view View) {
	if navItem := view.NavItem(); navItem != nil {
		vm.ModalNavDrawer.AddNavItem(materials.NavItem{
			Tag:  id,
			Name: navItem.Name,
			Icon: navItem.Icon,
		})
	}
	vm.views[id] = view
	view.SetManager(vm)
}

func (vm *viewManager) RequestViewSwitch(id ViewID) {
	vm.Push(vm.current)
	vm.current = id
	vm.ModalNavDrawer.SetNavDestination(id)
	view := vm.views[vm.current]
	if showBar, title, actions, overflow := view.AppBarData(); showBar {
		vm.AppBar.Title = title
		vm.AppBar.SetActions(actions, overflow)
	}
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

func (vm *viewManager) HandleBackNavigation(event *system.CommandEvent) {
	if len(vm.viewStack) < 1 {
		event.Cancel = false
		return
	}
	vm.Pop()
	event.Cancel = true
}

func (vm *viewManager) Push(id ViewID) {
	vm.viewStack = append(vm.viewStack, id)
}

func (vm *viewManager) Pop() {
	finalIndex := len(vm.viewStack) - 1
	vm.current, vm.viewStack = vm.viewStack[finalIndex], vm.viewStack[:finalIndex]
	vm.ModalNavDrawer.SetNavDestination(vm.current)
	vm.window.Invalidate()
}

func (vm *viewManager) Layout(gtx layout.Context) layout.Dimensions {
	if vm.AppBar.NavigationClicked(gtx) {
		vm.ModalNavDrawer.Appear(gtx.Now)
	}
	if vm.ModalNavDrawer.NavDestinationChanged() {
		vm.RequestViewSwitch(vm.ModalNavDrawer.CurrentNavDestination().(ViewID))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return vm.layoutProfileTimings(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			if !vm.themeing {
				gtx.Constraints.Min = gtx.Constraints.Max
				return vm.layoutCurrentView(gtx)
			}
			return layout.Flex{}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					gtx.Constraints.Max.X /= 2
					gtx.Constraints.Min = gtx.Constraints.Max
					return vm.layoutCurrentView(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					return vm.layoutThemeing(gtx)
				}),
			)
		}),
	)
}

func (vm *viewManager) layoutCurrentView(gtx layout.Context) layout.Dimensions {
	var barOp op.CallOp
	view := vm.views[vm.current]
	displayBar, _, _, _ := view.AppBarData()
	dimensions := layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			if displayBar {
				// Calculate the dimensions of the app bar, but wait to lay it out
				// until later.
				macro := op.Record(gtx.Ops)
				barDims := vm.AppBar.Layout(gtx)
				barOp = macro.Stop()
				return barDims
			}
			return layout.Dimensions{}
		}),
		layout.Flexed(1, func(gtx C) D {
			view.Update(gtx)
			return view.Layout(gtx)
		}),
	)
	// Lay out the app bar *after* the view so that the overflow can expand
	// on top of the underlying view.
	if displayBar {
		barOp.Add(gtx.Ops)
	}
	// Lay out the nav drawer after the app bar so that it can expand over
	// the app bar.
	vm.ModalNavDrawer.Layout(gtx)
	return dimensions
}

func (vm *viewManager) layoutProfileTimings(gtx layout.Context) layout.Dimensions {
	if !vm.profiling {
		return D{}
	}
	for _, e := range gtx.Events(vm) {
		if e, ok := e.(profile.Event); ok {
			vm.profile = e
		}
	}
	profile.Op{Tag: vm}.Add(gtx.Ops)
	var mstats runtime.MemStats
	runtime.ReadMemStats(&mstats)
	mallocs := mstats.Mallocs - vm.lastMallocs
	vm.lastMallocs = mstats.Mallocs
	text := fmt.Sprintf("m: %d %s", mallocs, vm.profile.Timings)
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			return sprigTheme.DrawRect(gtx,
				vm.Theme.Background.Light,
				f32.Point{
					X: float32(gtx.Constraints.Min.X),
					Y: float32(gtx.Constraints.Min.Y),
				},
				0)
		}),
		layout.Stacked(func(gtx C) D {
			return layout.Inset{Top: unit.Dp(4), Left: unit.Dp(4)}.Layout(gtx, func(gtx C) D {
				return material.Body1(vm.Theme.Theme, text).Layout(gtx)
			})
		}),
	)
}

func (vm *viewManager) SetProfiling(isProfiling bool) {
	vm.profiling = isProfiling
}

func (vm *viewManager) SetThemeing(isThemeing bool) {
	vm.themeing = isThemeing
}

func (vm *viewManager) layoutThemeing(gtx C) D {
	vm.themeView.Update(gtx)
	return vm.themeView.Layout(gtx)
}
