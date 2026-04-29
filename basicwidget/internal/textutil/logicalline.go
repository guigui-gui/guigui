// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"image"
	"iter"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// A "logical line" is a hard-break-delimited slice of the source text: it
// contains at most one hard line break, and only at its very end. The empty
// string is also a valid logical line. Logical lines are layout-independent;
// the visual lines that result from rendering them depend on the width and
// the autoWrap setting. The functions below are the per-logical-line
// counterparts of the whole-document Measure / Position / Index helpers in
// textutil.go and let callers shape one logical line at a time without
// rescanning the entire document.

// visualLinesFromLogicalLine yields the visual lines that result from rendering
// one logical line at the given width. Positions in the yielded values are
// relative to the start of logicalLine, not to any global document offset.
//
// If autoWrap is false, exactly one visual line is yielded: logicalLine
// itself (including any trailing hard break). If autoWrap is true and the
// line fits within width, exactly one visual line is yielded as well.
// Otherwise the line is wrapped at width-based break opportunities.
//
// An empty logicalLine yields a single empty visual line. A logicalLine that
// contains a mid-line hard break violates the contract; the iterator stops
// at the first mandatory break it encounters.
func visualLinesFromLogicalLine(width int, logicalLine string, autoWrap bool, advance func(str string) float64) iter.Seq[visualLine] {
	// Fast path: a single visual line. Avoids invoking the segmenter for
	// short content that fits, including the empty-line case.
	if !autoWrap || width == math.MaxInt || advance(logicalLine) <= float64(width) {
		return func(yield func(visualLine) bool) {
			yield(visualLine{pos: 0, str: logicalLine})
		}
	}

	return func(yield func(visualLine) bool) {
		seg := pushSegmenter()
		defer popSegmenter()
		// initSegmenterWithString may sanitize the input if it isn't valid UTF-8.
		// Operate on the (possibly sanitized) string so byte offsets reported
		// by the segmenter align with the slices we yield.
		sanitized := initSegmenterWithString(seg, logicalLine)
		it := seg.LineIterator()

		var vlStart, vlEnd, pos int
		for it.Next() {
			l := it.Line()
			segment := string(l.Text)
			if vlEnd-vlStart > 0 {
				candidate := sanitized[vlStart : vlEnd+len(segment)]
				if advance(candidate[:len(candidate)-tailingLineBreakLen(candidate)]) > float64(width) {
					if !yield(visualLine{pos: pos, str: sanitized[vlStart:vlEnd]}) {
						return
					}
					pos += vlEnd - vlStart
					vlStart = vlEnd
				}
			}
			vlEnd += len(segment)
			if l.IsMandatoryBreak {
				// At most one mandatory break is expected, at the very end.
				yield(visualLine{pos: pos, str: sanitized[vlStart:vlEnd]})
				return
			}
		}
		// No trailing break: emit the remaining content as the final visual line.
		if vlEnd-vlStart > 0 {
			yield(visualLine{pos: pos, str: sanitized[vlStart:vlEnd]})
		}
	}
}

// MeasureLogicalLineHeight returns the rendered height of one logical line
// at the given width. This is the per-logical-line counterpart of
// [MeasureHeight] and is used by virtualized layout to size lines one at a
// time without scanning the whole document.
func MeasureLogicalLineHeight(width int, logicalLine string, autoWrap bool, face text.Face, lineHeight float64, tabWidth float64, keepTailingSpace bool) float64 {
	return lineHeight * float64(VisualLineCountForLogicalLine(width, logicalLine, autoWrap, face, tabWidth, keepTailingSpace))
}

// VisualLineCountForLogicalLine returns the number of visual lines one
// logical line wraps into at the given width. With autoWrap off (or when
// the line fits) the result is always 1.
func VisualLineCountForLogicalLine(width int, logicalLine string, autoWrap bool, face text.Face, tabWidth float64, keepTailingSpace bool) int {
	var count int
	for range visualLinesFromLogicalLine(width, logicalLine, autoWrap, func(s string) float64 {
		return advance(s, face, tabWidth, keepTailingSpace)
	}) {
		count++
	}
	return count
}

// MeasureLogicalLine returns the rendered width and height of one logical
// line at the given width. Per-logical-line counterpart of [Measure].
func MeasureLogicalLine(width int, logicalLine string, autoWrap bool, face text.Face, lineHeight float64, tabWidth float64, keepTailingSpace bool, ellipsisString string) (float64, float64) {
	var maxWidth, height float64
	for l := range visualLinesFromLogicalLine(width, logicalLine, autoWrap, func(s string) float64 {
		return advance(s, face, tabWidth, keepTailingSpace)
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
func TextPositionFromIndexInLogicalLine(width int, logicalLine string, index int, options *Options) (position0, position1 TextPosition, count int) {
	if index < 0 || index > len(logicalLine) {
		return TextPosition{}, TextPosition{}, 0
	}
	return textPositionFromIndex(width, logicalLine, slices.Collect(visualLinesFromLogicalLine(width, logicalLine, options.AutoWrap, func(s string) float64 {
		return advance(s, options.Face, options.TabWidth, options.KeepTailingSpace)
	})), index, options)
}

// TextIndexFromPositionInLogicalLine returns the byte offset within one logical line
// closest to the given position. The position's Y is relative to the top of
// the logical line. Counterpart of [TextIndexFromPosition].
func TextIndexFromPositionInLogicalLine(width int, position image.Point, logicalLine string, options *Options) int {
	// Determine the visual line first.
	padding := textPadding(options.Face, options.LineHeight)
	n := int((float64(position.Y) + padding) / options.LineHeight)

	var pos int
	var vlStr string
	var vlIndex int
	for l := range visualLinesFromLogicalLine(width, logicalLine, options.AutoWrap, func(s string) float64 {
		return advance(s, options.Face, options.TabWidth, options.KeepTailingSpace)
	}) {
		vlStr = l.str
		pos = l.pos
		if vlIndex >= n {
			break
		}
		vlIndex++
	}

	// Determine the index within the visual line.
	left := oneLineLeft(width, vlStr, options.Face, options.HorizontalAlign, options.TabWidth, options.KeepTailingSpace)
	var prevA float64
	var clusterFound bool
	for _, c := range visibleCulsters(vlStr, options.Face) {
		a := advance(vlStr[:c.EndIndexInBytes], options.Face, options.TabWidth, true)
		if (float64(position.X) - left) < (prevA + (a-prevA)/2) {
			pos += c.StartIndexInBytes
			clusterFound = true
			break
		}
		prevA = a
	}
	if !clusterFound {
		pos += len(vlStr)
		pos -= tailingLineBreakLen(vlStr)
	}

	return pos
}
