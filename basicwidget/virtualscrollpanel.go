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

var virtualScrollPanelEventScroll guigui.EventKey = guigui.GenerateEventKey()

// virtualScrollContent is the interface that [virtualScrollPanel] requires
// from its content widget.
type virtualScrollContent interface {
	guigui.Widget

	// contentWidth returns the rendered content width.
	contentWidth(context *guigui.Context) int

	// itemCount returns the total number of scrollable items.
	itemCount() int

	// measureItemHeight returns the rendered height of the item at the given
	// index. For an in-range index (0 <= index < itemCount()) the returned
	// value must be >= 0; zero is a valid height. For an out-of-range index
	// it must return -1.
	measureItemHeight(context *guigui.Context, index int) int

	// viewportPaddingY returns the total vertical padding the content
	// reserves inside the panel viewport.
	viewportPaddingY(context *guigui.Context) int
}

// virtualScrollPanel is a scroll panel that uses virtual scrolling: it tracks
// the topmost visible item's index and pixel offset instead of measuring all
// items to compute total content height.
//
// "Item" here is whatever unit the [virtualScrollContent] implementation
// chooses — a list row, a logical line of text, etc.
type virtualScrollPanel struct {
	guigui.DefaultWidget

	content    virtualScrollContent
	scrollHBar scrollBar
	scrollVBar virtualScrollVBar

	horizontalBarHidden bool

	// topItemIndex is the index of the topmost visible item.
	topItemIndex int

	// topItemOffset is the pixel offset of the top item's top edge relative
	// to the viewport top. This is typically <= 0.
	topItemOffset int

	// offsetX is the horizontal scroll offset.
	offsetX float64

	// scrolledTopItemIndex, scrolledTopItemOffset, and scrolledOffsetX record the
	// scroll position last emitted via virtualScrollPanelEventScroll, so the event
	// is dispatched only when the position actually changes.
	scrolledTopItemIndex  int
	scrolledTopItemOffset int
	scrolledOffsetX       float64

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

	// Vertical scroll animation state. vAnimCount > 0 means an animation
	// is in flight; it counts down from scrollAnimMaxCount() to 0. Each
	// tick eases vAnimDelta into topItemOffset; the final tick snaps to
	// (vAnimTargetIndex, vAnimTargetOffset).
	vAnimTargetIndex  int
	vAnimTargetOffset int
	vAnimDelta        int
	vAnimAppliedDelta int
	vAnimCount        int

	scrollHBarCount int
	scrollVBarCount int

	// estimatedItemHeight is the average item height computed during the
	// most recent layout, used to estimate scroll bar thumb size and
	// viewport item count.
	estimatedItemHeight int

	// estimatedTotalHeightInPixels is the estimated total content height: the
	// per-item average over the sample window scaled to the full item count.
	// Float-valued so the thumb size does not step with estimatedItemHeight's
	// rounding; converges to accurateTotalHeightInPixels once the window
	// covers every item.
	estimatedTotalHeightInPixels float64

	// allHeightsMeasured reports whether every item's height was measured this
	// layout, making the accurate* fields below exact rather than estimated.
	allHeightsMeasured bool

	// accurateTotalHeightInPixels is the exact total content height in
	// pixels. Valid only when allHeightsMeasured.
	accurateTotalHeightInPixels int

	// accurateHeightAboveTopInPixels is the exact pixel sum of items
	// 0..topItemIndex-1; it does not include topItemOffset. Valid only
	// when allHeightsMeasured.
	accurateHeightAboveTopInPixels int

	// atBottom reports whether the most recent layout left the list scrolled to
	// its end, with no content extending past the viewport bottom.
	atBottom bool

	// Scroll wheel state for bar visibility.
	lastWheelX float64
	lastWheelY float64

	onceDraw bool
}

