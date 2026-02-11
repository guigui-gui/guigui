// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

var (
	panelEventScroll guigui.EventKey = guigui.GenerateEventKey()
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

func (p *Panel) OnScroll(callback func(context *guigui.Context, offsetX, offsetY float64)) {
	p.panel.OnScroll(callback)
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
	p.panel.setScrolVisible(visible)
}

func (p *Panel) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&p.panel)
	context.SetClipChildren(&p.panel, true)
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

	scrollWheel scrollWheel
	content     guigui.Widget
	scrollHBar  scrollBar
	scrollVBar  scrollBar

	border             panelBorder
	style              PanelStyle
	contentConstraints PanelContentConstraints
	scrollHidden       bool

	offsetX             float64
	offsetY             float64
	nextOffsetSet       bool
	isNextOffsetDelta   bool
	nextOffsetX         float64
	nextOffsetY         float64
	scrollBarCount      int
	contentSizeAtLayout image.Point
}

func (p *panel) OnScroll(callback func(context *guigui.Context, offsetX, offsetY float64)) {
	guigui.AddEventHandler(p, panelEventScroll, callback)
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
	return p.offsetX, p.offsetY
}

// SetScrollOffsetByDelta sets the offset by adding dx and dy to the current offset.
func (p *panel) SetScrollOffsetByDelta(dx, dy float64) {
	p.nextOffsetSet = true
	p.isNextOffsetDelta = true
	p.nextOffsetX = dx
	p.nextOffsetY = dy
}

// SetScrollOffset sets the offset to (x, y).
func (p *panel) SetScrollOffset(x, y float64) {
	p.setScrollOffset(x, y)
}

func (p *panel) setScrollOffset(x, y float64) {
	if p.offsetX == x && p.offsetY == y {
		return
	}
	p.nextOffsetSet = true
	p.isNextOffsetDelta = false
	p.nextOffsetX = x
	p.nextOffsetY = y
}

func (p *panel) setScrolVisible(visible bool) {
	p.scrollHidden = !visible
}

func (p *panel) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&p.scrollWheel)
	if p.content != nil {
		adder.AddChild(p.content)
	}
	adder.AddChild(&p.scrollHBar)
	adder.AddChild(&p.scrollVBar)
	adder.AddChild(&p.border)

	p.border.panel = p
	context.SetVisible(&p.scrollWheel, !p.scrollHidden)
	context.SetVisible(&p.scrollHBar, !p.scrollHidden)
	context.SetVisible(&p.scrollVBar, !p.scrollHidden)

	p.scrollWheel.setOffsetGetSetter(p)
	p.scrollHBar.setOffsetGetSetter(p)
	p.scrollHBar.setHorizontal(true)
	p.scrollVBar.setOffsetGetSetter(p)
	p.scrollVBar.setHorizontal(false)

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
		p.contentSizeAtLayout = p.contentSize(context, widgetBounds)
		p.scrollWheel.setContentSize(p.contentSizeAtLayout)
		p.scrollHBar.setContentSize(p.contentSizeAtLayout)
		p.scrollVBar.setContentSize(p.contentSizeAtLayout)
		p.SetScrollOffset(p.adjustOffset(context, widgetBounds, p.offsetX, p.offsetY))

		pt := bounds.Min.Add(image.Pt(int(p.offsetX), int(p.offsetY)))
		layouter.LayoutWidget(p.content, image.Rectangle{
			Min: pt,
			Max: pt.Add(p.contentSizeAtLayout),
		})
	} else {
		p.contentSizeAtLayout = image.Point{}
		p.SetScrollOffset(p.adjustOffset(context, widgetBounds, p.offsetX, p.offsetY))
	}

	layouter.LayoutWidget(&p.border, bounds)
	layouter.LayoutWidget(&p.scrollWheel, widgetBounds.Bounds())
	layouter.LayoutWidget(&p.scrollHBar, p.horizontalBarBounds(context, widgetBounds))
	layouter.LayoutWidget(&p.scrollVBar, p.verticalBarBounds(context, widgetBounds))

	hb, vb := p.thumbBounds(context, widgetBounds)
	p.scrollHBar.setThumbBounds(hb)
	p.scrollVBar.setThumbBounds(vb)
}

