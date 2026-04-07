// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

// listPanel is a specialized panel for list widgets that uses virtual scrolling.
// Instead of measuring all items to compute the total content height,
// it tracks the topmost visible item index and its pixel offset.
//
// Scroll wheel input is handled directly (delta applied to topItemOffset).
// Scroll bar dragging maps position directly to item index.
// This avoids lossy round-trips through virtual pixel offsets.
type listPanel[T comparable] struct {
	guigui.DefaultWidget

	content    *listContent[T]
	scrollHBar scrollBar
	scrollVBar listVScrollBar[T]

	// topItemIndex is the index (in the available items list) of the topmost visible item.
	topItemIndex int

	// topItemOffset is the pixel offset of the top item's top edge
	// relative to the viewport top. This is typically <= 0.
	topItemOffset int

	// offsetX is the horizontal scroll offset.
	offsetX float64

	// Pending horizontal offset changes.
	nextOffsetXSet     bool
	nextOffsetXIsDelta bool
	nextOffsetX        float64

	// Pending vertical position changes.
	// When nextTopItemIsDelta, nextDeltaY is applied to topItemOffset.
	// Otherwise, nextTopItemIndex/nextTopItemOffset replace the current values.
	nextTopItemSet     bool
	nextTopItemIsDelta bool
	nextDeltaY         float64
	nextTopItemIndex   int
	nextTopItemOffset  int

	scrollBarCount int

	// estimatedItemHeight is the average item height computed during the most
	// recent layout, used to estimate scroll bar thumb size and viewport item count.
	estimatedItemHeight int

	// Scroll wheel state for bar visibility.
	lastWheelY float64
}

func (p *listPanel[T]) setContent(content *listContent[T]) {
	p.content = content
}

// scrollOffset returns (offsetX, 0). Only the horizontal offset is pixel-based.
// Vertical scroll is managed via topItemIndex/topItemOffset.
func (p *listPanel[T]) scrollOffset() (float64, float64) {
	return p.offsetX, 0
}

// setScrollOffset is called by the horizontal scroll bar.
// Only horizontal offset is meaningful here.
func (p *listPanel[T]) setScrollOffset(x, _ float64) {
	if p.offsetX == x {
		return
	}
	p.nextOffsetXSet = true
	p.nextOffsetXIsDelta = false
	p.nextOffsetX = x
}

// setScrollOffsetByDelta adjusts the horizontal offset by dx
// and the vertical position by dy pixels.
func (p *listPanel[T]) setScrollOffsetByDelta(dx, dy float64) {
	if dx != 0 {
		if p.nextOffsetXSet && p.nextOffsetXIsDelta {
			p.nextOffsetX += dx
		} else {
			p.nextOffsetXSet = true
			p.nextOffsetXIsDelta = true
			p.nextOffsetX = dx
		}
	}
	if dy != 0 {
		if p.nextTopItemSet && p.nextTopItemIsDelta {
			p.nextDeltaY += dy
		} else {
			p.nextTopItemSet = true
			p.nextTopItemIsDelta = true
			p.nextDeltaY = dy
		}
	}
}

// setTopItem sets the vertical scroll position directly by available-item index and offset.
func (p *listPanel[T]) setTopItem(index, offset int) {
	p.nextTopItemSet = true
	p.nextTopItemIsDelta = false
	p.nextTopItemIndex = index
	p.nextTopItemOffset = offset
}

// topItem returns the current vertical scroll state.
func (p *listPanel[T]) topItem() (int, int) {
	return p.topItemIndex, p.topItemOffset
}

// forceSetTopItem writes the top item position directly, bypassing and clearing
// any pending vertical offset. Used by layout code that has already measured items.
func (p *listPanel[T]) forceSetTopItem(index, offset int) {
	p.topItemIndex = index
	p.topItemOffset = offset
	p.nextTopItemSet = false
	p.nextTopItemIsDelta = false
	p.nextDeltaY = 0
	p.nextTopItemIndex = 0
	p.nextTopItemOffset = 0
}

// setEstimatedItemHeight records the estimated item height, used to
// estimate scroll bar thumb size and viewport item count.
func (p *listPanel[T]) setEstimatedItemHeight(h int) {
	p.estimatedItemHeight = h
}