func (p *virtualScrollPanel) WriteStateKey(w *guigui.StateKeyWriter) {
	w.WriteInt64(int64(p.topItemIndex))
	w.WriteInt64(int64(p.topItemOffset))
	w.WriteFloat64(p.offsetX)
}

func (p *virtualScrollPanel) setContent(content virtualScrollContent) {
	p.content = content
}

// setHorizontalBarHidden sets whether the horizontal scroll bar thumb is
// suppressed regardless of content width.
func (p *virtualScrollPanel) setHorizontalBarHidden(hidden bool) {
	p.horizontalBarHidden = hidden
}

// scrollOffset returns (offsetX, 0). Only the horizontal offset is pixel-based.
// Vertical scroll is managed via topItemIndex/topItemOffset.
func (p *virtualScrollPanel) scrollOffset() (float64, float64) {
	return p.offsetX, 0
}

// OnScroll sets the handler invoked when the scroll position changes.
func (p *virtualScrollPanel) OnScroll(callback func(context *guigui.Context, topItemIndex, topItemOffset int, offsetX float64)) {
	guigui.SetEventHandler(p, virtualScrollPanelEventScroll, callback)
}

// forceSetScrollOffsetX sets the horizontal scroll offset.
func (p *virtualScrollPanel) forceSetScrollOffsetX(x float64) {
	if p.offsetX == x {
		return
	}
	p.nextOffsetXSet = true
	p.nextOffsetXIsDelta = false
	p.nextOffsetX = x
}

// forceSetScrollOffset satisfies [scrollOffsetGetSetter]. Y is ignored because
// vertical scroll is item-based, not pixel-based.
func (p *virtualScrollPanel) forceSetScrollOffset(x, _ float64) {
	p.forceSetScrollOffsetX(x)
}

