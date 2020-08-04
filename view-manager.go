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
	"git.sr.ht/~whereswaldon/materials"
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type ViewManager interface {
	// request that the primary view be switched to the view with the given ID
	RequestViewSwitch(ViewID)
	// associate a view with an ID
	RegisterView(id ViewID, view View)
	// trigger an asynchronous paste operation
	RequestClipboardPaste()
	// handle a paste operation from the platform by dispatching it to a view
	HandleClipboard(contents string)
	// set the system clipboard to the value
	UpdateClipboard(string)
	// handle logical "back" navigation operations
	HandleBackNavigation(*system.CommandEvent)
	// trigger a contextual app menu with the given title and actions
	RequestContextualBar(gtx layout.Context, title string, actions []materials.AppBarAction, overflow []materials.OverflowAction)
	// request that any contextual menu disappear
	DismissContextualBar(gtx layout.Context)
	// request that an app bar overflow menu disappear
	DismissOverflow(gtx layout.Context)
	// get the tag of a selected overflow message
	SelectedOverflowTag() interface{}
	// render the interface
	Layout(gtx layout.Context) layout.Dimensions
	// enable graphics profiling
	SetProfiling(bool)
	// enable live theme editing
	SetThemeing(bool)
	// apply settings changes relevant to the UI
	ApplySettings(Settings)
}

type viewManager struct {
	views   map[ViewID]View
	current ViewID
	window  *app.Window
	Theme   *sprigTheme.Theme

	*materials.ModalLayer
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
	modal := materials.NewModal()
	vm := &viewManager{
		views:          make(map[ViewID]View),
		window:         window,
		profiling:      profile,
		Theme:          theme,
		themeView:      NewThemeEditorView(theme),
		ModalLayer:     modal,
		ModalNavDrawer: materials.NewModalNav(theme.Theme, modal, "Sprig", "Arbor chat client"),
		AppBar:         materials.NewAppBar(theme.Theme, modal),
	}
	vm.AppBar.NavigationIcon = icons.MenuIcon
	return vm
}

func (vm *viewManager) ApplySettings(settings Settings) {
	anchor := materials.Top
	if settings.BottomAppBar {
		anchor = materials.Bottom
	}
	vm.AppBar.Anchor = anchor
	vm.ModalNavDrawer.Anchor = anchor
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
	view.BecomeVisible()
}

func (vm *viewManager) RequestContextualBar(gtx layout.Context, title string, actions []materials.AppBarAction, overflow []materials.OverflowAction) {
	vm.AppBar.SetContextualActions(actions, overflow)
	vm.AppBar.StartContextual(gtx.Now, title)
}

func (vm *viewManager) DismissContextualBar(gtx layout.Context) {
	vm.AppBar.StopContextual(gtx.Now)
}

func (vm *viewManager) DismissOverflow(gtx layout.Context) {
	vm.AppBar.CloseOverflowMenu(gtx.Now)
}

func (vm *viewManager) SelectedOverflowTag() interface{} {
	return vm.AppBar.SelectedOverflowAction()
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
		vm.ModalNavDrawer.ToggleVisibility(gtx.Now)
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
	view := vm.views[vm.current]
	view.Update(gtx)
	displayBar, _, _, _ := view.AppBarData()
	bar := layout.Rigid(func(gtx C) D {
		if displayBar {
			return vm.AppBar.Layout(gtx)
		}
		return layout.Dimensions{}
	})
	content := layout.Flexed(1, func(gtx C) D {
		return view.Layout(gtx)
	})
	flex := layout.Flex{
		Axis: layout.Vertical,
	}
	var dimensions layout.Dimensions
	if vm.AppBar.Anchor == materials.Top {
		dimensions = flex.Layout(gtx,
			bar,
			content,
		)
	} else {
		dimensions = flex.Layout(gtx,
			content,
			bar,
		)
	}
	vm.ModalLayer.Layout(gtx)
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
				label := material.Body1(vm.Theme.Theme, text)
				label.Font.Variant = "Mono"
				return label.Layout(gtx)
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
