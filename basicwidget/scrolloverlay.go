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

type scrollOverlayEventArgsScroll struct {
	OffsetX float64
	OffsetY float64
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

func scrollBarOpacity(count int) float64 {
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

// scrollOverlay is a widget that shows scroll bars overlayed on its content.
//
// scrollOverlay's bounds must be the same as its parent widget's bounds.
// Some methods of scrollOverlay takes widgetBounds parameter.
// You can pass the parent widget's widgetBounds to those methods.
type scrollOverlay struct {
	guigui.DefaultWidget

	contentSize image.Point
	offsetX     float64
	offsetY     float64

	lastSize              image.Point
	lastWheelX            float64
	lastWheelY            float64
	lastOffsetX           float64
	lastOffsetY           float64
	draggingX             bool
	draggingY             bool
	draggingStartPosition image.Point
	draggingStartOffsetX  float64
	draggingStartOffsetY  float64
	onceDraw              bool

	barCount int
}

func (s *scrollOverlay) Reset() {
	s.offsetX = 0
	s.offsetY = 0
}

// SetContentSize sets the size of the content inside the scrollOverlay.
//
// widgetBounds can be the parent widget's widgetBounds, assuming that scrollOverlay's bounds is the same as its parent widget's bounds.
func (s *scrollOverlay) SetContentSize(context *guigui.Context, widgetBounds *guigui.WidgetBounds, contentSize image.Point) {
	if s.contentSize == contentSize {
		return
	}

	s.contentSize = contentSize
	s.adjustOffset(context, widgetBounds)
	guigui.RequestRedraw(s)
}

// SetOffsetByDelta sets the offset by adding dx and dy to the current offset.
//
// widgetBounds can be the parent widget's widgetBounds, assuming that scrollOverlay's bounds is the same as its parent widget's bounds.
func (s *scrollOverlay) SetOffsetByDelta(context *guigui.Context, widgetBounds *guigui.WidgetBounds, contentSize image.Point, dx, dy float64) {
	s.SetOffset(context, widgetBounds, contentSize, s.offsetX+dx, s.offsetY+dy)
}

// SetOffset sets the offset to (x, y).
//
// widgetBounds can be the parent widget's widgetBounds, assuming that scrollOverlay's bounds is the same as its parent widget's bounds.
func (s *scrollOverlay) SetOffset(context *guigui.Context, widgetBounds *guigui.WidgetBounds, contentSize image.Point, x, y float64) {
	s.SetContentSize(context, widgetBounds, contentSize)

	x, y = s.doAdjustOffset(context, widgetBounds, x, y)
	if s.offsetX == x && s.offsetY == y {
		return
	}
	s.offsetX = x
	s.offsetY = y
	if s.onceDraw {
		s.showBars(context, widgetBounds)
	}
	guigui.RequestRedraw(s)
}

func (s *scrollOverlay) setDragging(draggingX, draggingY bool) {
	if s.draggingX == draggingX && s.draggingY == draggingY {
		return
	}

	s.draggingX = draggingX
	s.draggingY = draggingY
}

func adjustedWheel() (float64, float64) {
	x, y := ebiten.Wheel()
	switch runtime.GOOS {
	case "darwin":
		x *= 2
		y *= 2
	}
	return x, y
}

func (s *scrollOverlay) isWidgetHitAtCursor(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	if !widgetBounds.IsHitAtCursor() {
		return false
	}
	if !s.isBarVisible(context, widgetBounds) {
		return true
	}
	return !s.isCursorInEdgeArea(context, widgetBounds)
}

// handlePointingInput process pointing input for scrollOverlay as Widget.HandlePointingInput does.
//
// Guigui's input handling system does not invoke this method automatically.
// Instead, handlePointingInput must be invoked from the parent widget's HandlePointingInput method.
// This is becausethe scroll of the widget that is closest to the leaf in the tree should be handled first.
// If scrollOverlay is treated as an independent widget, the input handling order would become counterintuitive.
func (s *scrollOverlay) handlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	hovered := widgetBounds.IsHitAtCursor()
	if hovered {
		dx, dy := adjustedWheel()
		s.lastWheelX = dx
		s.lastWheelY = dy
	} else {
		s.lastWheelX = 0
		s.lastWheelY = 0
	}

	if !s.draggingX && !s.draggingY && hovered && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		hb, vb := s.barBounds(context, widgetBounds)
		if image.Pt(x, y).In(hb) {
			s.setDragging(true, s.draggingY)
			s.draggingStartPosition.X = x
			s.draggingStartOffsetX = s.offsetX
		} else if image.Pt(x, y).In(vb) {
			s.setDragging(s.draggingX, true)
			s.draggingStartPosition.Y = y
			s.draggingStartOffsetY = s.offsetY
		}
		if s.draggingX || s.draggingY {
			return guigui.HandleInputByWidget(s)
		}
	}

	if dx, dy := adjustedWheel(); dx != 0 || dy != 0 {
		s.setDragging(false, false)
	}

	if (s.draggingX || s.draggingY) && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		var dx, dy float64
		if s.draggingX {
			dx = float64(x - s.draggingStartPosition.X)
		}
		if s.draggingY {
			dy = float64(y - s.draggingStartPosition.Y)
		}
		if dx != 0 || dy != 0 {
			prevOffsetX := s.offsetX
			prevOffsetY := s.offsetY

			cs := widgetBounds.Bounds().Size()
			barWidth, barHeight := s.barSize(context, widgetBounds)
			if s.draggingX && barWidth > 0 && s.contentSize.X-cs.X > 0 {
				offsetPerPixel := float64(s.contentSize.X-cs.X) / (float64(cs.X) - barWidth)
				s.offsetX = s.draggingStartOffsetX + float64(-dx)*offsetPerPixel
			}
			if s.draggingY && barHeight > 0 && s.contentSize.Y-cs.Y > 0 {
				offsetPerPixel := float64(s.contentSize.Y-cs.Y) / (float64(cs.Y) - barHeight)
				s.offsetY = s.draggingStartOffsetY + float64(-dy)*offsetPerPixel
			}
			s.adjustOffset(context, widgetBounds)
			if prevOffsetX != s.offsetX || prevOffsetY != s.offsetY {
				guigui.DispatchEvent(s, &scrollOverlayEventArgsScroll{
					OffsetX: s.offsetX,
					OffsetY: s.offsetY,
				})
				guigui.RequestRedraw(s)
			}
		}
		return guigui.HandleInputByWidget(s)
	}

	if (s.draggingX || s.draggingY) && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		s.setDragging(false, false)
	}

	if dx, dy := adjustedWheel(); dx != 0 || dy != 0 {
		if !hovered {
			return guigui.HandleInputResult{}
		}
		s.setDragging(false, false)

		prevOffsetX := s.offsetX
		prevOffsetY := s.offsetY
		s.offsetX += dx * 4 * context.Scale()
		s.offsetY += dy * 4 * context.Scale()
		s.adjustOffset(context, widgetBounds)
		if prevOffsetX != s.offsetX || prevOffsetY != s.offsetY {
			guigui.DispatchEvent(s, &scrollOverlayEventArgsScroll{
				OffsetX: s.offsetX,
				OffsetY: s.offsetY,
			})
			guigui.RequestRedraw(s)
			return guigui.HandleInputByWidget(s)
		}
		return guigui.HandleInputResult{}
	}

	return guigui.HandleInputResult{}
}

