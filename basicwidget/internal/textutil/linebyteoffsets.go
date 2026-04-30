// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"slices"
	"strings"
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
	l.offsets = append(l.offsets[:0], 0)

	// Maintain the index of the next occurrence at or after cur for each
	// line-break lead byte. len(s) means "no further occurrence". After a
	// byte's first scan returns len(s), it stays that way for the rest of
	// the rebuild — so lead bytes that never appear in s (commonly 0x0B,
	// 0x0C, 0xC2, 0xE2 in editor text) cost a single IndexByte each total,
	// rather than one per line as in the FirstLineBreakPositionAndLen
	// loop.
	n := len(s)
	next := func(b byte, from int) int {
		if from >= n {
			return n
		}
		i := strings.IndexByte(s[from:], b)
		if i < 0 {
			return n
		}
		return from + i
	}

	nLF := next(0x0a, 0)
	nCR := next(0x0d, 0)
	nVT := next(0x0b, 0)
	nFF := next(0x0c, 0)
	nC2 := next(0xc2, 0)
	nE2 := next(0xe2, 0)

	cur := 0
	for cur < n {
		// Pick the earliest pending occurrence.
		best := nLF
		kind := 0
		if nCR < best {
			best = nCR
			kind = 1
		}
		if nVT < best {
			best = nVT
			kind = 2
		}
		if nFF < best {
			best = nFF
			kind = 3
		}
		if nC2 < best {
			best = nC2
			kind = 4
		}
		if nE2 < best {
			best = nE2
			kind = 5
		}
		if best == n {
			return
		}

		var ln int
		switch kind {
		case 0: // LF
			ln = 1
		case 1: // CR (possibly CRLF)
			ln = 1
			if best+1 < n && s[best+1] == 0x0a {
				ln = 2
			}
		case 2, 3: // VT, FF
			ln = 1
		case 4: // 0xC2 lead — only NEL (U+0085, 0xC2 0x85) is a line break.
			if best+1 < n && s[best+1] == 0x85 {
				ln = 2
			} else {
				// False positive (e.g. NBSP). Advance just this cache.
				nC2 = next(0xc2, best+1)
				continue
			}
		case 5: // 0xE2 lead — only LS (U+2028, 0xE2 0x80 0xA8) / PS (U+2029, 0xE2 0x80 0xA9) are breaks.
			if best+2 < n && s[best+1] == 0x80 && (s[best+2] == 0xa8 || s[best+2] == 0xa9) {
				ln = 3
			} else {
				nE2 = next(0xe2, best+1)
				continue
			}
		}

		cur = best + ln
		l.offsets = append(l.offsets, cur)

		// Refresh whichever caches were consumed.
		if nLF < cur {
			nLF = next(0x0a, cur)
		}
		if nCR < cur {
			nCR = next(0x0d, cur)
		}
		if nVT < cur {
			nVT = next(0x0b, cur)
		}
		if nFF < cur {
			nFF = next(0x0c, cur)
		}
		if nC2 < cur {
			nC2 = next(0xc2, cur)
		}
		if nE2 < cur {
			nE2 = next(0xe2, cur)
		}
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
