// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"image"
	"log/slog"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

// findWordBoundaries returns the byte range of the word containing idx,
// scanning only the logical line containing idx. Word-segmentation rules
// always break at line breaks (UAX #29 WB3a/3b), so a word never spans
// logical lines.
func (t *Text) findWordBoundaries(idx int) (start, end int) {
	line, lineStart := t.stringValueForLineContaining(idx)
	s, e := textutil.FindWordBoundaries(line, idx-lineStart)
	return s + lineStart, e + lineStart
}

// findWordBoundariesForDoubleClick returns the byte range a double click at idx
// selects: the line break at idx when the logical line ends there, and the word
// containing idx otherwise.
func (t *Text) findWordBoundariesForDoubleClick(idx int) (start, end int) {
	line, lineStart := t.stringValueForLineContaining(idx)
	// A double click resolves to the offset at the line's end wherever it
	// lands in the space after the line, so the break starting there is what
	// the click points at.
	if pos, l := textutil.FirstLineBreakPositionAndLen(line[idx-lineStart:]); pos == 0 {
		return idx, idx + l
	}
	s, e := textutil.FindWordBoundaries(line, idx-lineStart)
	return s + lineStart, e + lineStart
}

// prevPositionOnGraphemes returns the byte offset of the grapheme cluster
// boundary that immediately precedes position. Grapheme breaks always
// exist around line-break characters (UAX #29 GB4/GB5), so the previous
// boundary is always inside the logical line containing position-1.
func (t *Text) prevPositionOnGraphemes(position int) int {
	if position <= 0 {
		return position
	}
	line, lineStart := t.stringValueForLineContaining(position - 1)
	return lineStart + textutil.PrevPositionOnGraphemes(line, position-lineStart)
}

// nextPositionOnGraphemes returns the byte offset of the grapheme cluster
// boundary that immediately follows position. The next boundary is always
// inside the logical line containing position (cf. prevPositionOnGraphemes).
func (t *Text) nextPositionOnGraphemes(position int) int {
	if position >= t.store.TextLengthInBytes() {
		return position
	}
	line, lineStart := t.stringValueForLineContaining(position)
	return lineStart + textutil.NextPositionOnGraphemes(line, position-lineStart)
}

// prevWordStart returns the byte offset of the start of the last word before
// position, or 0 when no earlier word exists.
func (t *Text) prevWordStart(position int) int {
	// Step back over graphemes until the one just before position lies in a
	// word; [Text.findWordBoundaries] then yields that word's start. Both
	// helpers scan only the relevant logical line, so this crosses line breaks
	// without materializing the document.
	for position > 0 {
		prev := t.prevPositionOnGraphemes(position)
		if s, e := t.findWordBoundaries(prev); e > s {
			return s
		}
		position = prev
	}
	return 0
}

// nextWordEnd returns the byte offset of the end of the first word at or after
// position, or the text length when no further word exists.
func (t *Text) nextWordEnd(position int) int {
	// Step forward over graphemes until position lies in a word;
	// [Text.findWordBoundaries] then yields that word's end. Both helpers scan
	// only the relevant logical line, so this crosses line breaks without
	// materializing the document.
	total := t.store.TextLengthInBytes()
	for position < total {
		if _, e := t.findWordBoundaries(position); e > position {
			return e
		}
		next := t.nextPositionOnGraphemes(position)
		if next <= position {
			break
		}
		position = next
	}
	return total
}

// paragraphStart returns the byte offset of the beginning of the logical line
// containing position, or of the previous logical line when position is
// already at a line start.
func (t *Text) paragraphStart(position int) int {
	lineIndex := t.LineIndexFromTextIndexInBytes(position)
	lineStart := t.LineStartInBytes(lineIndex)
	if position > lineStart {
		return lineStart
	}
	if lineIndex > 0 {
		return t.LineStartInBytes(lineIndex - 1)
	}
	return 0
}

// paragraphEnd returns the byte offset of the end of the logical line
// containing position, excluding its trailing line break, or of the next
// logical line when position is already at a line end.
func (t *Text) paragraphEnd(position int) int {
	lineIndex := t.LineIndexFromTextIndexInBytes(position)
	lineEnd := t.logicalLineContentEnd(lineIndex)
	if position < lineEnd {
		return lineEnd
	}
	if lineIndex+1 < t.LineCount() {
		return t.logicalLineContentEnd(lineIndex + 1)
	}
	return t.store.TextLengthInBytes()
}

