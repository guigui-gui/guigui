// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"github.com/guigui-gui/guigui/basicwidget/internal/font"
)

// Insertion is a zero-width position in the laid-out text carrying the face
// of the text that will be inserted there. It takes part in the height of the
// visual line holding it, as a character with that face would. A face is all
// it carries: nothing is drawn at the position, so the colors and decorations
// of a [StyleRun] would have nothing to apply to.
type Insertion struct {
	// Face is the face the inserted text uses. The zero value means there is
	// no insertion.
	Face font.Face

	// IndexInBytes is the insertion's byte offset in the laid-out text.
	IndexInBytes int
}

// isZero reports whether i is absent.
func (i *Insertion) isZero() bool {
	return i.Face.ID() == 0
}

// onVisualLine reports whether the visual line holding lineStr, starting at
// lineStartInBytes, holds i. endsText reports whether the line is the last of
// the text it belongs to; otherwise a position at the line's end belongs to
// the following line, where the caret renders.
func (i *Insertion) onVisualLine(lineStartInBytes int, lineStr string, endsText bool) bool {
	if i.isZero() {
		return false
	}
	if i.IndexInBytes < lineStartInBytes {
		return false
	}
	breakLen := tailingLineBreakLen(lineStr)
	contentEnd := lineStartInBytes + len(lineStr) - breakLen
	if i.IndexInBytes < contentEnd {
		return true
	}
	if i.IndexInBytes > contentEnd {
		return false
	}
	return endsText || breakLen > 0
}

// scaleOnVisualLine returns the factor by which i scales the height of the
// visual line holding lineStr, relative to def's size. The factor is 1 when
// the line does not hold i, and never below 1.
func (i *Insertion) scaleOnVisualLine(def font.Face, lineStartInBytes int, lineStr string, endsText bool) float64 {
	if !i.onVisualLine(lineStartInBytes, lineStr, endsText) {
		return 1
	}
	defSize := def.Attributes().Size
	if defSize <= 0 {
		return 1
	}
	return max(i.Face.Attributes().Size/defSize, 1)
}
