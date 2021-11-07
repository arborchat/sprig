package pages

import (
	"image"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/x/component"
	"git.sr.ht/~gioverse/skel/router"
	"git.sr.ht/~gioverse/skel/scheduler"
	"git.sr.ht/~gioverse/skel/window"
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

const (
	settingsPage string = "settings"
)

// Window is the main event loop for application windows.
func Window(w *app.Window, bus scheduler.Connection) error {
	sth := sprigTheme.New()
	sp := &Settings{Th: sth, Conn: bus}
	r := router.Router{
		Pages: map[string]router.Page{
			settingsPage: sp,
		},
	}
	r.Push(settingsPage)

	modal := component.NewModal()
	nav := component.NewModalNav(modal, "Sprig", "Arbor Chat Client")
	nav.AddNavItem(sp.NavItem())
	nonModalVis := component.VisibilityAnimation{
		State: component.Visible,
	}
	resize := component.Resize{
		Ratio: 0.3,
	}
	var ops op.Ops
	for {
		select {
		case event := <-w.Events():
			switch event := event.(type) {
			case system.DestroyEvent:
				return event.Err
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, event)
				paint.Fill(gtx.Ops, sth.Background.Default.Bg)
				if gtx.Constraints.Max.X > gtx.Px(unit.Dp(500)) {
					// Lay out the nav non-modally.
					resize.Layout(gtx,
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
					r.Layout(gtx)
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
			if r.Update(update) {
				w.Invalidate()
			}
		}
	}
}
