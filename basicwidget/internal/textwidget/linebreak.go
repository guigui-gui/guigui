// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"strings"

	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

// isLineBreak reports whether text is a single line break.
func isLineBreak(text string) bool {
	pos, l := textutil.FirstLineBreakPositionAndLen(text)
	return pos == 0 && l == len(text) && l > 0
}

// replaceNewLinesWithSpace returns text with every line break replaced by a
// single space, along with the byte offsets start and end remapped to the
// replaced text.
func replaceNewLinesWithSpace(text string, start, end int) (string, int, int) {
	var buf strings.Builder
	for {
		pos, len := textutil.FirstLineBreakPositionAndLen(text)
		if len == 0 {
			buf.WriteString(text)
			break
		}
		buf.WriteString(text[:pos])
		origLen := buf.Len()
		buf.WriteString(" ")
		if diff := len - 1; diff > 0 {
			if origLen < start {
				if start >= origLen+len {
					start -= diff
				} else {
					// This is a very rare case, e.g. the position is in between '\r' and '\n'.
					start = origLen + 1
				}
			}
			if origLen < end {
				if end >= origLen+len {
					end -= diff
				} else {
					end = origLen + 1
				}
			}
		}
		text = text[pos+len:]
	}
	text = buf.String()

	return text, start, end
}
