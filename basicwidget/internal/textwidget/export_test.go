// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"image/color"
	"slices"

	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
	"github.com/guigui-gui/guigui/internal/clipboard"
)

func ReplaceNewLinesWithSpace(text string, start, end int) (string, int, int) {
	return replaceNewLinesWithSpace(text, start, end)
}

func ShiftClickAnchor(start, end int, shiftSide SelectionSide, idx int) int {
	return shiftClickAnchor(start, end, shiftSide, idx)
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

func (t *Text) StyleRuns() []textstyle.Run {
	return slices.Collect(t.ensureStyleRuns().All())
}

func (t *Text) SetColorInRange(start, end int, clr color.Color) {
	t.ensureStyleRuns().SetColor(start, end, clr)
}

func (t *Text) SetUnderlineInRange(start, end int, underline bool) {
	t.ensureStyleRuns().SetUnderline(start, end, underline)
}

func (t *Text) UnsetColorInRange(start, end int) {
	t.ensureStyleRuns().UnsetColor(start, end)
}

func (t *Text) ResetStylesInRange(start, end int) {
	t.ensureStyleRuns().Reset(start, end)
}

func (t *Text) ReplaceTextAt(text string, start, end int) {
	t.replaceTextAt(text, start, end, nil)
}

func AppendFaceRunsThroughComposition(dst, src []textutil.FaceRun, selStart, selEnd, compLen int) []textutil.FaceRun {
	return appendFaceRunsThroughComposition(dst, src, selStart, selEnd, compLen)
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
