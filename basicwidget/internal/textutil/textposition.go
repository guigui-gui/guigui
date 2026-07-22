// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"image"
	"iter"
	"math"
)

type TextPosition struct {
	X      float64
	Top    float64
	Bottom float64
}

// logicalLineAndCaretPosition maps index to its logical line through m, shapes that
// one line, and returns the line-local caret position(s). pos0 and pos1 are
// line-local: Top and Bottom are measured from the line's top, not the
// document top. count is 1, or 2 at a soft-wrap boundary; count==0 means
// index is out of range.
func logicalLineAndCaretPosition(m *logicalLineMeasurer, p *TextLayoutParams, index int) (logicalLineIdx, indexInLine int, pos0, pos1 TextPosition, count int) {
	if index < 0 || index > p.RenderingTextLength {
		return 0, 0, TextPosition{}, TextPosition{}, 0
	}
	logicalLineIdx = m.logicalLineIndexForRenderingIndex(index)
	renderingLineStart, renderingLineEnd := m.renderingRange(logicalLineIdx)
	line := p.RenderingTextRange(renderingLineStart, renderingLineEnd)
	indexInLine = index - renderingLineStart

	if p.Style.WrapMode != WrapModeNone && !faceRunsIntersect(p.Style.FaceRuns, renderingLineStart, renderingLineStart+len(line)) {
		if vlStarts, ok := cachedVisualLineStarts(p.Width, line, p.Style.WrapMode, p.Style.Face, p.Style.TabWidth, p.Style.KeepTailingSpace); ok {
			pos0, pos1, count = textPositionFromIndexInVisualLines(p.Width, visualLinesFromStarts(line, vlStarts), renderingLineStart, indexInLine, &p.Style)
			if count == 0 {
				return 0, 0, TextPosition{}, TextPosition{}, 0
			}
			return logicalLineIdx, indexInLine, pos0, pos1, count
		}
	}

	pos0, pos1, count = TextPositionFromIndexInLogicalLine(p.Width, line, renderingLineStart, indexInLine, &p.Style)
	if count == 0 {
		return 0, 0, TextPosition{}, TextPosition{}, 0
	}
	return logicalLineIdx, indexInLine, pos0, pos1, count
}

// PositionWithinLogicalLine returns the caret's logical-line index and its
// visual position(s). pos.Top / pos.Bottom are measured from the start of the
// line at lineIdx, not the document top.
//
// count==0 when the result is unavailable: index out of range, no precomputed
// logical-line offsets, empty document, or composition straddling a logical-line
// boundary. Callers needing the slow whole-document fallback in that case should
// call [TextPositionFromIndex].
func PositionWithinLogicalLine(p *TextLayoutParams, index int) (lineIdx int, position0, position1 TextPosition, count int) {
	m, ok := newLogicalLineMeasurer(p)
	if !ok {
		return 0, TextPosition{}, TextPosition{}, 0
	}
	logicalLineIdx, _, pos0, pos1, c := logicalLineAndCaretPosition(m, p, index)
	if c == 0 {
		return 0, TextPosition{}, TextPosition{}, 0
	}
	return logicalLineIdx, pos0, pos1, c
}

