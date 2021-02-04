package main

import (
	"log"
	"sort"
	"strings"
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

// Sub describes the state of a subscription to a community across many
// connected relays.
type Sub struct {
	*forest.Community
	ActiveHostingRelays []string
	Subbed              widget.Bool
}

type SubscriptionView struct {
	manager ViewManager

	core.App

	Subs           []Sub
	ConnectionList layout.List

	Updates chan []Sub
}

var _ View = &SubscriptionView{}

func NewSubscriptionView(app core.App) View {
	c := &SubscriptionView{
		App:     app,
		Updates: make(chan []Sub, 1),
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

func (c *SubscriptionView) Update(gtx layout.Context) {
outer:
	for {
		select {
		case c.Subs = <-c.Updates:
		default:
			break outer
		}
	}
	var changes []Sub
	for i := range c.Subs {
		sub := &c.Subs[i]
		if sub.Subbed.Changed() {
			changes = append(changes, *sub)
		}
	}
	if len(changes) > 0 {
		go c.implementChanges(changes)
	}
}

func (c *SubscriptionView) implementChanges(changes []Sub) {
	for _, sub := range changes {
		for _, addr := range sub.ActiveHostingRelays {
			timeout := time.NewTicker(time.Second * 5)
			worker := c.Sprout().WorkerFor(addr)
			var subFunc func(*forest.Community, <-chan time.Time) error
			var sessionFunc func(*fields.QualifiedHash)
			if !sub.Subbed.Value {
				subFunc = worker.SendUnsubscribe
				sessionFunc = worker.Unsubscribe
			} else {
				subFunc = worker.SendSubscribe
				sessionFunc = worker.Subscribe
			}
			if err := subFunc(sub.Community, timeout.C); err != nil {
				log.Printf("Failed changing sub for %s to %v on relay %s", sub.ID(), sub.Subbed.Value, addr)
			} else {
				sessionFunc(sub.Community.ID())
				log.Printf("Changed subscription for %s to %v on relay %s", sub.ID(), sub.Subbed.Value, addr)
			}
		}
	}
	c.refresh()
}

func (c *SubscriptionView) BecomeVisible() {
	go c.refresh()
}

func (c *SubscriptionView) refresh() {
	for _, conn := range c.Sprout().Connections() {
		func() {
			worker := c.Sprout().WorkerFor(conn)
			worker.Session.RLock()
			defer worker.Session.RUnlock()
			communities := map[string]Sub{}
			response, err := worker.SendList(fields.NodeTypeCommunity, 1024, time.NewTicker(time.Second*5).C)
			if err != nil {
				log.Printf("Failed listing communities on worker %s: %v", conn, err)
			} else {
				for _, n := range response.Nodes {
					n, isCommunity := n.(*forest.Community)
					if !isCommunity {
						continue
					}
					id := n.ID().String()
					existing, ok := communities[id]
					if !ok {
						existing = Sub{
							Community: n,
						}
					}
					existing.ActiveHostingRelays = append(existing.ActiveHostingRelays, conn)
					communities[id] = existing

				}
			}
			var out []Sub
			for id := range worker.Session.Communities {
				data := communities[id.String()]
				data.Subbed.Value = true
				communities[id.String()] = data
			}
			for _, sub := range communities {
				out = append(out, sub)
			}
			sort.Slice(out, func(i, j int) bool {
				iID := out[i].Community.ID().String()
				jID := out[j].Community.ID().String()
				return strings.Compare(iID, jID) < 0
			})
			c.Updates <- out
			c.manager.RequestInvalidate()
		}()
	}
}

func (c *SubscriptionView) Layout(gtx layout.Context) layout.Dimensions {
	c.Update(gtx)
	sTheme := c.Theme().Current()
	theme := sTheme.Theme

	inset := layout.UniformInset(unit.Dp(4))

	return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx C) D {
		return c.ConnectionList.Layout(gtx, len(c.Subs), func(gtx C, index int) D {
			sub := &c.Subs[index]
			var children []layout.FlexChild
			children = append(children, layout.Rigid(func(gtx C) D {
				return layout.Flex{}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return inset.Layout(gtx, func(gtx C) D {
							return material.Switch(theme, &sub.Subbed).Layout(gtx)
						})
					}),
					layout.Rigid(func(gtx C) D {
						return inset.Layout(gtx, func(gtx C) D {
							return sprigTheme.CommunityName(theme, sub.Community).Layout(gtx)
						})
					}),
				)
			}))
			for _, relay := range sub.ActiveHostingRelays {
				children = append(children, layout.Rigid(func(gtx C) D {
					return inset.Layout(gtx, func(gtx C) D {
						return material.Body2(theme, relay).Layout(gtx)
					})
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

func (c *SubscriptionView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
