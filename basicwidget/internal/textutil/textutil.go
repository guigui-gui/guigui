// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package textutil

import (
	"fmt"
	"image"
	"iter"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func nextIndentPosition(position float64, indentWidth float64) float64 {
	if indentWidth == 0 {
		return position
	}
	// TODO: The calculation should consider the center and right alignment (#162).
	return float64(int(position/indentWidth)+1) * indentWidth
}

func advance(str string, face text.Face, tabWidth float64, keepTailingSpace bool) float64 {
	var hasLineBreak bool
	if !keepTailingSpace {
		str = strings.TrimRightFunc(str, unicode.IsSpace)
	} else if l := tailingLineBreakLen(str); l > 0 {
		str = str[:len(str)-l]
		hasLineBreak = true
	}
	if tabWidth == 0 {
		return text.Advance(str, face)
	}
	var width float64
	for {
		head, tail, ok := strings.Cut(str, "\t")
		width += text.Advance(head, face)
		if !ok {
			break
		}
		width = nextIndentPosition(width, tabWidth)
		str = tail
	}
	if hasLineBreak {
		// Always add the advance of a space for the line break for a consistent behavior.
		width += text.Advance(" ", face)
	}
	return width
}

// truncateWithEllipsis truncates str so that str + ellipsis fits within maxWidth.
// The truncation is done at grapheme cluster boundaries.
// If ellipsis itself is wider than maxWidth, the ellipsis string is returned as-is.
func truncateWithEllipsis(str string, ellipsis string, maxWidth float64, face text.Face, tabWidth float64) string {
	if advance(str, face, tabWidth, false) <= maxWidth {
		return str
	}

	// Find the longest prefix of str that fits within maxWidth - ellipsisWidth.
	ellipsisWidth := advance(ellipsis, face, tabWidth, false)
	targetWidth := maxWidth - ellipsisWidth
	if targetWidth <= 0 {
		return ellipsis
	}

	var lastFittingEnd int
	seg := pushSegmenter()
	defer popSegmenter()
	str = initSegmenterWithString(seg, str)
	it := seg.GraphemeIterator()
	var bytePos int
	for it.Next() {
		g := it.Grapheme()
		s := string(g.Text)
		candidateEnd := bytePos + len(s)
		if advance(str[:candidateEnd], face, tabWidth, false) > targetWidth {
			break
		}
		lastFittingEnd = candidateEnd
		bytePos = candidateEnd
	}

	return str[:lastFittingEnd] + ellipsis
}

type Options struct {
	AutoWrap         bool
	Face             text.Face
	LineHeight       float64
	HorizontalAlign  HorizontalAlign
	VerticalAlign    VerticalAlign
	TabWidth         float64
	KeepTailingSpace bool
	EllipsisString   string
}

type HorizontalAlign int

const (
	HorizontalAlignStart HorizontalAlign = iota
	HorizontalAlignCenter
	HorizontalAlignEnd
	HorizontalAlignLeft
	HorizontalAlignRight
)

type VerticalAlign int

const (
	VerticalAlignTop VerticalAlign = iota
	VerticalAlignMiddle
	VerticalAlignBottom
)

func visibleCulsters(str string, face text.Face) []text.Glyph {
	return text.AppendGlyphs(nil, str, face, nil)
}

// visualLine is one rendered row of pixels: the unit yielded by visualLines
// and consumed by Draw / position helpers. With autoWrap on, a single
// hard-break-delimited "logical line" can decompose into multiple visualLines.
type visualLine struct {
	pos int
	str string
}

// cachedSingleVisualLineSeq avoids closure allocation for the common
// single-visual-line case. The value is set before returning seq, and seq
// yields it without capturing any per-call state.
type cachedSingleVisualLineSeq struct {
	visualLine visualLine
	seq        iter.Seq[visualLine]
}

var theCachedSingleVisualLineSeq cachedSingleVisualLineSeq

func init() {
	theCachedSingleVisualLineSeq.seq = func(yield func(visualLine) bool) {
		yield(theCachedSingleVisualLineSeq.visualLine)
	}
}

// visualLines yields the visual sublines (rendered rows) of str at the given
// width. With autoWrap on, hard-break-delimited segments may further split
// at width-based wrap opportunities.
func visualLines(width int, str string, autoWrap bool, advance func(str string) float64) iter.Seq[visualLine] {
	// Fast path: single visual line that fits within width.
	// Returns a cached iter.Seq to avoid closure allocation.
	if p, _ := FirstLineBreakPositionAndLen(str); p == -1 {
		if !autoWrap || width == math.MaxInt || advance(str) <= float64(width) {
			theCachedSingleVisualLineSeq.visualLine = visualLine{pos: 0, str: str}
			return theCachedSingleVisualLineSeq.seq
		}
	}

	return func(yield func(visualLine) bool) {
		origStr := str

		if !autoWrap {
			var pos int
			for pos < len(str) {
				p, l := FirstLineBreakPositionAndLen(str[pos:])
				if p == -1 {
					if !yield(visualLine{
						pos: pos,
						str: str[pos:],
					}) {
						return
					}
					break
				}
				if !yield(visualLine{
					pos: pos,
					str: str[pos : pos+p+l],
				}) {
					return
				}
				pos += p + l
			}
		} else {
			var lineStart int
			var lineEnd int
			var pos int

			seg := pushSegmenter()
			defer popSegmenter()
			// If str is not valid UTF-8, the sanitized string returned here
			// is what the segmenter actually iterates over. Use it as origStr
			// so byte offsets from the segmenter align with the string being
			// sliced.
			str = initSegmenterWithString(seg, str)
			origStr = str
			it := seg.LineIterator()
			for it.Next() {
				l := it.Line()
				segment := string(l.Text)
				if lineEnd-lineStart > 0 {
					candidate := origStr[lineStart : lineEnd+len(segment)]
					// TODO: Consider a line alignment and/or editable/selectable states when calculating the width.
					if advance(candidate[:len(candidate)-tailingLineBreakLen(candidate)]) > float64(width) {
						if !yield(visualLine{
							pos: pos,
							str: origStr[lineStart:lineEnd],
						}) {
							return
						}
						pos += lineEnd - lineStart
						lineStart = lineEnd
					}
				}
				lineEnd += len(segment)
				if l.IsMandatoryBreak {
					if !yield(visualLine{
						pos: pos,
						str: origStr[lineStart:lineEnd],
					}) {
						return
					}
					pos += lineEnd - lineStart
					lineStart = lineEnd
				}
			}

			if lineEnd-lineStart > 0 {
				if !yield(visualLine{
					pos: pos,
					str: origStr[lineStart:lineEnd],
				}) {
					return
				}
				pos += lineEnd - lineStart
				lineStart = lineEnd
			}
		}

		// If the string ends with a line break, or an empty line, add an extra empty line.
		if tailingLineBreakLen(origStr) > 0 || origStr == "" {
			if !yield(visualLine{
				pos: len(origStr),
			}) {
				return
			}
		}
	}
}

func oneLineLeft(width int, vlStr string, face text.Face, hAlign HorizontalAlign, tabWidth float64, keepTailingSpace bool) float64 {
	w := advance(vlStr[:len(vlStr)-tailingLineBreakLen(vlStr)], face, tabWidth, keepTailingSpace)
	switch hAlign {
	case HorizontalAlignStart, HorizontalAlignLeft:
		// For RTL languages, HorizontalAlignStart should be the same as HorizontalAlignRight.
		return 0
	case HorizontalAlignCenter:
		return (float64(width) - w) / 2
	case HorizontalAlignEnd, HorizontalAlignRight:
		// For RTL languages, HorizontalAlignEnd should be the same as HorizontalAlignLeft.
		return float64(width) - w
	default:
		panic(fmt.Sprintf("textutil: invalid HorizontalAlign: %d", hAlign))
	}
}

func FirstLineBreakPositionAndLen(str string) (pos, length int) {
	// Find the earliest line-break lead byte using strings.IndexByte
	// (SIMD-accelerated). Each scan operates on a shrinking prefix, so
	// once a common newline (LF) is found, the remaining searches for
	// rarer line-break characters cover only the bytes before it.
	bestPos := -1
	bestLen := 0
	upper := len(str)

	// 0x0A LF — most common in editor text; scan first to bound the rest.
	if i := strings.IndexByte(str, 0x0a); i >= 0 {
		bestPos = i
		bestLen = 1
		upper = i
	}

	// 0x0D CR (possibly part of CRLF).
	if upper > 0 {
		if i := strings.IndexByte(str[:upper], 0x0d); i >= 0 {
			n := 1
			if i+1 < len(str) && str[i+1] == 0x0a {
				n = 2
			}
			bestPos = i
			bestLen = n
			upper = i
		}
	}

	// 0x0B VT, 0x0C FF — rare.
	if upper > 0 {
		if i := strings.IndexByte(str[:upper], 0x0b); i >= 0 {
			bestPos = i
			bestLen = 1
			upper = i
		}
	}
	if upper > 0 {
		if i := strings.IndexByte(str[:upper], 0x0c); i >= 0 {
			bestPos = i
			bestLen = 1
			upper = i
		}
	}

	// 0xC2 — lead of NEL (U+0085, encoded as C2 85). 0xC2 followed by any
	// other byte is a different code point (e.g. NBSP), so loop past
	// false positives.
	for from := 0; from < upper; {
		i := strings.IndexByte(str[from:upper], 0xc2)
		if i < 0 {
			break
		}
		idx := from + i
		if idx+1 < len(str) && str[idx+1] == 0x85 {
			bestPos = idx
			bestLen = 2
			upper = idx
			break
		}
		from = idx + 1
	}

	// 0xE2 — lead of LS/PS (U+2028/2029, encoded as E2 80 A8/A9). Same
	// false-positive handling: 0xE2 also leads many other code points
	// (including common CJK punctuation), so step past mismatches.
	for from := 0; from < upper; {
		i := strings.IndexByte(str[from:upper], 0xe2)
		if i < 0 {
			break
		}
		idx := from + i
		if idx+2 < len(str) && str[idx+1] == 0x80 {
			if c := str[idx+2]; c == 0xa8 || c == 0xa9 {
				bestPos = idx
				bestLen = 3
				upper = idx
				break
			}
		}
		from = idx + 1
	}

	return bestPos, bestLen
}

// LastLineBreakPositionAndLen returns the position and the byte length of the last line break in str.
// If no line break is found, it returns (-1, 0).
func LastLineBreakPositionAndLen(str string) (pos, length int) {
	for i := len(str); i > 0; {
		r, s := utf8.DecodeLastRuneInString(str[:i])
		if s == 0 {
			break
		}
		i -= s
		if r == 0x000b || r == 0x000c {
			return i, 1
		}
		if r == 0x0085 {
			return i, 2
		}
		if r == 0x2028 || r == 0x2029 {
			return i, 3
		}
		if r == 0x000a {
			// \r\n
			if i > 0 {
				r2, s2 := utf8.DecodeLastRuneInString(str[:i])
				if s2 > 0 && r2 == 0x000d {
					return i - s2, 2
				}
			}
			return i, 1
		}
		if r == 0x000d {
			return i, 1
		}
	}
	return -1, 0
}

func tailingLineBreakLen(str string) int {
	// Hard-code the check here.
	// See also: https://en.wikipedia.org/wiki/Newline#Unicode
	if r, s := utf8.DecodeLastRuneInString(str); s > 0 {
		if r == 0x000b || r == 0x000c || r == 0x000d || r == 0x0085 || r == 0x2028 || r == 0x2029 {
			return s
		}
		if r == 0x000a {
			// \r\n
			if r, s := utf8.DecodeLastRuneInString(str[:len(str)-s]); s > 0 && r == 0x000d {
				return 2
			}
			return 1
		}
	}
	return 0
}

func trimTailingLineBreak(str string) string {
	for {
		c := tailingLineBreakLen(str)
		if c == 0 {
			break
		}
		str = str[:len(str)-c]
	}
	return str
}

// visualLineCount returns the number of visual lines str produces at the
// given width.
func visualLineCount(width int, str string, autoWrap bool, face text.Face, tabWidth float64, keepTailingSpace bool) int {
	// Fast path: single visual line that fits within width.
	// This avoids allocating a closure for the advance function.
	if p, _ := FirstLineBreakPositionAndLen(str); p == -1 {
		if !autoWrap || width == math.MaxInt || advance(str, face, tabWidth, keepTailingSpace) <= float64(width) {
			return 1
		}
	}

	var count int
	for range visualLines(width, str, autoWrap, func(str string) float64 {
		return advance(str, face, tabWidth, keepTailingSpace)
	}) {
		count++
	}
	return count
}

// MeasureHeight is like [Measure] but only returns height. When width does not
// need to be computed, this avoids per-visual-line shaping calls and is
// dramatically cheaper for very long text (e.g. a multi-megabyte editor
// buffer).
func MeasureHeight(width int, str string, autoWrap bool, face text.Face, lineHeight float64, tabWidth float64, keepTailingSpace bool) float64 {
	return lineHeight * float64(visualLineCount(width, str, autoWrap, face, tabWidth, keepTailingSpace))
}

func Measure(width int, str string, autoWrap bool, face text.Face, lineHeight float64, tabWidth float64, keepTailingSpace bool, ellipsisString string) (float64, float64) {
	var maxWidth, height float64
	for l := range visualLines(width, str, autoWrap, func(str string) float64 {
		return advance(str, face, tabWidth, keepTailingSpace)
	}) {
		vlStr := l.str
		if !keepTailingSpace {
			vlStr = trimTailingLineBreak(vlStr)
		}
		vlWidth := advance(vlStr, face, tabWidth, keepTailingSpace)
		if ellipsisString != "" && vlWidth > float64(width) {
			vlStr = truncateWithEllipsis(vlStr, ellipsisString, float64(width), face, tabWidth)
			vlWidth = advance(vlStr, face, tabWidth, false)
		}
		maxWidth = max(maxWidth, vlWidth)
		// The text is already shifted by (lineHeight - (m.HAscent + m.Descent)) / 2.
		// Thus, just counting the visual line number is enough.
		height += lineHeight
	}
	return maxWidth, height
}

func textPadding(face text.Face, lineHeight float64) float64 {
	m := face.Metrics()
	padding := (lineHeight - (m.HAscent + m.HDescent)) / 2
	return padding
}

func textPositionYOffset(size image.Point, str string, options *Options) float64 {
	yOffset := textPadding(options.Face, options.LineHeight)
	switch options.VerticalAlign {
	case VerticalAlignTop:
	case VerticalAlignMiddle:
		c := visualLineCount(size.X, str, options.AutoWrap, options.Face, options.TabWidth, options.KeepTailingSpace)
		textHeight := options.LineHeight * float64(c)
		yOffset += (float64(size.Y) - textHeight) / 2
	case VerticalAlignBottom:
		c := visualLineCount(size.X, str, options.AutoWrap, options.Face, options.TabWidth, options.KeepTailingSpace)
		textHeight := options.LineHeight * float64(c)
		yOffset += float64(size.Y) - textHeight
	}
	return yOffset
}

func FindWordBoundaries(text string, idx int) (start, end int) {
	seg := pushSegmenter()
	defer popSegmenter()
	if sanitized := initSegmenterWithString(seg, text); sanitized != text {
		// Invalid UTF-8: byte offsets reported by the segmenter are into
		// the sanitized string and would not be meaningful in the original.
		return idx, idx
	}
	it := seg.WordIterator()

	for it.Next() {
		w := it.Word()
		wordStart := w.OffsetInBytes
		wordEnd := w.OffsetInBytes + w.LengthInBytes
		if wordStart <= idx && idx <= wordEnd {
			return wordStart, wordEnd
		}
	}

	return idx, idx
}
