package main

import (
	"log"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/sprout-go"

	"golang.org/x/exp/shiny/materialdesign/icons"
)

func main() {
	gofont.Register()
	go func() {
		w := app.NewWindow()
		if err := eventLoop(w); err != nil {
			log.Println(err)
			return
		}
	}()
	app.Main()
}

func eventLoop(w *app.Window) error {
	appState := NewAppState()
	gtx := layout.NewContext(w.Queue())
	for {
		switch event := (<-w.Events()).(type) {
		case system.DestroyEvent:
			return event.Err
		case system.FrameEvent:
			gtx.Reset(event.Config, event.Size)
			Layout(appState, gtx)
			event.Frame(gtx.Ops)
		}
	}
}

type AppState struct {
	Settings
	ArborState
	UIState
	*material.Theme
}

func NewAppState() *AppState {
	memStore := forest.NewMemoryStore()
	return &AppState{
		ArborState: ArborState{
			SubscribableStore: sprout.NewSubscriberStore(memStore),
		},
		Theme: material.NewTheme(),
	}
}

func (appState *AppState) Update(gtx *layout.Context) {
	appState.UIState.Update(&appState.Settings, gtx)
}

type ArborState struct {
	sprout.SubscribableStore
}

type Settings struct {
	Address string
}

type ViewID int

const (
	ConnectForm ViewID = iota
	CommunityMenu
)

type UIState struct {
	CurrentView ViewID
	ConnectFormState
	CommunityMenuState
}

func (ui *UIState) Update(config *Settings, gtx *layout.Context) {
	switch ui.CurrentView {
	case ConnectForm:
		if ui.ConnectFormState.ConnectButton.Clicked(gtx) {
			config.Address = ui.ConnectFormState.Editor.Text()
			ui.CurrentView = CommunityMenu
		}
	case CommunityMenu:
		if ui.CommunityMenuState.BackButton.Clicked(gtx) {
			ui.CurrentView = ConnectForm
		}
	}
}

type ConnectFormState struct {
	widget.Editor
	ConnectButton widget.Button
}

type CommunityMenuState struct {
	BackButton    widget.Button
	CommunityList layout.List
}

func Layout(appState *AppState, gtx *layout.Context) {
	ui := &appState.UIState
	switch ui.CurrentView {
	case ConnectForm:
		LayoutConnectForm(appState, gtx)
	case CommunityMenu:
		LayoutCommunityMenu(appState, gtx)
	default:
	}
}

func LayoutConnectForm(appState *AppState, gtx *layout.Context) {
	ui := &appState.UIState
	theme := appState.Theme
	layout.Center.Layout(gtx, func() {
		layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func() {
				layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
					theme.Body1("Arbor Relay Address:").Layout(gtx)
				})
			}),
			layout.Rigid(func() {
				layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
					theme.Editor("HOST:PORT").Layout(gtx, &(ui.Editor))
				})
			}),
			layout.Rigid(func() {
				layout.Center.Layout(gtx, func() {
					layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
						theme.Button("Connect").Layout(gtx, &(ui.ConnectButton))
					})
				})
			}),
		)
	})
}

var BackIcon *material.Icon = func() *material.Icon {
	icon, _ := material.NewIcon(icons.NavigationArrowBack)
	return icon
}()

func LayoutCommunityMenu(appState *AppState, gtx *layout.Context) {
	ui := &appState.UIState
	theme := appState.Theme
	layout.NW.Layout(gtx, func() {
		layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
			theme.IconButton(BackIcon).Layout(gtx, &ui.CommunityMenuState.BackButton)
		})
	})
	layout.Center.Layout(gtx, func() {
		layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func() {
				layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
					theme.Body1("Choose communities to join:").Layout(gtx)
				})
			}),
			layout.Rigid(func() {
				ui.CommunityMenuState.CommunityList.Layout(gtx, 3, func(index int) {
				})
			}),
		)
	})
}
