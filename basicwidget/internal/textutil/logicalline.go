// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"image"
	"iter"
	"math"
	"slices"
	"unicode/utf8"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
)

// A "logical line" is a hard-break-delimited slice of the source text: it
// contains at most one hard line break, and only at its very end. The empty
// string is also a valid logical line. Logical lines are layout-independent;
// the visual lines that result from rendering them depend on the width and
// the wrap mode. The functions below are the per-logical-line counterparts
// of the whole-document Measure / Position / Index helpers in textutil.go
// and let callers shape one logical line at a time without rescanning the
// entire document.

// visualLinesFromLogicalLine yields the visual lines that result from rendering
// one logical line at the given width. Positions in the yielded values are
// relative to the start of logicalLine, not to any global document offset.
//
// If wrapMode is [WrapModeNone], exactly one visual line is yielded: logicalLine
// itself (including any trailing hard break). For other wrap modes, if the
// line fits within width, exactly one visual line is yielded as well.
// Otherwise the line is wrapped at break opportunities determined by wrapMode.
//
// An empty logicalLine yields a single empty visual line. A logicalLine that
// contains a mid-line hard break violates the contract; the iterator stops
// at the first mandatory break it encounters.
func visualLinesFromLogicalLine(width int, logicalLine string, wrapMode WrapMode, advance func(str string, indexInBytes int) float64) iter.Seq[visualLine] {
	// Fast path: a single visual line. Avoids invoking the segmenter for
	// short content that fits, including the empty-line case.
	if wrapMode == WrapModeNone || width == math.MaxInt || advance(logicalLine, len(logicalLine)) <= float64(width) {
		return func(yield func(visualLine) bool) {
			yield(visualLine{pos: 0, str: logicalLine})
		}
	}

	// This per-logical-line path does not use the layout cache: it has only an
	// advance closure (the cache is keyed on a font.Face), and it serves the
	// content the cache rejects. The cache requires valid UTF-8, so operate on
	// the sanitized copy; the yielded slices index into it.
	sanitized := sanitizedForCache(logicalLine)
	return visualLinesFromStarts(sanitized, visualLineStarts(width, sanitized, wrapMode, advance))
}

// visualLineStarts yields the start byte offset of each visual line that line
// wraps into at the given width, in order. line must be valid UTF-8. The first
// value is always 0, the number of values equals the visual-line count, and
// visual line i spans line[start_i:start_{i+1}] (the last spans to len(line)).
// This describes exactly the same wrapping as visualLinesFromLogicalLine.
func visualLineStarts(width int, line string, wrapMode WrapMode, advance func(str string, indexInBytes int) float64) iter.Seq[int] {
	return func(yield func(int) bool) {
		// Fast path: a single visual line. Mirrors visualLinesFromLogicalLine.
		if wrapMode == WrapModeNone || width == math.MaxInt || advance(line, len(line)) <= float64(width) {
			yield(0)
			return
		}

		if !yield(0) {
			return
		}
		var vlStart, vlEnd int
		// emit consumes the next segment, identified only by its byte length, and
		// starts a new visual line whenever the accumulated content would overflow
		// width. Returns cont=false at a mandatory break (the contract allows at
		// most one, at the very end) or once the consumer stops. A mandatory break
		// never starts a fresh visual line: its start offset is already yielded.
		emit := func(segLenInBytes int, isMandatoryBreak bool) (cont bool) {
			if vlEnd-vlStart > 0 {
				candidate := line[vlStart : vlEnd+segLenInBytes]
				if advance(candidate, len(candidate)-tailingLineBreakLen(candidate)) > float64(width) {
					vlStart = vlEnd
					if !yield(vlStart) {
						return false
					}
				}
			}
			vlEnd += segLenInBytes
			return !isMandatoryBreak
		}

		// WrapModeNormal wraps at line-break opportunities, WrapModeAnywhere at
		// grapheme boundaries; both feed the same packing loop. A logical line
		// has at most a trailing hard break, so the mandatory-break flag is
		// taken from each segment's own trailing line break.
		boundaries := theSegmentCache.softLineBreakBoundaries
		if wrapMode != WrapModeNormal {
			boundaries = theSegmentCache.graphemeBoundaries
		}
		var segStart int
		for end := range boundaries(line) {
			segText := line[segStart:end]
			if !emit(end-segStart, tailingLineBreakLen(segText) > 0) {
				break
			}
			segStart = end
		}
	}
}

// visualLinesFromStarts yields the visual lines described by the start offsets in
// vlStarts (as produced by visualLineStarts) over line. line must be
// the same string the offsets were computed against. No shaping is performed.
func visualLinesFromStarts(line string, vlStarts iter.Seq[int]) iter.Seq[visualLine] {
	return func(yield func(visualLine) bool) {
		started := false
		var prev int
		for s := range vlStarts {
			if !started {
				prev = s
				started = true
				continue
			}
			if !yield(visualLine{pos: prev, str: line[prev:s]}) {
				return
			}
			prev = s
		}
		if started {
			yield(visualLine{pos: prev, str: line[prev:]})
		}
	}
}