func (p *listPanel[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(p.content)
	adder.AddWidget(&p.scrollHBar)
	adder.AddWidget(&p.scrollVBar)

	// Horizontal scroll bar uses the standard scrollOffsetGetSetter interface.
	p.scrollHBar.setOffsetGetSetter(p)
	p.scrollHBar.setHorizontal(true)
	p.scrollVBar.panel = p

	context.SetClipChildren(p, true)
	context.DelegateFocus(p, p.content)

	return nil
}

// HandlePointingInput handles scroll wheel input directly,
// applying vertical deltas to topItemOffset without virtual offset conversion.
func (p *listPanel[T]) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	// Handle scroll wheel.
	if widgetBounds.IsHitAtCursor() {
		wheelX, wheelY := adjustedWheel()
		p.lastWheelY = wheelY
		if wheelX != 0 || wheelY != 0 {
			dx := wheelX * scrollWheelSpeed(context)
			dy := wheelY * scrollWheelSpeed(context)
			p.setScrollOffsetByDelta(dx, dy)
			return guigui.HandleInputByWidget(p)
		}
	} else {
		p.lastWheelY = 0
	}

	return guigui.HandleInputResult{}
}

func (p *listPanel[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	p.applyPendingScrollOffset()

	bounds := widgetBounds.Bounds()

	// listContent.layoutItems handles clamping and normalization of
	// topItemIndex/topItemOffset, so we don't need to do it here.

	// Compute horizontal content size for scroll bar.
	cw := p.content.contentWidth()
	if cw == 0 {
		cw = bounds.Dx()
	}

	// Adjust horizontal offset.
	maxOffsetX := float64(min(bounds.Dx()-cw, 0))
	p.offsetX = min(max(p.offsetX, maxOffsetX), 0)

	// Layout the content widget at the panel bounds with the horizontal offset.
	// The listContent will use topItemIndex/topItemOffset to position items.
	pt := bounds.Min.Add(image.Pt(int(p.offsetX), 0))
	contentSize := image.Pt(cw, bounds.Dy())
	layouter.LayoutWidget(p.content, image.Rectangle{
		Min: pt,
		Max: pt.Add(contentSize),
	})

	// Set content size for horizontal scroll bar only.
	hContentSize := image.Pt(cw, bounds.Dy())
	p.scrollHBar.setContentSize(hContentSize)

	layouter.LayoutWidget(&p.scrollHBar, p.horizontalBarBounds(context, widgetBounds))
	layouter.LayoutWidget(&p.scrollVBar, p.verticalBarBounds(context, widgetBounds))

	hb, vb := p.thumbBounds(context, widgetBounds)
	p.scrollHBar.setThumbBounds(hb)
	p.scrollVBar.setThumbBounds(vb)
}

func (p *listPanel[T]) applyPendingScrollOffset() {
	if p.nextOffsetXSet {
		if p.nextOffsetXIsDelta {
			p.offsetX += p.nextOffsetX
		} else {
			p.offsetX = p.nextOffsetX
		}
		p.nextOffsetXSet = false
		p.nextOffsetXIsDelta = false
		p.nextOffsetX = 0
	}
	if p.nextTopItemSet {
		if p.nextTopItemIsDelta {
			p.topItemOffset += int(p.nextDeltaY)
		} else {
			p.topItemIndex = p.nextTopItemIndex
			p.topItemOffset = p.nextTopItemOffset
		}
		p.nextTopItemSet = false
		p.nextTopItemIsDelta = false
		p.nextDeltaY = 0
		p.nextTopItemIndex = 0
		p.nextTopItemOffset = 0
	}
}

func (p *listPanel[T]) horizontalBarBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	bounds := widgetBounds.Bounds()
	bounds.Min.Y = max(bounds.Min.Y, bounds.Max.Y-scrollBarAreaSize(context))
	return bounds
}

func (p *listPanel[T]) verticalBarBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	bounds := widgetBounds.Bounds()
	bounds.Min.X = max(bounds.Min.X, bounds.Max.X-scrollBarAreaSize(context))
	return bounds
}