// TextPositionFromIndex returns the visual position(s) for index in the
// rendering text. The Y origin is the visual line at
// (p.LogicalLineIndexHint, p.VisualLineIndexHint); count is 1, or 2 at line-
// break boundaries.
func TextPositionFromIndex(p *TextLayoutParams, index int) (position0, position1 TextPosition, count int) {
	m, ok := newLogicalLineMeasurer(p)
	if !ok {
		str := p.RenderingTextRange(0, p.RenderingTextLength)
		vls := visualLines(p.Width, str, p.Style.WrapMode, func(s string, strStartInBytes, endIndexInBytes int) float64 {
			return advanceWithFaces(s, strStartInBytes, endIndexInBytes, p.Style.Face, p.Style.FaceRuns, p.Style.TabWidth, p.Style.KeepTailingSpace)
		})
		return textPositionFromIndexInVisualLines(p.Width, vls, 0, index, &p.Style)
	}

	logicalLineIdx, indexInLine, pos0, pos1, c := logicalLineAndCaretPosition(m, p, index)
	if c == 0 {
		return TextPosition{}, TextPosition{}, 0
	}

	// visualLineIndexAt walks from the caller-supplied hint to
	// targetLine, accumulating per-line wrap counts so the result
	// is the visual-line index where targetLine starts in the
	// caller's coordinate system.
	hintLine := min(max(p.LogicalLineIndexHint, 0), m.logicalLineCount-1)
	visualLineIndexAt := func(targetLine int) int {
		v := p.VisualLineIndexHint
		if targetLine == hintLine {
			return v
		}
		if targetLine > hintLine {
			for i := hintLine; i < targetLine; i++ {
				v += m.visualLineCount(i)
			}
			return v
		}
		for i := hintLine - 1; i >= targetLine; i-- {
			v -= m.visualLineCount(i)
		}
		return v
	}
	precedingVisualLines := visualLineIndexAt(logicalLineIdx)
	yOffset := p.Style.LineHeight * float64(precedingVisualLines)

	pos0.Top += yOffset
	pos0.Bottom += yOffset
	if c == 2 {
		pos1.Top += yOffset
		pos1.Bottom += yOffset
	}

	// Hard-line-break boundary: when index is at the very start of a non-
	// first logical line, the unrestricted walk reports two positions —
	// tail of the previous line plus head of this one. The per-logical
	// call only sees the head (c == 1, with pos0 at indexInLine==0). Pull
	// the tail position from the previous logical line and rebuild as
	// (pos0=tail, pos1=head, count=2). Soft-wrap boundaries within a
	// single logical line are already handled by
	// [TextPositionFromIndexInLogicalLine].
	if c == 1 && indexInLine == 0 && logicalLineIdx > 0 {
		prevLogicalLineIdx := logicalLineIdx - 1
		prevRenderingLineStart, prevRenderingLineEnd := m.renderingRange(prevLogicalLineIdx)
		prevLine := p.RenderingTextRange(prevRenderingLineStart, prevRenderingLineEnd)
		prevPos0, _, prevCount := TextPositionFromIndexInLogicalLine(p.Width, prevLine, prevRenderingLineStart, len(prevLine), &p.Style)
		if prevCount > 0 {
			prevYOffset := p.Style.LineHeight * float64(visualLineIndexAt(prevLogicalLineIdx))
			prevPos0.Top += prevYOffset
			prevPos0.Bottom += prevYOffset
			pos1 = pos0
			pos0 = prevPos0
			c = 2
		}
	}
	return pos0, pos1, c
}

// textPositionFromIndexInVisualLines returns the visual position(s) at byte
// offset index within the visual lines vls, where index is relative to vls'
// first byte. style's face runs use whole-text byte offsets; vlsStartInBytes
// is the whole-text byte offset of vls' first byte, rebasing vls-relative
// offsets for the face-run lookups. count is 1, or 2 when index lands on a
// line-break boundary, in which case position0 is the tail of one visual
// line and position1 the head of the next. An out-of-range index yields
// count 0.
func textPositionFromIndexInVisualLines(width int, vls iter.Seq[visualLine], vlsStartInBytes, index int, style *Style) (position0, position1 TextPosition, count int) {
	var y, y0, y1 float64
	// indexInVisualLine0/1 are index relative to the matched visual line's
	// start.
	var indexInVisualLine0, indexInVisualLine1 int
	// vlStartInBytes0/1 are the matched visual line's start in the whole
	// text.
	var vlStartInBytes0, vlStartInBytes1 int
	var line0, line1 string
	var found0, found1 bool
	for l := range vls {
		// When auto wrap is on or the string ends with a line break, there can be two positions:
		// one in the tail of the previous line and one in the head of the next line.
		if index == l.pos+len(l.str) {
			if !found0 {
				found0 = true
				line0 = l.str
				indexInVisualLine0 = index - l.pos
				vlStartInBytes0 = vlsStartInBytes + l.pos
				y0 = y
			} else {
				// A previous line already matched as the tail position; this line
				// (typically an empty trailing line for a string ending in a line break)
				// is the head of the next line.
				found1 = true
				line1 = l.str
				indexInVisualLine1 = index - l.pos
				vlStartInBytes1 = vlsStartInBytes + l.pos
				y1 = y
				break
			}
		} else if l.pos <= index && index < l.pos+len(l.str) {
			found1 = true
			line1 = l.str
			indexInVisualLine1 = index - l.pos
			vlStartInBytes1 = vlsStartInBytes + l.pos
			y1 = y
			break
		}
		y += style.LineHeight
	}

	if !found0 && !found1 {
		return TextPosition{}, TextPosition{}, 0
	}

	paddingY := textPadding(style.Face.TextFace(), style.LineHeight)

	var pos0, pos1 TextPosition
	if found0 {
		x0 := oneLineLeft(width, line0, vlStartInBytes0, style)
		x0 += advanceWithFaces(line0, vlStartInBytes0, indexInVisualLine0, style.Face, style.FaceRuns, style.TabWidth, true)
		pos0 = TextPosition{
			X:      x0,
			Top:    y0 + paddingY,
			Bottom: y0 + style.LineHeight - paddingY,
		}
	}
	if found1 {
		x1 := oneLineLeft(width, line1, vlStartInBytes1, style)
		x1 += advanceWithFaces(line1, vlStartInBytes1, indexInVisualLine1, style.Face, style.FaceRuns, style.TabWidth, true)
		pos1 = TextPosition{
			X:      x1,
			Top:    y1 + paddingY,
			Bottom: y1 + style.LineHeight - paddingY,
		}
	}
	if found0 && !found1 {
		return pos0, TextPosition{}, 1
	}
	if found1 && !found0 {
		return pos1, TextPosition{}, 1
	}
	return pos0, pos1, 2
}

