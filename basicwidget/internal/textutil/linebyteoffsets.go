// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"slices"
)

// LineByteOffsets holds the byte offsets where each logical line (segment
// separated by hard line breaks) begins within a source string. It is a
// precomputed sidecar that enables O(log n) line<->byte-offset lookups
// without rescanning the text.
//
// After RebuildFromString returns, the first entry is always 0 and the
// entries are strictly increasing. A trailing line break in the source
// string creates an extra empty line at the end (e.g. "abc\n" has two
// logical lines).
type LineByteOffsets struct {
	offsets []int
}

// RebuildFromString discards any current contents and rescans s for logical
// line starts. It is O(len(s)).
func (l *LineByteOffsets) RebuildFromString(s string) {
	l.offsets = l.offsets[:0]
	l.offsets = append(l.offsets, 0)
	var pos int
	for {
		p, n := FirstLineBreakPositionAndLen(s[pos:])
		if p == -1 {
			return
		}
		pos += p + n
		l.offsets = append(l.offsets, pos)
	}
}

// Reset clears the offsets. After Reset, LineCount returns 0; callers that
// expect at least one line must rebuild first.
func (l *LineByteOffsets) Reset() {
	l.offsets = l.offsets[:0]
}

// LineCount returns the number of logical lines.
//
// The empty string has one logical line. A trailing line break creates an
// extra empty line, so "abc\n" has two logical lines.
func (l *LineByteOffsets) LineCount() int {
	return len(l.offsets)
}

// ByteOffsetByLineIndex returns the byte offset of the start of the i-th logical
// line. Panics if i is out of range.
func (l *LineByteOffsets) ByteOffsetByLineIndex(i int) int {
	return l.offsets[i]
}

// LineIndexForByteOffset returns the index of the logical line that contains
// byteOffset. byteOffset is clamped: negative values map to line 0 and values
// past the text map to the last line.
func (l *LineByteOffsets) LineIndexForByteOffset(byteOffset int) int {
	i, found := slices.BinarySearch(l.offsets, byteOffset)
	if found {
		return i
	}
	// Not found: i is the insertion position - the smallest index with
	// offsets[i] > byteOffset, or len(offsets) if byteOffset is past every
	// recorded line start. The line containing byteOffset is therefore i-1,
	// except when i == 0 (byteOffset < 0, or offsets is empty), which clamps
	// to line 0.
	if i == 0 {
		return 0
	}
	return i - 1
}
