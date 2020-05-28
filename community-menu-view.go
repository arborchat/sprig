package main

import (
	"log"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/sprig/icons"
)

type CommunityMenuView struct {
	manager ViewManager

	*Settings
	*ArborState
	*material.Theme

	BackButton     widget.Clickable
	IdentityButton widget.Clickable
	CommunityList  layout.List
	CommunityBoxes []widget.Bool
	ViewButton     widget.Clickable
}

var _ View = &CommunityMenuView{}

func NewCommunityMenuView(settings *Settings, arborState *ArborState, theme *material.Theme) View {
	c := &CommunityMenuView{
		Settings:   settings,
		ArborState: arborState,
		Theme:      theme,
	}
	return c
}

func (c *CommunityMenuView) HandleClipboard(contents string) {
}

func (c *CommunityMenuView) Update(gtx *layout.Context) {
	if c.BackButton.Clicked(gtx) {
		c.manager.RequestViewSwitch(ConnectForm)
	}
	for i := range c.CommunityBoxes {
		box := &c.CommunityBoxes[i]
		if box.Update(gtx) {
			log.Println("updated")
		}
	}
	if c.ViewButton.Clicked(gtx) {
		c.manager.RequestViewSwitch(ReplyView)
	}
	if c.IdentityButton.Clicked(gtx) {
		c.manager.RequestViewSwitch(IdentityForm)
	}
}

func (c *CommunityMenuView) Layout(gtx *layout.Context) {
	theme := c.Theme
	c.CommunityList.Axis = layout.Vertical
	layout.NW.Layout(gtx, func() {
		layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
			material.IconButton(theme, icons.BackIcon).Layout(gtx, &c.BackButton)
		})
	})
	width := gtx.Constraints.Width.Constrain(gtx.Px(unit.Dp(200)))
	layout.Center.Layout(gtx, func() {
		gtx.Constraints.Width.Max = width
		layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func() {
				if c.Settings.ActiveIdentity != nil {
					material.Body1(c.Theme, "Identity: "+c.Settings.ActiveIdentity.String()).Layout(gtx)
				} else {
					material.Button(c.Theme, "Create new Identity").Layout(gtx, &c.IdentityButton)
				}
			}),
			layout.Rigid(func() {
				gtx.Constraints.Width.Max = width
				layout.UniformInset(unit.Dp(4)).Layout(gtx, func() {
					material.Body1(theme, "Choose communities to join:").Layout(gtx)
				})
			}),
			layout.Rigid(func() {
				c.ArborState.CommunityList.WithCommunities(func(communities []*forest.Community) {
					gtx.Constraints.Width.Max = width
					newCommunities := len(communities) - len(c.CommunityBoxes)
					for ; newCommunities > 0; newCommunities-- {
						c.CommunityBoxes = append(c.CommunityBoxes, widget.Bool{})
					}
					c.CommunityList.Layout(gtx, len(communities), func(index int) {
						gtx.Constraints.Width.Max = width
						community := communities[index]
						checkbox := &c.CommunityBoxes[index]
						layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func() {
								layout.Flex{}.Layout(gtx,
									layout.Rigid(func() {
										layout.UniformInset(unit.Dp(8)).Layout(gtx, func() {
											box := material.CheckBox(theme, "")
											box.Layout(gtx, checkbox)
										})
									}),
									layout.Rigid(func() {
										layout.UniformInset(unit.Dp(8)).Layout(gtx, func() {
											material.H6(theme, string(community.Name.Blob)).Layout(gtx)
										})
									}),
								)
							}),
							layout.Rigid(func() {
								layout.UniformInset(unit.Dp(8)).Layout(gtx, func() {
									material.Body2(theme, community.ID().String()).Layout(gtx)
								})
							}),
						)
					})
				})
			}),
			layout.Rigid(func() {
				gtx.Constraints.Width.Max = width
				layout.Center.Layout(gtx, func() {
					gtx.Constraints.Width.Max = width
					material.Button(theme, "View These Communities").Layout(gtx, &c.ViewButton)
				})
			}),
		)
	})
}

func (c *CommunityMenuView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