func (p *listPanel[T]) isScrolling() bool {
	return p.lastWheelY != 0
}

func (p *listPanel[T]) isBarVisible(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	if p.isScrolling() {
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

func (p *listPanel[T]) startShowingBarsIfNeeded(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
	if hb, vb := p.thumbBounds(context, widgetBounds); hb.Empty() && vb.Empty() {
		return
	}

	switch {
	case p.scrollBarCount >= scrollBarMaxCount()-scrollBarFadingInTime():
	case p.scrollBarCount >= scrollBarFadingOutTime():
		p.scrollBarCount = scrollBarMaxCount() - scrollBarFadingInTime()
	case p.scrollBarCount > 0:
		p.scrollBarCount = scrollBarMaxCount() - scrollBarFadingInTime()
	default:
		p.scrollBarCount = scrollBarMaxCount()
	}
}

func (p *listPanel[T]) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	shouldShowBar := p.isBarVisible(context, widgetBounds)

	if p.applyPendingScrollOffsetInTick() {
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

	alpha := scrollThumbOpacity(p.scrollBarCount)
	p.scrollHBar.setAlpha(alpha)
	p.scrollVBar.setAlpha(alpha)

	return nil
}

func (p *listPanel[T]) applyPendingScrollOffsetInTick() bool {
	if !p.nextOffsetXSet && !p.nextTopItemSet {
		return false
	}

	oldOffsetX := p.offsetX
	oldTopItemIndex := p.topItemIndex
	oldTopItemOffset := p.topItemOffset

	p.applyPendingScrollOffset()

	changed := p.offsetX != oldOffsetX || p.topItemIndex != oldTopItemIndex || p.topItemOffset != oldTopItemOffset
	if changed {
		guigui.RequestRebuild(p)
	}
	return changed
}

// vThumbHeight returns the vertical thumb height.
// Returns 0 if no items have been measured yet or no thumb should be shown.
func (p *listPanel[T]) vThumbHeight(context *guigui.Context, widgetBounds *guigui.WidgetBounds, totalCount int) float64 {
	if p.estimatedItemHeight <= 0 || totalCount == 0 {
		return 0
	}
	bounds := widgetBounds.Bounds()
	padding := scrollThumbPadding(context)
	viewportItems := float64(bounds.Dy()) / float64(p.estimatedItemHeight)
	if viewportItems >= float64(totalCount) {
		return 0
	}
	barHeight := (float64(bounds.Dy()) - 2*padding) * viewportItems / float64(totalCount)
	return max(barHeight, scrollThumbStrokeWidth(context))
}

func (p *listPanel[T]) thumbBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (image.Rectangle, image.Rectangle) {
	bounds := widgetBounds.Bounds()
	padding := scrollThumbPadding(context)

	var horizontalBarBounds, verticalBarBounds image.Rectangle

	// Horizontal thumb.
	if cw := p.content.contentWidth(); cw > bounds.Dx() {
		barWidth := (float64(bounds.Dx()) - 2*padding) * float64(bounds.Dx()) / float64(cw)
		barWidth = max(barWidth, scrollThumbStrokeWidth(context))

		rate := -p.offsetX / float64(cw-bounds.Dx())
		x0 := float64(bounds.Min.X) + padding + rate*(float64(bounds.Dx())-2*padding-barWidth)
		x1 := x0 + barWidth
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

	// Vertical thumb — position based directly on topItemIndex.
	totalCount := p.content.availableItemCount()
	if barHeight := p.vThumbHeight(context, widgetBounds, totalCount); barHeight > 0 {
		// barHeight > 0 guarantees estimatedItemHeight > 0 (see vThumbHeight),
		// so the division below is safe.
		viewportItems := float64(bounds.Dy()) / float64(p.estimatedItemHeight)
		maxIndex := float64(totalCount) - viewportItems
		if maxIndex < 1 {
			maxIndex = 1
		}
		var rate float64
		if float64(p.topItemIndex)+viewportItems >= float64(totalCount) {
			// The last item is visible — thumb should be at the bottom.
			rate = 1
		} else {
			rate = float64(p.topItemIndex) / maxIndex
			rate = min(max(rate, 0), 1)
		}
		y0 := float64(bounds.Min.Y) + padding + rate*(float64(bounds.Dy())-2*padding-barHeight)
		y1 := y0 + barHeight
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

// listVScrollBar is a child widget that draws and handles input for
// the vertical scroll bar of a listPanel. It maps drag position directly
// to item index, avoiding lossy virtual offset conversions.
type listVScrollBar[T comparable] struct {
	guigui.DefaultWidget

	panel       *listPanel[T]
	thumbBounds image.Rectangle
	alpha       float64

	dragging              bool
	draggingStartPosition int
	draggingStartIndex    int
	onceDraw              bool
}

func (s *listVScrollBar[T]) setThumbBounds(bounds image.Rectangle) {
	if s.thumbBounds == bounds {
		return
	}
	s.thumbBounds = bounds
	guigui.RequestRedraw(s)
}

func (s *listVScrollBar[T]) setAlpha(alpha float64) {
	if s.alpha == alpha {
		return
	}
	s.alpha = alpha
	if !s.thumbBounds.Empty() {
		guigui.RequestRedraw(s)
	}
}

func (s *listVScrollBar[T]) isDragging() bool {
	return s.dragging
}

func (s *listVScrollBar[T]) isOnceDrawn() bool {
	return s.onceDraw
}

func (s *listVScrollBar[T]) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	totalCount := s.panel.content.availableItemCount()
	if totalCount == 0 {
		return guigui.HandleInputResult{}
	}

	bounds := widgetBounds.Bounds()
	padding := scrollThumbPadding(context)
	// barHeight > 0 guarantees estimatedItemHeight > 0 (see vThumbHeight),
	// so divisions by estimatedItemHeight below are safe.
	barHeight := s.panel.vThumbHeight(context, widgetBounds, totalCount)
	if barHeight <= 0 {
		return guigui.HandleInputResult{}
	}
	trackHeight := float64(bounds.Dy()) - 2*padding - barHeight

	if !s.dragging && widgetBounds.IsHitAtCursor() && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		tb := s.thumbBounds
		topIdx, _ := s.panel.topItem()

		// Check the cross-axis: cursor must be on the scroll bar's side.
		if x >= tb.Min.X || x >= bounds.Min.X {
			if !tb.Empty() && y >= tb.Min.Y && y < tb.Max.Y {
				// Clicked on thumb — start dragging.
				s.dragging = true
				s.draggingStartPosition = y
				s.draggingStartIndex = topIdx
				return guigui.HandleInputByWidget(s)
			}
			// Clicked on track — jump by page.
			if !tb.Empty() {
				pageItems := bounds.Dy() / s.panel.estimatedItemHeight
				if y < tb.Min.Y {
					s.panel.setTopItem(max(0, topIdx-pageItems), 0)
				} else {
					s.panel.setTopItem(min(totalCount-1, topIdx+pageItems), 0)
				}
				return guigui.HandleInputByWidget(s)
			}
		}
	}

	if wheelX, wheelY := adjustedWheel(); wheelX != 0 || wheelY != 0 {
		s.dragging = false
	}

	if s.dragging && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		_, y := ebiten.CursorPosition()
		dy := y - s.draggingStartPosition
		if dy != 0 && trackHeight > 0 {
			// Map pixel drag to item index. Use totalCount-1 as the range
			// so dragging to the bottom always reaches the last item.
			// The bottom-gap fix in layoutItems will correct any overshoot.
			indexPerPixel := float64(totalCount-1) / trackHeight
			deltaItems := float64(dy) * indexPerPixel
			newIdx := s.draggingStartIndex + int(math.Round(deltaItems))
			newIdx = max(0, min(totalCount-1, newIdx))
			s.panel.forceSetTopItem(newIdx, 0)
			guigui.RequestRebuild(s.panel)
		}
		return guigui.HandleInputByWidget(s)
	}

	if s.dragging && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		s.dragging = false
	}

	return guigui.HandleInputResult{}
}

func (s *listVScrollBar[T]) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	return ebiten.CursorShapeDefault, true
}

func (s *listVScrollBar[T]) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
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
