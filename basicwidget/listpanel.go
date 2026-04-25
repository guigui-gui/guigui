// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image"

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

	// Vertical scroll animation state. The animation interpolates a pixel
	// delta (vAnimDelta) over scrollAnimMaxCount() ticks using easeOutQuad,
	// applying the eased increment to topItemOffset on each tick. topItemIndex
	// is left untouched during the animation; normalizeTopItem advances it
	// between ticks using real measured heights, which keeps the visible
	// scroll smooth when items have heterogeneous heights. The final tick
	// snaps (topItemIndex, topItemOffset) to (vAnimTargetIndex, vAnimTargetOffset)
	// to ensure an exact landing position.
	// vAnimCount counts down from scrollAnimMaxCount() to 0; a positive value
	// indicates an animation is in flight.
	vAnimTargetIndex  int
	vAnimTargetOffset int
	vAnimDelta        int
	vAnimAppliedDelta int
	vAnimCount        int

	scrollHBarCount int
	scrollVBarCount int

	// estimatedItemHeight is the average item height computed during the most
	// recent layout, used to estimate scroll bar thumb size and viewport item count.
	estimatedItemHeight int

	// Scroll wheel state for bar visibility.
	lastWheelX float64
	lastWheelY float64

	onceDraw bool
}

func (p *listPanel[T]) WriteStateKey(w *guigui.StateKeyWriter) {
	w.WriteInt64(int64(p.topItemIndex))
	w.WriteInt64(int64(p.topItemOffset))
	w.WriteFloat64(p.offsetX)
}

func (p *listPanel[T]) setContent(content *listContent[T]) {
	p.content = content
}

// scrollOffset returns (offsetX, 0). Only the horizontal offset is pixel-based.
// Vertical scroll is managed via topItemIndex/topItemOffset.
func (p *listPanel[T]) scrollOffset() (float64, float64) {
	return p.offsetX, 0
}

// forceSetScrollOffsetX sets the horizontal scroll offset.
func (p *listPanel[T]) forceSetScrollOffsetX(x float64) {
	if p.offsetX == x {
		return
	}
	p.nextOffsetXSet = true
	p.nextOffsetXIsDelta = false
	p.nextOffsetX = x
}

// forceSetScrollOffset satisfies [scrollOffsetGetSetter], used by the horizontal
// scroll bar. Y is ignored because vertical scroll is item-based, not
// pixel-based (see [listPanel.forceSetTopItem] / [listPanel.setTopItem]).
func (p *listPanel[T]) forceSetScrollOffset(x, _ float64) {
	p.forceSetScrollOffsetX(x)
}

// forceSetScrollOffsetByDelta adjusts the horizontal offset by dx and the
// vertical position by dy pixels, without animation. Direct user input (wheel)
// cancels any in-flight vertical animation.
func (p *listPanel[T]) forceSetScrollOffsetByDelta(dx, dy float64) {
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
		p.vAnimCount = 0
		if p.nextTopItemSet && p.nextTopItemIsDelta {
			p.nextDeltaY += dy
		} else {
			p.nextTopItemSet = true
			p.nextTopItemIsDelta = true
			p.nextDeltaY = dy
		}
	}
}

// setTopItem animates the vertical scroll position toward the given
// available-item index and offset. Falls back to an instant set when
// no item-height estimate is available yet, or before the first Draw.
func (p *listPanel[T]) setTopItem(index, offset int) {
	estH := p.estimatedItemHeight
	if estH <= 0 || !p.onceDraw {
		p.vAnimCount = 0
		p.nextTopItemSet = true
		p.nextTopItemIsDelta = false
		p.nextTopItemIndex = index
		p.nextTopItemOffset = offset
		return
	}
	if p.vAnimCount > 0 && index == p.vAnimTargetIndex && offset == p.vAnimTargetOffset {
		return
	}
	if index == p.topItemIndex && offset == p.topItemOffset {
		// The caller is asking to scroll to the current top item. If an
		// animation is in flight, don't restart it toward its own mid-flight
		// position — that would freeze the scroll. Mirrors the guard in
		// panel.SetScrollOffset; see the comment there.
		return
	}
	// Compute the total pixel delta from current to target.
	// When index == p.topItemIndex (the typical case for arrow-key navigation
	// scrolling within a page), estH cancels and the delta equals
	// p.topItemOffset - offset exactly, regardless of the height estimate.
	// For cross-index animations, the delta uses estH and may be approximate;
	// the final-tick snap to (vAnimTargetIndex, vAnimTargetOffset) corrects it.
	currentScroll := p.topItemIndex*estH - p.topItemOffset
	targetScroll := index*estH - offset
	// Animation supersedes any pending instant change.
	p.nextTopItemSet = false
	p.nextTopItemIsDelta = false
	p.nextDeltaY = 0
	p.nextTopItemIndex = 0
	p.nextTopItemOffset = 0
	p.vAnimTargetIndex = index
	p.vAnimTargetOffset = offset
	p.vAnimDelta = targetScroll - currentScroll
	p.vAnimAppliedDelta = 0
	p.vAnimCount = scrollAnimMaxCount()
}

