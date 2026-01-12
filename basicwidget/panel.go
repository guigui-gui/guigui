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

type PanelBorders struct {
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

func (p *Panel) SetBorders(borders PanelBorders) {
	p.panel.SetBorders(borders)
}

func (p *Panel) SetAutoBorder(auto bool) {
	p.panel.SetAutoBorder(auto)
}

func (p *Panel) scrollOffset() (float64, float64) {
	return p.panel.scrollOffset()
}

func (p *Panel) SetScrollOffset(offsetX, offsetY float64) {
	p.panel.SetScrollOffset(offsetX, offsetY)
}

func (p *Panel) SetScrollOffsetByDelta(offsetXDelta, offsetYDelta float64) {
	p.panel.SetScrollOffsetByDelta(offsetXDelta, offsetYDelta)
}

func (p *Panel) setScrolBarVisible(visible bool) {
	p.panel.setScrolBarVisible(visible)
}

func (p *Panel) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&p.panel)
	context.SetContainer(&p.panel, true)
	return nil
}

func (p *Panel) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
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
	scrollBarHidden   bool
}

func (p *panel) SetContent(widget guigui.Widget) {
	p.content = widget
}

func (p *panel) SetStyle(typ PanelStyle) {
	if p.style == typ {
		return
	}
	p.style = typ
	guigui.RequestRebuild(p)
}

func (p *panel) SetContentConstraints(c PanelContentConstraints) {
	p.contentConstraints = c
}

func (p *panel) SetBorders(borders PanelBorders) {
	p.border.setBorders(borders)
}

func (p *panel) SetAutoBorder(auto bool) {
	p.border.SetAutoBorder(auto)
}

func (p *panel) scrollOffset() (float64, float64) {
	// As the next offset might not be a valid offset, return the current offset.
	return p.scrollOverlay.Offset()
}

func (p *panel) nextScrollOffsetDelta() (float64, float64) {
	if !p.hasNextOffset {
		return 0, 0
	}
	if p.isNextOffsetDelta {
		return p.nextOffsetX, p.nextOffsetY
	}
	x, y := p.scrollOverlay.Offset()
	return p.nextOffsetX - x, p.nextOffsetY - y
}

func (p *panel) SetScrollOffset(offsetX, offsetY float64) {
	if x, y := p.scrollOffset(); x == offsetX && y == offsetY {
		return
	}
	p.hasNextOffset = true
	p.nextOffsetX = offsetX
	p.nextOffsetY = offsetY
	p.isNextOffsetDelta = false
	guigui.RequestRebuild(p)
}

func (p *panel) SetScrollOffsetByDelta(offsetXDelta, offsetYDelta float64) {
	if dx, dy := p.nextScrollOffsetDelta(); dx == offsetXDelta && dy == offsetYDelta {
		return
	}
	p.hasNextOffset = true
	p.nextOffsetX = offsetXDelta
	p.nextOffsetY = offsetYDelta
	p.isNextOffsetDelta = true
	guigui.RequestRebuild(p)
}

func (p *panel) setScrolBarVisible(visible bool) {
	p.scrollBarHidden = !visible
}

func (p *panel) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if p.content != nil {
		adder.AddChild(p.content)
	}
	adder.AddChild(&p.scrollOverlay)
	adder.AddChild(&p.border)
	if p.content == nil {
		return nil
	}
	p.border.scrollOverlay = &p.scrollOverlay

	context.SetVisible(&p.scrollOverlay, !p.scrollBarHidden)

	return nil
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

func (p *panel) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	bounds := widgetBounds.Bounds()
	if p.content != nil {
		if p.hasNextOffset {
			if p.isNextOffsetDelta {
				p.scrollOverlay.SetOffsetByDelta(p.nextOffsetX, p.nextOffsetY)
			} else {
				p.scrollOverlay.SetOffset(p.nextOffsetX, p.nextOffsetY)
			}
			p.hasNextOffset = false
			p.nextOffsetX = 0
			p.nextOffsetY = 0
		}

		contentSize := p.contentSize(context, widgetBounds)
		p.scrollOverlay.SetContentSize(context, widgetBounds, contentSize)

		offsetX, offsetY := p.scrollOverlay.Offset()
		pt := bounds.Min.Add(image.Pt(int(offsetX), int(offsetY)))
		layouter.LayoutWidget(p.content, image.Rectangle{
			Min: pt,
			Max: pt.Add(contentSize),
		})
	}
	layouter.LayoutWidget(&p.scrollOverlay, bounds)
	layouter.LayoutWidget(&p.border, bounds)
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
	borders       PanelBorders
	autoBorder    bool
}

func (b *panelBorder) setBorders(borders PanelBorders) {
	if b.borders == borders {
		return
	}
	b.borders = borders
	guigui.RequestRebuild(b)
}

func (b *panelBorder) SetAutoBorder(auto bool) {
	if b.autoBorder == auto {
		return
	}
	b.autoBorder = auto
	guigui.RequestRebuild(b)
}

func (p *panelBorder) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if p.scrollOverlay == nil && p.borders == (PanelBorders{}) {
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
		r = p.scrollOverlay.scrollRange(widgetBounds)
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
