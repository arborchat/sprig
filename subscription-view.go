package main

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	materials "gioui.org/x/component"
	"git.sr.ht/~whereswaldon/sprig/core"
	"git.sr.ht/~whereswaldon/sprig/icons"
)

type SubscriptionGroup struct {
	Connection   string
	Subscribable map[string]bool
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
			communities := map[string]bool{}
			for id := range worker.Session.Communities {
				communities[id.String()] = true
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
			var children []layout.FlexChild
			children = append(children, layout.Rigid(func(gtx C) D {
				return material.H3(theme, c.ConnectionNames[index]).Layout(gtx)
			}))
			for community := range c.Connections[c.ConnectionNames[index]].Subscribable {
				children = append(children, layout.Rigid(func(gtx C) D {
					return material.Body1(theme, community).Layout(gtx)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

func (c *SubscriptionView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