func (s *scrollOverlay) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	x, y := ebiten.CursorPosition()
	hb, vb := s.barBounds(context, widgetBounds)
	if image.Pt(x, y).In(hb) || image.Pt(x, y).In(vb) {
		return ebiten.CursorShapeDefault, true
	}
	return 0, false
}

func (s *scrollOverlay) Offset() (float64, float64) {
	return s.offsetX, s.offsetY
}

func (s *scrollOverlay) adjustOffset(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
	s.offsetX, s.offsetY = s.doAdjustOffset(context, widgetBounds, s.offsetX, s.offsetY)
}

func (s *scrollOverlay) doAdjustOffset(context *guigui.Context, widgetBounds *guigui.WidgetBounds, x, y float64) (float64, float64) {
	r := s.scrollRange(context, widgetBounds)
	x = min(max(x, float64(r.Min.X)), float64(r.Max.X))
	y = min(max(y, float64(r.Min.Y)), float64(r.Max.Y))
	return x, y
}

func (s *scrollOverlay) scrollRange(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	bounds := widgetBounds.Bounds()
	return image.Rectangle{
		Min: image.Pt(min(bounds.Dx()-s.contentSize.X, 0), min(bounds.Dy()-s.contentSize.Y, 0)),
		Max: image.Pt(0, 0),
	}
}

func (s *scrollOverlay) hasBars(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	hb, vb := s.barBounds(context, widgetBounds)
	return !hb.Empty() || !vb.Empty()
}

