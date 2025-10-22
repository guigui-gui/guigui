// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

type PanelStyle int

const (
	PanelStyleDefault PanelStyle = iota
	PanelStyleSide
)

type PanelContentConstraints int

const (
	PanelContentConstraintsNone PanelContentConstraints = iota
	PanelContentConstraintsFixedWidth
	PanelContentConstraintsFixedHeight
)

type PanelBorder struct {
	Start  bool
	Top    bool
	End    bool
	Bottom bool
}

type Panel struct {
	guigui.DefaultWidget

	content            guigui.Widget
	scollOverlay       scrollOverlay
	border             panelBorder
	style              PanelStyle
	contentConstraints PanelContentConstraints

	hasNextOffset     bool
	nextOffsetX       float64
	nextOffsetY       float64
	isNextOffsetDelta bool
}

func (p *Panel) SetContent(widget guigui.Widget) {
	p.content = widget
}

func (p *Panel) SetStyle(typ PanelStyle) {
	if p.style == typ {
		return
	}
	p.style = typ
	guigui.RequestRedraw(p)
}

func (p *Panel) SetContentConstraints(c PanelContentConstraints) {
	p.contentConstraints = c
}

func (p *Panel) SetBorders(borders PanelBorder) {
	p.border.setBorders(borders)
}

func (p *Panel) SetAutoBorder(auto bool) {
	p.border.SetAutoBorder(auto)
}

func (p *Panel) SetScrollOffset(offsetX, offsetY float64) {
	p.hasNextOffset = true
	p.nextOffsetX = offsetX
	p.nextOffsetY = offsetY
	p.isNextOffsetDelta = false
}

func (p *Panel) SetScrollOffsetByDelta(offsetXDelta, offsetYDelta float64) {
	p.hasNextOffset = true
	p.nextOffsetX = offsetXDelta
	p.nextOffsetY = offsetYDelta
	p.isNextOffsetDelta = true
}

func (p *Panel) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	if p.content != nil {
		adder.AddChild(p.content)
	}
	adder.AddChild(&p.scollOverlay)
	adder.AddChild(&p.border)
}

func (p *Panel) contentSize(context *guigui.Context) image.Point {
	switch p.contentConstraints {
	case PanelContentConstraintsNone:
		return p.content.Measure(context, guigui.Constraints{})
	case PanelContentConstraintsFixedWidth:
		w := context.Bounds(p).Dx()
		return p.content.Measure(context, guigui.FixedWidthConstraints(w))
	case PanelContentConstraintsFixedHeight:
		h := context.Bounds(p).Dy()
		return p.content.Measure(context, guigui.FixedHeightConstraints(h))
	default:
		panic(fmt.Sprintf("basicwidget: unknown PanelContentConstraints value: %d", p.contentConstraints))
	}
}

func (p *Panel) Update(context *guigui.Context) error {
	if p.content == nil {
		return nil
	}

	contentSize := p.contentSize(context)
	if p.hasNextOffset {
		if p.isNextOffsetDelta {
			p.scollOverlay.SetOffsetByDelta(context, contentSize, p.nextOffsetX, p.nextOffsetY)
		} else {
			p.scollOverlay.SetOffset(context, contentSize, p.nextOffsetX, p.nextOffsetY)
		}
		p.hasNextOffset = false
		p.nextOffsetX = 0
		p.nextOffsetY = 0
	}

	p.scollOverlay.SetContentSize(context, contentSize)
	p.border.scrollOverlay = &p.scollOverlay

	return nil
}

func (p *Panel) Layout(context *guigui.Context, widget guigui.Widget) image.Rectangle {
	switch widget {
	case p.content:
		offsetX, offsetY := p.scollOverlay.Offset()
		pt := context.Bounds(p).Min.Add(image.Pt(int(offsetX), int(offsetY)))
		return image.Rectangle{
			Min: pt,
			Max: pt.Add(p.contentSize(context)),
		}
	case &p.scollOverlay:
		return context.Bounds(p)
	case &p.border:
		return context.Bounds(p)
	}
	return image.Rectangle{}
}

func (p *Panel) HandlePointingInput(context *guigui.Context) guigui.HandleInputResult {
	return p.scollOverlay.handlePointingInput(context)
}

func (p *Panel) Draw(context *guigui.Context, dst *ebiten.Image) {
	switch p.style {
	case PanelStyleSide:
		dst.Fill(draw.Color(context.ColorMode(), draw.ColorTypeBase, 0.9))
	}
}

type panelBorder struct {
	guigui.DefaultWidget

	scrollOverlay *scrollOverlay
	borders       PanelBorder
	autoBorder    bool
}

func (b *panelBorder) setBorders(borders PanelBorder) {
	if b.borders == borders {
		return
	}
	b.borders = borders
	guigui.RequestRedraw(b)
}

func (b *panelBorder) SetAutoBorder(auto bool) {
	if b.autoBorder == auto {
		return
	}
	b.autoBorder = auto
	guigui.RequestRedraw(b)
}

func (p *panelBorder) Draw(context *guigui.Context, dst *ebiten.Image) {
	if p.scrollOverlay == nil && p.borders == (PanelBorder{}) {
		return
	}

	// Render borders.
	strokeWidth := float32(1 * context.Scale())
	bounds := context.Bounds(p)
	x0 := float32(bounds.Min.X)
	x1 := float32(bounds.Max.X)
	y0 := float32(bounds.Min.Y)
	y1 := float32(bounds.Max.Y)
	var offsetX, offsetY float64
	var r image.Rectangle
	if p.scrollOverlay != nil {
		offsetX, offsetY = p.scrollOverlay.Offset()
		r = p.scrollOverlay.scrollRange(context)
	}
	clr := draw.Color(context.ColorMode(), draw.ColorTypeBase, 0.8)
	if (p.scrollOverlay != nil && p.autoBorder && offsetX < float64(r.Max.X)) || p.borders.Start {
		vector.StrokeLine(dst, x0+strokeWidth/2, y0, x0+strokeWidth/2, y1, strokeWidth, clr, false)
	}
	if (p.scrollOverlay != nil && p.autoBorder && offsetY < float64(r.Max.Y)) || p.borders.Top {
		vector.StrokeLine(dst, x0, y0+strokeWidth/2, x1, y0+strokeWidth/2, strokeWidth, clr, false)
	}
	if (p.scrollOverlay != nil && p.autoBorder && offsetX > float64(r.Min.X)) || p.borders.End {
		vector.StrokeLine(dst, x1-strokeWidth/2, y0, x1-strokeWidth/2, y1, strokeWidth, clr, false)
	}
	if (p.scrollOverlay != nil && p.autoBorder && offsetY > float64(r.Min.Y)) || p.borders.Bottom {
		vector.StrokeLine(dst, x0, y1-strokeWidth/2, x1, y1-strokeWidth/2, strokeWidth, clr, false)
	}
}