// forceSetScrollOffsetByDelta adjusts the horizontal offset by dx and the
// vertical position by dy pixels, without animation. Direct user input (wheel)
// cancels any in-flight vertical animation.
func (p *virtualScrollPanel) forceSetScrollOffsetByDelta(dx, dy float64) {
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

// setTopItem animates the vertical scroll position toward the given item
// index and offset. Falls back to an instant set when no item-height
// estimate is available yet, or before the first Draw.
func (p *virtualScrollPanel) setTopItem(index, offset int) {
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
func (p *virtualScrollPanel) topItem() (int, int) {
	return p.topItemIndex, p.topItemOffset
}

// forceSetTopItem writes the top item position directly.
//
// When cancelAnimation is true, any pending vertical change and in-flight
// animation are cleared. When cancelAnimation is false, the animation is
// preserved and the caller must ensure no pending vertical change is queued
// (verified by the assert).
func (p *virtualScrollPanel) forceSetTopItem(index, offset int, cancelAnimation bool) {
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

// layoutTopItem resolves the panel's (topItemIndex, topItemOffset) to its
// canonical layout-ready form, commits it via forceSetTopItem, and returns the
// committed values. It also records whether the resolved position is at the end
// of the content (p.atBottom).
//
// apparentItemHeight returns the per-item height as it appears in the viewport;
// callers wrap [virtualScrollContent.measureItemHeight] to apply per-item
// visual effects such as a list's expand-animation scaling.
//
// viewportInner is the content area height: panel bounds minus
// content.viewportPaddingY.
func (p *virtualScrollPanel) layoutTopItem(context *guigui.Context, viewportInner int, apparentItemHeight func(ai int) int) (idx, offset int) {
	n := p.content.itemCount()
	idx = p.topItemIndex
	offset = p.topItemOffset
	if n == 0 {
		idx, offset = 0, 0
		p.forceSetTopItem(idx, offset, false)
		p.atBottom = true
		return
	}
	if idx >= n {
		idx = n - 1
		offset = 0
	}
	if idx < 0 {
		idx = 0
		offset = 0
	}

	// During a scroll animation, substitute estimatedItemHeight in the
	// normalize walks to avoid re-measuring every item the eased pixel delta
	// passes over (wrapped text shapes each line). The settling Layout
	// (vAnimCount == 0) walks apparentItemHeight again.
	measure := func(i int) int {
		if p.vAnimCount > 0 && p.estimatedItemHeight > 0 {
			return p.estimatedItemHeight
		}
		return apparentItemHeight(i)
	}
	for offset < 0 && idx < n-1 {
		ih := measure(idx)
		if -offset >= ih {
			offset += ih
			idx++
			continue
		}
		break
	}
	for offset > 0 && idx > 0 {
		idx--
		offset -= measure(idx)
	}
	if idx == 0 && offset > 0 {
		offset = 0
	}

	y := offset
	var reachedEnd bool
	for ai := idx; ai < n; ai++ {
		if y >= viewportInner {
			break
		}
		y += apparentItemHeight(ai)
		if ai == n-1 {
			reachedEnd = true
		}
	}
	if reachedEnd {
		if gap := viewportInner - y; gap > 0 {
			offset += gap
			for offset > 0 && idx > 0 {
				idx--
				offset -= apparentItemHeight(idx)
			}
			if idx == 0 && offset > 0 {
				offset = 0
			}
		}
	}

	p.forceSetTopItem(idx, offset, false)
	// The end is on screen only if the walk reached the last item without its
	// bottom (y) passing the viewport: a tall last item can be touched yet still
	// extend below, leaving room to scroll further.
	p.atBottom = reachedEnd && y <= viewportInner
	return idx, offset
}

// updateHeightMetrics samples item heights and refreshes the cached
// scroll-bar metrics: estimatedItemHeight always, and the accurate* fields when
// every item's height was measured.
func (p *virtualScrollPanel) updateHeightMetrics(context *guigui.Context, panelBounds image.Rectangle) {
	totalCount := p.content.itemCount()
	if totalCount == 0 {
		p.estimatedItemHeight = 0
		p.estimatedTotalHeightInPixels = 0
		p.allHeightsMeasured = false
		p.accurateTotalHeightInPixels = 0
		p.accurateHeightAboveTopInPixels = 0
		return
	}
	// Skip mid-animation: estimatedItemHeight was captured into vAnimDelta at
	// animation start, and the thumb size can stay frozen for the brief
	// animation window. The settling Layout (vAnimCount == 0) refreshes both.
	if p.vAnimCount > 0 {
		return
	}

	// Measure every item when the list is small enough, making the accurate*
	// metrics exact for pixel-accurate thumb positioning. The threshold is a
	// fixed count, not a test of whether the sliding sample window covers every
	// item: window coverage depends on the scroll position, so it would flip
	// allHeightsMeasured mid-scroll and snap the thumb between positioning modes.
	const maxFullyMeasuredItemCount = 256
	allMeasured := totalCount <= maxFullyMeasuredItemCount
	var start, end int
	if allMeasured {
		start = 0
		end = totalCount - 1
	} else {
		// Count the items filling the viewport by walking measured heights, not by
		// dividing by estimatedItemHeight: sizing the window from the estimate it
		// feeds can oscillate as the estimate crosses integer boundaries. These
		// items are on-screen, so their heights are already cached.
		var viewportCount int
		var y int
		for i := p.topItemIndex; i < totalCount && y < panelBounds.Dy(); i++ {
			h := p.content.measureItemHeight(context, i)
			if h < 0 {
				break
			}
			y += h
			viewportCount++
		}
		viewportCount = max(1, viewportCount)

		// Sample heights from a window spanning at least 10 items, and 5 viewports, on each side of the top item.
		extendCount := max(10, 5*viewportCount)
		start = max(0, p.topItemIndex-extendCount)
		end = min(totalCount-1, p.topItemIndex+viewportCount+extendCount)
	}

	var sum, count, heightAboveTop int
	for i := start; i <= end; i++ {
		h := p.content.measureItemHeight(context, i)
		if h < 0 {
			allMeasured = false
			continue
		}
		sum += h
		count++
		if i < p.topItemIndex {
			heightAboveTop += h
		}
	}
	if count > 0 {
		p.estimatedItemHeight = sum / count
		p.estimatedTotalHeightInPixels = float64(sum) / float64(count) * float64(totalCount)
	}
	p.allHeightsMeasured = allMeasured
	if allMeasured {
		p.accurateTotalHeightInPixels = sum
		p.accurateHeightAboveTopInPixels = heightAboveTop
	} else {
		p.accurateTotalHeightInPixels = 0
		p.accurateHeightAboveTopInPixels = 0
	}
}

func (p *virtualScrollPanel) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
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
func (p *virtualScrollPanel) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if !widgetBounds.IsHitAtCursor() {
		p.lastWheelX = 0
		p.lastWheelY = 0
		return guigui.HandleInputResult{}
	}

	wheelX, wheelY := adjustedWheel()
	if wheelX == 0 && wheelY == 0 {
		p.lastWheelX = 0
		p.lastWheelY = 0
		return guigui.HandleInputResult{}
	}

	bounds := widgetBounds.Bounds()
	dx := wheelX * scrollWheelSpeed(context)
	dy := wheelY * scrollWheelSpeed(context)

	// Horizontal scroll is pixel-based; clamp against the content width as
	// Layout does to learn whether the offset would move.
	cw := p.content.contentWidth(context)
	if cw == 0 {
		cw = bounds.Dx()
	}
	minX := float64(min(bounds.Dx()-cw, 0))
	scrollX := dx != 0 && min(max(p.offsetX+dx, minX), 0) != p.offsetX

	// Vertical scroll is item-based. A positive dy scrolls toward the top, a
	// negative dy toward the bottom.
	var scrollY bool
	switch {
	case dy > 0:
		scrollY = p.topItemIndex > 0 || p.topItemOffset < 0
	case dy < 0:
		scrollY = !p.atBottom
	}

	// Report scrolling only on an axis that actually moved, so a stale axis
	// does not keep its scroll bar visible.
	if scrollX {
		p.lastWheelX = wheelX
	} else {
		p.lastWheelX = 0
		dx = 0
	}
	if scrollY {
		p.lastWheelY = wheelY
	} else {
		p.lastWheelY = 0
		dy = 0
	}

	if !scrollX && !scrollY {
		// Already at the scroll limit in the wheel's direction. Don't consume
		// the event, so it can fall through to a scrollable widget behind this
		// one.
		return guigui.HandleInputResult{}
	}

	p.forceSetScrollOffsetByDelta(dx, dy)
	return guigui.HandleInputByWidget(p)
}

func (p *virtualScrollPanel) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	p.onceDraw = true
}

func (p *virtualScrollPanel) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	p.applyPendingScrollOffset()

	bounds := widgetBounds.Bounds()

	// The content's layout handles clamping and normalization of
	// topItemIndex/topItemOffset, so we don't need to do it here.

	// Compute horizontal content size for scroll bar.
	cw := p.content.contentWidth(context)
	if cw == 0 {
		cw = bounds.Dx()
	}

	// Adjust horizontal offset.
	maxOffsetX := float64(min(bounds.Dx()-cw, 0))
	p.offsetX = min(max(p.offsetX, maxOffsetX), 0)

	// Layout the content widget at the panel bounds with the horizontal offset.
	// The content uses topItemIndex/topItemOffset to position items.
	pt := bounds.Min.Add(image.Pt(int(p.offsetX), 0))
	contentSize := image.Pt(cw, bounds.Dy())
	layouter.LayoutWidget(p.content, image.Rectangle{
		Min: pt,
		Max: pt.Add(contentSize),
	})
	p.updateHeightMetrics(context, bounds)

	// Set content size for horizontal scroll bar only.
	hContentSize := image.Pt(cw, bounds.Dy())
	p.scrollHBar.setContentSize(hContentSize)

	layouter.LayoutWidget(&p.scrollHBar, p.horizontalBarBounds(context, widgetBounds))
	p.scrollVBar.setPanelBounds(bounds)
	layouter.LayoutWidget(&p.scrollVBar, p.verticalBarBounds(context, widgetBounds))

	hb, vb := p.thumbBounds(context, widgetBounds)
	p.scrollHBar.setThumbBounds(hb)
	p.scrollVBar.setThumbBounds(vb)
}

