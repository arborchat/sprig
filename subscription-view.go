package main

import (
	"log"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	materials "gioui.org/x/component"
	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/sprig/core"
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type SubscriptionGroup struct {
	Connection   string
	Subscribable map[string]*SubscriptionData
}

type SubscriptionData struct {
	Subscribed widget.Bool
	*forest.Community
}

type SubscriptionView struct {
	manager ViewManager

	core.App

	ConnectionNames []string
	Connections     map[string]SubscriptionGroup
	ConnectionList  layout.List

	Updates chan SubscriptionGroup
}

var _ View = &SubscriptionView{}

func NewSubscriptionView(app core.App) View {
	c := &SubscriptionView{
		App:         app,
		Updates:     make(chan SubscriptionGroup, 1),
		Connections: make(map[string]SubscriptionGroup),
	}
	c.ConnectionList.Axis = layout.Vertical
	return c
}

func (c *SubscriptionView) AppBarData() (bool, string, []materials.AppBarAction, []materials.OverflowAction) {
	return true, "Subscriptions", []materials.AppBarAction{}, []materials.OverflowAction{}
}

func (c *SubscriptionView) NavItem() *materials.NavItem {
	return &materials.NavItem{
		Name: "Subscriptions",
		Icon: icons.SubscriptionIcon,
	}
}

func (c *SubscriptionView) HandleClipboard(contents string) {
}

func (c *SubscriptionView) Update(gtx layout.Context) {
outer:
	for {
		select {
		case update := <-c.Updates:
			c.Connections[update.Connection] = update
		default:
			break outer
		}
	}
	var changes []SubscriptionData
	for _, group := range c.Connections {
		for _, subdata := range group.Subscribable {
			if subdata.Subscribed.Changed() {
				changes = append(changes, *subdata)
			}
		}
	}
	if len(changes) > 0 {
		go c.implementChanges(changes)
	}
}

func (c *SubscriptionView) implementChanges(changes []SubscriptionData) {
}

func (c *SubscriptionView) BecomeVisible() {
	connections := c.Sprout().Connections()
	for _, conn := range connections {
		if _, ok := c.Connections[conn]; !ok {
			c.Connections[conn] = SubscriptionGroup{}
		}
	}
	c.ConnectionNames = connections
	go c.refresh(connections)
}

func (c *SubscriptionView) refresh(connections []string) {
	for _, conn := range connections {
		func() {
			worker := c.Sprout().WorkerFor(conn)
			worker.Session.RLock()
			defer worker.Session.RUnlock()
			communities := map[string]*SubscriptionData{}
			response, err := worker.SendList(fields.NodeTypeCommunity, 1024, time.NewTicker(time.Second*5).C)
			if err != nil {
				log.Printf("Failed listing communities on worker %s: %v", conn, err)
			} else {
				for _, n := range response.Nodes {
					n, isCommunity := n.(*forest.Community)
					if !isCommunity {
						continue
					}
					subdata := SubscriptionData{
						Community: n,
					}
					subdata.Subscribed.Value = false
					communities[n.ID().String()] = &subdata

				}
			}
			for id := range worker.Session.Communities {
				data := communities[id.String()]
				data.Subscribed.Value = true
				communities[id.String()] = data
			}
			c.Updates <- SubscriptionGroup{
				Connection:   conn,
				Subscribable: communities,
			}
			c.manager.RequestInvalidate()
		}()
	}
}

func (c *SubscriptionView) Layout(gtx layout.Context) layout.Dimensions {
	c.Update(gtx)
	sTheme := c.Theme().Current()
	theme := sTheme.Theme

	return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx C) D {
		return c.ConnectionList.Layout(gtx, len(c.ConnectionNames), func(gtx C, index int) D {
			name := c.ConnectionNames[index]
			connection := c.Connections[name]
			var children []layout.FlexChild
			children = append(children, layout.Rigid(func(gtx C) D {
				return material.H5(theme, name).Layout(gtx)
			}))
			for community, subdata := range connection.Subscribable {
				children = append(children, layout.Rigid(func(gtx C) D {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return material.Switch(theme, &subdata.Subscribed).Layout(gtx)
						}),
						layout.Flexed(1, func(gtx C) D {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx C) D {
									return sprigTheme.CommunityName(theme, connection.Subscribable[community].Community).Layout(gtx)
								}),
								layout.Rigid(func(gtx C) D {
									return material.Body2(theme, community).Layout(gtx)
								}),
							)
						}),
					)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

func (c *SubscriptionView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
