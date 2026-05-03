// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"bytes"
	"io"
	"slices"
)

// streamState tracks the trailing partial-line-break state across a
// streaming rebuild. Bytes whose line-break classification depends on
// not-yet-seen lookahead are deferred via this state.
type streamState uint8

const (
	streamStateNormal streamState = iota
	streamStateAfterCR
	streamStateAfterC2
	streamStateAfterE2
	streamStateAfterE280
)

// lineByteOffsetsWriter is the [io.Writer] handed to the scan callback
// inside [LineByteOffsets.Rebuild]. It owns the transient streaming-scan
// state so that nothing about the partial-break tracking leaks into
// [LineByteOffsets]'s exported surface or persists past Rebuild.
type lineByteOffsetsWriter struct {
	l         *LineByteOffsets
	streamOff int
	state     streamState
}

// Write scans p for logical-line starts and updates the streaming state.
func (w *lineByteOffsetsWriter) Write(p []byte) (int, error) {
	n := len(p)
	if n == 0 {
		return 0, nil
	}

	i := 0

	// Phase 1: resolve any pending state carried over from a previous
	// Write using leading bytes of p. Loop because resolving one state
	// may transition into another (e.g. AfterE2 → AfterE280).
	for i < n && w.state != streamStateNormal {
		switch w.state {
		case streamStateAfterCR:
			if p[i] == 0x0a {
				w.l.offsets = append(w.l.offsets, w.streamOff+i+1)
				i++
			} else {
				// Bare CR ended at the previous Write's last byte; the
				// new line starts here. Don't consume p[i] — it will be
				// reprocessed in Normal state below.
				w.l.offsets = append(w.l.offsets, w.streamOff+i)
			}
			w.state = streamStateNormal
		case streamStateAfterC2:
			if p[i] == 0x85 {
				w.l.offsets = append(w.l.offsets, w.streamOff+i+1)
				i++
			}
			w.state = streamStateNormal
		case streamStateAfterE2:
			if p[i] == 0x80 {
				w.state = streamStateAfterE280
				i++
			} else {
				w.state = streamStateNormal
			}
		case streamStateAfterE280:
			if p[i] == 0xa8 || p[i] == 0xa9 {
				w.l.offsets = append(w.l.offsets, w.streamOff+i+1)
				i++
			}
			w.state = streamStateNormal
		}
	}

	// Phase 2: scan p[i:n] in Normal state with cached IndexByte. Once a
	// lead byte's first scan returns n it stays at n for the rest of this
	// chunk, so leads that never appear in p (commonly 0x0B, 0x0C, 0xC2,
	// 0xE2 in editor text) cost a single IndexByte each total rather than
	// one per line. The cache is local to this Write; cross-Write caching
	// is impossible since p is gone after we return.
	if i < n {
		next := func(b byte, from int) int {
			if from >= n {
				return n
			}
			k := bytes.IndexByte(p[from:], b)
			if k < 0 {
				return n
			}
			return from + k
		}
		nLF := next(0x0a, i)
		nVT := next(0x0b, i)
		nFF := next(0x0c, i)
		nCR := next(0x0d, i)
		nC2 := next(0xc2, i)
		nE2 := next(0xe2, i)

		cur := i
	Loop:
		for cur < n {
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
				break
			}

			var ln int
			switch kind {
			case 0, 2, 3: // LF, VT, FF
				ln = 1
			case 1: // CR (possibly CRLF)
				if best+1 == n {
					w.state = streamStateAfterCR
					break Loop
				}
				if p[best+1] == 0x0a {
					ln = 2
				} else {
					ln = 1
				}
			case 4: // 0xC2 lead — only NEL (U+0085, 0xC2 0x85) is a line break.
				if best+1 == n {
					w.state = streamStateAfterC2
					break Loop
				}
				if p[best+1] == 0x85 {
					ln = 2
				} else {
					nC2 = next(0xc2, best+1)
					continue
				}
			case 5: // 0xE2 lead — only LS (U+2028) / PS (U+2029) are breaks.
				if best+1 == n {
					w.state = streamStateAfterE2
					break Loop
				}
				if p[best+1] != 0x80 {
					nE2 = next(0xe2, best+1)
					continue
				}
				if best+2 == n {
					w.state = streamStateAfterE280
					break Loop
				}
				if p[best+2] == 0xa8 || p[best+2] == 0xa9 {
					ln = 3
				} else {
					nE2 = next(0xe2, best+1)
					continue
				}
			}

			cur = best + ln
			w.l.offsets = append(w.l.offsets, w.streamOff+cur)

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

	w.streamOff += n
	return n, nil
}

// LineByteOffsets holds the byte offsets where each logical line (segment
// separated by hard line breaks) begins within a source string. It is a
// precomputed sidecar that enables O(log n) line<->byte-offset lookups
// without rescanning the text.
//
// After a rebuild the first entry is always 0 and the entries are
// strictly increasing. A trailing line break in the source string
// creates an extra empty line at the end (e.g. "abc\n" has two logical
// lines).
type LineByteOffsets struct {
	offsets []int
}

// Rebuild discards any current contents and rescans the bytes written by
// scan for logical-line starts. The [io.Writer] passed to scan accepts
// bytes in any number of chunks.
//
// Rebuild produces the same offsets RebuildFromString would for the
// concatenated bytes, but never requires those bytes to be materialized
// contiguously. Any error from scan is returned unchanged after the
// trailing partial-break state has been flushed.
func (l *LineByteOffsets) Rebuild(scan func(io.Writer) error) error {
	l.offsets = append(l.offsets[:0], 0)
	w := &lineByteOffsetsWriter{l: l}
	err := scan(w)
	if w.state == streamStateAfterCR {
		l.offsets = append(l.offsets, w.streamOff)
	}
	return err
}

// RebuildFromString discards any current contents and rescans s for logical
// line starts. It is a convenience wrapper around Rebuild.
func (l *LineByteOffsets) RebuildFromString(s string) {
	_ = l.Rebuild(func(w io.Writer) error {
		_, err := w.Write([]byte(s))
		return err
	})
}

// Reset clears the offsets. After Reset, LineCount returns 0; callers that
// expect at least one line must rebuild first.
func (l *LineByteOffsets) Reset() {
	l.offsets = l.offsets[:0]
}

// Replace updates the offsets to reflect a splice that replaced the source
// text's [start, end) byte range with newText. The caller must invoke
// Replace immediately after applying the same splice to the underlying
// source.
//
// startCtx is the up-to-2-byte slice of pre-splice text immediately before
// start (positions [max(0, start-2), start)); it is used to derive the
// scanner's entry state at start so a partial break ending at the splice
// boundary disambiguates correctly. endCtx is the up-to-3-byte slice of
// post-splice text immediately after newText (positions [start+len(newText),
// start+len(newText)+3)); it provides the lookahead the scanner needs for
// breaks formed at the splice's far boundary. atEOT must be true iff
// endCtx reaches end-of-text (i.e. start+len(newText)+3 ≥ post-splice
// length).
//
// After Replace returns, the offsets describe the post-splice text.
func (l *LineByteOffsets) Replace(newText string, start, end int, startCtx, endCtx string, atEOT bool) {
	n := len(l.offsets)
	if n == 0 {
		return
	}
	delta := len(newText) - (end - start)

	// Derive the scanner's entry state at start from the trailing bytes of
	// startCtx. AfterE280 needs both bytes; the others depend only on the
	// last byte. Anything else leaves the scanner in Normal state.
	state := streamStateNormal
	if m := len(startCtx); m > 0 {
		switch startCtx[m-1] {
		case 0x0d:
			state = streamStateAfterCR
		case 0xc2:
			state = streamStateAfterC2
		case 0xe2:
			state = streamStateAfterE2
		case 0x80:
			if m >= 2 && startCtx[m-2] == 0xe2 {
				state = streamStateAfterE280
			}
		}
	}

	// Replace offsets at indices [replaceStart, shiftStart) by re-emitting
	// them from the rescan. replaceStart starts one past the line
	// containing start (whose offset is ≤ start and stays put) and is
	// pulled back by one when the scanner enters in AfterCR state and
	// offsets[replaceStart-1] equals start — the original break's CRLF/
	// bare-CR disambiguation can flip once the byte at start changes.
	// shiftStart is the first line whose offset is far enough past end
	// (≥ end+3, the longest break length) that no break of any kind
	// can have its lead byte inside the splice and its end byte at
	// offsets[shiftStart]-1.
	startLine := l.LineIndexForByteOffset(start)
	replaceStart := startLine + 1
	if state == streamStateAfterCR && startLine > 0 && l.offsets[startLine] == start {
		replaceStart = startLine
	}
	shiftStart, _ := slices.BinarySearch(l.offsets, end+3)

	var shifted []int
	if shiftStart < n {
		shifted = make([]int, n-shiftStart)
		for i := range shifted {
			shifted[i] = l.offsets[shiftStart+i] + delta
		}
	}

	l.offsets = l.offsets[:replaceStart]
	w := &lineByteOffsetsWriter{l: l, streamOff: start, state: state}
	_, _ = w.Write([]byte(newText + endCtx))
	if atEOT && w.state == streamStateAfterCR {
		l.offsets = append(l.offsets, w.streamOff)
	}
	if len(shifted) > 0 && len(l.offsets) > replaceStart && l.offsets[len(l.offsets)-1] == shifted[0] {
		// Last rescan emit coincides with the first shifted entry (e.g.,
		// a CRLF spanning the rescan boundary); drop the duplicate so
		// the result stays strictly increasing.
		l.offsets = l.offsets[:len(l.offsets)-1]
	}
	l.offsets = append(l.offsets, shifted...)
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
