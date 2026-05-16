// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package textutil

import (
	"fmt"
	"image"
	"iter"
	"math"
	"slices"
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

func advance(str string, indexInBytes int, face text.Face, tabWidth float64, keepTailingSpace bool) float64 {
	end := indexInBytes
	var hasLineBreak bool
	if !keepTailingSpace {
		for end > 0 {
			r, s := utf8.DecodeLastRuneInString(str[:end])
			if s == 0 || !unicode.IsSpace(r) {
				break
			}
			end -= s
		}
	} else if l := tailingLineBreakLen(str[:end]); l > 0 {
		end -= l
		hasLineBreak = true
	}
	if tabWidth == 0 {
		return text.AdvanceAt(str, end, face)
	}
	var width float64
	var pos int
	for pos < end {
		i := strings.IndexByte(str[pos:end], '\t')
		if i < 0 {
			head := str[pos:end]
			width += text.AdvanceAt(head, len(head), face)
			break
		}
		tabIdx := pos + i
		head := str[pos:tabIdx]
		width += text.AdvanceAt(head, len(head), face)
		width = nextIndentPosition(width, tabWidth)
		pos = tabIdx + 1
	}
	if hasLineBreak {
		// Always add the advance of a space for the line break for a consistent behavior.
		width += text.AdvanceAt(" ", 1, face)
	}
	return width
}

// truncateWithEllipsis truncates str so that str + ellipsis fits within maxWidth.
// The truncation is done at grapheme cluster boundaries.
// If ellipsis itself is wider than maxWidth, the ellipsis string is returned as-is.
func truncateWithEllipsis(str string, ellipsis string, maxWidth float64, face text.Face, tabWidth float64) string {
	if advance(str, len(str), face, tabWidth, false) <= maxWidth {
		return str
	}

	// Find the longest prefix of str that fits within maxWidth - ellipsisWidth.
	ellipsisWidth := advance(ellipsis, len(ellipsis), face, tabWidth, false)
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
		if advance(str, candidateEnd, face, tabWidth, false) > targetWidth {
			break
		}
		lastFittingEnd = candidateEnd
		bytePos = candidateEnd
	}

	return str[:lastFittingEnd] + ellipsis
}

type Options struct {
	WrapMode         WrapMode
	Face             text.Face
	LineHeight       float64
	HorizontalAlign  HorizontalAlign
	VerticalAlign    VerticalAlign
	TabWidth         float64
	KeepTailingSpace bool
	EllipsisString   string
}

// WrapMode selects how visual lines wrap when text exceeds the available
// width. The basicwidget package mirrors this enum for its public API.
type WrapMode int

const (
	// WrapModeNone disables automatic wrapping.
	WrapModeNone WrapMode = iota
	// WrapModeWord wraps at Unicode line break opportunities.
	WrapModeWord
	// WrapModeAnywhere wraps at any grapheme cluster boundary.
	WrapModeAnywhere
)

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

// appendVisibleGlyphs appends the shaped clusters of str to glyphs. Each '\t'
// is emitted as a synthetic [text.LazyGlyph] (Image returns nil) whose
// AdvanceX spans to the next tab stop. Glyph rasterization is deferred — this
// function is used only for layout and hit testing, never for drawing.
func appendVisibleGlyphs(glyphs []text.LazyGlyph, str string, face text.Face, tabWidth float64) []text.LazyGlyph {
	var originX float64
	var byteOffset int
	for {
		head, tail, ok := strings.Cut(str, "\t")
		before := len(glyphs)
		glyphs = text.AppendLazyGlyphs(glyphs, head, face, nil)
		for i := before; i < len(glyphs); i++ {
			glyphs[i].StartIndexInBytes += byteOffset
			glyphs[i].EndIndexInBytes += byteOffset
			glyphs[i].OriginX += originX
		}
		byteOffset += len(head)
		if !ok {
			break
		}
		// The guard handles empty heads (leading or consecutive tabs)
		// where AppendLazyGlyphs returned nothing.
		tabStart := originX
		if n := len(glyphs); n > before {
			last := glyphs[n-1]
			tabStart = last.OriginX + last.AdvanceX
		}
		nextX := nextIndentPosition(tabStart, tabWidth)
		// Placeholder glyph for the tab: imager and ImageBounds stay zero so
		// Image() returns nil, but the byte range and advance let hit-testing
		// and cursor positioning treat the tab like a real cluster.
		glyphs = append(glyphs, text.LazyGlyph{
			StartIndexInBytes: byteOffset,
			EndIndexInBytes:   byteOffset + 1,
			OriginX:           tabStart,
			AdvanceX:          nextX - tabStart,
		})
		byteOffset++
		originX = nextX
		str = tail
	}
	return glyphs
}

