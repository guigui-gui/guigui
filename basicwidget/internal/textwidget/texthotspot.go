// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"image"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/piecetable"
)

var (
	textEventHotspotDown guigui.EventKey = guigui.GenerateEventKey()
	textEventHotspotUp   guigui.EventKey = guigui.GenerateEventKey()
)

// TextRange is a byte range of a text value.
type TextRange = piecetable.TextRange

// SetHotspotRanges sets the hotspot ranges: over their rectangles the cursor
// turns into a pointer, and mouse presses and releases fire the hotspot down
// and up events. The ranges follow the text through edits like the ranged
// styles. While the value is editable, the hotspots are inert.
func (t *Text) SetHotspotRanges(ranges []TextRange) {
	t.hotspotRanges = append(t.hotspotRanges[:0], ranges...)
	t.hotspotRangesValidGeneration = t.store.Generation()
}

// AppendHotspotRanges appends the current hotspot ranges to dst and returns
// the extended slice, reflecting the adjustments made for edits since
// [Text.SetHotspotRanges].
func (t *Text) AppendHotspotRanges(dst []TextRange) []TextRange {
	return append(dst, t.ensureHotspotRanges()...)
}

// ensureHotspotRanges brings the hotspot ranges up to date with the store's
// content and returns them. Positional edits since the last call are
// replayed so the ranges keep covering the same text; mutations without a
// positional record (whole-value replacements) clear the ranges. Undo and
// redo reinstall the ranges from their history snapshots via
// [Text.restoreRangedState] instead.
func (t *Text) ensureHotspotRanges() []TextRange {
	gen := t.store.Generation()
	if t.hotspotRangesValidGeneration == gen {
		return t.hotspotRanges
	}
	defer func() {
		t.textEditsBuf = slices.Delete(t.textEditsBuf, 0, len(t.textEditsBuf))
	}()
	var covered bool
	t.textEditsBuf, covered = t.store.appendEditsSince(t.textEditsBuf, t.hotspotRangesValidGeneration)
	if covered {
		for _, e := range t.textEditsBuf {
			t.hotspotRanges = replaceTextRanges(t.hotspotRanges, e.start, e.end, e.newLen)
		}
	} else {
		t.hotspotRanges = slices.Delete(t.hotspotRanges, 0, len(t.hotspotRanges))
	}
	t.hotspotRangesValidGeneration = gen
	return t.hotspotRanges
}

// replaceTextRanges adjusts ranges for a replacement of the byte range
// [start, end) with newLen bytes of text, with the same movement rules as
// [textstyle.Runs.Replace]: ranges keep covering the text around the
// replacement, an insertion inside a range or at its end extends the range,
// and replacing text starting inside a range stays part of it. A range whose
// text is fully replaced is removed.
func replaceTextRanges(ranges []TextRange, start, end, newLen int) []TextRange {
	result := ranges[:0]
	for _, r := range ranges {
		if adjusted, ok := replaceTextRange(r, start, end, newLen); ok {
			result = append(result, adjusted)
		}
	}
	return result
}

// replaceTextRange adjusts one range for a replacement of the byte range
// [start, end) with newLen bytes of text, with the movement rules of
// [replaceTextRanges]. ok is false when the range's text is fully replaced.
func replaceTextRange(r TextRange, start, end, newLen int) (adjusted TextRange, ok bool) {
	start = max(start, 0)
	end = max(end, start)
	newLen = max(newLen, 0)
	if start == end && newLen == 0 {
		return r, true
	}
	delta := newLen - (end - start)

	switch {
	case start == end && r.StartInBytes < start && r.EndInBytes >= start:
		r.EndInBytes += delta
		return r, true
	case r.EndInBytes <= start:
		return r, true
	case r.StartInBytes >= end:
		r.StartInBytes += delta
		r.EndInBytes += delta
		return r, true
	default:
		adjusted := TextRange{
			StartInBytes: end + delta,
			EndInBytes:   end + delta,
		}
		if r.StartInBytes <= start {
			adjusted.StartInBytes = r.StartInBytes
			adjusted.EndInBytes = start + newLen
		}
		if r.EndInBytes > end {
			adjusted.EndInBytes = r.EndInBytes + delta
		}
		return adjusted, adjusted.StartInBytes < adjusted.EndInBytes
	}
}