// logicalLineContentEnd returns the byte offset of the end of the lineIndex-th
// logical line's content, excluding its trailing line break.
func (t *Text) logicalLineContentEnd(lineIndex int) int {
	if lineIndex+1 >= t.LineCount() {
		return t.store.TextLengthInBytes()
	}
	lineEnd := t.LineStartInBytes(lineIndex + 1)
	lineStart := t.LineStartInBytes(lineIndex)
	// The trailing line break is at most a few bytes, so inspect a short
	// suffix rather than materializing the whole line.
	suffix := t.stringValueWithRange(max(lineStart, lineEnd-4), lineEnd)
	if i, l := textutil.LastLineBreakPositionAndLen(suffix); i >= 0 && i+l == len(suffix) {
		return lineEnd - l
	}
	return lineEnd
}

// nextWordStart returns the byte offset of the start of the next word after
// position, or the text length when no later word exists.
func (t *Text) nextWordStart(position int) int {
	total := t.store.TextLengthInBytes()
	// Skip past the word under the caret, then to the first following word
	// start. findWordBoundaries and the grapheme steppers scan only the
	// relevant logical line, so this crosses line breaks without materializing
	// the document.
	if _, e := t.findWordBoundaries(position); e > position {
		position = e
	}
	for position < total {
		if s, e := t.findWordBoundaries(position); s == position && e > position {
			return position
		}
		next := t.nextPositionOnGraphemes(position)
		if next <= position {
			break
		}
		position = next
	}
	return total
}

// visualLineStart returns the byte offset of the first index on the visual
// (wrapped) line holding the caret at position, and whether the caret's line is
// laid out. A far-left probe clamps to that line's start.
func (t *Text) visualLineStart(context *guigui.Context, widgetBounds *guigui.WidgetBounds, position int) (int, bool) {
	pos, ok := t.textPosition(context, widgetBounds.Bounds(), position, false)
	if !ok {
		return 0, false
	}
	y := int((pos.Top + pos.Bottom) / 2)
	idx := t.textIndexFromPosition(context, widgetBounds.Bounds(), image.Pt(math.MinInt32, y), false)
	if idx < 0 {
		return 0, false
	}
	return idx, true
}

// visualLineEnd returns the byte offset of the last index on the visual
// (wrapped) line holding the caret at position, excluding a trailing line
// break, and whether the caret's line is laid out.
func (t *Text) visualLineEnd(context *guigui.Context, widgetBounds *guigui.WidgetBounds, position int) (int, bool) {
	pos, ok := t.textPosition(context, widgetBounds.Bounds(), position, false)
	if !ok {
		return 0, false
	}
	y := int((pos.Top + pos.Bottom) / 2)
	idx := t.textIndexFromPosition(context, widgetBounds.Bounds(), image.Pt(math.MaxInt32, y), false)
	if idx < 0 {
		return 0, false
	}
	return idx, true
}

// navigateBackward moves the caret backward to target(position), or extends the
// selection there under Shift. position is the moving end; target reporting
// ok=false leaves the selection unchanged.
func (t *Text) navigateBackward(shift bool, target func(position int) (int, bool)) {
	start, end := t.store.Selection()
	position := start
	if shift && t.shiftSelectionSide == SelectionSideEnd {
		position = end
	}
	tgt, ok := target(position)
	if !ok {
		return
	}
	switch {
	case !shift:
		t.setSelection(tgt, tgt, SelectionSideNone, true)
	case t.shiftSelectionSide == SelectionSideEnd:
		t.setSelection(start, tgt, SelectionSideEnd, true)
	default:
		t.setSelection(tgt, end, SelectionSideStart, true)
	}
}

// navigateForward mirrors [Text.navigateBackward] in the forward direction.
func (t *Text) navigateForward(shift bool, target func(position int) (int, bool)) {
	start, end := t.store.Selection()
	position := end
	if shift && t.shiftSelectionSide == SelectionSideStart {
		position = start
	}
	tgt, ok := target(position)
	if !ok {
		return
	}
	switch {
	case !shift:
		t.setSelection(tgt, tgt, SelectionSideNone, true)
	case t.shiftSelectionSide == SelectionSideStart:
		t.setSelection(tgt, end, SelectionSideStart, true)
	default:
		t.setSelection(start, tgt, SelectionSideEnd, true)
	}
}