var theCachedGlyphs []text.LazyGlyph

// indexFromXInVisualLine returns the byte index within vlStr at the cluster
// boundary nearest target, where target is the click X measured from the
// visual line's left edge.
func indexFromXInVisualLine(vlStr string, target float64, options *Options) int {
	theCachedGlyphs = appendVisibleGlyphs(theCachedGlyphs[:0], vlStr, options.Face, options.TabWidth)
	// Drop imager refs on exit so the pooled slice doesn't pin face state.
	defer func() {
		theCachedGlyphs = slices.Delete(theCachedGlyphs, 0, len(theCachedGlyphs))
	}()
	var origin float64
	if len(theCachedGlyphs) > 0 {
		origin = theCachedGlyphs[0].OriginX
	}
	var prevA float64
	for _, c := range theCachedGlyphs {
		a := c.OriginX + c.AdvanceX - origin
		if target < (prevA + (a-prevA)/2) {
			return c.StartIndexInBytes
		}
		prevA = a
	}
	return len(vlStr) - tailingLineBreakLen(vlStr)
}

// visualLine is one rendered row of pixels: the unit yielded by visualLines
// and consumed by Draw / position helpers. With wrapping on, a single
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
// width. For [WrapModeWord] and [WrapModeAnywhere], hard-break-delimited
// segments may further split at width-based wrap opportunities — at Unicode
// line break opportunities for the former, at any grapheme cluster boundary
// for the latter.
func visualLines(width int, str string, wrapMode WrapMode, advance func(str string, indexInBytes int) float64) iter.Seq[visualLine] {
	// Fast path: single visual line that fits within width.
	// Returns a cached iter.Seq to avoid closure allocation.
	if p, _ := FirstLineBreakPositionAndLen(str); p == -1 {
		if wrapMode == WrapModeNone || width == math.MaxInt || advance(str, len(str)) <= float64(width) {
			theCachedSingleVisualLineSeq.visualLine = visualLine{pos: 0, str: str}
			return theCachedSingleVisualLineSeq.seq
		}
	}

	return func(yield func(visualLine) bool) {
		origStr := str

		switch wrapMode {
		case WrapModeNone:
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
		case WrapModeWord, WrapModeAnywhere:
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

			emit := func(segment string, isMandatoryBreak bool) bool {
				if lineEnd-lineStart > 0 {
					candidate := origStr[lineStart : lineEnd+len(segment)]
					// TODO: Consider a line alignment and/or editable/selectable states when calculating the width.
					if advance(candidate, len(candidate)-tailingLineBreakLen(candidate)) > float64(width) {
						if !yield(visualLine{
							pos: pos,
							str: origStr[lineStart:lineEnd],
						}) {
							return false
						}
						pos += lineEnd - lineStart
						lineStart = lineEnd
					}
				}
				lineEnd += len(segment)
				if isMandatoryBreak {
					if !yield(visualLine{
						pos: pos,
						str: origStr[lineStart:lineEnd],
					}) {
						return false
					}
					pos += lineEnd - lineStart
					lineStart = lineEnd
				}
				return true
			}

			if wrapMode == WrapModeWord {
				it := seg.LineIterator()
				for it.Next() {
					l := it.Line()
					if !emit(string(l.Text), l.IsMandatoryBreak) {
						return
					}
				}
			} else {
				it := seg.GraphemeIterator()
				for it.Next() {
					g := it.Grapheme()
					t := string(g.Text)
					// A hard line break is its own grapheme cluster
					// (CRLF stays a single cluster), so detecting a
					// trailing break length covers all hard breaks.
					if !emit(t, tailingLineBreakLen(t) > 0) {
						return
					}
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
	switch hAlign {
	case HorizontalAlignStart, HorizontalAlignLeft:
		// For RTL languages, HorizontalAlignStart should be the same as HorizontalAlignRight.
		return 0
	case HorizontalAlignCenter:
		w := advance(vlStr, len(vlStr)-tailingLineBreakLen(vlStr), face, tabWidth, keepTailingSpace)
		return (float64(width) - w) / 2
	case HorizontalAlignEnd, HorizontalAlignRight:
		// For RTL languages, HorizontalAlignEnd should be the same as HorizontalAlignLeft.
		w := advance(vlStr, len(vlStr)-tailingLineBreakLen(vlStr), face, tabWidth, keepTailingSpace)
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

// TrimPartialUTF8Prefix drops any continuation bytes (10xxxxxx) at the start
// of s that would have belonged to a UTF-8 sequence sliced before s.
func TrimPartialUTF8Prefix(s string) string {
	for len(s) > 0 && !utf8.RuneStart(s[0]) {
		s = s[1:]
	}
	return s
}

// TrimPartialUTF8Suffix drops a partial UTF-8 sequence at the end of s when
// the slice was cut before the sequence's continuation bytes were included.
func TrimPartialUTF8Suffix(s string) string {
	n := len(s)
	if n == 0 || s[n-1] < 0x80 {
		return s
	}
	// Scan back through up to 4 bytes (the UTF-8 maximum) for the sequence's
	// lead byte and check whether enough continuation bytes follow it.
	for i := 1; i <= 4 && i <= n; i++ {
		b := s[n-i]
		if b < 0x80 {
			// ASCII ends its own 1-byte sequence, so the i-1 continuation
			// bytes the walk just passed over (s[n-i+1:n]) have no lead.
			// Keep through the ASCII byte and drop those stray bytes.
			return s[:n-i+1]
		}
		if b < 0xC0 {
			// Continuation byte (10xxxxxx) - keep walking.
			continue
		}
		var expected int
		switch {
		case b >= 0xF0:
			expected = 4
		case b >= 0xE0:
			expected = 3
		default: // b >= 0xC0
			expected = 2
		}
		if i == expected {
			return s
		}
		return s[:n-i]
	}
	return s
}

// visualLineCount returns the number of visual lines str produces at the
// given width.
func visualLineCount(width int, str string, wrapMode WrapMode, face text.Face, tabWidth float64, keepTailingSpace bool) int {
	// Fast path: single visual line that fits within width.
	// This avoids allocating a closure for the advance function.
	if p, _ := FirstLineBreakPositionAndLen(str); p == -1 {
		if wrapMode == WrapModeNone || width == math.MaxInt || advance(str, len(str), face, tabWidth, keepTailingSpace) <= float64(width) {
			return 1
		}
	}

	var count int
	for range visualLines(width, str, wrapMode, func(str string, indexInBytes int) float64 {
		return advance(str, indexInBytes, face, tabWidth, keepTailingSpace)
	}) {
		count++
	}
	return count
}

// MeasureHeight is like [Measure] but only returns height. When width does not
// need to be computed, this avoids per-visual-line shaping calls and is
// dramatically cheaper for very long text (e.g. a multi-megabyte editor
// buffer).
func MeasureHeight(width int, str string, wrapMode WrapMode, face text.Face, lineHeight float64, tabWidth float64, keepTailingSpace bool) float64 {
	return lineHeight * float64(visualLineCount(width, str, wrapMode, face, tabWidth, keepTailingSpace))
}

func Measure(width int, str string, wrapMode WrapMode, face text.Face, lineHeight float64, tabWidth float64, keepTailingSpace bool, ellipsisString string) (float64, float64) {
	var maxWidth, height float64
	for l := range visualLines(width, str, wrapMode, func(str string, indexInBytes int) float64 {
		return advance(str, indexInBytes, face, tabWidth, keepTailingSpace)
	}) {
		vlStr := l.str
		if !keepTailingSpace {
			vlStr = trimTailingLineBreak(vlStr)
		}
		vlWidth := advance(vlStr, len(vlStr), face, tabWidth, keepTailingSpace)
		if ellipsisString != "" && vlWidth > float64(width) {
			vlStr = truncateWithEllipsis(vlStr, ellipsisString, float64(width), face, tabWidth)
			vlWidth = advance(vlStr, len(vlStr), face, tabWidth, false)
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
		c := visualLineCount(size.X, str, options.WrapMode, options.Face, options.TabWidth, options.KeepTailingSpace)
		textHeight := options.LineHeight * float64(c)
		yOffset += (float64(size.Y) - textHeight) / 2
	case VerticalAlignBottom:
		c := visualLineCount(size.X, str, options.WrapMode, options.Face, options.TabWidth, options.KeepTailingSpace)
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