// AppendBoundsOfTextRange appends the bounding rectangles covering the byte
// range [start, end) of the rendering text to dst and returns the extended
// slice, one rectangle per crossed visual line, in order. A visual line
// holding an empty part of the range appends nothing. Coordinates are
// relative to the text layout origin, rounded outward to integers.
// Endpoints outside the text are clamped.
func AppendBoundsOfTextRange(dst []image.Rectangle, p *TextLayoutParams, start, end int) []image.Rectangle {
	start = max(start, 0)
	end = min(end, p.RenderingTextLength)
	if start >= end {
		return dst
	}

	m, ok := newLogicalLineMeasurer(p)
	if !ok {
		str := p.RenderingTextRange(0, p.RenderingTextLength)
		var vls []visualLine
		for vl := range visualLines(p.Width, str, p.Style.WrapMode, func(str string, strStartInBytes, endIndexInBytes int) float64 {
			return advanceWithFaces(str, strStartInBytes, endIndexInBytes, p.Style.Face, p.Style.FaceRuns, p.Style.TabWidth, p.Style.KeepTailingSpace)
		}) {
			vls = append(vls, vl)
		}
		return appendBoundsOfRangeInVisualLines(dst, p.Width, vls, 0, 0, start, end, &p.Style)
	}

	firstLogicalLineIdx := m.logicalLineIndexForRenderingIndex(start)
	var yOrigin float64
	for i := range firstLogicalLineIdx {
		yOrigin += p.Style.LineHeight * float64(m.visualLineCount(i))
	}
	var vls []visualLine
	for i := firstLogicalLineIdx; i < m.logicalLineCount; i++ {
		renderingLineStart, renderingLineEnd := m.renderingRange(i)
		if renderingLineStart >= end {
			break
		}
		line := p.RenderingTextRange(renderingLineStart, renderingLineEnd)
		vls = vls[:0]
		for vl := range visualLinesFromLogicalLine(p.Width, line, p.Style.WrapMode, p.Style.Face, p.Style.FaceRuns, renderingLineStart, p.Style.TabWidth, p.Style.KeepTailingSpace) {
			vls = append(vls, vl)
		}
		lineRangeStart := max(start-renderingLineStart, 0)
		lineRangeEnd := min(end-renderingLineStart, len(line))
		dst = appendBoundsOfRangeInVisualLines(dst, p.Width, vls, renderingLineStart, yOrigin, lineRangeStart, lineRangeEnd, &p.Style)
		yOrigin += p.Style.LineHeight * float64(len(vls))
	}
	return dst
}

// appendBoundsOfRangeInVisualLines appends the rectangles covering
// [start, end) within the visual lines vls to dst, one per crossed visual
// line holding a non-empty part of the range. start, end, and each
// visualLine's pos are relative to vls' first byte; vlsStartInBytes is the
// whole-text byte offset of that byte. yOrigin is added to the vertical
// coordinates.
func appendBoundsOfRangeInVisualLines(dst []image.Rectangle, width int, vls []visualLine, vlsStartInBytes int, yOrigin float64, start, end int, style *Style) []image.Rectangle {
	for _, vl := range vls {
		segStart := max(start, vl.pos)
		segEnd := min(end, vl.pos+len(vl.str))
		if segStart >= segEnd {
			continue
		}
		posStart, posEnd, ok := rangePositionsInVisualLines(width, vls, vlsStartInBytes, segStart, segEnd, style)
		if !ok {
			continue
		}
		dst = append(dst, image.Rect(
			int(math.Floor(posStart.X)),
			int(math.Floor(yOrigin+posStart.Top)),
			int(math.Ceil(posEnd.X)),
			int(math.Ceil(yOrigin+posStart.Bottom)),
		))
	}
	return dst
}
