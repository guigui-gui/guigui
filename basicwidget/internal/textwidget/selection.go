// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

// SelectionSide identifies one endpoint of a selection.
type SelectionSide int

const (
	SelectionSideNone SelectionSide = iota
	SelectionSideStart
	SelectionSideEnd
)

// shiftClickAnchor returns the byte offset of the selection [start, end]
// (start <= end) that stays fixed when a Shift+click at idx extends it; the
// opposite end becomes the new moving end. shiftSide is the endpoint currently
// moved by Shift, or [SelectionSideNone] when none is tracked.
func shiftClickAnchor(start, end int, shiftSide SelectionSide, idx int) int {
	switch {
	case start == end:
		// A bare caret becomes the anchor.
		return start
	case shiftSide == SelectionSideStart:
		return end
	case shiftSide == SelectionSideEnd:
		return start
	default:
		// A selection without a tracked moving end (e.g. a word or select-all
		// selection): keep the endpoint farther from the click as the anchor so
		// the selection extends toward the click.
		ds := idx - start
		if ds < 0 {
			ds = -ds
		}
		de := end - idx
		if de < 0 {
			de = -de
		}
		if ds >= de {
			return start
		}
		return end
	}
}
