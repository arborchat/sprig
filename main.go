package main

import (
	"crypto/tls"
	"fmt"
	"log"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/sprout-go"
	"git.sr.ht/~whereswaldon/wisteria/archive"
)

func init() {
	gofont.Register()
}

func main() {
	window := app.NewWindow(app.Title("Sprig"))
	theme := material.NewTheme()
	gtx := layout.NewContext(window.Queue())
	list := &layout.List{
		Axis:        layout.Vertical,
		ScrollToEnd: true,
	}
	go func() {
		done, archive, store, err := LaunchWorker()
		if err != nil {
			log.Fatalf("Failed launching worker: %v", err)
		}
		store.SubscribeToNewMessages(func(forest.Node) {
			archive.Sort()
			window.Invalidate()
		})
		for e := range window.Events() {
			switch e := e.(type) {
			case system.DestroyEvent:
				close(done)
				log.Fatalf("Got destroy event: %v", e)
			case system.FrameEvent:
				gtx.Reset(e.Config, e.Size)
				if list.Dragging() {
					key.HideInputOp{}.Add(gtx.Ops)
				}
				layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func() {
						title := theme.H1("sprig")
						title.Alignment = text.Middle
						title.Layout(gtx)
					}),
					layout.Flexed(1, func() {
						listLen := len(archive.ReplyList)
						list.Layout(gtx, listLen, func(i int) {
							layout.UniformInset(unit.Dp(8)).Layout(gtx, func() {
								message := string(archive.ReplyList[i].Content.Blob)
								element := theme.Body1(message)
								element.Layout(gtx)
							})
						})
					}))
				e.Frame(gtx.Ops)
			}
		}
	}()
	app.Main()
}

func LaunchWorker() (chan<- struct{}, *archive.Archive, sprout.SubscribableStore, error) {
	arch, err := archive.NewArchive(forest.NewMemoryStore())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed wrapping store in archive: %w", err)
	}
	store := sprout.NewSubscriberStore(arch)
	const address = "arbor.chat:7117"
	conn, err := tls.Dial("tcp", address, &tls.Config{})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed dialing %s: %w", address, err)
	}
	doneChan := make(chan struct{})
	worker, err := sprout.NewWorker(doneChan, conn, store)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed launching worker: %w", err)
	}
	go worker.Run()
	go worker.BootstrapLocalStore(1024)
	return doneChan, arch, store, nil
}
