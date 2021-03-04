package main

import (
	"image"
	"log"
	"sort"
	"strings"
	"time"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	materials "gioui.org/x/component"
	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/latest"
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

	SubStateManager
	ConnectionList layout.List

	Refresh widget.Clickable
}

var _ View = &SubscriptionView{}

func NewSubscriptionView(app core.App) View {
	c := &SubscriptionView{
		App: app,
	}
	c.SubStateManager = NewSubStateManager(app, func() {
		c.manager.RequestInvalidate()
	})
	c.ConnectionList.Axis = layout.Vertical
	return c
}

func (c *SubscriptionView) AppBarData() (bool, string, []materials.AppBarAction, []materials.OverflowAction) {
	return true, "Subscriptions", []materials.AppBarAction{
		materials.SimpleIconAction(c.Theme().Current().Theme, &c.Refresh, icons.RefreshIcon, materials.OverflowAction{
			Name: "Refresh",
			Tag:  &c.Refresh,
		}),
	}, []materials.OverflowAction{}
}

func (c *SubscriptionView) NavItem() *materials.NavItem {
	return &materials.NavItem{
		Tag:  c,
		Name: "Subscriptions",
		Icon: icons.SubscriptionIcon,
	}
}

func (c *SubscriptionView) Update(gtx layout.Context) {
	c.SubStateManager.Update()
	if c.Refresh.Clicked() {
		c.SubStateManager.Refresh()
	}
}
func (c *SubscriptionView) Layout(gtx layout.Context) layout.Dimensions {
	c.Update(gtx)
	sTheme := c.Theme().Current()
	theme := sTheme.Theme

	return layout.UniformInset(unit.Dp(4)).Layout(gtx, SubscriptionList(theme, &c.ConnectionList, c.Subs).Layout)
}

func (c *SubscriptionView) SetManager(mgr ViewManager) {
	c.manager = mgr
}

func (c *SubscriptionView) BecomeVisible() {
	c.SubStateManager.Refresh()
}

// SubscriptionCardStyle configures the presentation of a card with controls and
// information about a subscription.
type SubscriptionCardStyle struct {
	*material.Theme
	*Sub
	// Inset is applied to each element of the card and can be used to
	// control their minimum spacing relative to one another.
	layout.Inset
}

func SubscriptionCard(th *material.Theme, state *Sub) SubscriptionCardStyle {
	return SubscriptionCardStyle{
		Sub:   state,
		Inset: layout.UniformInset(unit.Dp(4)),
		Theme: th,
	}
}

func (s SubscriptionCardStyle) Layout(gtx C) D {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			rr := float32(gtx.Px(unit.Dp(4)))
			outline := clip.UniformRRect(layout.FRect(image.Rectangle{Max: gtx.Constraints.Min}), rr).Op(gtx.Ops)
			paint.FillShape(gtx.Ops, s.Theme.Bg, outline)
			return D{}
		}),
		layout.Stacked(func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return s.Inset.Layout(gtx, func(gtx C) D {
				return layout.Flex{
					Spacing: layout.SpaceBetween,
				}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return s.Inset.Layout(gtx, func(gtx C) D {
							return material.Switch(s.Theme, &s.Subbed).Layout(gtx)
						})
					}),
					layout.Rigid(func(gtx C) D {
						return s.Inset.Layout(gtx, func(gtx C) D {
							return sprigTheme.CommunityName(s.Theme, s.Community).Layout(gtx)
						})
					}),
					layout.Rigid(func(gtx C) D {
						return s.Inset.Layout(gtx, func(gtx C) D {
							return material.Body2(s.Theme, strings.Join(s.ActiveHostingRelays, "\n")).Layout(gtx)
						})
					}),
				)
			})
		}),
	)
}

// SubscriptionListStyle lays out a scrollable list of subscription cards.
type SubscriptionListStyle struct {
	*material.Theme
	layout.Inset
	ConnectionList *layout.List
	Subs           []Sub
}

func SubscriptionList(th *material.Theme, list *layout.List, subs []Sub) SubscriptionListStyle {
	return SubscriptionListStyle{
		Inset:          layout.UniformInset(unit.Dp(4)),
		Theme:          th,
		ConnectionList: list,
		Subs:           subs,
	}
}

