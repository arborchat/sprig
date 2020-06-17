package main

import (
	"fmt"
	"runtime"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/profile"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type ViewManager interface {
	RequestViewSwitch(ViewID)
	RegisterView(ViewID, View)
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
		views:     make(map[ViewID]View),
		window:    window,
		profiling: profile,
		Theme:     theme,
		themeView: NewThemeEditorView(theme),
	}
	return vm
}

func (vm *viewManager) RegisterView(id ViewID, view View) {
	vm.views[id] = view
	view.SetManager(vm)
}

func (vm *viewManager) RequestViewSwitch(id ViewID) {
	vm.Push(vm.current)
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
	vm.window.Invalidate()
}

func (vm *viewManager) Layout(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return vm.layoutProfileTimings(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			if !vm.themeing {
				gtx.Constraints.Min = gtx.Constraints.Max
				vm.views[vm.current].Update(gtx)
				return vm.views[vm.current].Layout(gtx)
			}
			return layout.Flex{}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					gtx.Constraints.Max.X /= 2
					gtx.Constraints.Min = gtx.Constraints.Max
					vm.views[vm.current].Update(gtx)
					return vm.views[vm.current].Layout(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					return vm.layoutThemeing(gtx)
				}),
			)
		}),
	)
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