// topItem returns the current vertical scroll state.
func (p *listPanel[T]) topItem() (int, int) {
	return p.topItemIndex, p.topItemOffset
}

// forceSetTopItem writes the top item position directly.
//
// When cancelAnimation is true, any pending vertical change and in-flight
// animation are cleared. Used by direct/user input (e.g. scroll-bar drag)
// that should supersede the animation.
//
// When cancelAnimation is false, the animation is preserved. This is layout
// bookkeeping — normalization derives canonical (index, offset) from real
// item heights so that readers like the scroll bar thumb see a consistent
// position, and that derivation must not cancel the animation target. Callers
// must ensure no pending vertical change is queued (verified by the assert).
// In practice Layout runs applyPendingScrollOffset before child layout.
func (p *listPanel[T]) forceSetTopItem(index, offset int, cancelAnimation bool) {
	if !cancelAnimation && p.nextTopItemSet {
		panic("basicwidget: forceSetTopItem(cancelAnimation=false) called with a pending vertical change; callers must run applyPendingScrollOffset first")
	}
	p.topItemIndex = index
	p.topItemOffset = offset
	p.nextTopItemSet = false
	p.nextTopItemIsDelta = false
	p.nextDeltaY = 0
	p.nextTopItemIndex = 0
	p.nextTopItemOffset = 0
	if cancelAnimation {
		p.vAnimCount = 0
	}
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
		p.lastWheelX = wheelX
		p.lastWheelY = wheelY
		if wheelX != 0 || wheelY != 0 {
			dx := wheelX * scrollWheelSpeed(context)
			dy := wheelY * scrollWheelSpeed(context)
			p.forceSetScrollOffsetByDelta(dx, dy)
			return guigui.HandleInputByWidget(p)
		}
	} else {
		p.lastWheelX = 0
		p.lastWheelY = 0
	}

	return guigui.HandleInputResult{}
}

func (p *listPanel[T]) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	p.onceDraw = true
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

func (p *listPanel[T]) isScrollingX() bool {
	return p.lastWheelX != 0
}

func (p *listPanel[T]) isScrollingY() bool {
	return p.lastWheelY != 0
}

func (p *listPanel[T]) isHBarVisible(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	if p.isScrollingX() {
		return true
	}
	if p.scrollHBar.isDragging() {
		return true
	}
	if !widgetBounds.IsHitAtCursor() {
		return false
	}
	pt := image.Pt(ebiten.CursorPosition())
	return pt.In(p.horizontalBarBounds(context, widgetBounds))
}

func (p *listPanel[T]) isVBarVisible(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	if p.isScrollingY() {
		return true
	}
	if p.scrollVBar.isDragging() {
		return true
	}
	if !widgetBounds.IsHitAtCursor() {
		return false
	}
	pt := image.Pt(ebiten.CursorPosition())
	return pt.In(p.verticalBarBounds(context, widgetBounds))
}

func (p *listPanel[T]) startShowingHBarIfNeeded(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
	if hb, _ := p.thumbBounds(context, widgetBounds); hb.Empty() {
		return
	}
	p.scrollHBarCount = startShowingBarCount(p.scrollHBarCount)
}

func (p *listPanel[T]) startShowingVBarIfNeeded(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
	if _, vb := p.thumbBounds(context, widgetBounds); vb.Empty() {
		return
	}
	p.scrollVBarCount = startShowingBarCount(p.scrollVBarCount)
}

func (p *listPanel[T]) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	shouldShowHBar := p.isHBarVisible(context, widgetBounds)
	shouldShowVBar := p.isVBarVisible(context, widgetBounds)
	// lastWheelX/Y are a one-tick signal: HandlePointingInput only runs on ticks
	// with pointing activity, so without this reset a stopped wheel would keep
	// isScrollingX/Y() true until the cursor next moves.
	p.lastWheelX = 0
	p.lastWheelY = 0

	hChanged, vChanged := p.applyPendingScrollOffsetInTick()
	if p.advanceScrollAnimation() {
		vChanged = true
	}
	if hChanged && p.scrollHBar.isOnceDrawn() {
		shouldShowHBar = true
	}
	if vChanged && p.scrollVBar.isOnceDrawn() {
		shouldShowVBar = true
	}

	oldHOpacity := scrollThumbOpacity(p.scrollHBarCount)
	oldVOpacity := scrollThumbOpacity(p.scrollVBarCount)
	if shouldShowHBar {
		p.startShowingHBarIfNeeded(context, widgetBounds)
	}
	if shouldShowVBar {
		p.startShowingVBarIfNeeded(context, widgetBounds)
	}
	newHOpacity := scrollThumbOpacity(p.scrollHBarCount)
	newVOpacity := scrollThumbOpacity(p.scrollVBarCount)

	if newHOpacity != oldHOpacity || newVOpacity != oldVOpacity {
		guigui.RequestRedraw(p)
	}

	if p.scrollHBarCount > 0 {
		if !shouldShowHBar || p.scrollHBarCount != scrollBarMaxCount()-scrollBarFadingInTime() {
			p.scrollHBarCount--
		}
	}
	if p.scrollVBarCount > 0 {
		if !shouldShowVBar || p.scrollVBarCount != scrollBarMaxCount()-scrollBarFadingInTime() {
			p.scrollVBarCount--
		}
	}

	p.scrollHBar.setAlpha(scrollThumbOpacity(p.scrollHBarCount))
	p.scrollVBar.setAlpha(scrollThumbOpacity(p.scrollVBarCount))

	return nil
}

