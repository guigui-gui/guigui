// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"image"
)

// textDragState tracks an in-progress mouse selection drag on a [Text].
type textDragState struct {
	// dragging reports whether a selection drag is in progress.
	dragging bool

	// startPlus1 is the drag anchor range's start byte offset plus 1. The
	// zero value means no anchor.
	startPlus1 int

	// endPlus1 is the drag anchor range's end byte offset plus 1. The zero
	// value means no anchor.
	endPlus1 int

	// pressPosition is the cursor position at the press that started the
	// drag. It is meaningful only while dragging is true.
	pressPosition image.Point

	// cursorMoved reports whether the cursor has left pressPosition at some
	// point during the drag.
	cursorMoved bool

	// clickCounter distinguishes double- and triple-clicks from repeated
	// single clicks.
	clickCounter clickCounter
}

// start begins a drag at the pressed cursor position, anchored at the byte
// range [anchorStart, anchorEnd). A negative offset means no anchor.
func (d *textDragState) start(pressPosition image.Point, anchorStart, anchorEnd int) {
	d.dragging = true
	d.pressPosition = pressPosition
	d.cursorMoved = false
	d.startPlus1 = anchorStart + 1
	d.endPlus1 = anchorEnd + 1
}

// isDragging reports whether a selection drag is in progress.
func (d *textDragState) isDragging() bool {
	return d.dragging
}

// trackCursorMovement records that the cursor has left the pressed position
// when cursorPosition differs from it during a drag.
func (d *textDragState) trackCursorMovement(cursorPosition image.Point) {
	if d.dragging && cursorPosition != d.pressPosition {
		d.cursorMoved = true
	}
}

// extendedSelection returns the anchor range extended to include the text
// index idx.
func (d *textDragState) extendedSelection(idx int) (start, end int) {
	start, end = idx, idx
	if d.startPlus1-1 >= 0 {
		start = min(start, d.startPlus1-1)
	}
	if d.endPlus1-1 >= 0 {
		end = max(end, d.endPlus1-1)
	}
	return start, end
}

// clearAnchor clears the anchor range.
func (d *textDragState) clearAnchor() {
	d.startPlus1 = 0
	d.endPlus1 = 0
}

// moved reports whether a drag is in progress and the cursor has moved from
// the pressed position, either earlier in the drag or to cursorPosition now.
func (d *textDragState) moved(cursorPosition image.Point) bool {
	if !d.dragging {
		return false
	}
	return d.cursorMoved || cursorPosition != d.pressPosition
}

// click records a click at textIndex on tick and returns the updated
// consecutive-click count.
func (d *textDragState) click(tick int64, textIndex int, leftClick bool) int {
	return d.clickCounter.click(tick, textIndex, leftClick)
}

// resetClickCount clears the consecutive-click count while recording a click
// at textIndex on tick, so a following click is not treated as a double- or
// triple-click.
func (d *textDragState) resetClickCount(tick int64, textIndex int) {
	d.clickCounter.reset(tick, textIndex)
}

// reset ends any drag and clears the anchor range.
func (d *textDragState) reset() {
	d.dragging = false
	d.startPlus1 = 0
	d.endPlus1 = 0
	d.pressPosition = image.Point{}
	d.cursorMoved = false
}
