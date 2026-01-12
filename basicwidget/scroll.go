// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"image"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

func adjustedWheel() (float64, float64) {
	x, y := ebiten.Wheel()
	switch runtime.GOOS {
	case "darwin":
		x *= 2
		y *= 2
	}
	return x, y
}

func scrollBarFadingInTime() int {
	return ebiten.TPS() / 15
}

func scrollBarFadingOutTime() int {
	return ebiten.TPS() / 5
}

func scrollBarShowingTime() int {
	return ebiten.TPS() / 2
}

func scrollBarMaxCount() int {
	return scrollBarFadingInTime() + scrollBarShowingTime() + scrollBarFadingOutTime()
}

func scrollThumbOpacity(count int) float64 {
	switch {
	case scrollBarMaxCount()-scrollBarFadingInTime() <= count:
		c := count - (scrollBarMaxCount() - scrollBarFadingInTime())
		return 1 - float64(c)/float64(scrollBarFadingInTime())
	case scrollBarFadingOutTime() <= count:
		return 1
	default:
		return float64(count) / float64(scrollBarFadingOutTime())
	}
}

func scrollThumbStrokeWidth(context *guigui.Context) float64 {
	return 8 * context.Scale()
}

func scrollThumbPadding(context *guigui.Context) float64 {
	return 2 * context.Scale()
}

func scrollThumbSize(context *guigui.Context, widgetBounds *guigui.WidgetBounds, contentSize image.Point) (float64, float64) {
	bounds := widgetBounds.Bounds()
	padding := scrollThumbPadding(context)

	var w, h float64
	if contentSize.X > bounds.Dx() {
		w = (float64(bounds.Dx()) - 2*padding) * float64(bounds.Dx()) / float64(contentSize.X)
		w = max(w, scrollThumbStrokeWidth(context))
	}
	if contentSize.Y > bounds.Dy() {
		h = (float64(bounds.Dy()) - 2*padding) * float64(bounds.Dy()) / float64(contentSize.Y)
		h = max(h, scrollThumbStrokeWidth(context))
	}
	return w, h
}

type panelScroll struct {
	guigui.DefaultWidget

	scrollWheel scrollWheel
	scrollHBar  scrollBar
	scrollVBar  scrollBar

	contentSize       image.Point
	offsetX           float64
	offsetY           float64
	nextOffsetSet     bool
	isNextOffsetDelta bool
	nextOffsetX       float64
	nextOffsetY       float64
	scrollBarCount    int
}

// setContentSize sets the size of the content inside the scrollOverlay.
//
// widgetBounds can be the parent widget's widgetBounds, assuming that scrollOverlay's bounds is the same as its parent widget's bounds.
func (p *panelScroll) setContentSize(widgetBounds *guigui.WidgetBounds, contentSize image.Point) {
	if p.contentSize == contentSize {
		return
	}

	p.contentSize = contentSize
	p.scrollWheel.setContentSize(contentSize)
	p.scrollHBar.setContentSize(contentSize)
	p.scrollVBar.setContentSize(contentSize)
	p.setOffset(p.adjustOffset(widgetBounds, p.offsetX, p.offsetY))
	guigui.RequestRebuild(p)
}

// setOffsetByDelta sets the offset by adding dx and dy to the current offset.
func (p *panelScroll) setOffsetByDelta(dx, dy float64) {
	p.nextOffsetSet = true
	p.isNextOffsetDelta = true
	p.nextOffsetX = dx
	p.nextOffsetY = dy
}

// setOffset sets the offset to (x, y).
func (p *panelScroll) setOffset(x, y float64) {
	if p.offsetX == x && p.offsetY == y {
		return
	}
	p.nextOffsetSet = true
	p.isNextOffsetDelta = false
	p.nextOffsetX = x
	p.nextOffsetY = y
}

func (p *panelScroll) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&p.scrollWheel)
	adder.AddChild(&p.scrollHBar)
	adder.AddChild(&p.scrollVBar)

	p.scrollWheel.setOffsetGetSetter(p)
	p.scrollHBar.setOffsetGetSetter(p)
	p.scrollHBar.setHorizontal(true)
	p.scrollVBar.setOffsetGetSetter(p)
	p.scrollVBar.setHorizontal(false)

	context.SetFloat(&p.scrollHBar, true)
	context.SetFloat(&p.scrollVBar, true)

	return nil
}

