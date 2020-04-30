package main

import (
	"log"
	"sort"
	"strings"
	"sync"

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
	appState.SubscribableStore.SubscribeToNewMessages(func(n forest.Node) {
		w.Invalidate()
	})
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
	appState.UIState.Update(&appState.Settings, &appState.ArborState, gtx)
}

type ArborState struct {
	sync.Once
	sprout.SubscribableStore

	communities []*forest.Community

	workerLock sync.Mutex
	workerDone chan struct{}
	workerLog  *log.Logger
}

func (a *ArborState) init() {
	a.Once.Do(func() {
		a.SubscribableStore.SubscribeToNewMessages(func(node forest.Node) {
			if community, ok := node.(*forest.Community); ok {
				index := sort.Search(len(a.communities), func(i int) bool {
					return a.communities[i].ID().Equals(community.ID())
				})
				if index >= len(a.communities) {
					a.communities = append(a.communities, community)
					sort.SliceStable(a.communities, func(i, j int) bool {
						return strings.Compare(string(a.communities[i].Name.Blob), string(a.communities[j].Name.Blob)) < 0
					})
				}
			}
		})
	})
}

func (a *ArborState) RestartWorker(address string) {
	a.init()
	a.workerLock.Lock()
	defer a.workerLock.Unlock()
	if a.workerDone != nil {
		close(a.workerDone)
	}
	a.workerDone = make(chan struct{})
	a.workerLog = log.New(log.Writer(), "worker "+address, log.LstdFlags|log.Lshortfile)
	go sprout.LaunchSupervisedWorker(a.workerDone, address, a.SubscribableStore, nil, a.workerLog)
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

func (ui *UIState) Update(config *Settings, arborState *ArborState, gtx *layout.Context) {
	switch ui.CurrentView {
	case ConnectForm:
		if ui.ConnectFormState.ConnectButton.Clicked(gtx) {
			config.Address = ui.ConnectFormState.Editor.Text()
			arborState.RestartWorker(config.Address)
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
	BackButton     widget.Button
	CommunityList  layout.List
	CommunityBoxes []widget.CheckBox
}

func Layout(appState *AppState, gtx *layout.Context) {
	appState.Update(gtx)
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
				layout.Center.Layout(gtx, func() {
					layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
						theme.Body1("Arbor Relay Address:").Layout(gtx)
					})
				})
			}),
			layout.Rigid(func() {
				layout.Center.Layout(gtx, func() {
					layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
						theme.Editor("HOST:PORT").Layout(gtx, &(ui.Editor))
					})
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
	ui.CommunityList.Axis = layout.Vertical
	theme := appState.Theme
	layout.NW.Layout(gtx, func() {
		layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
			theme.IconButton(BackIcon).Layout(gtx, &ui.CommunityMenuState.BackButton)
		})
	})
	width := gtx.Constraints.Width.Constrain(gtx.Px(unit.Dp(200)))
	layout.Center.Layout(gtx, func() {
		gtx.Constraints.Width.Max = width
		layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func() {
				gtx.Constraints.Width.Max = width
				layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
					theme.Body1("Choose communities to join:").Layout(gtx)
				})
			}),
			layout.Rigid(func() {
				gtx.Constraints.Width.Max = width
				newCommunities := len(appState.communities) - len(ui.CommunityMenuState.CommunityBoxes)
				for ; newCommunities > 0; newCommunities-- {
					ui.CommunityMenuState.CommunityBoxes = append(ui.CommunityMenuState.CommunityBoxes, widget.CheckBox{})
				}
				ui.CommunityMenuState.CommunityList.Layout(gtx, len(appState.communities), func(index int) {
					gtx.Constraints.Width.Max = width
					community := appState.communities[index]
					checkbox := &ui.CommunityMenuState.CommunityBoxes[index]
					layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func() {
							layout.Flex{}.Layout(gtx,
								layout.Rigid(func() {
									layout.UniformInset(unit.Dp(8)).Layout(gtx, func() {
										box := appState.Theme.CheckBox("")
										box.Layout(gtx, checkbox)
									})
								}),
								layout.Rigid(func() {
									layout.UniformInset(unit.Dp(8)).Layout(gtx, func() {
										theme.H6(string(community.Name.Blob)).Layout(gtx)
									})
								}),
							)
						}),
						layout.Rigid(func() {
							layout.UniformInset(unit.Dp(8)).Layout(gtx, func() {
								appState.Theme.Body2(community.ID().String()).Layout(gtx)
							})
						}),
					)
				})
			}),
		)
	})
}