func (p *virtualScrollPanel) applyPendingScrollOffset() {
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

func (p *virtualScrollPanel) horizontalBarBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	bounds := widgetBounds.Bounds()
	bounds.Min.Y = max(bounds.Min.Y, bounds.Max.Y-scrollBarAreaSize(context))
	return bounds
}

func (p *virtualScrollPanel) verticalBarBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	bounds := widgetBounds.Bounds()
	bounds.Min.X = max(bounds.Min.X, bounds.Max.X-scrollBarAreaSize(context))
	return bounds
}

func (p *virtualScrollPanel) isScrollingX() bool {
	return p.lastWheelX != 0
}

func (p *virtualScrollPanel) isScrollingY() bool {
	return p.lastWheelY != 0
}

func (p *virtualScrollPanel) isHBarVisible(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
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

func (p *virtualScrollPanel) isVBarVisible(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
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

func (p *virtualScrollPanel) startShowingHBarIfNeeded(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
	if hb, _ := p.thumbBounds(context, widgetBounds); hb.Empty() {
		return
	}
	p.scrollHBarCount = startShowingBarCount(p.scrollHBarCount)
}

func (p *virtualScrollPanel) startShowingVBarIfNeeded(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
	if _, vb := p.thumbBounds(context, widgetBounds); vb.Empty() {
		return
	}
	p.scrollVBarCount = startShowingBarCount(p.scrollVBarCount)
}

func (p *virtualScrollPanel) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
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
	p.dispatchScrollEventIfNeeded()
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

// advanceScrollAnimation advances the vertical scroll animation by one tick and
// reports whether an animation was in flight. Each tick applies the eased
// increment of vAnimDelta to topItemOffset only; topItemIndex is updated
// between ticks by the content's normalization using real measured heights. The
// final tick snaps (topItemIndex, topItemOffset) to the exact target.
func (p *virtualScrollPanel) advanceScrollAnimation() bool {
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
func (p *virtualScrollPanel) applyPendingScrollOffsetInTick() (bool, bool) {
	if !p.nextOffsetXSet && !p.nextTopItemSet {
		return false, false
	}

	oldOffsetX := p.offsetX
	oldTopItemIndex := p.topItemIndex
	oldTopItemOffset := p.topItemOffset

	p.applyPendingScrollOffset()

	// topItemIndex/topItemOffset/offsetX are in WriteStateKey,
	// so the rebuild that re-invokes Layout is triggered automatically.
	hChanged := p.offsetX != oldOffsetX
	vChanged := p.topItemIndex != oldTopItemIndex || p.topItemOffset != oldTopItemOffset
	return hChanged, vChanged
}

// dispatchScrollEventIfNeeded emits virtualScrollPanelEventScroll when the
// scroll position has changed since the last dispatch. Comparing against a
// stored snapshot catches every write path, including direct writes such as a
// scroll-bar drag that does not set a pending change.
func (p *virtualScrollPanel) dispatchScrollEventIfNeeded() {
	if p.topItemIndex == p.scrolledTopItemIndex &&
		p.topItemOffset == p.scrolledTopItemOffset &&
		p.offsetX == p.scrolledOffsetX {
		return
	}
	p.scrolledTopItemIndex = p.topItemIndex
	p.scrolledTopItemOffset = p.topItemOffset
	p.scrolledOffsetX = p.offsetX
	guigui.DispatchEvent(p, virtualScrollPanelEventScroll, p.topItemIndex, p.topItemOffset, p.offsetX)
}

// vThumbHeight returns the vertical thumb height.
// Returns 0 if no items have been measured yet or no thumb should be shown.
func (p *virtualScrollPanel) vThumbHeight(context *guigui.Context, panelBounds image.Rectangle, totalCount int) float64 {
	if totalCount == 0 {
		return 0
	}
	padding := scrollThumbPadding(context)
	trackLength := float64(panelBounds.Dy()) - 2*padding
	// Size the thumb proportionally to the estimated total content height. It
	// equals the exact total whenever the sample window covers every item, so
	// the size stays continuous across the boundary where positioning switches
	// modes.
	totalHeight := p.estimatedTotalHeightInPixels
	if totalHeight <= float64(panelBounds.Dy()) {
		return 0
	}
	barHeight := trackLength * float64(panelBounds.Dy()) / totalHeight
	return max(barHeight, scrollThumbMinSize(context, trackLength))
}

// bottomFracIdx returns the fracIdx reached when the last item's bottom
// aligns with the viewport bottom.
func (p *virtualScrollPanel) bottomFracIdx(context *guigui.Context, viewportHeight int) float64 {
	totalCount := p.content.itemCount()
	measure := func(i int) int {
		return p.content.measureItemHeight(context, i)
	}
	return bottomFracIdx(measure, totalCount, viewportHeight-p.content.viewportPaddingY(context))
}

// bottomFracIdx is the free-function core of
// [virtualScrollPanel.bottomFracIdx]. measure must return -1 for out-of-range
// indices and a non-negative height otherwise.
func bottomFracIdx(measure func(index int) int, totalCount, viewportHeight int) float64 {
	if totalCount == 0 || viewportHeight <= 0 {
		return 0
	}
	var accum int
	idx := totalCount - 1
	var h int
	for idx >= 0 {
		h = measure(idx)
		if h < 0 {
			return 0
		}
		accum += h
		if accum >= viewportHeight {
			break
		}
		idx--
	}
	if accum < viewportHeight {
		return 0
	}
	if h <= 0 {
		return float64(idx)
	}
	topOff := viewportHeight - accum
	return float64(idx) + float64(-topOff)/float64(h)
}

func (p *virtualScrollPanel) thumbBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (image.Rectangle, image.Rectangle) {
	bounds := widgetBounds.Bounds()
	padding := scrollThumbPadding(context)

	var horizontalBarBounds, verticalBarBounds image.Rectangle

	// Horizontal thumb.
	if cw := p.content.contentWidth(context); !p.horizontalBarHidden && cw > bounds.Dx() {
		trackLength := float64(bounds.Dx()) - 2*padding
		barWidth := trackLength * float64(bounds.Dx()) / float64(cw)
		barWidth = max(barWidth, scrollThumbMinSize(context, trackLength))

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

	// Vertical thumb — position based on the document-space scroll position.
	totalCount := p.content.itemCount()
	if barHeight := p.vThumbHeight(context, bounds, totalCount); barHeight > 0 {
		// barHeight > 0 guarantees totalCount > 0; when not allHeightsMeasured
		// it also guarantees estimatedItemHeight > 0 (see vThumbHeight).
		var rate float64
		if p.allHeightsMeasured {
			// Pixel-accurate mapping: every wheel pixel moves the thumb
			// the same track distance, regardless of item heterogeneity.
			viewportEff := bounds.Dy() - p.content.viewportPaddingY(context)
			maxScrollPos := p.accurateTotalHeightInPixels - viewportEff
			if maxScrollPos > 0 {
				scrollPos := p.accurateHeightAboveTopInPixels - p.topItemOffset
				rate = min(max(float64(scrollPos)/float64(maxScrollPos), 0), 1)
			}
		} else {
			// fracIdx is the top item index plus the fraction of that item
			// scrolled off the top, using the item's measured height. This
			// stays continuous across the (topItemIndex+1, topItemOffset+h_top)
			// normalization, avoiding the (estH - h_actual) jump per boundary
			// that a pixel-based formula would have on heterogeneous content.
			fracIdx := float64(p.topItemIndex)
			if h := p.content.measureItemHeight(context, p.topItemIndex); h > 0 {
				fracIdx += float64(-p.topItemOffset) / float64(h)
			}
			maxFracIdx := p.bottomFracIdx(context, bounds.Dy())
			if maxFracIdx > 0 {
				rate = min(max(fracIdx/maxFracIdx, 0), 1)
			}
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

// virtualScrollVBar is a child widget that draws and handles input for the
// vertical scroll bar of a [virtualScrollPanel].
type virtualScrollVBar struct {
	guigui.DefaultWidget

	panel       *virtualScrollPanel
	thumbBounds image.Rectangle
	alpha       float64

	// panelBoundsRect is the parent panel's bounds, captured by
	// [virtualScrollPanel.Layout]. It is unset until Layout has run.
	panelBoundsRect image.Rectangle

	dragging              bool
	draggingStartPosition int
	draggingStartIndex    int
	draggingStartOffset   int
	onceDraw              bool
}

func (s *virtualScrollVBar) setPanelBounds(rect image.Rectangle) {
	s.panelBoundsRect = rect
}

func (s *virtualScrollVBar) setThumbBounds(bounds image.Rectangle) {
	if s.thumbBounds == bounds {
		return
	}
	s.thumbBounds = bounds
	guigui.RequestRedraw(s)
}

func (s *virtualScrollVBar) setAlpha(alpha float64) {
	if s.alpha == alpha {
		return
	}
	s.alpha = alpha
	if !s.thumbBounds.Empty() {
		guigui.RequestRedraw(s)
	}
}

func (s *virtualScrollVBar) isDragging() bool {
	return s.dragging
}

func (s *virtualScrollVBar) isOnceDrawn() bool {
	return s.onceDraw
}

func (s *virtualScrollVBar) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	totalCount := s.panel.content.itemCount()
	if totalCount == 0 {
		return guigui.HandleInputResult{}
	}

	bounds := widgetBounds.Bounds()
	padding := scrollThumbPadding(context)
	// barHeight > 0 guarantees estimatedItemHeight > 0 (see vThumbHeight),
	// so divisions by estimatedItemHeight below are safe.
	barHeight := s.panel.vThumbHeight(context, s.panelBoundsRect, totalCount)
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
			// Clicked on track — jump by one viewport in pixels.
			if !tb.Empty() {
				deltaPx := bounds.Dy()
				if y < tb.Min.Y {
					deltaPx = -deltaPx
				}
				measure := func(i int) int {
					h := s.panel.content.measureItemHeight(context, i)
					if h < 0 {
						return s.panel.estimatedItemHeight
					}
					return h
				}
				newIdx, newOff := topItemAfterPixelScroll(measure, totalCount, topIdx, topOff, deltaPx)
				s.panel.setTopItem(newIdx, newOff)
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
			if s.panel.allHeightsMeasured {
				// Map cursor delta to a pixel-space scroll delta and walk
				// item heights to recover (idx, off). The drag-start scroll
				// position is recomputed each tick from draggingStartIndex
				// rather than captured at drag start; the mode can flip if
				// itemCount crosses the full-measure threshold mid-drag, but
				// allHeightsMeasured implies totalCount is below it so the
				// walk is cheap.
				measure := func(i int) int {
					return s.panel.content.measureItemHeight(context, i)
				}
				var startScroll int
				for i := 0; i < s.draggingStartIndex; i++ {
					if h := measure(i); h > 0 {
						startScroll += h
					}
				}
				startScroll -= s.draggingStartOffset
				viewportEff := s.panelBoundsRect.Dy() - s.panel.content.viewportPaddingY(context)
				maxScrollPos := s.panel.accurateTotalHeightInPixels - viewportEff
				if maxScrollPos > 0 {
					newScrollPos := min(max(startScroll+int(float64(dy)*float64(maxScrollPos)/trackHeight), 0), maxScrollPos)
					newIdx, newOff := accurateTopItemFromScrollPos(measure, totalCount, newScrollPos)
					s.panel.forceSetTopItem(newIdx, newOff, true)
				}
			} else {
				// Map cursor drag to a fractional-index delta, the inverse of
				// the forward formula in [virtualScrollPanel.thumbBounds]:
				// the full track length corresponds to maxFracIdx items.
				maxFracIdx := s.panel.bottomFracIdx(context, s.panelBoundsRect.Dy())
				if maxFracIdx > 0 {
					startFrac := float64(s.draggingStartIndex)
					if h := s.panel.content.measureItemHeight(context, s.draggingStartIndex); h > 0 {
						startFrac += float64(-s.draggingStartOffset) / float64(h)
					}
					newFrac := startFrac + float64(dy)*maxFracIdx/trackHeight
					if newFrac < 0 {
						newFrac = 0
					}
					if newFrac > maxFracIdx {
						newFrac = maxFracIdx
					}
					newIdx := int(newFrac)
					if newIdx >= totalCount {
						newIdx = totalCount - 1
					}
					frac := newFrac - float64(newIdx)
					var newOff int
					if h := s.panel.content.measureItemHeight(context, newIdx); h > 0 {
						newOff = -int(frac * float64(h))
					}
					s.panel.forceSetTopItem(newIdx, newOff, true)
				}
			}
		}
		return guigui.HandleInputByWidget(s)
	}

	if s.dragging && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		s.dragging = false
	}

	return guigui.HandleInputResult{}
}

func (s *virtualScrollVBar) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	return ebiten.CursorShapeDefault, true
}

func (s *virtualScrollVBar) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
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

// accurateTopItemFromScrollPos returns (topIndex, topOffset) for an absolute
// pixel scroll position scrollPos, the distance from the top of item 0 to the
// top of the viewport. measure must return a non-negative height for every
// in-range index. The returned offset follows
// [virtualScrollPanel.topItemOffset]'s convention.
func accurateTopItemFromScrollPos(measure func(index int) int, totalCount, scrollPos int) (newIndex, newOffset int) {
	if totalCount == 0 || scrollPos <= 0 {
		return 0, 0
	}
	rem := scrollPos
	var idx int
	for idx < totalCount-1 {
		h := measure(idx)
		if h < 0 {
			return idx, 0
		}
		if rem < h {
			return idx, -rem
		}
		rem -= h
		idx++
	}
	return idx, -rem
}

// topItemAfterPixelScroll returns (topIndex, topOffset) after scrolling by
// deltaPx (positive = forward) from (startIndex, startOffset). The returned
// offset follows [virtualScrollPanel.topItemOffset]'s convention; measure must
// return a non-negative height for each in-range index. The returned index
// stays within [0, totalCount-1].
func topItemAfterPixelScroll(measure func(index int) int, totalCount, startIndex, startOffset, deltaPx int) (newIndex, newOffset int) {
	newIndex = startIndex
	newOffset = startOffset - deltaPx
	if deltaPx > 0 {
		// Scrolling forward: advance newIndex while newOffset is more
		// negative than the current item's height.
		for newIndex < totalCount-1 {
			h := measure(newIndex)
			if -newOffset < h {
				break
			}
			newOffset += h
			newIndex++
		}
	} else if deltaPx < 0 {
		// Scrolling backward: retreat newIndex while newOffset is positive.
		for newOffset > 0 && newIndex > 0 {
			newIndex--
			h := measure(newIndex)
			newOffset -= h
		}
		if newIndex == 0 && newOffset > 0 {
			newOffset = 0
		}
	}
	return
}