func (t *Text) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	cursorPosition := image.Pt(ebiten.CursorPosition())
	hotspotResult := t.handleHotspotPointingInput(context, widgetBounds, cursorPosition)

	if !t.selectable && !t.editable {
		return hotspotResult
	}
	if t.dragState.isDragging() {
		t.dragState.trackCursorMovement(cursorPosition)
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			idx := t.textIndexFromPosition(context, widgetBounds.Bounds(), cursorPosition, false)
			start, end := t.dragState.extendedSelection(idx)
			// idx is the dragged-to position; record whichever endpoint it
			// became as the moving end so a subsequent Shift+click or
			// Shift+arrow extends from the opposite, anchored end. While the
			// cursor stays inside a word- or line-selection, idx matches
			// neither endpoint and no moving end is tracked.
			var shiftSide SelectionSide
			switch idx {
			case start:
				shiftSide = SelectionSideStart
			case end:
				shiftSide = SelectionSideEnd
			}
			if t.setSelection(start, end, shiftSide, true) {
				return guigui.HandleInputByWidget(t)
			} else {
				return guigui.AbortHandlingInputByWidget(t)
			}
		}
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			t.dragState.reset()
			return guigui.HandleInputByWidget(t)
		}
		return guigui.AbortHandlingInputByWidget(t)
	}

	left := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	right := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	if left || right {
		if widgetBounds.IsHitAtCursor() {
			t.handleClick(context, widgetBounds.Bounds(), cursorPosition, left)
			if left {
				return guigui.HandleInputByWidget(t)
			}
			return guigui.HandleInputResult{}
		}
		context.SetFocused(t, false)
	}

	if !context.IsFocused(t) {
		if t.store.IsFocused() {
			t.store.Blur()
		}
		return guigui.HandleInputResult{}
	}
	// The field auto-commits text input via Ebitengine's BeforeUpdate hook whenever
	// it is focused, so only focus it when this widget actually accepts edits.
	if t.editable {
		t.store.Focus()
	} else if t.store.IsFocused() {
		t.store.Blur()
	}

	if !t.editable && !t.selectable {
		return guigui.HandleInputResult{}
	}

	return guigui.HandleInputResult{}
}

func (t *Text) handleClick(context *guigui.Context, textBounds image.Rectangle, cursorPosition image.Point, leftClick bool) {
	idx := t.textIndexFromPosition(context, textBounds, cursorPosition, false)

	// Shift+click on a text that already holds a cursor moves one end of the
	// selection to the clicked position and keeps the opposite end anchored.
	// Dragging afterwards keeps extending from the same anchor.
	if leftClick && idx >= 0 && ebiten.IsKeyPressed(ebiten.KeyShift) && context.IsFocusedOrHasFocusedDescendant(t) {
		selStart, selEnd := t.store.Selection()
		anchor := shiftClickAnchor(selStart, selEnd, t.shiftSelectionSide, idx)
		t.dragState.start(cursorPosition, anchor, anchor)
		t.setSelection(anchor, idx, SelectionSideEnd, false)
		context.SetFocused(t, true)
		// Reset the click count so a following plain click is not treated as a
		// double- or triple-click.
		t.dragState.resetClickCount(ebiten.Tick(), idx)
		return
	}

	var clickCount int
	if t.hotspotPressed {
		// A press on a hotspot is always an individual click: double- and
		// triple-clicks must not select a word or the whole text there.
		t.dragState.resetClickCount(ebiten.Tick(), idx)
		clickCount = 1
	} else {
		clickCount = t.dragState.click(ebiten.Tick(), idx, leftClick)
	}

	switch clickCount {
	case 1:
		if leftClick {
			t.dragState.start(cursorPosition, idx, idx)
		} else {
			t.dragState.reset()
		}
		if leftClick || !context.IsFocusedOrHasFocusedDescendant(t) {
			if start, end := t.store.Selection(); start != idx || end != idx {
				t.setSelection(idx, idx, SelectionSideNone, false)
			}
		}
	case 2:
		start, end := t.findWordBoundariesForDoubleClick(idx)
		t.dragState.start(cursorPosition, start, end)
		t.setSelection(start, end, SelectionSideNone, false)
	case 3:
		t.doSelectAll()
	}

	context.SetFocused(t, true)
}

