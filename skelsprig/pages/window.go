package pages

import (
	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"git.sr.ht/~gioverse/skel/router"
	"git.sr.ht/~gioverse/skel/scheduler"
	"git.sr.ht/~gioverse/skel/window"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// Window is the main event loop for application windows.
func Window(w *app.Window, bus scheduler.Connection) error {
	sth := sprigTheme.New()
	r := router.Router{
		Pages: map[string]router.Page{
			"settings": &Settings{Th: sth, Conn: bus},
		},
	}
	r.Push("settings")
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
				r.Layout(gtx)
				event.Frame(&ops)
			}
		case update := <-bus.Output():
			window.Update(w, update)
			if r.Update(update) {
				w.Invalidate()
			}
		}
	}
}
