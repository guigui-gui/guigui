// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"image"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
)

var (
	textEventHotspotDown guigui.EventKey = guigui.GenerateEventKey()
	textEventHotspotUp   guigui.EventKey = guigui.GenerateEventKey()
)

// TextRange is a byte range of a text value.
type TextRange struct {
	// StartInBytes is the inclusive start of the range in bytes.
	StartInBytes int

	// EndInBytes is the exclusive end of the range in bytes.
	EndInBytes int
}

// SetHotspotRanges sets the hotspot ranges: over their rectangles the cursor
// turns into a pointer, and mouse presses and releases fire the hotspot down
// and up events.
func (t *Text) SetHotspotRanges(ranges []TextRange) {
	t.hotspotRanges = append(t.hotspotRanges[:0], ranges...)
}

// OnHotspotDown sets the event handler that is called when the left mouse
// button is pressed on a hotspot range. The handler is given the pressed
// range.
func (t *Text) OnHotspotDown(f func(context *guigui.Context, textRange TextRange)) {
	guigui.SetEventHandler(t, textEventHotspotDown, f)
}

// OnHotspotUp sets the event handler that is called when the left mouse
// button is released on the hotspot range it was pressed on. For selectable
// or editable text, the click is canceled when the cursor moved or a
// selection was made between the press and the release: a drag selects, it
// does not click. The handler is given the released range.
func (t *Text) OnHotspotUp(f func(context *guigui.Context, textRange TextRange)) {
	guigui.SetEventHandler(t, textEventHotspotUp, f)
}

// hotspotRangeAt returns the hotspot range whose rectangles contain
// position, and whether one exists. IsHitAt covers visibility, clipping, and
// obscuring by higher-layer widgets such as popups.
func (t *Text) hotspotRangeAt(context *guigui.Context, widgetBounds *guigui.WidgetBounds, position image.Point) (TextRange, bool) {
	if len(t.hotspotRanges) == 0 {
		return TextRange{}, false
	}
	if !widgetBounds.IsHitAt(position) {
		return TextRange{}, false
	}
	defer func() {
		t.hotspotBoundsBuf = slices.Delete(t.hotspotBoundsBuf, 0, len(t.hotspotBoundsBuf))
	}()
	for _, r := range t.hotspotRanges {
		t.hotspotBoundsBuf = t.AppendBoundsOfTextRange(t.hotspotBoundsBuf[:0], context, widgetBounds.Bounds(), r.StartInBytes, r.EndInBytes)
		if slices.ContainsFunc(t.hotspotBoundsBuf, position.In) {
			return r, true
		}
	}
	return TextRange{}, false
}

// handleHotspotPointingInput fires the hotspot down and up events for the
// pointer state at cursorPosition: down on a press on a hotspot range, up on
// a release on the same range with no selection in between. The result is
// non-empty when an event fired.
func (t *Text) handleHotspotPointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds, cursorPosition image.Point) guigui.HandleInputResult {
	if len(t.hotspotRanges) == 0 {
		return guigui.HandleInputResult{}
	}
	// IsMouseButtonJustPressed and IsMouseButtonJustReleased can be true at the same time as of Ebitengine v2.9.
	// Check both.
	var fired bool
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if r, ok := t.hotspotRangeAt(context, widgetBounds, cursorPosition); ok {
			t.hotspotPressed = true
			t.pressedHotspotRange = r
			guigui.DispatchEvent(t, textEventHotspotDown, r)
			fired = true
		}
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && t.hotspotPressed {
		t.hotspotPressed = false
		if r, ok := t.hotspotRangeAt(context, widgetBounds, cursorPosition); ok && r == t.pressedHotspotRange && !t.dragState.moved(cursorPosition) {
			if start, end := t.store.Selection(); start == end {
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