func (p *panel) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	switch p.style {
	case PanelStyleSide:
		dst.Fill(basicwidgetdraw.BackgroundSecondaryColor(context.ColorMode()))
	}
}

func (p *panel) horizontalBarBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	bounds := widgetBounds.Bounds()
	bounds.Min.Y = max(bounds.Min.Y, bounds.Max.Y-UnitSize(context)/2)
	return bounds
}

func (p *panel) verticalBarBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	bounds := widgetBounds.Bounds()
	bounds.Min.X = max(bounds.Min.X, bounds.Max.X-UnitSize(context)/2)
	return bounds
}

func (p *panel) offset() (float64, float64) {
	// As the next offset might not be a valid offset, return the current offset.
	return p.offsetX, p.offsetY
}

func (p *panel) adjustOffset(context *guigui.Context, widgetBounds *guigui.WidgetBounds, x, y float64) (float64, float64) {
	r := p.scrollRange(context, widgetBounds)
	x = min(max(x, float64(r.Min.X)), float64(r.Max.X))
	y = min(max(y, float64(r.Min.Y)), float64(r.Max.Y))
	return x, y
}

func (p *panel) scrollRange(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	bounds := widgetBounds.Bounds()
	cs := p.contentSizeAtLayout
	return image.Rectangle{
		Min: image.Pt(min(bounds.Dx()-cs.X, 0), min(bounds.Dy()-cs.Y, 0)),
		Max: image.Pt(0, 0),
	}
}

func (p *panel) isBarVisible(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	if p.scrollWheel.isScrolling() {
		return true
	}
	if p.scrollHBar.isDragging() || p.scrollVBar.isDragging() {
		return true
	}
	if !widgetBounds.IsHitAtCursor() {
		return false
	}
	pt := image.Pt(ebiten.CursorPosition())
	if pt.In(p.horizontalBarBounds(context, widgetBounds)) {
		return true
	}
	if pt.In(p.verticalBarBounds(context, widgetBounds)) {
		return true
	}
	return false
}

func (p *panel) startShowingBarsIfNeeded(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
	if hb, vb := p.thumbBounds(context, widgetBounds); hb.Empty() && vb.Empty() {
		return
	}

	switch {
	case p.scrollBarCount >= scrollBarMaxCount()-scrollBarFadingInTime():
		// If the scroll bar is being fading in, do nothing.
	case p.scrollBarCount >= scrollBarFadingOutTime():
		// If the scroll bar is shown, reset the count.
		p.scrollBarCount = scrollBarMaxCount() - scrollBarFadingInTime()
	case p.scrollBarCount > 0:
		// If the scroll bar is fading out, reset the count.
		p.scrollBarCount = scrollBarMaxCount() - scrollBarFadingInTime()
	default:
		p.scrollBarCount = scrollBarMaxCount()
	}
}

func (p *panel) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	shouldShowBar := p.isBarVisible(context, widgetBounds)

	if p.nextOffsetSet {
		var newOffsetX, newOffsetY float64
		if p.isNextOffsetDelta {
			newOffsetX = p.offsetX + p.nextOffsetX
			newOffsetY = p.offsetY + p.nextOffsetY
		} else {
			newOffsetX = p.nextOffsetX
			newOffsetY = p.nextOffsetY
		}
		newOffsetX, newOffsetY = p.adjustOffset(context, widgetBounds, newOffsetX, newOffsetY)
		if p.offsetX != newOffsetX || p.offsetY != newOffsetY {
			p.offsetX = newOffsetX
			p.offsetY = newOffsetY
			guigui.DispatchEvent(p, panelEventScroll, p.offsetX, p.offsetY)
			// Rebuilding the widget tree is needed to invoke this panel's Layout (#298).
			guigui.RequestRebuild(p)
		}
		p.nextOffsetSet = false
		p.nextOffsetX = 0
		p.nextOffsetY = 0
		p.isNextOffsetDelta = false
		if p.scrollHBar.isOnceDrawn() || p.scrollVBar.isOnceDrawn() {
			shouldShowBar = true
		}
	}

	oldOpacity := scrollThumbOpacity(p.scrollBarCount)
	if shouldShowBar {
		p.startShowingBarsIfNeeded(context, widgetBounds)
	}
	newOpacity := scrollThumbOpacity(p.scrollBarCount)

	if newOpacity != oldOpacity {
		guigui.RequestRedraw(p)
	}

	if p.scrollBarCount > 0 {
		if !shouldShowBar || p.scrollBarCount != scrollBarMaxCount()-scrollBarFadingInTime() {
			p.scrollBarCount--
		}
	}

	alpha := scrollThumbOpacity(p.scrollBarCount) * 3 / 4
	p.scrollHBar.setAlpha(alpha)
	p.scrollVBar.setAlpha(alpha)

	return nil
}