// advanceScrollAnimation advances the vertical scroll animation by one tick.
// Each tick applies the eased increment of vAnimDelta to topItemOffset only;
// topItemIndex is updated by normalizeTopItem between ticks using real measured
// heights. This avoids visual jumps when items have heterogeneous heights — a
// virtual-pixel-space interpolation can otherwise step topItemIndex on a tick
// where the actual item heights say it should not yet have advanced (or vice
// versa), producing a backward jump in the rendered position. The final tick
// snaps (topItemIndex, topItemOffset) to the exact target so any approximation
// in vAnimDelta (cross-index animations using estH) lands cleanly.
func (p *listPanel[T]) advanceScrollAnimation() bool {
	if p.vAnimCount <= 0 {
		return false
	}
	p.vAnimCount--
	if p.vAnimCount <= 0 {
		p.topItemIndex = p.vAnimTargetIndex
		p.topItemOffset = p.vAnimTargetOffset
		return true
	}
	max := scrollAnimMaxCount()
	t := easeOutQuad(float64(max-p.vAnimCount) / float64(max))
	// Track the cumulative integer delta so float→int truncation doesn't
	// accumulate across ticks.
	desired := int(float64(p.vAnimDelta) * t)
	delta := desired - p.vAnimAppliedDelta
	p.vAnimAppliedDelta = desired
	p.topItemOffset -= delta
	return true
}

// applyPendingScrollOffsetInTick applies pending offsets and reports whether
// the horizontal and vertical positions changed, respectively.
func (p *listPanel[T]) applyPendingScrollOffsetInTick() (bool, bool) {
	if !p.nextOffsetXSet && !p.nextTopItemSet {
		return false, false
	}

	oldOffsetX := p.offsetX
	oldTopItemIndex := p.topItemIndex
	oldTopItemOffset := p.topItemOffset

	p.applyPendingScrollOffset()

	// topItemIndex/topItemOffset/offsetX are in the listPanel's WriteStateKey,
	// so the rebuild that re-invokes Layout is triggered automatically.
	hChanged := p.offsetX != oldOffsetX
	vChanged := p.topItemIndex != oldTopItemIndex || p.topItemOffset != oldTopItemOffset
	return hChanged, vChanged
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
			// Include topItemOffset so the thumb moves smoothly between items.
			// topItemOffset is typically <= 0; negating it gives the fraction scrolled into the current item.
			topItemH := p.content.measureAvailableItemHeight(context, p.topItemIndex)
			if topItemH <= 0 {
				topItemH = p.estimatedItemHeight
			}
			fractionalIndex := float64(p.topItemIndex) + float64(-p.topItemOffset)/float64(topItemH)
			rate = fractionalIndex / maxIndex
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
	draggingStartOffset   int
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
		topIdx, topOff := s.panel.topItem()

		// Check the cross-axis: cursor must be on the scroll bar's side.
		if x >= tb.Min.X || x >= bounds.Min.X {
			if !tb.Empty() && y >= tb.Min.Y && y < tb.Max.Y {
				// Clicked on thumb — start dragging.
				s.dragging = true
				s.draggingStartPosition = y
				s.draggingStartIndex = topIdx
				s.draggingStartOffset = topOff
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
			// Map pixel drag to item index. Use the same maxIndex as
			// thumbBounds so that the thumb tracks the cursor 1:1.
			viewportItems := float64(bounds.Dy()) / float64(s.panel.estimatedItemHeight)
			maxIndex := float64(totalCount) - viewportItems
			if maxIndex < 1 {
				maxIndex = 1
			}
			indexPerPixel := maxIndex / trackHeight
			deltaItems := float64(dy) * indexPerPixel
			// Use fractional position to compute both index and sub-item offset.
			// Use the actual height of the start item (matching thumbBounds) so
			// the start fraction agrees with the thumb position on screen.
			startItemH := s.panel.content.measureAvailableItemHeight(context, s.draggingStartIndex)
			if startItemH <= 0 {
				startItemH = s.panel.estimatedItemHeight
			}
			startFraction := float64(s.draggingStartIndex) + float64(-s.draggingStartOffset)/float64(startItemH)
			newFraction := startFraction + deltaItems
			newFraction = min(max(newFraction, 0), float64(totalCount-1))
			newIdx := int(newFraction)
			// Use the actual height of the target item for the sub-item offset.
			newItemH := s.panel.content.measureAvailableItemHeight(context, newIdx)
			if newItemH <= 0 {
				newItemH = s.panel.estimatedItemHeight
			}
			newOffset := -int((newFraction - float64(newIdx)) * float64(newItemH))
			s.panel.forceSetTopItem(newIdx, newOffset, true)
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