func (s SubscriptionListStyle) Layout(gtx layout.Context) layout.Dimensions {
	return s.ConnectionList.Layout(gtx, len(s.Subs),
		func(gtx C, index int) D {
			return s.Inset.Layout(gtx, func(gtx C) D {
				return SubscriptionCard(s.Theme, &s.Subs[index]).Layout(gtx)
			})
		})
}

// SubStateManager supervises and updates the list of subscribed communities
type SubStateManager struct {
	core.App
	invalidate func()
	latest.Worker
	Subs []Sub
}

// NewSubStateManager creates a new manager. The invalidate function is provided
// to it as a way to signal when the UI should be updated as a result of it
// finishing work.
func NewSubStateManager(app core.App, invalidate func()) SubStateManager {
	s := SubStateManager{App: app, invalidate: invalidate}
	s.Worker = latest.NewWorker(func(in interface{}) interface{} {
		return s.reconcileSubscriptions(in.([]Sub))
	})
	return s
}

// Update checks whether the backend has new results from the background worker
// goroutine and updates internal state to reflect those results. It should
// always be invoked before using the Subs field directly.
func (c *SubStateManager) Update() {
outer:
	for {
		select {
		case newSubs := <-c.Worker.Raw():
			c.Subs = newSubs.([]Sub)
			log.Println("updated subs", c.Subs)
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
		c.Worker.Push(changes)
	}
}

// Refresh requests the background goroutine to initiate an update of the Subs
// field.
func (c *SubStateManager) Refresh() {
	c.Worker.Push([]Sub(nil))
}

func (c *SubStateManager) reconcileSubscriptions(changes []Sub) []Sub {
	for _, sub := range changes {
		for _, addr := range sub.ActiveHostingRelays {
			timeout := time.NewTicker(time.Second * 5)
			worker := c.Sprout().WorkerFor(addr)
			var subFunc func(*forest.Community, <-chan time.Time) error
			var sessionFunc func(*fields.QualifiedHash)
			if !sub.Subbed.Value {
				subFunc = worker.SendUnsubscribe
				sessionFunc = worker.Unsubscribe
				c.Settings().RemoveSubscription(sub.Community.ID().String())
			} else {
				subFunc = worker.SendSubscribe
				sessionFunc = worker.Subscribe
				c.Settings().AddSubscription(sub.Community.ID().String())
				go core.BootstrapSubscribed(worker, []string{sub.Community.ID().String()})
			}
			if err := subFunc(sub.Community, timeout.C); err != nil {
				log.Printf("Failed changing sub for %s to %v on relay %s", sub.ID(), sub.Subbed.Value, addr)
			} else {
				sessionFunc(sub.Community.ID())
				log.Printf("Changed subscription for %s to %v on relay %s", sub.ID(), sub.Subbed.Value, addr)
			}
			go c.Settings().Persist()
		}
	}
	subs := c.refreshSubs()
	c.invalidate()
	return subs
}

func (c *SubStateManager) refreshSubs() []Sub {
	var out []Sub
	communities := map[string]Sub{}
	for _, conn := range c.Sprout().Connections() {
		func() {
			worker := c.Sprout().WorkerFor(conn)
			worker.Session.RLock()
			defer worker.Session.RUnlock()
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
			for id := range worker.Session.Communities {
				data := communities[id.String()]
				data.Subbed.Value = true
				communities[id.String()] = data
			}
		}()
	}
	for _, sub := range c.Settings().Subscriptions() {
		if _, alreadyInList := communities[sub]; alreadyInList {
			continue
		}
		var hash fields.QualifiedHash
		hash.UnmarshalText([]byte(sub))
		communityNode, has, err := c.Arbor().Store().GetCommunity(&hash)
		if err != nil {
			log.Printf("Settings indicate a subscription to %v, but loading it from local store failed: %v", sub, err)
			continue
		} else if !has {
			log.Printf("Settings indicate a subscription to %v, but it is not present in the local store.", sub)
			continue
		}
		community, ok := communityNode.(*forest.Community)
		if !ok {
			log.Printf("Settings indicate a subscription to %v, but it is not a community.", sub)
			continue
		}
		communities[sub] = Sub{
			Community:           community,
			Subbed:              widget.Bool{Value: true},
			ActiveHostingRelays: []string{"no known hosting relays"},
		}
	}
	for _, sub := range communities {
		out = append(out, sub)
	}
	sort.Slice(out, func(i, j int) bool {
		iID := out[i].Community.ID().String()
		jID := out[j].Community.ID().String()
		return strings.Compare(iID, jID) < 0
	})
	return out
}
