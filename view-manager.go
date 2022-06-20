package main

import (
	"fmt"
	"image"
	"runtime"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/profile"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	materials "gioui.org/x/component"

	"git.sr.ht/~whereswaldon/sprig/core"
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type ViewManager interface {
	// request that the primary view be switched to the view with the given ID
	RequestViewSwitch(ViewID)
	// set the primary view to be the view with the given ID. This does not
	// preserve the history of the previous view, so back navigation will not
	// work.
	SetView(ViewID)
	// associate a view with an ID
	RegisterView(id ViewID, view View)
	// register that a given view handles a given kind of intent
	RegisterIntentHandler(id ViewID, intent IntentID)
	// finds a view that can handle the intent and Pushes that view
	ExecuteIntent(intent Intent) bool
	// request a screen invalidation from outside of a render context
	RequestInvalidate()
	// handle logical "back" navigation operations
	HandleBackNavigation(*key.Event)
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
	ApplySettings(core.SettingsService)
}

type viewManager struct {
	views   map[ViewID]View
	current ViewID
	window  *app.Window

	core.App

	*materials.ModalLayer
	materials.NavDrawer
	navAnim materials.VisibilityAnimation
	*materials.ModalNavDrawer
	*materials.AppBar

	intentToView map[IntentID]ViewID

	// track the tag of the overflow action selected within the last frame
	selectedOverflowTag interface{}

	// tracking the handling of "back" events
	viewStack []ViewID

	// dock the navigation drawer?
	dockDrawer bool

	// runtime profiling data
	profiling   bool
	profile     profile.Event
	lastMallocs uint64

	// runtime themeing state
	themeing  bool
	themeView View
}

func NewViewManager(window *app.Window, app core.App) ViewManager {
	modal := materials.NewModal()
	drawer := materials.NewNav("Sprig", "Arbor chat client")
	vm := &viewManager{
		App:          app,
		views:        make(map[ViewID]View),
		window:       window,
		themeView:    NewThemeEditorView(app),
		ModalLayer:   modal,
		NavDrawer:    drawer,
		intentToView: make(map[IntentID]ViewID),
		navAnim: materials.VisibilityAnimation{
			Duration: time.Millisecond * 250,
			State:    materials.Invisible,
		},
		AppBar: materials.NewAppBar(modal),
	}
	vm.ModalNavDrawer = materials.ModalNavFrom(&vm.NavDrawer, vm.ModalLayer)
	vm.AppBar.NavigationIcon = icons.MenuIcon
	return vm
}

func (vm *viewManager) RequestInvalidate() {
	vm.window.Invalidate()
}

func (vm *viewManager) ApplySettings(settings core.SettingsService) {
	anchor := materials.Top
	if settings.BottomAppBar() {
		anchor = materials.Bottom
	}
	vm.AppBar.Anchor = anchor
	vm.ModalNavDrawer.Anchor = anchor
	vm.dockDrawer = settings.DockNavDrawer()
	vm.App.Theme().SetDarkMode(settings.DarkMode())

	vm.ModalNavDrawer = materials.ModalNavFrom(&vm.NavDrawer, vm.ModalLayer)
	vm.themeView.BecomeVisible()

	if settings.DarkMode() {
		vm.NavDrawer.AlphaPalette = materials.AlphaPalette{
			Hover:    100,
			Selected: 150,
		}
	} else {
		vm.NavDrawer.AlphaPalette = materials.AlphaPalette{
			Hover:    25,
			Selected: 50,
		}
	}
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

func (vm *viewManager) RegisterIntentHandler(id ViewID, intentID IntentID) {
	vm.intentToView[intentID] = id
}

func (vm *viewManager) ExecuteIntent(intent Intent) bool {
	view, ok := vm.intentToView[intent.ID]
	if !ok {
		return false
	}
	vm.Push(view)
	vm.views[view].HandleIntent(intent)
	return true
}

func (vm *viewManager) SetView(id ViewID) {
	vm.current = id
	//vm.ModalNavDrawer.SetNavDestination(id)
	view := vm.views[vm.current]
	if showBar, title, actions, overflow := view.AppBarData(); showBar {
		vm.AppBar.Title = title
		vm.AppBar.SetActions(actions, overflow)
	}
	vm.NavDrawer.SetNavDestination(id)
	view.BecomeVisible()
}

func (vm *viewManager) RequestViewSwitch(id ViewID) {
	vm.Push(vm.current)
	vm.SetView(id)
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
	return vm.selectedOverflowTag
}

func (vm *viewManager) HandleBackNavigation(event *key.Event) {
	if len(vm.viewStack) > 0 {
		vm.Pop()
	}
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
	vm.selectedOverflowTag = nil
	for _, event := range vm.AppBar.Events(gtx) {
		switch event := event.(type) {
		case materials.AppBarNavigationClicked:
			if vm.dockDrawer {
				vm.navAnim.ToggleVisibility(gtx.Now)
			} else {
				vm.navAnim.Disappear(gtx.Now)
				vm.ModalNavDrawer.ToggleVisibility(gtx.Now)
			}
		case materials.AppBarOverflowActionClicked:
			vm.selectedOverflowTag = event.Tag
		}
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
	th := vm.App.Theme().Current()
	banner := func(gtx C) D {
		switch bannerConfig := vm.App.Banner().Top().(type) {
		case *core.LoadingBanner:
			secondary := th.Secondary.Default
			th := *(th.Theme)
			th.ContrastFg = th.Fg
			th.ContrastBg = th.Bg
			th.Palette = sprigTheme.ApplyAsNormal(th.Palette, secondary)
			return layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx C) D {
					paint.FillShape(gtx.Ops, th.Bg, clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Op())
					return D{Size: gtx.Constraints.Min}
				}),
				layout.Stacked(func(gtx C) D {
					return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx C) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return layout.Flex{Spacing: layout.SpaceAround}.Layout(gtx,
							layout.Rigid(material.Body1(&th, bannerConfig.Text).Layout),
							layout.Rigid(material.Loader(&th).Layout),
						)
					})
				}),
			)
		default:
			return D{}
		}
	}

	bar := layout.Rigid(func(gtx C) D {
		if displayBar {
			return vm.AppBar.Layout(gtx, th.Theme, "Navigation", "More")
		}
		return layout.Dimensions{}
	})
	content := layout.Flexed(1, func(gtx C) D {
		return layout.Flex{}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				gtx.Constraints.Max.X /= 3
				return vm.NavDrawer.Layout(gtx, th.Theme, &vm.navAnim)
			}),
			layout.Flexed(1, func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(banner),
					layout.Flexed(1.0, view.Layout),
				)
			}),
		)
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
	vm.ModalLayer.Layout(gtx, th.Theme)
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
			return sprigTheme.Rect{
				Color: vm.App.Theme().Current().Background.Light.Bg,
				Size: f32.Point{
					X: float32(gtx.Constraints.Min.X),
					Y: float32(gtx.Constraints.Min.Y),
				},
			}.Layout(gtx)
		}),
		layout.Stacked(func(gtx C) D {
			return layout.Inset{Top: unit.Dp(4), Left: unit.Dp(4)}.Layout(gtx, func(gtx C) D {
				label := material.Body1(vm.App.Theme().Current().Theme, text)
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