func (p *panelScroll) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&p.scrollWheel, widgetBounds.Bounds())
	layouter.LayoutWidget(&p.scrollHBar, p.horizontalBarBounds(context, widgetBounds))
	layouter.LayoutWidget(&p.scrollVBar, p.verticalBarBounds(context, widgetBounds))

	p.setOffset(p.adjustOffset(widgetBounds, p.offsetX, p.offsetY))

	hb, vb := p.thumbBounds(context, widgetBounds)
	p.scrollHBar.setThumbBounds(hb)
	p.scrollVBar.setThumbBounds(vb)
}

func (p *panelScroll) horizontalBarBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	bounds := widgetBounds.Bounds()
	bounds.Min.Y = max(bounds.Min.Y, bounds.Max.Y-UnitSize(context)/2)
	return bounds
}

func (p *panelScroll) verticalBarBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	bounds := widgetBounds.Bounds()
	bounds.Min.X = max(bounds.Min.X, bounds.Max.X-UnitSize(context)/2)
	return bounds
}

func (p *panelScroll) offset() (float64, float64) {
	// As the next offset might not be a valid offset, return the current offset.
	return p.offsetX, p.offsetY
}

func (p *panelScroll) adjustOffset(widgetBounds *guigui.WidgetBounds, x, y float64) (float64, float64) {
	r := p.scrollRange(widgetBounds)
	x = min(max(x, float64(r.Min.X)), float64(r.Max.X))
	y = min(max(y, float64(r.Min.Y)), float64(r.Max.Y))
	return x, y
}

func (p *panelScroll) scrollRange(widgetBounds *guigui.WidgetBounds) image.Rectangle {
	bounds := widgetBounds.Bounds()
	return image.Rectangle{
		Min: image.Pt(min(bounds.Dx()-p.contentSize.X, 0), min(bounds.Dy()-p.contentSize.Y, 0)),
		Max: image.Pt(0, 0),
	}
}

