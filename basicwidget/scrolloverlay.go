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

// scrollOverlay is a widget that shows scroll bars overlayed on its content.
//
// scrollOverlay's bounds must be the same as its parent widget's bounds.
// Some methods of scrollOverlay takes widgetBounds parameter.
// You can pass the parent widget's widgetBounds to those methods.
type scrollOverlay struct {
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

	lastSize image.Point
	onceDraw bool

	barCount int
}

func (s *scrollOverlay) Reset() {
	if s.offsetX == 0 && s.offsetY == 0 {
		return
	}
	s.offsetX = 0
	s.offsetY = 0
	s.nextOffsetSet = false
	s.nextOffsetX = 0
	s.nextOffsetY = 0
	s.isNextOffsetDelta = false
	guigui.RequestRebuild(s)
}

// SetContentSize sets the size of the content inside the scrollOverlay.
//
// widgetBounds can be the parent widget's widgetBounds, assuming that scrollOverlay's bounds is the same as its parent widget's bounds.
func (s *scrollOverlay) SetContentSize(context *guigui.Context, widgetBounds *guigui.WidgetBounds, contentSize image.Point) {
	if s.contentSize == contentSize {
		return
	}

	s.contentSize = contentSize
	s.scrollWheel.setContentSize(contentSize)
	s.scrollHBar.setContentSize(contentSize)
	s.scrollVBar.setContentSize(contentSize)
	s.SetOffset(s.adjustOffset(widgetBounds, s.offsetX, s.offsetY))
	guigui.RequestRebuild(s)
}

// SetOffsetByDelta sets the offset by adding dx and dy to the current offset.
func (s *scrollOverlay) SetOffsetByDelta(dx, dy float64) {
	s.nextOffsetSet = true
	s.isNextOffsetDelta = true
	s.nextOffsetX = dx
	s.nextOffsetY = dy
}

// SetOffset sets the offset to (x, y).
func (s *scrollOverlay) SetOffset(x, y float64) {
	if s.offsetX == x && s.offsetY == y {
		return
	}
	s.nextOffsetSet = true
	s.isNextOffsetDelta = false
	s.nextOffsetX = x
	s.nextOffsetY = y
}

func (s *scrollOverlay) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&s.scrollWheel)
	adder.AddChild(&s.scrollHBar)
	adder.AddChild(&s.scrollVBar)

	s.scrollWheel.setOffsetGetSetter(s)
	s.scrollHBar.setOffsetGetSetter(s)
	s.scrollHBar.setHorizontal(true)
	s.scrollVBar.setOffsetGetSetter(s)
	s.scrollVBar.setHorizontal(false)

	context.SetFloat(&s.scrollHBar, true)
	context.SetFloat(&s.scrollVBar, true)

	return nil
}

func (s *scrollOverlay) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&s.scrollWheel, widgetBounds.Bounds())
	{
		bounds := widgetBounds.Bounds()
		bounds.Min.Y = max(bounds.Min.Y, bounds.Max.Y-UnitSize(context)/2)
		layouter.LayoutWidget(&s.scrollHBar, bounds)
	}
	{
		bounds := widgetBounds.Bounds()
		bounds.Min.X = max(bounds.Min.X, bounds.Max.X-UnitSize(context)/2)
		layouter.LayoutWidget(&s.scrollVBar, bounds)
	}

	if cs := widgetBounds.Bounds().Size(); s.lastSize != cs {
		s.SetOffset(s.adjustOffset(widgetBounds, s.offsetX, s.offsetY))
		s.lastSize = cs
	}

	hb, vb := s.thumbBounds(context, widgetBounds)
	s.scrollHBar.setThumbBounds(hb)
	s.scrollVBar.setThumbBounds(vb)
}

func (s *scrollOverlay) Offset() (float64, float64) {
	// As the next offset might not be a valid offset, return the current offset.
	return s.offsetX, s.offsetY
}

func (s *scrollOverlay) adjustOffset(widgetBounds *guigui.WidgetBounds, x, y float64) (float64, float64) {
	r := s.scrollRange(widgetBounds)
	x = min(max(x, float64(r.Min.X)), float64(r.Max.X))
	y = min(max(y, float64(r.Min.Y)), float64(r.Max.Y))
	return x, y
}

func (s *scrollOverlay) scrollRange(widgetBounds *guigui.WidgetBounds) image.Rectangle {
	bounds := widgetBounds.Bounds()
	return image.Rectangle{
		Min: image.Pt(min(bounds.Dx()-s.contentSize.X, 0), min(bounds.Dy()-s.contentSize.Y, 0)),
		Max: image.Pt(0, 0),
	}
}

func (s *scrollOverlay) isBarVisible(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	if hb, vb := s.thumbBounds(context, widgetBounds); hb.Empty() && vb.Empty() {
		return false
	}

	if s.scrollWheel.isScrolling() {
		return true
	}
	if s.scrollHBar.isDragging() || s.scrollVBar.isDragging() {
		return true
	}
	return s.isCursorInEdgeArea(context, widgetBounds)
}

