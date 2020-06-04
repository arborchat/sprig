package main

import (
	"image"
	"log"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type CommunityMenuView struct {
	manager ViewManager

	*Settings
	*ArborState
	*material.Theme

	BackButton      widget.Clickable
	IdentityButton  widget.Clickable
	CommunityList   layout.List
	CommunityBoxes  []widget.Bool
	ViewButton      widget.Clickable
	ProfilingSwitch widget.Bool
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

func (c *CommunityMenuView) Update(gtx layout.Context) {
	if c.BackButton.Clicked() {
		c.manager.RequestViewSwitch(ConnectFormID)
	}
	for i := range c.CommunityBoxes {
		box := &c.CommunityBoxes[i]
		if box.Changed() {
			log.Println("updated")
		}
	}
	if c.ViewButton.Clicked() {
		c.manager.RequestViewSwitch(ReplyViewID)
	}
	if c.IdentityButton.Clicked() {
		c.manager.RequestViewSwitch(IdentityFormID)
	}
	if c.ProfilingSwitch.Changed() {
		c.manager.SetProfiling(c.ProfilingSwitch.Value)
	}
}

func (c *CommunityMenuView) Layout(gtx layout.Context) layout.Dimensions {
	theme := c.Theme
	c.CommunityList.Axis = layout.Vertical
	layout.NW.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(4)).Layout(
			gtx,
			material.IconButton(theme, &c.BackButton, icons.BackIcon).Layout,
		)
	})
	width := gtx.Constraints.Constrain(image.Point{X: gtx.Px(unit.Dp(200))}).X
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = width
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if c.Settings.ActiveIdentity != nil {
					id, _ := c.Settings.Identity()
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.Body1(c.Theme, "Identity:").Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(12)}.Layout(gtx,
								sprigTheme.AuthorName(c.Theme, id).Layout)
						}),
					)
				} else {
					return material.Button(c.Theme, &c.IdentityButton, "Create new Identity").Layout(gtx)
				}
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = width
				return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.Body1(theme, "Choose communities to join:").Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				var dims layout.Dimensions
				c.ArborState.CommunityList.WithCommunities(func(communities []*forest.Community) {
					gtx.Constraints.Max.X = width
					newCommunities := len(communities) - len(c.CommunityBoxes)
					for ; newCommunities > 0; newCommunities-- {
						c.CommunityBoxes = append(c.CommunityBoxes, widget.Bool{})
					}
					dims = c.CommunityList.Layout(gtx, len(communities), func(gtx layout.Context, index int) layout.Dimensions {
						gtx.Constraints.Max.X = width
						community := communities[index]
						checkbox := &c.CommunityBoxes[index]
						return layout.Flex{}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx,
									material.CheckBox(theme, checkbox, "").Layout,
								)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx,
									sprigTheme.CommunityName(c.Theme, community).Layout,
								)
							}),
						)
					})
				})
				return dims
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = width
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = width
					return material.Button(theme, &c.ViewButton, "View These Communities").Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = width
				in := layout.UniformInset(unit.Dp(8))
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = width
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return in.Layout(gtx, material.Body1(c.Theme, "Profile layout:").Layout)
						}),
						layout.Rigid(func(gtx C) D {
							return in.Layout(gtx, material.Switch(theme, &c.ProfilingSwitch).Layout)
						}),
					)
				})
			}),
		)
	})
}

func (c *CommunityMenuView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
