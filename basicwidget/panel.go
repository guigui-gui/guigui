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

	panel panel
}

func (p *Panel) SetContent(widget guigui.Widget) {
	p.panel.SetContent(widget)
}

func (p *Panel) SetStyle(typ PanelStyle) {
	p.panel.SetStyle(typ)
}

func (p *Panel) SetContentConstraints(c PanelContentConstraints) {
	p.panel.SetContentConstraints(c)
}

func (p *Panel) SetBorders(borders PanelBorder) {
	p.panel.SetBorders(borders)
}

func (p *Panel) SetAutoBorder(auto bool) {
	p.panel.SetAutoBorder(auto)
}

func (p *Panel) SetScrollOffset(offsetX, offsetY float64) {
	p.panel.SetScrollOffset(offsetX, offsetY)
}

func (p *Panel) SetScrollOffsetByDelta(offsetXDelta, offsetYDelta float64) {
	p.panel.SetScrollOffsetByDelta(offsetXDelta, offsetYDelta)
}

func (p *Panel) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&p.panel)
}

func (p *Panel) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&p.panel, widgetBounds.Bounds())
}

func (p *Panel) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return p.panel.Measure(context, constraints)
}

type panel struct {
	guigui.DefaultWidget

	content            guigui.Widget
	scrollOverlay      scrollOverlay
	border             panelBorder
	style              PanelStyle
	contentConstraints PanelContentConstraints

	hasNextOffset     bool
	nextOffsetX       float64
	nextOffsetY       float64
	isNextOffsetDelta bool
}

func (p *panel) SetContent(widget guigui.Widget) {
	p.content = widget
}

func (p *panel) SetStyle(typ PanelStyle) {
	if p.style == typ {
		return
	}
	p.style = typ
	guigui.RequestRedraw(p)
}

func (p *panel) SetContentConstraints(c PanelContentConstraints) {
	p.contentConstraints = c
}

func (p *panel) SetBorders(borders PanelBorder) {
	p.border.setBorders(borders)
}

func (p *panel) SetAutoBorder(auto bool) {
	p.border.SetAutoBorder(auto)
}

func (p *panel) SetScrollOffset(offsetX, offsetY float64) {
	p.hasNextOffset = true
	p.nextOffsetX = offsetX
	p.nextOffsetY = offsetY
	p.isNextOffsetDelta = false
}

func (p *panel) SetScrollOffsetByDelta(offsetXDelta, offsetYDelta float64) {
	p.hasNextOffset = true
	p.nextOffsetX = offsetXDelta
	p.nextOffsetY = offsetYDelta
	p.isNextOffsetDelta = true
}

func (p *panel) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	if p.content != nil {
		adder.AddChild(p.content)
	}
	adder.AddChild(&p.scrollOverlay)
	adder.AddChild(&p.border)
}

func (p *panel) contentSize(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Point {
	switch p.contentConstraints {
	case PanelContentConstraintsNone:
		return p.content.Measure(context, guigui.Constraints{})
	case PanelContentConstraintsFixedWidth:
		w := widgetBounds.Bounds().Dx()
		return p.content.Measure(context, guigui.FixedWidthConstraints(w))
	case PanelContentConstraintsFixedHeight:
		h := widgetBounds.Bounds().Dy()
		return p.content.Measure(context, guigui.FixedHeightConstraints(h))
	default:
		panic(fmt.Sprintf("basicwidget: unknown PanelContentConstraints value: %d", p.contentConstraints))
	}
}

func (p *panel) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if p.content == nil {
		return nil
	}

	contentSize := p.contentSize(context, widgetBounds)
	if p.hasNextOffset {
		if p.isNextOffsetDelta {
			p.scrollOverlay.SetOffsetByDelta(context, widgetBounds, contentSize, p.nextOffsetX, p.nextOffsetY)
		} else {
			p.scrollOverlay.SetOffset(context, widgetBounds, contentSize, p.nextOffsetX, p.nextOffsetY)
		}
		p.hasNextOffset = false
		p.nextOffsetX = 0
		p.nextOffsetY = 0
	}

	p.scrollOverlay.SetContentSize(context, widgetBounds, contentSize)
	p.border.scrollOverlay = &p.scrollOverlay

	return nil
}

func (p *panel) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	bounds := widgetBounds.Bounds()
	if p.content != nil {
		offsetX, offsetY := p.scrollOverlay.Offset()
		pt := bounds.Min.Add(image.Pt(int(offsetX), int(offsetY)))
		layouter.LayoutWidget(p.content, image.Rectangle{
			Min: pt,
			Max: pt.Add(p.contentSize(context, widgetBounds)),
		})
	}
	layouter.LayoutWidget(&p.scrollOverlay, bounds)
	layouter.LayoutWidget(&p.border, bounds)
}

func (p *panel) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	return p.scrollOverlay.handlePointingInput(context, widgetBounds)
}

func (p *panel) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
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

func (p *panelBorder) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if p.scrollOverlay == nil && p.borders == (PanelBorder{}) {
		return
	}

	// Render borders.
	strokeWidth := float32(1 * context.Scale())
	bounds := widgetBounds.Bounds()
	x0 := float32(bounds.Min.X)
	x1 := float32(bounds.Max.X)
	y0 := float32(bounds.Min.Y)
	y1 := float32(bounds.Max.Y)
	var offsetX, offsetY float64
	var r image.Rectangle
	if p.scrollOverlay != nil {
		offsetX, offsetY = p.scrollOverlay.Offset()
		r = p.scrollOverlay.scrollRange(context, widgetBounds)
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