func (p *panel) thumbBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (image.Rectangle, image.Rectangle) {
	bounds := widgetBounds.Bounds()

	cs := p.contentSizeAtLayout
	offsetX, offsetY := p.offset()
	barWidth, barHeight := scrollThumbSize(context, widgetBounds, cs)

	padding := scrollThumbPadding(context)

	var horizontalBarBounds, verticalBarBounds image.Rectangle
	if cs.X > bounds.Dx() {
		rate := -offsetX / float64(cs.X-bounds.Dx())
		x0 := float64(bounds.Min.X) + padding + rate*(float64(bounds.Dx())-2*padding-barWidth)
		x1 := x0 + float64(barWidth)
		var y0, y1 float64
		if scrollThumbStrokeWidth(context) > float64(bounds.Dy())*0.3 {
			y0 = float64(bounds.Max.Y) - float64(bounds.Dy())*0.3
			y1 = float64(bounds.Max.Y)
		} else {
			y0 = float64(bounds.Max.Y) - padding - scrollThumbStrokeWidth(context)
			y1 = float64(bounds.Max.Y) - padding
		}
		horizontalBarBounds = image.Rect(int(x0), int(y0), int(x1), int(y1))
	}
	if cs.Y > bounds.Dy() {
		rate := -offsetY / float64(cs.Y-bounds.Dy())
		y0 := float64(bounds.Min.Y) + padding + rate*(float64(bounds.Dy())-2*padding-barHeight)
		y1 := y0 + float64(barHeight)
		var x0, x1 float64
		if scrollThumbStrokeWidth(context) > float64(bounds.Dx())*0.3 {
			x0 = float64(bounds.Max.X) - float64(bounds.Dx())*0.3
			x1 = float64(bounds.Max.X)
		} else {
			x0 = float64(bounds.Max.X) - padding - scrollThumbStrokeWidth(context)
			x1 = float64(bounds.Max.X) - padding
		}
		verticalBarBounds = image.Rect(int(x0), int(y0), int(x1), int(y1))
	}
	return horizontalBarBounds, verticalBarBounds
}

type panelBorder struct {
	guigui.DefaultWidget

	panel      *panel
	borders    PanelBorders
	autoBorder bool
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
	if p.panel == nil && p.borders == (PanelBorders{}) {
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
	if p.panel != nil {
		offsetX, offsetY = p.panel.offset()
		r = p.panel.scrollRange(context, widgetBounds)
	}
	clr := draw.Color(context.ColorMode(), draw.ColorTypeBase, 0.8)
	if (p.panel != nil && p.autoBorder && offsetX < float64(r.Max.X)) || p.borders.Start {
		vector.StrokeLine(dst, x0+strokeWidth/2, y0, x0+strokeWidth/2, y1, strokeWidth, clr, false)
	}
	if (p.panel != nil && p.autoBorder && offsetY < float64(r.Max.Y)) || p.borders.Top {
		vector.StrokeLine(dst, x0, y0+strokeWidth/2, x1, y0+strokeWidth/2, strokeWidth, clr, false)
	}
	if (p.panel != nil && p.autoBorder && offsetX > float64(r.Min.X)) || p.borders.End {
		vector.StrokeLine(dst, x1-strokeWidth/2, y0, x1-strokeWidth/2, y1, strokeWidth, clr, false)
	}
	if (p.panel != nil && p.autoBorder && offsetY > float64(r.Min.Y)) || p.borders.Bottom {
		vector.StrokeLine(dst, x0, y1-strokeWidth/2, x1, y1-strokeWidth/2, strokeWidth, clr, false)
	}
}
