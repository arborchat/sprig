package pages

import (
	"image"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"git.sr.ht/~gioverse/skel/router"
	"git.sr.ht/~gioverse/skel/scheduler"
	"git.sr.ht/~gioverse/skel/window"
	"git.sr.ht/~whereswaldon/sprig/skelsprig/banner"
	"git.sr.ht/~whereswaldon/sprig/skelsprig/platform"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// NavigablePage is a page that should be displayed in top-level navigation.
type NavigablePage interface {
	router.Page
	NavItem() component.NavItem
}

// AppBarPage is a page that provides actions for the app bar.
type AppBarPage interface {
	router.Page
	Actions() ([]component.AppBarAction, []component.OverflowAction)
}

// StandalonePage is implemented by pages that want full screen control,
// with no navigation or app bar.
type StandalonePage interface {
	router.Page
	StandalonePage()
}

const (
	settingsPage string = "settings"
	setupPage    string = "setup"
)

// Window is the main event loop for application windows.
func Window(w *app.Window, bus scheduler.Connection) error {
	sth := sprigTheme.New()
	settingsP := &Settings{Th: sth, Conn: bus}
	setupP := &Setup{Th: sth, Conn: bus}
	r := router.Router{
		Pages: map[string]router.Page{
			settingsPage: settingsP,
			setupPage:    setupP,
		},
	}

	modal := component.NewModal()
	bar := component.NewAppBar(modal)
	nav := component.NewModalNav(modal, "Sprig", "Arbor Chat Client")
	nav.AddNavItem(settingsP.NavItem())
	nonModalVis := component.VisibilityAnimation{
		State: component.Visible,
	}
	resize := component.Resize{
		Ratio: 0.3,
	}
	if platform.Mobile {
		bar.Anchor = component.Bottom
	} else {
		bar.Anchor = component.Top
	}
	nav.Anchor = bar.Anchor

	// Set up initial route.
	r.Push(setupPage)
	if p, ok := r.Current().(AppBarPage); ok {
		bar.SetActions(p.Actions())
	} else {
		bar.SetActions(nil, nil)
	}
	if p, ok := r.Current().(NavigablePage); ok {
		bar.Title = p.NavItem().Name
	} else {
		bar.Title = ""
	}

	var banServ *banner.Service

	var ops op.Ops
	for {
		select {
		case event := <-w.Events():
			switch event := event.(type) {
			case system.DestroyEvent:
				return event.Err
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, event)

				if nav.NavDestinationChanged() {
					r.Push(nav.CurrentNavDestination().(string))
					if p, ok := r.Current().(AppBarPage); ok {
						bar.SetActions(p.Actions())
					} else {
						bar.SetActions(nil, nil)
					}
					if p, ok := r.Current().(NavigablePage); ok {
						bar.Title = p.NavItem().Name
					} else {
						bar.Title = ""
					}
				}

				paint.Fill(gtx.Ops, sth.Background.Default.Bg)
				if _, ok := r.Current().(StandalonePage); ok {
					r.Layout(gtx)
				} else {
					bar := layout.Rigid(func(gtx C) D {
						return bar.Layout(gtx, sth.Theme)
					})
					content := layout.Flexed(1, func(gtx C) D {
						return layout.Stack{}.Layout(gtx,
							layout.Stacked(func(gtx C) D {
								if gtx.Constraints.Max.X > gtx.Px(unit.Dp(500)) {
									// Lay out the nav non-modally.
									return resize.Layout(gtx,
										func(gtx C) D {
											return nav.NavDrawer.Layout(gtx, sth.Theme, &nonModalVis)
										},
										func(gtx C) D {
											return r.Layout(gtx)
										},
										func(gtx C) D {
											size := image.Point{
												X: gtx.Px(unit.Dp(4)),
												Y: gtx.Constraints.Max.Y,
											}
											return D{Size: size}
										},
									)
								} else {
									// Lay out the nav in a modal drawer.
									return r.Layout(gtx)
								}
							}),
							layout.Expanded(func(gtx C) D {
								if banServ == nil {
									return D{}
								}
								top := banServ.Top()
								return layoutBanner(gtx, sth, top)
							}),
						)
					})
					var elements []layout.FlexChild
					if platform.Mobile {
						elements = []layout.FlexChild{content, bar}
					} else {
						elements = []layout.FlexChild{bar, content}
					}
					layout.Flex{Axis: layout.Vertical}.Layout(gtx, elements...)
				}
				event.Frame(&ops)
			case key.Event:
				if event.Name == "N" && event.Modifiers.Contain(key.ModCtrl) {
					bus.Message(window.CreateWindowRequest{
						WindowFunc: Window,
						Options:    []app.Option{app.Title("Sprig")},
					})
				}
			}
		case update := <-bus.Output():
			window.Update(w, update)
			switch update := update.(type) {
			case SetupCompleteEvent:
				r.Push(settingsPage)
				w.Invalidate()
			case banner.Event:
				banServ = update.Service
			}
			if r.Update(update) {
				w.Invalidate()
			}
		}
	}
}

func layoutBanner(gtx C, th *sprigTheme.Theme, b banner.Banner) D {
	switch bannerConfig := b.(type) {
	case *banner.LoadingBanner:
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
