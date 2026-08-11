// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"image/color"
	"slices"

	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
	"github.com/guigui-gui/guigui/clipboard"
)

func ReplaceNewLinesWithSpace(text string, start, end int) (string, int, int) {
	return replaceNewLinesWithSpace(text, start, end)
}

func ShiftClickAnchor(start, end int, shiftSide SelectionSide, idx int) int {
	return shiftClickAnchor(start, end, shiftSide, idx)
}

func (t *Text) FindWordBoundariesForDoubleClick(idx int) (start, end int) {
	return t.findWordBoundariesForDoubleClick(idx)
}

func (t *Text) PrevWordStart(position int) int {
	return t.prevWordStart(position)
}

func (t *Text) NextWordEnd(position int) int {
	return t.nextWordEnd(position)
}

func (t *Text) NextWordStart(position int) int {
	return t.nextWordStart(position)
}

func (t *Text) ParagraphStart(position int) int {
	return t.paragraphStart(position)
}

func (t *Text) ParagraphEnd(position int) int {
	return t.paragraphEnd(position)
}

func (t *Text) OverrideStyleRuns() []textstyle.Run {
	return slices.Collect(t.ensureOverrideStyleRuns().All())
}

func (t *Text) SetColorInRange(start, end int, clr color.Color) {
	t.ensureOverrideStyleRuns().SetColor(start, end, clr)
}

func (t *Text) SetUnderlineInRange(start, end int, underline bool) {
	t.ensureOverrideStyleRuns().SetUnderline(start, end, underline)
}

func (t *Text) UnsetColorInRange(start, end int) {
	t.ensureOverrideStyleRuns().UnsetColor(start, end)
}

func (t *Text) ResetStylesInRange(start, end int) {
	t.ensureOverrideStyleRuns().Reset(start, end)
}

func (t *Text) ReplaceTextAt(text string, start, end int, styleRuns *textstyle.Runs) {
	t.replaceTextAt(text, start, end, styleRuns)
}

// CommitTextByIME inserts text at the current selection through the store's
// IME commit path.
func (t *Text) CommitTextByIME(text string) {
	t.ensureStoreCallbacks()
	t.store.commitText(text)
}

// InsertionStyle returns the current insertion style.
func (t *Text) InsertionStyle() textstyle.Style {
	return t.insertionStyle
}

// HandleFocusChanged invokes the widget's focus-changed handler.
func (t *Text) HandleFocusChanged(focused bool) {
	t.handleFocusChanged(nil, focused)
}

// SetCompositionByIME replaces the active composition through the store's IME
// composition path.
func (t *Text) SetCompositionByIME(text string, selStartInBytes, selEndInBytes int) {
	t.ensureStoreCallbacks()
	t.store.setComposition(text, selStartInBytes, selEndInBytes)
}

// RenderingStyleRuns returns the ranged style overrides in rendering-text byte
// offsets.
func (t *Text) RenderingStyleRuns() []textstyle.Run {
	return slices.Collect(t.renderingStyleRuns().All())
}

// SetClipboardForTest substitutes the OS clipboard access with read and
// write, and returns a function restoring the real access.
func SetClipboardForTest(read func() (clipboard.Contents, error), write func(clipboard.Contents) error) (restore func()) {
	origRead, origWrite := clipboardRead, clipboardWrite
	clipboardRead = read
	clipboardWrite = write
	return func() {
		clipboardRead = origRead
		clipboardWrite = origWrite
	}
}

// ResetRichClipboardForTest empties the process-global rich clipboard slot.
func ResetRichClipboardForTest() {
	theRichClipboard.text = ""
	theRichClipboard.styleRuns.Clear()
}