func (t *Text) HandleButtonInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	r := t.handleButtonInput(context, widgetBounds)
	// Adjust the scroll offset right after handling the input so that
	// the scroll delta is applied during the next Build & Layout pass
	// within the same tick, avoiding a one-tick wobble.
	if r.IsHandled() && (t.selectable || t.editable) {
		if dx, dy := t.adjustScrollOffset(context, widgetBounds); dx != 0 || dy != 0 {
			guigui.DispatchEvent(t, textEventScrollDelta, dx, dy)
		}
	}
	return r
}

// updateIMEComposer pumps the IME composer for one tick and folds a resulting
// composition or commit into the cached text size and the value-changed
// listeners. It reports whether the IME consumed input this tick.
func (t *Text) updateIMEComposer(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	t.ensureStoreCallbacks()
	start, _ := t.store.Selection()
	if pos, ok := t.textPosition(context, widgetBounds.Bounds(), start, false); ok {
		t.store.SetBounds(image.Rect(int(pos.X), int(pos.Top), int(pos.X+1), int(pos.Bottom)))
	}
	processed, err := t.store.Update()
	if err != nil {
		slog.Error(err.Error())
	}
	if processed {
		// Reset the cached size before the scroll offset is adjusted so the text size is correct.
		t.resetCachedTextSize()
		t.dispatchValueChanged(false, false)
	}
	return processed
}

