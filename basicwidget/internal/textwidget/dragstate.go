// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

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

	// clickCounter distinguishes double- and triple-clicks from repeated
	// single clicks.
	clickCounter clickCounter
}

// reset ends any drag and clears the anchor range.
func (d *textDragState) reset() {
	d.dragging = false
	d.startPlus1 = 0
	d.endPlus1 = 0
}
