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

func scrollBarStrokeWidth(context *guigui.Context) float64 {
	return 8 * context.Scale()
}

func scrollBarPadding(context *guigui.Context) float64 {
	return 2 * context.Scale()
}

// scrollOverlay is a widget that shows scroll bars overlayed on its content.
//
// scrollOverlay's bounds must be the same as its parent widget's bounds.
// Some methods of scrollOverlay takes widgetBounds parameter.
// You can pass the parent widget's widgetBounds to those methods.
type scrollOverlay struct {
	guigui.DefaultWidget

	hBar scrollOverlayBar
	vBar scrollOverlayBar

	contentSize image.Point
	offsetX     float64
	offsetY     float64

	lastSize              image.Point
	lastWheelX            float64
	lastWheelY            float64
	draggingX             bool
	draggingY             bool
	draggingStartPosition image.Point
	draggingStartOffsetX  float64
	draggingStartOffsetY  float64
	onceDraw              bool

	barCount int
}

func (s *scrollOverlay) Reset() {
	if s.offsetX == 0 && s.offsetY == 0 {
		return
	}
	s.offsetX = 0
	s.offsetY = 0
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
	offsetX, offsetY := s.adjustOffset(widgetBounds, s.offsetX, s.offsetY)
	s.SetOffset(context, widgetBounds, s.contentSize, offsetX, offsetY)
	guigui.RequestRebuild(s)
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

	x, y = s.adjustOffset(widgetBounds, x, y)
	if s.offsetX == x && s.offsetY == y {
		return
	}
	s.offsetX = x
	s.offsetY = y
	if s.onceDraw {
		s.startShowingBarsIfNeeded(context, widgetBounds)
	}
	guigui.RequestRebuild(s)
}

func (s *scrollOverlay) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&s.hBar)
	adder.AddChild(&s.vBar)

	// TODO: After moving HandlePointingInput to scrollOverlayBar, enable Z delta setting.
	// context.SetZDelta(&s.hBar, 1)
	// context.SetZDelta(&s.vBar, 1)

	return nil
}

func (s *scrollOverlay) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	cs := widgetBounds.Bounds().Size()
	if s.lastSize != cs {
		offsetX, offsetY := s.adjustOffset(widgetBounds, s.offsetX, s.offsetY)
		s.SetOffset(context, widgetBounds, s.contentSize, offsetX, offsetY)
		s.lastSize = cs
	}

	hb, vb := s.barBounds(context, widgetBounds)
	if !hb.Empty() {
		bounds := widgetBounds.Bounds()
		bounds.Min.Y = hb.Min.Y
		bounds.Max.Y = hb.Max.Y
		layouter.LayoutWidget(&s.hBar, bounds)
		s.hBar.setThumbBounds(hb)
	}
	if !vb.Empty() {
		bounds := widgetBounds.Bounds()
		bounds.Min.X = vb.Min.X
		bounds.Max.X = vb.Max.X
		layouter.LayoutWidget(&s.vBar, bounds)
		s.vBar.setThumbBounds(vb)
	}
}

func (s *scrollOverlay) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	hovered := widgetBounds.IsHitAtCursor()

	wheelX, wheelY := adjustedWheel()
	if hovered {
		s.lastWheelX = wheelX
		s.lastWheelY = wheelY
	} else {
		s.lastWheelX = 0
		s.lastWheelY = 0
	}

	if !s.draggingX && !s.draggingY && hovered && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		hb, vb := s.barBounds(context, widgetBounds)
		if !hb.Empty() && y >= hb.Min.Y {
			s.draggingX = true
			s.draggingStartPosition.X = x
			s.draggingStartOffsetX = s.offsetX
		} else if !vb.Empty() && x >= vb.Min.X {
			s.draggingY = true
			s.draggingStartPosition.Y = y
			s.draggingStartOffsetY = s.offsetY
		}
		if s.draggingX || s.draggingY {
			return guigui.HandleInputByWidget(s)
		}
	}

	if wheelX != 0 || wheelY != 0 {
		s.draggingX = false
		s.draggingY = false
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
			offsetX := s.offsetX
			offsetY := s.offsetY

			cs := widgetBounds.Bounds().Size()
			barWidth, barHeight := s.barSize(context, widgetBounds)
			if s.draggingX && barWidth > 0 && s.contentSize.X-cs.X > 0 {
				offsetPerPixel := float64(s.contentSize.X-cs.X) / (float64(cs.X) - barWidth)
				offsetX = s.draggingStartOffsetX + float64(-dx)*offsetPerPixel
			}
			if s.draggingY && barHeight > 0 && s.contentSize.Y-cs.Y > 0 {
				offsetPerPixel := float64(s.contentSize.Y-cs.Y) / (float64(cs.Y) - barHeight)
				offsetY = s.draggingStartOffsetY + float64(-dy)*offsetPerPixel
			}
			s.SetOffset(context, widgetBounds, s.contentSize, offsetX, offsetY)
		}
		return guigui.HandleInputByWidget(s)
	}

	if (s.draggingX || s.draggingY) && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		s.draggingX = false
		s.draggingY = false
	}

	if wheelX != 0 || wheelY != 0 {
		if !hovered {
			return guigui.HandleInputResult{}
		}

		// TODO: If there is an inner scrollOverlay, wheels should not be handled here (#204).
		offsetX := s.offsetX
		offsetY := s.offsetY
		offsetX += wheelX * 4 * context.Scale()
		offsetY += wheelY * 4 * context.Scale()
		s.SetOffset(context, widgetBounds, s.contentSize, offsetX, offsetY)
		return guigui.HandleInputResult{}
	}

	return guigui.HandleInputResult{}
}