// OnHotspotDown sets the event handler that is called when the left mouse
// button is pressed on a hotspot range. The handler is given the pressed
// range.
func (t *Text) OnHotspotDown(f func(context *guigui.Context, textRange TextRange)) {
	guigui.SetEventHandler(t, textEventHotspotDown, f)
}

// OnHotspotUp sets the event handler that is called when the left mouse
// button is released on the hotspot range it was pressed on. For selectable
// text, the click is canceled when the cursor moved or a selection was made
// between the press and the release: a drag selects, it does not click. The
// handler is given the released range.
func (t *Text) OnHotspotUp(f func(context *guigui.Context, textRange TextRange)) {
	guigui.SetEventHandler(t, textEventHotspotUp, f)
}

// hotspotRangeAt returns the hotspot range whose rectangles contain
// position, and whether one exists. An editable value has no active
// hotspots. IsHitAt covers visibility, clipping, and obscuring by
// higher-layer widgets such as popups.
func (t *Text) hotspotRangeAt(context *guigui.Context, widgetBounds *guigui.WidgetBounds, position image.Point) (TextRange, bool) {
	if t.editable {
		return TextRange{}, false
	}
	ranges := t.ensureHotspotRanges()
	if len(ranges) == 0 {
		return TextRange{}, false
	}
	if !widgetBounds.IsHitAt(position) {
		return TextRange{}, false
	}
	defer func() {
		t.hotspotBoundsBuf = slices.Delete(t.hotspotBoundsBuf, 0, len(t.hotspotBoundsBuf))
	}()
	for _, r := range ranges {
		t.hotspotBoundsBuf = t.AppendBoundsOfTextRange(t.hotspotBoundsBuf[:0], context, widgetBounds.Bounds(), r.StartInBytes, r.EndInBytes)
		if slices.ContainsFunc(t.hotspotBoundsBuf, position.In) {
			return r, true
		}
	}
	return TextRange{}, false
}

// handleHotspotPointingInput fires the hotspot down and up events for the
// pointer state at cursorPosition: down on a press on a hotspot range, up on
// a release on the same range with no selection made during the press. The
// result is non-empty when an event fired.
func (t *Text) handleHotspotPointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds, cursorPosition image.Point) guigui.HandleInputResult {
	if t.editable || len(t.ensureHotspotRanges()) == 0 {
		return guigui.HandleInputResult{}
	}
	// IsMouseButtonJustPressed and IsMouseButtonJustReleased can be true at the same time as of Ebitengine v2.9.
	// Check both.
	var fired bool
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if r, ok := t.hotspotRangeAt(context, widgetBounds, cursorPosition); ok {
			t.hotspotPressed = true
			t.pressedHotspotRange = r
			t.hotspotPressSelectionStart, t.hotspotPressSelectionEnd = t.store.Selection()
			guigui.DispatchEvent(t, textEventHotspotDown, r)
			fired = true
		}
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && t.hotspotPressed {
		t.hotspotPressed = false
		if r, ok := t.hotspotRangeAt(context, widgetBounds, cursorPosition); ok && r == t.pressedHotspotRange && !t.dragState.moved(cursorPosition) {
			// A selection made during the press cancels the click: a drag
			// selects, it does not click. A selection predating the press
			// (possibly left invisible by an earlier editable state) does
			// not.
			start, end := t.store.Selection()
			if start == end || start == t.hotspotPressSelectionStart && end == t.hotspotPressSelectionEnd {
				guigui.DispatchEvent(t, textEventHotspotUp, r)
				fired = true
			}
		}
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		t.hotspotPressed = false
	}
	if fired {
		return guigui.HandleInputByWidget(t)
	}
	return guigui.HandleInputResult{}
}