func (p *panelScroll) isBarVisible(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
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

func (p *panelScroll) startShowingBarsIfNeeded(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
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

func (p *panelScroll) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
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
		newOffsetX, newOffsetY = p.adjustOffset(widgetBounds, newOffsetX, newOffsetY)
		if p.offsetX != newOffsetX || p.offsetY != newOffsetY {
			p.offsetX = newOffsetX
			p.offsetY = newOffsetY
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

func (p *panelScroll) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	// This is a dummy Draw implementation.
	// Without this, scrollOverly would always fail hit testing.
}

func (p *panelScroll) thumbBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (image.Rectangle, image.Rectangle) {
	bounds := widgetBounds.Bounds()

	offsetX, offsetY := p.offset()
	barWidth, barHeight := scrollThumbSize(context, widgetBounds, p.contentSize)

	padding := scrollThumbPadding(context)

	var horizontalBarBounds, verticalBarBounds image.Rectangle
	if p.contentSize.X > bounds.Dx() {
		rate := -offsetX / float64(p.contentSize.X-bounds.Dx())
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
	if p.contentSize.Y > bounds.Dy() {
		rate := -offsetY / float64(p.contentSize.Y-bounds.Dy())
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

type offsetGetSetter interface {
	offset() (float64, float64)
	setOffset(x, y float64)
}

type scrollWheel struct {
	guigui.DefaultWidget

	offsetGetSetter offsetGetSetter
	contentSize     image.Point
	lastWheelX      float64
	lastWheelY      float64
}

func (s *scrollWheel) setOffsetGetSetter(offsetGetSetter offsetGetSetter) {
	if s.offsetGetSetter == offsetGetSetter {
		return
	}
	s.offsetGetSetter = offsetGetSetter
	guigui.RequestRebuild(s)
}

func (s *scrollWheel) setContentSize(size image.Point) {
	if s.contentSize == size {
		return
	}
	s.contentSize = size
	guigui.RequestRebuild(s)
}

func (s *scrollWheel) isScrolling() bool {
	return s.lastWheelX != 0 || s.lastWheelY != 0
}

func (s *scrollWheel) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if s.offsetGetSetter == nil {
		return guigui.HandleInputResult{}
	}

	if !widgetBounds.IsHitAtCursor() {
		s.lastWheelX = 0
		s.lastWheelY = 0
		return guigui.HandleInputResult{}
	}

	wheelX, wheelY := adjustedWheel()
	s.lastWheelX = wheelX
	s.lastWheelY = wheelY

	if wheelX != 0 || wheelY != 0 {
		offsetX, offsetY := s.offsetGetSetter.offset()
		offsetX += wheelX * 4 * context.Scale()
		offsetY += wheelY * 4 * context.Scale()
		s.offsetGetSetter.setOffset(offsetX, offsetY)
		return guigui.HandleInputResult{}
	}

	return guigui.HandleInputResult{}
}

type scrollBar struct {
	guigui.DefaultWidget

	offsetGetSetter offsetGetSetter
	horizontal      bool
	thumbBounds     image.Rectangle
	contentSize     image.Point
	alpha           float64

	dragging              bool
	draggingStartPosition int
	draggingStartOffset   float64
	onceDraw              bool
}

func (s *scrollBar) setOffsetGetSetter(offsetGetSetter offsetGetSetter) {
	if s.offsetGetSetter == offsetGetSetter {
		return
	}
	s.offsetGetSetter = offsetGetSetter
	guigui.RequestRebuild(s)
}

func (s *scrollBar) setHorizontal(horizontal bool) {
	if s.horizontal == horizontal {
		return
	}
	s.horizontal = horizontal
	guigui.RequestRebuild(s)
}

func (s *scrollBar) setThumbBounds(bounds image.Rectangle) {
	if s.thumbBounds == bounds {
		return
	}
	s.thumbBounds = bounds
	guigui.RequestRedraw(s)
}

func (s *scrollBar) setContentSize(size image.Point) {
	if s.contentSize == size {
		return
	}
	s.contentSize = size
	guigui.RequestRebuild(s)
}

func (s *scrollBar) setAlpha(alpha float64) {
	if s.alpha == alpha {
		return
	}
	s.alpha = alpha
	if !s.thumbBounds.Empty() {
		guigui.RequestRedraw(s)
	}
}

func (s *scrollBar) isDragging() bool {
	return s.dragging
}

func (s *scrollBar) isOnceDrawn() bool {
	return s.onceDraw
}

func (s *scrollBar) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if s.offsetGetSetter == nil {
		return guigui.HandleInputResult{}
	}

	if !s.dragging && widgetBounds.IsHitAtCursor() && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if tb := s.thumbBounds; !tb.Empty() {
			x, y := ebiten.CursorPosition()
			offsetX, offsetY := s.offsetGetSetter.offset()
			if s.horizontal && y >= tb.Min.Y && x >= tb.Min.X && x < tb.Max.X {
				s.dragging = true
				s.draggingStartPosition = x
				s.draggingStartOffset = offsetX
			} else if !s.horizontal && x >= tb.Min.X && y >= tb.Min.Y && y < tb.Max.Y {
				s.dragging = true
				s.draggingStartPosition = y
				s.draggingStartOffset = offsetY
			}
		}
		if s.dragging {
			return guigui.HandleInputByWidget(s)
		}
	}

	if wheelX, wheelY := adjustedWheel(); wheelX != 0 || wheelY != 0 {
		s.dragging = false
	}

	if s.dragging && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		var dx, dy float64
		if s.dragging {
			x, y := ebiten.CursorPosition()
			if s.horizontal {
				dx = float64(x - s.draggingStartPosition)
			} else {
				dy = float64(y - s.draggingStartPosition)
			}
		}
		if dx != 0 || dy != 0 {
			offsetX, offsetY := s.offsetGetSetter.offset()

			cs := widgetBounds.Bounds().Size()
			barWidth, barHeight := scrollThumbSize(context, widgetBounds, s.contentSize)
			if s.horizontal && s.dragging && barWidth > 0 && s.contentSize.X-cs.X > 0 {
				offsetPerPixel := float64(s.contentSize.X-cs.X) / (float64(cs.X) - barWidth)
				offsetX = s.draggingStartOffset + float64(-dx)*offsetPerPixel
			}
			if !s.horizontal && s.dragging && barHeight > 0 && s.contentSize.Y-cs.Y > 0 {
				offsetPerPixel := float64(s.contentSize.Y-cs.Y) / (float64(cs.Y) - barHeight)
				offsetY = s.draggingStartOffset + float64(-dy)*offsetPerPixel
			}
			s.offsetGetSetter.setOffset(offsetX, offsetY)
		}
		return guigui.HandleInputByWidget(s)
	}

	if s.dragging && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		s.dragging = false
	}
	return guigui.HandleInputResult{}
}

func (s *scrollBar) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	return ebiten.CursorShapeDefault, true
}

func (s *scrollBar) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if s.thumbBounds.Empty() {
		return
	}
	if s.alpha == 0 {
		return
	}
	s.onceDraw = true
	barColor := draw.Color(context.ColorMode(), draw.ColorTypeBase, 0.2)
	barColor = draw.ScaleAlpha(barColor, s.alpha)
	basicwidgetdraw.DrawRoundedRect(context, dst, s.thumbBounds, barColor, RoundedCornerRadius(context))
}