func (s *scrollOverlay) Offset() (float64, float64) {
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
	if hb, vb := s.barBounds(context, widgetBounds); hb.Empty() && vb.Empty() {
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

func (s *scrollOverlay) startShowingBarsIfNeeded(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
	if hb, vb := s.barBounds(context, widgetBounds); hb.Empty() && vb.Empty() {
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

	oldOpacity := scrollBarOpacity(s.barCount)
	if shouldShowBar {
		s.startShowingBarsIfNeeded(context, widgetBounds)
	}
	newOpacity := scrollBarOpacity(s.barCount)

	if newOpacity != oldOpacity {
		guigui.RequestRedraw(s)
	}

	if s.barCount > 0 {
		if !shouldShowBar || s.barCount != scrollBarMaxCount()-scrollBarFadingInTime() {
			oldOpacity = scrollBarOpacity(s.barCount)
			s.barCount--
			newOpacity = scrollBarOpacity(s.barCount)
			if newOpacity != oldOpacity {
				guigui.RequestRedraw(s)
			}
		}
	}

	alpha := scrollBarOpacity(s.barCount) * 3 / 4
	s.hBar.setAlpha(alpha)
	s.vBar.setAlpha(alpha)

	return nil
}

func (s *scrollOverlay) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	s.onceDraw = true
}

func (s *scrollOverlay) barSize(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (float64, float64) {
	bounds := widgetBounds.Bounds()
	padding := scrollBarPadding(context)

	var w, h float64
	if s.contentSize.X > bounds.Dx() {
		w = (float64(bounds.Dx()) - 2*padding) * float64(bounds.Dx()) / float64(s.contentSize.X)
		w = max(w, scrollBarStrokeWidth(context))
	}
	if s.contentSize.Y > bounds.Dy() {
		h = (float64(bounds.Dy()) - 2*padding) * float64(bounds.Dy()) / float64(s.contentSize.Y)
		h = max(h, scrollBarStrokeWidth(context))
	}
	return w, h
}

func (s *scrollOverlay) barBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (image.Rectangle, image.Rectangle) {
	bounds := widgetBounds.Bounds()

	offsetX, offsetY := s.Offset()
	barWidth, barHeight := s.barSize(context, widgetBounds)

	padding := scrollBarPadding(context)

	var horizontalBarBounds, verticalBarBounds image.Rectangle
	if s.contentSize.X > bounds.Dx() {
		rate := -offsetX / float64(s.contentSize.X-bounds.Dx())
		x0 := float64(bounds.Min.X) + padding + rate*(float64(bounds.Dx())-2*padding-barWidth)
		x1 := x0 + float64(barWidth)
		var y0, y1 float64
		if scrollBarStrokeWidth(context) > float64(bounds.Dy())*0.3 {
			y0 = float64(bounds.Max.Y) - float64(bounds.Dy())*0.3
			y1 = float64(bounds.Max.Y)
		} else {
			y0 = float64(bounds.Max.Y) - padding - scrollBarStrokeWidth(context)
			y1 = float64(bounds.Max.Y) - padding
		}
		horizontalBarBounds = image.Rect(int(x0), int(y0), int(x1), int(y1))
	}
	if s.contentSize.Y > bounds.Dy() {
		rate := -offsetY / float64(s.contentSize.Y-bounds.Dy())
		y0 := float64(bounds.Min.Y) + padding + rate*(float64(bounds.Dy())-2*padding-barHeight)
		y1 := y0 + float64(barHeight)
		var x0, x1 float64
		if scrollBarStrokeWidth(context) > float64(bounds.Dx())*0.3 {
			x0 = float64(bounds.Max.X) - float64(bounds.Dx())*0.3
			x1 = float64(bounds.Max.X)
		} else {
			x0 = float64(bounds.Max.X) - padding - scrollBarStrokeWidth(context)
			x1 = float64(bounds.Max.X) - padding
		}
		verticalBarBounds = image.Rect(int(x0), int(y0), int(x1), int(y1))
	}
	return horizontalBarBounds, verticalBarBounds
}

type scrollOverlayBar struct {
	guigui.DefaultWidget

	thumbBounds image.Rectangle
	alpha       float64
}

func (s *scrollOverlayBar) setThumbBounds(bounds image.Rectangle) {
	if s.thumbBounds == bounds {
		return
	}
	s.thumbBounds = bounds
	guigui.RequestRedraw(s)
}

func (s *scrollOverlayBar) setAlpha(alpha float64) {
	if s.alpha == alpha {
		return
	}
	s.alpha = alpha
	guigui.RequestRedraw(s)
}

func (s *scrollOverlayBar) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	return ebiten.CursorShapeDefault, true
}

func (s *scrollOverlayBar) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
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