func (t *Text) handleButtonInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if t.onHandleButtonInput != nil {
		if r := t.onHandleButtonInput(context, widgetBounds); r.IsHandled() {
			return r
		}
	}

	if !t.selectable && !t.editable {
		return guigui.HandleInputResult{}
	}

	mode := context.KeyBindingMode()
	// commandMode also selects the navigation layout: Command and Option with the
	// arrow keys, and Home and End that scroll without moving the caret.
	commandMode := mode == guigui.KeyBindingModeCommand
	shortcutModifierPressed := ebiten.IsKeyPressed(mode.ShortcutModifierKey())
	emacsKeymap := mode.UsesEmacsKeymap()

	if t.editable {
		if t.updateIMEComposer(context, widgetBounds) {
			return guigui.HandleInputByWidget(t)
		}

		// Do not accept key inputs when compositing.
		if _, _, ok := t.store.CompositionSelection(); ok {
			return guigui.HandleInputByWidget(t)
		}

		// For Windows key binds, see:
		// https://support.microsoft.com/en-us/windows/keyboard-shortcuts-in-windows-dcc61a57-8ff0-cffe-9796-cb9706c75eec#textediting

		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			if t.IsMultiline() {
				t.replaceTextAtSelection("\n")
			} else {
				t.commit(true)
			}
			return guigui.HandleInputByWidget(t)
		case IsKeyRepeating(ebiten.KeyBackspace) ||
			emacsKeymap && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyH):
			start, end := t.store.Selection()
			if start != end {
				t.replaceTextAtSelection("")
			} else if start > 0 {
				pos := t.prevPositionOnGraphemes(start)
				t.replaceTextAt("", pos, start, nil)
			}
			return guigui.HandleInputByWidget(t)
		case ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyD):
			// Delete
			start, end := t.store.Selection()
			if start != end {
				t.replaceTextAtSelection("")
			} else if emacsKeymap && end < t.store.TextLengthInBytes() {
				pos := t.nextPositionOnGraphemes(end)
				t.replaceTextAt("", start, pos, nil)
			}
			return guigui.HandleInputByWidget(t)
		case IsKeyRepeating(ebiten.KeyDelete):
			// Delete one cluster
			if start, end := t.store.Selection(); end < t.store.TextLengthInBytes() {
				pos := t.nextPositionOnGraphemes(end)
				t.replaceTextAt("", start, pos, nil)
			}
			return guigui.HandleInputByWidget(t)
		// The Emacs key theme binds Control+W to a cut as well. The macOS text
		// system leaves Control+W unbound, so Command mode does not take it.
		case shortcutModifierPressed && IsKeyRepeating(ebiten.KeyX) ||
			mode == guigui.KeyBindingModeControlEmacs && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyW):
			t.Cut()
			return guigui.HandleInputByWidget(t)
		case shortcutModifierPressed && ebiten.IsKeyPressed(ebiten.KeyShift) && IsKeyRepeating(ebiten.KeyV):
			// Paste without styles. An additionally held Option also lands
			// here, covering the macOS Paste and Match Style chord.
			t.PasteWithoutStyles()
			return guigui.HandleInputByWidget(t)
		case shortcutModifierPressed && IsKeyRepeating(ebiten.KeyV):
			t.Paste()
			return guigui.HandleInputByWidget(t)
		// Where the Emacs keymap is in effect, Control+Y is a yank, so redo is
		// only Shift and the shortcut modifier with Z.
		case shortcutModifierPressed && ebiten.IsKeyPressed(ebiten.KeyShift) && IsKeyRepeating(ebiten.KeyZ) ||
			!emacsKeymap && shortcutModifierPressed && IsKeyRepeating(ebiten.KeyY):
			t.Redo()
			return guigui.HandleInputByWidget(t)
		case shortcutModifierPressed && IsKeyRepeating(ebiten.KeyZ):
			t.Undo()
			return guigui.HandleInputByWidget(t)
		}
	}

	switch {
	// macOS: Command+Arrow moves to a visual-line or document extreme;
	// Option+Arrow moves by word or paragraph. Shift extends the selection.
	case commandMode && ebiten.IsKeyPressed(ebiten.KeyMeta) && IsKeyRepeating(ebiten.KeyLeft):
		t.navigateBackward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.visualLineStart(context, widgetBounds, position)
		})
		return guigui.HandleInputByWidget(t)
	case commandMode && ebiten.IsKeyPressed(ebiten.KeyMeta) && IsKeyRepeating(ebiten.KeyRight):
		t.navigateForward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.visualLineEnd(context, widgetBounds, position)
		})
		return guigui.HandleInputByWidget(t)
	case commandMode && ebiten.IsKeyPressed(ebiten.KeyMeta) && IsKeyRepeating(ebiten.KeyUp):
		t.navigateBackward(ebiten.IsKeyPressed(ebiten.KeyShift), func(int) (int, bool) {
			return 0, true
		})
		return guigui.HandleInputByWidget(t)
	case commandMode && ebiten.IsKeyPressed(ebiten.KeyMeta) && IsKeyRepeating(ebiten.KeyDown):
		t.navigateForward(ebiten.IsKeyPressed(ebiten.KeyShift), func(int) (int, bool) {
			return t.store.TextLengthInBytes(), true
		})
		return guigui.HandleInputByWidget(t)
	case commandMode && ebiten.IsKeyPressed(ebiten.KeyAlt) && IsKeyRepeating(ebiten.KeyLeft):
		t.navigateBackward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.prevWordStart(position), true
		})
		return guigui.HandleInputByWidget(t)
	case commandMode && ebiten.IsKeyPressed(ebiten.KeyAlt) && IsKeyRepeating(ebiten.KeyRight):
		t.navigateForward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.nextWordEnd(position), true
		})
		return guigui.HandleInputByWidget(t)
	case commandMode && ebiten.IsKeyPressed(ebiten.KeyAlt) && IsKeyRepeating(ebiten.KeyUp):
		t.navigateBackward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.paragraphStart(position), true
		})
		return guigui.HandleInputByWidget(t)
	case commandMode && ebiten.IsKeyPressed(ebiten.KeyAlt) && IsKeyRepeating(ebiten.KeyDown):
		t.navigateForward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.paragraphEnd(position), true
		})
		return guigui.HandleInputByWidget(t)
	// macOS: Shift+Home/End extend the selection to the start/end of the text.
	// Plain Home/End scroll without moving the caret; they are left unhandled
	// here and handled by the virtualizing parent after bubbling up.
	case commandMode && ebiten.IsKeyPressed(ebiten.KeyShift) && IsKeyRepeating(ebiten.KeyHome):
		t.navigateBackward(true, func(int) (int, bool) {
			return 0, true
		})
		return guigui.HandleInputByWidget(t)
	case commandMode && ebiten.IsKeyPressed(ebiten.KeyShift) && IsKeyRepeating(ebiten.KeyEnd):
		t.navigateForward(true, func(int) (int, bool) {
			return t.store.TextLengthInBytes(), true
		})
		return guigui.HandleInputByWidget(t)
	// Windows/Linux: Ctrl+Arrow moves by word, Home/End to line head/tail,
	// Ctrl+Home/End to document head/tail. Shift extends the selection.
	case !commandMode && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyLeft):
		t.navigateBackward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.prevWordStart(position), true
		})
		return guigui.HandleInputByWidget(t)
	case !commandMode && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyRight):
		t.navigateForward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.nextWordStart(position), true
		})
		return guigui.HandleInputByWidget(t)
	case !commandMode && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyHome):
		t.navigateBackward(ebiten.IsKeyPressed(ebiten.KeyShift), func(int) (int, bool) {
			return 0, true
		})
		return guigui.HandleInputByWidget(t)
	case !commandMode && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyEnd):
		t.navigateForward(ebiten.IsKeyPressed(ebiten.KeyShift), func(int) (int, bool) {
			return t.store.TextLengthInBytes(), true
		})
		return guigui.HandleInputByWidget(t)
	case !commandMode && IsKeyRepeating(ebiten.KeyHome):
		t.navigateBackward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.visualLineStart(context, widgetBounds, position)
		})
		return guigui.HandleInputByWidget(t)
	case !commandMode && IsKeyRepeating(ebiten.KeyEnd):
		t.navigateForward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.visualLineEnd(context, widgetBounds, position)
		})
		return guigui.HandleInputByWidget(t)
	// The Emacs key theme moves by word with Alt and B or F, while the macOS text
	// system places the same motion on Alt and Control. Both cases precede the
	// character motion below, whose guard does not test Alt and would otherwise
	// take the chord first.
	case mode == guigui.KeyBindingModeControlEmacs && ebiten.IsKeyPressed(ebiten.KeyAlt) && IsKeyRepeating(ebiten.KeyB) ||
		commandMode && ebiten.IsKeyPressed(ebiten.KeyAlt) && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyB):
		t.navigateBackward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.prevWordStart(position), true
		})
		return guigui.HandleInputByWidget(t)
	// Both keymaps stop at the word end, where Control with an arrow key stops at
	// the next word start.
	case mode == guigui.KeyBindingModeControlEmacs && ebiten.IsKeyPressed(ebiten.KeyAlt) && IsKeyRepeating(ebiten.KeyF) ||
		commandMode && ebiten.IsKeyPressed(ebiten.KeyAlt) && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyF):
		t.navigateForward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.nextWordEnd(position), true
		})
		return guigui.HandleInputByWidget(t)
	case IsKeyRepeating(ebiten.KeyLeft) ||
		emacsKeymap && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyB):
		start, end := t.store.Selection()
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			if t.shiftSelectionSide == SelectionSideEnd {
				pos := t.prevPositionOnGraphemes(end)
				t.setSelection(start, pos, SelectionSideEnd, true)
			} else {
				pos := t.prevPositionOnGraphemes(start)
				t.setSelection(pos, end, SelectionSideStart, true)
			}
		} else {
			if start != end {
				t.setSelection(start, start, SelectionSideNone, true)
			} else if start > 0 {
				pos := t.prevPositionOnGraphemes(start)
				t.setSelection(pos, pos, SelectionSideNone, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case IsKeyRepeating(ebiten.KeyRight) ||
		emacsKeymap && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyF):
		start, end := t.store.Selection()
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			if t.shiftSelectionSide == SelectionSideStart {
				pos := t.nextPositionOnGraphemes(start)
				t.setSelection(pos, end, SelectionSideStart, true)
			} else {
				pos := t.nextPositionOnGraphemes(end)
				t.setSelection(start, pos, SelectionSideEnd, true)
			}
		} else {
			if start != end {
				t.setSelection(end, end, SelectionSideNone, true)
			} else if start < t.store.TextLengthInBytes() {
				pos := t.nextPositionOnGraphemes(start)
				t.setSelection(pos, pos, SelectionSideNone, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case IsKeyRepeating(ebiten.KeyUp) ||
		emacsKeymap && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyP):
		shift := ebiten.IsKeyPressed(ebiten.KeyShift)
		var moveEnd bool
		start, end := t.store.Selection()
		idx := start
		if shift && t.shiftSelectionSide == SelectionSideEnd {
			idx = end
			moveEnd = true
		}
		if pos, ok := t.textPosition(context, widgetBounds.Bounds(), idx, false); ok {
			y, _ := t.adjacentLineYs(context, pos)
			nextIdx := t.textIndexFromPosition(context, widgetBounds.Bounds(), image.Pt(int(pos.X), int(y)), false)
			// A genuine move to the previous line lands on an earlier byte
			// offset. When the caret is already on the first line, the move is
			// clamped and round-trips to the same offset; move to the head of
			// the text instead.
			if nextIdx >= idx {
				nextIdx = 0
			}
			if shift {
				if moveEnd {
					t.setSelection(start, nextIdx, SelectionSideEnd, true)
				} else {
					t.setSelection(nextIdx, end, SelectionSideStart, true)
				}
			} else {
				t.setSelection(nextIdx, nextIdx, SelectionSideNone, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case IsKeyRepeating(ebiten.KeyDown) ||
		emacsKeymap && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyN):
		shift := ebiten.IsKeyPressed(ebiten.KeyShift)
		var moveStart bool
		start, end := t.store.Selection()
		idx := end
		if shift && t.shiftSelectionSide == SelectionSideStart {
			idx = start
			moveStart = true
		}
		if pos, ok := t.textPosition(context, widgetBounds.Bounds(), idx, false); ok {
			_, y := t.adjacentLineYs(context, pos)
			nextIdx := t.textIndexFromPosition(context, widgetBounds.Bounds(), image.Pt(int(pos.X), int(y)), false)
			// A genuine move to the next line lands on a later byte offset. When
			// the caret is already on the last line, the move is clamped and
			// round-trips to the same offset; move to the tail of the text
			// instead.
			if nextIdx <= idx {
				nextIdx = t.store.TextLengthInBytes()
			}
			if shift {
				if moveStart {
					t.setSelection(nextIdx, end, SelectionSideStart, true)
				} else {
					t.setSelection(start, nextIdx, SelectionSideEnd, true)
				}
			} else {
				t.setSelection(nextIdx, nextIdx, SelectionSideNone, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case emacsKeymap && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyA):
		idx := 0
		start, end := t.store.Selection()
		if i, l := textutil.LastLineBreakPositionAndLen(t.stringValueWithRange(0, start)); i >= 0 {
			idx = i + l
		}
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			t.setSelection(idx, end, SelectionSideStart, true)
		} else {
			t.setSelection(idx, idx, SelectionSideNone, true)
		}
		return guigui.HandleInputByWidget(t)
	case emacsKeymap && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyE):
		idx := t.store.TextLengthInBytes()
		start, end := t.store.Selection()
		if i, _ := textutil.FirstLineBreakPositionAndLen(t.stringValueWithRange(end, -1)); i >= 0 {
			idx = end + i
		}
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			t.setSelection(start, idx, SelectionSideEnd, true)
		} else {
			t.setSelection(idx, idx, SelectionSideNone, true)
		}
		return guigui.HandleInputByWidget(t)
	// Where the Emacs keymap shares the shortcut modifier with Control, its
	// beginning-of-line binding above takes Control+A and select all has no chord.
	case shortcutModifierPressed && IsKeyRepeating(ebiten.KeyA):
		t.doSelectAll()
		return guigui.HandleInputByWidget(t)
	case shortcutModifierPressed && IsKeyRepeating(ebiten.KeyC):
		// Copy
		t.Copy()
		return guigui.HandleInputByWidget(t)
	case emacsKeymap && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyK):
		// 'Kill' the text after the caret or the selection.
		start, end := t.store.Selection()
		if start == end {
			i, l := textutil.FirstLineBreakPositionAndLen(t.stringValueWithRange(start, -1))
			if i < 0 {
				end = t.store.TextLengthInBytes()
			} else if i == 0 {
				end = start + l
			} else {
				end = start + i
			}
		}
		t.tmpClipboard = t.stringValueWithRange(start, end)
		t.replaceTextAt("", start, end, nil)
		return guigui.HandleInputByWidget(t)
	case emacsKeymap && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyY):
		// 'Yank' the killed text.
		if t.tmpClipboard != "" {
			t.replaceTextAtSelection(t.tmpClipboard)
		}
		return guigui.HandleInputByWidget(t)
	}

	return guigui.HandleInputResult{}
}

func (t *Text) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	cursorPosition := image.Pt(ebiten.CursorPosition())
	if !t.dragState.moved(cursorPosition) {
		if _, ok := t.hotspotRangeAt(context, widgetBounds, cursorPosition); ok {
			return ebiten.CursorShapePointer, true
		}
	}
	if t.selectable || t.editable {
		return ebiten.CursorShapeText, true
	}
	return 0, false
}
