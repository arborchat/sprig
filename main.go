package main

import (
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
				layout.Inset{
					Top:    e.Insets.Top,
					Bottom: e.Insets.Bottom,
					Left:   e.Insets.Left,
					Right:  e.Insets.Right,
				}.Layout(gtx,
					func() {
						layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func() {
								title := theme.H3("sprig")
								title.Alignment = text.Start
								title.Layout(gtx)
							}),
							layout.Flexed(1, func() {
								listLen := len(archive.ReplyList)
								list.Layout(gtx, listLen, func(i int) {
									layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
										reply := archive.ReplyList[i]
										Layout(gtx, theme, reply, store)
									})
								})
							}),
						)
					},
				)
				e.Frame(gtx.Ops)
			}
		}
	}()
	app.Main()
}

func Layout(gtx *layout.Context, theme *material.Theme, reply *forest.Reply, store forest.Store) {
	author, has, err := store.GetIdentity(&reply.Author)
	if err != nil {
		log.Printf("failed finding %s in store: %v", &reply.Author, err)
	}
	if !has {
		author = &forest.Identity{}
	}
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func() {
			layout.Flex{}.Layout(gtx,
				layout.Rigid(func() {
					label := theme.Label(unit.Dp(10), string(author.(*forest.Identity).Name.Blob)+"  ")
					label.Font.Weight = text.Bold
					label.Layout(gtx)
				}),
				layout.Rigid(func() {
					theme.Label(unit.Dp(10), reply.Created.Time().Local().Format("2006/01/02 15:04")).Layout(gtx)
				}),
			)
		}),
		layout.Rigid(func() {
			element := theme.Body1(string(reply.Content.Blob))
			element.Layout(gtx)
		}),
	)
}

func LaunchWorker() (chan<- struct{}, *archive.Archive, sprout.SubscribableStore, error) {
	arch, err := archive.NewArchive(forest.NewMemoryStore())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed wrapping store in archive: %w", err)
	}
	store := sprout.NewSubscriberStore(arch)
	const address = "arbor.chat:7117"
	doneChan := make(chan struct{})
	sprout.LaunchSupervisedWorker(doneChan, address, store, nil, log.New(log.Writer(), address+" ", log.LstdFlags))
	return doneChan, arch, store, nil
}
