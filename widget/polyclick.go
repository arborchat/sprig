package widget

import (
	"image"
	"time"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/widget"
)

// Polyclick can detect and report a variety of gesture interactions
// within a single pointer input area.
type Polyclick struct {
	// The zero value will pass through pointer events by default.
	NoPass bool
	gesture.Click
	clicks                     []widget.Click
	pressed, longPressReported bool
	pressStart                 time.Time
	currentTime                time.Time
}

func (p *Polyclick) update(gtx layout.Context) {
	p.currentTime = gtx.Now
	for _, event := range p.Click.Events(gtx) {
		switch event.Type {
		case gesture.TypeCancel:
			p.processCancel(event, gtx)
		case gesture.TypePress:
			p.processPress(event, gtx)
		case gesture.TypeClick:
			p.processClick(event, gtx)
		default:
			continue
		}
	}
}

func (p *Polyclick) processCancel(event gesture.ClickEvent, gtx layout.Context) {
	p.pressed = false
	p.longPressReported = false
}
func (p *Polyclick) processPress(event gesture.ClickEvent, gtx layout.Context) {
	p.pressed = true
	p.pressStart = gtx.Now
}
func (p *Polyclick) processClick(event gesture.ClickEvent, gtx layout.Context) {
	p.pressed = false
	if !p.longPressReported {
		p.clicks = append(p.clicks, widget.Click{
			Modifiers: event.Modifiers,
			NumClicks: event.NumClicks,
		})
	}
	p.longPressReported = false
}

func (p *Polyclick) Clicks() (out []widget.Click) {
	out, p.clicks = p.clicks, p.clicks[:0]
	return
}

func (p *Polyclick) LongPressed() bool {
	elapsed := p.currentTime.Sub(p.pressStart)
	if !p.longPressReported && p.pressed && elapsed > time.Millisecond*250 {
		p.longPressReported = true
		return true
	}
	return false
}

func (p *Polyclick) Layout(gtx layout.Context) layout.Dimensions {
	p.update(gtx)
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	p.Click.Add(gtx.Ops)
	if p.pressed {
		op.InvalidateOp{}.Add(gtx.Ops)
	}
	return layout.Dimensions{Size: gtx.Constraints.Min}
}