func (s *scrollOverlay) isCursorInEdgeArea(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	if !widgetBounds.IsHitAtCursor() {
		return false
	}
	pt := image.Pt(ebiten.CursorPosition())
	bounds := widgetBounds.Bounds()
	if s.contentSize.X > bounds.Dx() && bounds.Max.Y-UnitSize(context)/2 <= pt.Y-1 {
		return true
	}
	if s.contentSize.Y > bounds.Dy() && bounds.Max.X-UnitSize(context)/2 <= pt.X-1 {
		return true
	}
	return false
}

func (s *scrollOverlay) startShowingBarsIfNeeded(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
	if hb, vb := s.thumbBounds(context, widgetBounds); hb.Empty() && vb.Empty() {
		return
	}

	switch {
	case s.barCount >= scrollBarMaxCount()-scrollBarFadingInTime():
		// If the scroll bar is being fading in, do nothing.
	case s.barCount >= scrollBarFadingOutTime():
		// If the scroll bar is shown, reset the count.
		s.barCount = scrollBarMaxCount() - scrollBarFadingInTime()
	case s.barCount > 0:
		// If the scroll bar is fading out, reset the count.
		s.barCount = scrollBarMaxCount() - scrollBarFadingInTime()
	default:
		s.barCount = scrollBarMaxCount()
	}
}

func (s *scrollOverlay) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if s.nextOffsetSet {
		var newOffsetX, newOffsetY float64
		if s.isNextOffsetDelta {
			newOffsetX = s.offsetX + s.nextOffsetX
			newOffsetY = s.offsetY + s.nextOffsetY
		} else {
			newOffsetX = s.nextOffsetX
			newOffsetY = s.nextOffsetY
		}
		newOffsetX, newOffsetY = s.adjustOffset(widgetBounds, newOffsetX, newOffsetY)
		if s.offsetX != newOffsetX || s.offsetY != newOffsetY {
			s.offsetX = newOffsetX
			s.offsetY = newOffsetY
			guigui.RequestRebuild(s)
		}
		s.nextOffsetSet = false
		s.nextOffsetX = 0
		s.nextOffsetY = 0
		s.isNextOffsetDelta = false
	}

	shouldShowBar := s.isBarVisible(context, widgetBounds)

	oldOpacity := scrollThumbOpacity(s.barCount)
	if shouldShowBar {
		s.startShowingBarsIfNeeded(context, widgetBounds)
	}
	newOpacity := scrollThumbOpacity(s.barCount)

	if newOpacity != oldOpacity {
		guigui.RequestRedraw(s)
	}

	if s.barCount > 0 {
		if !shouldShowBar || s.barCount != scrollBarMaxCount()-scrollBarFadingInTime() {
			oldOpacity = scrollThumbOpacity(s.barCount)
			s.barCount--
			newOpacity = scrollThumbOpacity(s.barCount)
			if newOpacity != oldOpacity {
				guigui.RequestRedraw(s)
			}
		}
	}

	alpha := scrollThumbOpacity(s.barCount) * 3 / 4
	s.scrollHBar.setAlpha(alpha)
	s.scrollVBar.setAlpha(alpha)

	return nil
}

func (s *scrollOverlay) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	s.onceDraw = true
}

func (s *scrollOverlay) thumbBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (image.Rectangle, image.Rectangle) {
	bounds := widgetBounds.Bounds()

	offsetX, offsetY := s.Offset()
	barWidth, barHeight := scrollThumbSize(context, widgetBounds, s.contentSize)

	padding := scrollThumbPadding(context)

	var horizontalBarBounds, verticalBarBounds image.Rectangle
	if s.contentSize.X > bounds.Dx() {
		rate := -offsetX / float64(s.contentSize.X-bounds.Dx())
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
	if s.contentSize.Y > bounds.Dy() {
		rate := -offsetY / float64(s.contentSize.Y-bounds.Dy())
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
	Offset() (float64, float64)
	SetOffset(x, y float64)
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
		offsetX, offsetY := s.offsetGetSetter.Offset()
		offsetX += wheelX * 4 * context.Scale()
		offsetY += wheelY * 4 * context.Scale()
		s.offsetGetSetter.SetOffset(offsetX, offsetY)
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
	guigui.RequestRedraw(s)
}

func (s *scrollBar) isDragging() bool {
	return s.dragging
}

func (s *scrollBar) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if s.offsetGetSetter == nil {
		return guigui.HandleInputResult{}
	}

	if !s.dragging && widgetBounds.IsHitAtCursor() && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if tb := s.thumbBounds; !tb.Empty() {
			x, y := ebiten.CursorPosition()
			offsetX, offsetY := s.offsetGetSetter.Offset()
			if s.horizontal && y >= tb.Min.Y {
				s.dragging = true
				s.draggingStartPosition = x
				s.draggingStartOffset = offsetX
			} else if !s.horizontal && x >= tb.Min.X {
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
			offsetX, offsetY := s.offsetGetSetter.Offset()

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
			s.offsetGetSetter.SetOffset(offsetX, offsetY)
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
	barColor := draw.Color(context.ColorMode(), draw.ColorTypeBase, 0.2)
	barColor = draw.ScaleAlpha(barColor, s.alpha)
	basicwidgetdraw.DrawRoundedRect(context, dst, s.thumbBounds, barColor, RoundedCornerRadius(context))
}