// cachedVisualLineStarts returns the visual-line start offsets for line at the
// given layout parameters, memoized by content in [theLayoutCache]. The face's
// recipe fingerprints the entry; a face with no recipe (zero attributes)
// computes without caching. ok is false for non-UTF-8 lines (whose offsets would
// index into a sanitized copy, not line) so callers fall back to their shaping
// path; in that case vlStarts is nil.
func cachedVisualLineStarts(width int, line string, wrapMode WrapMode, face font.Face, tabWidth float64, keepTailingSpace bool) (vlStarts []int, ok bool) {
	k := layoutKey{
		text:             line,
		faceID:           face.ID(),
		width:            width,
		wrapMode:         wrapMode,
		tabWidth:         tabWidth,
		keepTailingSpace: keepTailingSpace,
	}
	if s, hit := theLayoutCache.get(k); hit {
		return s, true
	}
	if !utf8.ValidString(line) {
		return nil, false
	}
	// line is valid UTF-8, so the segmenter can run on it directly and the
	// offsets index into line without a sanitized copy.
	tf := face.TextFace()
	vlStarts = slices.Collect(visualLineStarts(width, line, wrapMode, func(s string, indexInBytes int) float64 {
		return advance(s, indexInBytes, tf, tabWidth, keepTailingSpace)
	}))
	theLayoutCache.put(k, vlStarts)
	return vlStarts, true
}

// MeasureLogicalLineHeight returns the rendered height of one logical line
// at the given width. This is the per-logical-line counterpart of
// [MeasureHeight] and is used by virtualized layout to size lines one at a
// time without scanning the whole document.
func MeasureLogicalLineHeight(width int, logicalLine string, wrapMode WrapMode, face font.Face, lineHeight float64, tabWidth float64, keepTailingSpace bool) float64 {
	return lineHeight * float64(VisualLineCountForLogicalLine(width, logicalLine, wrapMode, face, tabWidth, keepTailingSpace))
}

// VisualLineCountForLogicalLine returns the number of visual lines one
// logical line wraps into at the given width. With wrapMode set to
// [WrapModeNone] (or when the line fits) the result is always 1.
func VisualLineCountForLogicalLine(width int, logicalLine string, wrapMode WrapMode, face font.Face, tabWidth float64, keepTailingSpace bool) int {
	var count int
	for range visualLineStarts(width, sanitizedForCache(logicalLine), wrapMode, func(s string, indexInBytes int) float64 {
		return advance(s, indexInBytes, face.TextFace(), tabWidth, keepTailingSpace)
	}) {
		count++
	}
	return count
}

// CachedVisualLineCount is [VisualLineCountForLogicalLine] backed by the
// content-keyed layout cache. For [WrapModeNone] (no packing) or a non-UTF-8
// line it falls back to the uncached count. Use this for per-tick height
// measurement of a logical line whose wrap layout the other cached paths (draw,
// caret, hit-test) also touch, so they share one cache entry.
func CachedVisualLineCount(width int, logicalLine string, wrapMode WrapMode, face font.Face, tabWidth float64, keepTailingSpace bool) int {
	if wrapMode != WrapModeNone {
		if vlStarts, ok := cachedVisualLineStarts(width, logicalLine, wrapMode, face, tabWidth, keepTailingSpace); ok {
			return len(vlStarts)
		}
	}
	return VisualLineCountForLogicalLine(width, logicalLine, wrapMode, face, tabWidth, keepTailingSpace)
}

// MeasureLogicalLine returns the rendered width and height of one logical
// line at the given width. Per-logical-line counterpart of [Measure].
func MeasureLogicalLine(width int, logicalLine string, wrapMode WrapMode, face font.Face, lineHeight float64, tabWidth float64, keepTailingSpace bool, ellipsisString string) (float64, float64) {
	var maxWidth, height float64
	for l := range visualLinesFromLogicalLine(width, logicalLine, wrapMode, func(s string, indexInBytes int) float64 {
		return advance(s, indexInBytes, face.TextFace(), tabWidth, keepTailingSpace)
	}) {
		vlStr := l.str
		if !keepTailingSpace {
			vlStr = trimTailingLineBreak(vlStr)
		}
		vlWidth := advance(vlStr, len(vlStr), face.TextFace(), tabWidth, keepTailingSpace)
		if ellipsisString != "" && vlWidth > float64(width) {
			vlStr = truncateWithEllipsis(vlStr, ellipsisString, float64(width), face.TextFace(), tabWidth)
			vlWidth = advance(vlStr, len(vlStr), face.TextFace(), tabWidth, false)
		}
		maxWidth = max(maxWidth, vlWidth)
		height += lineHeight
	}
	return maxWidth, height
}

// TextPositionFromIndexInLogicalLine returns the visual position(s) within one logical
// line corresponding to the given byte index inside that line. The Y values
// are relative to the top of the logical line (so the caller can offset them
// by the line's origin Y). Counterpart of [TextPositionFromIndex].
//
// index is a byte offset in [0, len(logicalLine)]. Out-of-range values yield
// (TextPosition{}, TextPosition{}, 0).
func TextPositionFromIndexInLogicalLine(width int, logicalLine string, index int, style *Style) (position0, position1 TextPosition, count int) {
	if index < 0 || index > len(logicalLine) {
		return TextPosition{}, TextPosition{}, 0
	}
	return textPositionFromIndexInVisualLines(width, visualLinesFromLogicalLine(width, logicalLine, style.WrapMode, func(s string, indexInBytes int) float64 {
		return advance(s, indexInBytes, style.Face.TextFace(), style.TabWidth, style.KeepTailingSpace)
	}), index, style)
}

// TextIndexFromPositionInLogicalLine returns the byte offset within one logical line
// closest to the given position. The position's Y is relative to the top of
// the logical line. Counterpart of [TextIndexFromPosition].
func TextIndexFromPositionInLogicalLine(width int, position image.Point, logicalLine string, style *Style) int {
	return textIndexFromPositionInVisualLines(width, position, visualLinesFromLogicalLine(width, logicalLine, style.WrapMode, func(s string, indexInBytes int) float64 {
		return advance(s, indexInBytes, style.Face.TextFace(), style.TabWidth, style.KeepTailingSpace)
	}), style)
}