func (s *scrollOverlay) isBarVisible(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	if !s.hasBars(context, widgetBounds) {
		return false
	}

	if s.draggingX || s.draggingY {
		return true
	}
	if s.lastWheelX != 0 || s.lastWheelY != 0 {
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

func (s *scrollOverlay) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	cs := widgetBounds.Bounds().Size()
	if s.lastSize != cs {
		s.adjustOffset(context, widgetBounds)
		s.lastSize = cs
	}
}

func (s *scrollOverlay) showBars(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
	if !s.hasBars(context, widgetBounds) {
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
	shouldShowBar := s.isBarVisible(context, widgetBounds)

	if s.lastOffsetX != s.offsetX || s.lastOffsetY != s.offsetY {
		shouldShowBar = true
	}
	s.lastOffsetX = s.offsetX
	s.lastOffsetY = s.offsetY

	oldOpacity := scrollBarOpacity(s.barCount)
	if shouldShowBar {
		s.showBars(context, widgetBounds)
	}
	newOpacity := scrollBarOpacity(s.barCount)

	if newOpacity != oldOpacity {
		guigui.RequestRedraw(s)
	}

	if s.barCount == 0 {
		return nil
	}

	if shouldShowBar && s.barCount == scrollBarMaxCount()-scrollBarFadingInTime() {
		// Keep showing the bar.
		return nil
	}

	oldOpacity = scrollBarOpacity(s.barCount)
	s.barCount--
	newOpacity = scrollBarOpacity(s.barCount)
	if newOpacity != oldOpacity {
		guigui.RequestRedraw(s)
	}

	return nil
}

func (s *scrollOverlay) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	defer func() {
		s.onceDraw = true
	}()

	alpha := scrollBarOpacity(s.barCount) * 3 / 4
	if alpha == 0 {
		return
	}

	barColor := draw.Color(context.ColorMode(), draw.ColorTypeBase, 0.2)
	barColor = draw.ScaleAlpha(barColor, alpha)

	hb, vb := s.barBounds(context, widgetBounds)

	// Show a horizontal bar.
	if !hb.Empty() {
		basicwidgetdraw.DrawRoundedRect(context, dst, hb, barColor, RoundedCornerRadius(context))
	}

	// Show a vertical bar.
	if !vb.Empty() {
		basicwidgetdraw.DrawRoundedRect(context, dst, vb, barColor, RoundedCornerRadius(context))
	}
}

func scrollOverlayBarStrokeWidth(context *guigui.Context) float64 {
	return 8 * context.Scale()
}

func scrollOverlayPadding(context *guigui.Context) float64 {
	return 2 * context.Scale()
}

func (s *scrollOverlay) barSize(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (float64, float64) {
	bounds := widgetBounds.Bounds()
	padding := scrollOverlayPadding(context)

	var w, h float64
	if s.contentSize.X > bounds.Dx() {
		w = (float64(bounds.Dx()) - 2*padding) * float64(bounds.Dx()) / float64(s.contentSize.X)
		w = max(w, scrollOverlayBarStrokeWidth(context))
	}
	if s.contentSize.Y > bounds.Dy() {
		h = (float64(bounds.Dy()) - 2*padding) * float64(bounds.Dy()) / float64(s.contentSize.Y)
		h = max(h, scrollOverlayBarStrokeWidth(context))
	}
	return w, h
}

func (s *scrollOverlay) barBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (image.Rectangle, image.Rectangle) {
	bounds := widgetBounds.Bounds()

	offsetX, offsetY := s.Offset()
	barWidth, barHeight := s.barSize(context, widgetBounds)

	padding := scrollOverlayPadding(context)

	var horizontalBarBounds, verticalBarBounds image.Rectangle
	if s.contentSize.X > bounds.Dx() {
		rate := -offsetX / float64(s.contentSize.X-bounds.Dx())
		x0 := float64(bounds.Min.X) + padding + rate*(float64(bounds.Dx())-2*padding-barWidth)
		x1 := x0 + float64(barWidth)
		var y0, y1 float64
		if scrollOverlayBarStrokeWidth(context) > float64(bounds.Dy())*0.3 {
			y0 = float64(bounds.Max.Y) - float64(bounds.Dy())*0.3
			y1 = float64(bounds.Max.Y)
		} else {
			y0 = float64(bounds.Max.Y) - padding - scrollOverlayBarStrokeWidth(context)
			y1 = float64(bounds.Max.Y) - padding
		}
		horizontalBarBounds = image.Rect(int(x0), int(y0), int(x1), int(y1))
	}
	if s.contentSize.Y > bounds.Dy() {
		rate := -offsetY / float64(s.contentSize.Y-bounds.Dy())
		y0 := float64(bounds.Min.Y) + padding + rate*(float64(bounds.Dy())-2*padding-barHeight)
		y1 := y0 + float64(barHeight)
		var x0, x1 float64
		if scrollOverlayBarStrokeWidth(context) > float64(bounds.Dx())*0.3 {
			x0 = float64(bounds.Max.X) - float64(bounds.Dx())*0.3
			x1 = float64(bounds.Max.X)
		} else {
			x0 = float64(bounds.Max.X) - padding - scrollOverlayBarStrokeWidth(context)
			x1 = float64(bounds.Max.X) - padding
		}
		verticalBarBounds = image.Rect(int(x0), int(y0), int(x1), int(y1))
	}
	return horizontalBarBounds, verticalBarBounds
}
