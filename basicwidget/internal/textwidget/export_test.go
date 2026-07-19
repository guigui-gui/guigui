// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"slices"

	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
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
