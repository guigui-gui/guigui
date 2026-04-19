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
	case "windows":
		x *= 4
		y *= 4
	}
	return x, y
}

func scrollBarFadingInTime() int {
	return ebiten.TPS() / 20
}

func scrollBarFadingOutTime() int {
	return ebiten.TPS() / 10
}

func scrollBarShowingTime() int {
	return ebiten.TPS() / 2
}

func scrollBarMaxCount() int {
	return scrollBarFadingInTime() + scrollBarShowingTime() + scrollBarFadingOutTime()
}

func scrollThumbOpacity(count int) float64 {
	const maxOpacity = 0.75

	switch {
	case scrollBarMaxCount()-scrollBarFadingInTime() <= count:
		c := count - (scrollBarMaxCount() - scrollBarFadingInTime())
		return (1 - float64(c)/float64(scrollBarFadingInTime())) * maxOpacity
	case scrollBarFadingOutTime() <= count:
		return maxOpacity
	default:
		return float64(count) / float64(scrollBarFadingOutTime()) * maxOpacity
	}
}

func scrollWheelSpeed(context *guigui.Context) float64 {
	return 4 * context.Scale()
}

func scrollBarAreaSize(context *guigui.Context) int {
	return UnitSize(context) / 2
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

type scrollOffsetGetSetter interface {
	scrollOffset() (float64, float64)
	setScrollOffset(x, y float64)
}

type scrollWheel struct {
	guigui.DefaultWidget

	offsetGetSetter scrollOffsetGetSetter
	contentSize     image.Point
	lastWheelX      float64
	lastWheelY      float64
}

func (s *scrollWheel) WriteStateKey(w *guigui.StateKeyWriter) {
	writePoint(w, s.contentSize)
}

func (s *scrollWheel) setOffsetGetSetter(offsetGetSetter scrollOffsetGetSetter) {
	s.offsetGetSetter = offsetGetSetter
}

func (s *scrollWheel) setContentSize(size image.Point) {
	s.contentSize = size
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
		offsetX, offsetY := s.offsetGetSetter.scrollOffset()
		offsetX += wheelX * scrollWheelSpeed(context)
		offsetY += wheelY * scrollWheelSpeed(context)
		s.offsetGetSetter.setScrollOffset(offsetX, offsetY)
		// TODO: If the actual offset is not changed, this should not return HandleInputByWidget (#204).
		return guigui.HandleInputByWidget(s)
	}

	return guigui.HandleInputResult{}
}

type scrollBar struct {
	guigui.DefaultWidget

	offsetGetSetter scrollOffsetGetSetter
	horizontal      bool
	thumbBounds     image.Rectangle
	contentSize     image.Point
	alpha           float64

	dragging              bool
	draggingStartPosition int
	draggingStartOffset   float64
	onceDraw              bool
}

func (s *scrollBar) WriteStateKey(w *guigui.StateKeyWriter) {
	w.WriteBool(s.horizontal)
	writePoint(w, s.contentSize)
}

func (s *scrollBar) setOffsetGetSetter(offsetGetSetter scrollOffsetGetSetter) {
	s.offsetGetSetter = offsetGetSetter
}

func (s *scrollBar) setHorizontal(horizontal bool) {
	s.horizontal = horizontal
}

func (s *scrollBar) setThumbBounds(bounds image.Rectangle) {
	if s.thumbBounds == bounds {
		return
	}
	s.thumbBounds = bounds
	guigui.RequestRedraw(s)
}

func (s *scrollBar) setContentSize(size image.Point) {
	s.contentSize = size
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
			offsetX, offsetY := s.offsetGetSetter.scrollOffset()

			var pos, thumbMin, thumbMax int
			var offset float64
			var pageSize int
			if s.horizontal {
				pos = x
				thumbMin = tb.Min.X
				thumbMax = tb.Max.X
				offset = offsetX
				pageSize = widgetBounds.Bounds().Dx()
			} else {
				pos = y
				thumbMin = tb.Min.Y
				thumbMax = tb.Max.Y
				offset = offsetY
				pageSize = widgetBounds.Bounds().Dy()
			}

			// Check the cross-axis: the cursor must be on the scroll bar's side.
			if (s.horizontal && y >= tb.Min.Y) || (!s.horizontal && x >= tb.Min.X) {
				if pos >= thumbMin && pos < thumbMax {
					// Clicked on the thumb. Start dragging.
					s.dragging = true
					s.draggingStartPosition = pos
					s.draggingStartOffset = offset
				} else {
					// Clicked on the track area outside the thumb. Move by one page.
					if pos < thumbMin {
						offset += float64(pageSize)
					} else {
						offset -= float64(pageSize)
					}
					if s.horizontal {
						s.offsetGetSetter.setScrollOffset(offset, offsetY)
					} else {
						s.offsetGetSetter.setScrollOffset(offsetX, offset)
					}
					return guigui.HandleInputByWidget(s)
				}
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
			offsetX, offsetY := s.offsetGetSetter.scrollOffset()

			cs := widgetBounds.Bounds().Size()
			padding := scrollThumbPadding(context)
			barWidth, barHeight := scrollThumbSize(context, widgetBounds, s.contentSize)
			if s.horizontal && s.dragging && barWidth > 0 && s.contentSize.X-cs.X > 0 {
				trackWidth := float64(cs.X) - 2*padding - barWidth
				offsetPerPixel := float64(s.contentSize.X-cs.X) / trackWidth
				offsetX = s.draggingStartOffset + float64(-dx)*offsetPerPixel
			}
			if !s.horizontal && s.dragging && barHeight > 0 && s.contentSize.Y-cs.Y > 0 {
				trackHeight := float64(cs.Y) - 2*padding - barHeight
				offsetPerPixel := float64(s.contentSize.Y-cs.Y) / trackHeight
				offsetY = s.draggingStartOffset + float64(-dy)*offsetPerPixel
			}
			s.offsetGetSetter.setScrollOffset(offsetX, offsetY)
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
	barColor := draw.Color(context.ColorMode(), draw.SemanticColorBase, 0.2)
	barColor = draw.ScaleAlpha(barColor, s.alpha)
	basicwidgetdraw.DrawRoundedRect(context, dst, s.thumbBounds, barColor, RoundedCornerRadius(context))
}
