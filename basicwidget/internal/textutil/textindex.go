// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"image"
	"iter"
	"math"
	"slices"
)

// TextIndexFromPosition returns the byte offset in the rendering text
// closest to position. When p.PrecomputedLineByteOffsets is supplied, the
// visual-line walk is localized: it starts from
// (p.LogicalLineIndexHint, p.VisualLineIndexHint) and steps forward
// (or backward) one logical line at a time until the line covering
// position.Y is found. With the hint placed inside the viewport
// this costs O(visible lines) of typesetting per query, instead of
// the O(documentLen) full scan performed when no precomputed
// logical-line offsets are supplied.
//
// When an active IME composition splices into the rendering text, the
// precomputed committed-text logical-line offsets are reused:
// byte/visual-line shifts derived from [ComputeCompositionInfo] map
// between committed and rendering coordinates without rebuilding the
// offsets. Falls back to the unrestricted whole-document walk when the
// composition crosses a logical-line boundary, when no precomputed
// logical-line offsets are supplied, or when the document is empty. The
// fallback is observationally equivalent to the fast path.
func TextIndexFromPosition(p *TextLayoutParams, position image.Point) int {
	if p.PrecomputedLineByteOffsets == nil {
		return textIndexFromPosition(p.Width, position, p.RenderingTextRange(0, p.RenderingTextLength), &p.Style)
	}
	n := p.PrecomputedLineByteOffsets.LineCount()
	if n == 0 {
		return textIndexFromPosition(p.Width, position, p.RenderingTextRange(0, p.RenderingTextLength), &p.Style)
	}

	// Resolve composition shifts so the precomputed logical-line offsets are
	// usable as-is. selectionLineVisualCountDelta carries the wrap-
	// count difference between the rendering and committed selection
	// lines (0 for [WrapModeNone] or compositions that don't change the
	// wrap).
	var compInfo CompositionInfo
	var hasComp bool
	var selectionLineVisualCountDelta int
	if p.CompositionLen > 0 {
		selectionLineIdx := p.PrecomputedLineByteOffsets.LineIndexForByteOffset(p.SelectionStart)
		cs := p.PrecomputedLineByteOffsets.ByteOffsetByLineIndex(selectionLineIdx)
		byteDelta := p.CompositionLen - (p.SelectionEnd - p.SelectionStart)
		ce := p.RenderingTextLength - byteDelta
		if selectionLineIdx+1 < n {
			ce = p.PrecomputedLineByteOffsets.ByteOffsetByLineIndex(selectionLineIdx + 1)
		}
		// The selection-line slices are only valid when the selection
		// lies inside a single logical line; otherwise ce+byteDelta
		// underflows. When the selection crosses lines we leave them
		// empty — [ComputeCompositionInfo]'s own multi-line check
		// returns false before reading them, and the caller falls back
		// below.
		var committedSelectionLine, renderingSelectionLine string
		if p.Style.WrapMode != WrapModeNone && p.PrecomputedLineByteOffsets.LineIndexForByteOffset(p.SelectionEnd) == selectionLineIdx {
			committedSelectionLine = p.CommittedTextRange(cs, ce)
			renderingSelectionLine = p.RenderingTextRange(cs, ce+byteDelta)
		}

		info, ok := ComputeCompositionInfo(&CompositionInfoParams{
			CompositionText:        p.RenderingTextRange(p.SelectionStart, p.SelectionStart+p.CompositionLen),
			LineByteOffsets:        p.PrecomputedLineByteOffsets,
			SelectionStart:         p.SelectionStart,
			SelectionEnd:           p.SelectionEnd,
			WrapMode:               p.Style.WrapMode,
			CommittedSelectionLine: committedSelectionLine,
			RenderingSelectionLine: renderingSelectionLine,
			Face:                   p.Style.Face,
			LineHeight:             p.Style.LineHeight,
			TabWidth:               p.Style.TabWidth,
			KeepTailingSpace:       p.Style.KeepTailingSpace,
			WrapWidth:              p.Width,
		})
		if !ok {
			return textIndexFromPosition(p.Width, position, p.RenderingTextRange(0, p.RenderingTextLength), &p.Style)
		}
		compInfo = info
		hasComp = true

		if p.Style.WrapMode != WrapModeNone {
			committedCount := VisualLineCountForLogicalLine(p.Width, committedSelectionLine, p.Style.WrapMode, p.Style.Face, p.Style.TabWidth, p.Style.KeepTailingSpace)
			renderingCount := VisualLineCountForLogicalLine(p.Width, renderingSelectionLine, p.Style.WrapMode, p.Style.Face, p.Style.TabWidth, p.Style.KeepTailingSpace)
			selectionLineVisualCountDelta = renderingCount - committedCount
		}
	}

	// Target visual-line index from position.Y. Use floor so a Y just
	// above the hint's first visual line maps to a negative target and
	// drives the backward walk — int() truncation rounds toward zero
	// and would clamp such Ys onto the hint line, causing arrow-up at
	// the viewport top to stand still instead of crossing into the
	// previous logical line.
	padding := textPadding(p.Style.Face.TextFace(), p.Style.LineHeight)
	target := int(math.Floor((float64(position.Y) + padding) / p.Style.LineHeight))

	committedTextLen := p.RenderingTextLength
	if hasComp {
		committedTextLen -= compInfo.RenderingByteShift
	}

	m := &logicalLineMeasurer{
		offsets:            p.PrecomputedLineByteOffsets,
		logicalLineCount:   n,
		committedTextLen:   committedTextLen,
		renderingTextRange: p.RenderingTextRange,
		width:              p.Width,
		face:               p.Style.Face,
		tabWidth:           p.Style.TabWidth,
		keepTailingSpace:   p.Style.KeepTailingSpace,
		wrapMode:           p.Style.WrapMode,
		composition:        compInfo,
	}

	// Locate the committed logical line whose visual range covers
	// target by walking forward (or backward) from the caller-supplied
	// hint, measuring each logical line's wrap count until the running
	// visual offset crosses target. The hint lets the caller scope work
	// to the viewport — without it (zero values) the walk starts from
	// line 0 and degrades to O(documentLen). For [WrapModeNone] each
	// logical line is exactly one visual line so the walk is a simple
	// add/subtract, but it still needs to step from (hintLL, hintVL)
	// rather than treating target as an absolute line index — the
	// caller's coordinate system is whatever the hint says it is.
	hintLL := min(max(p.LogicalLineIndexHint, 0), n-1)
	hintVL := max(p.VisualLineIndexHint, 0)
	// Translate the committed-text hint into a rendering-text
	// visual offset by applying the composition delta when the
	// hint sits past the composition's line.
	if hasComp && hintLL > compInfo.LineIndex {
		hintVL += selectionLineVisualCountDelta
	}

	curLL := hintLL
	curVL := hintVL
	if target >= hintVL {
		for curLL < n-1 {
			c := m.visualLineCount(curLL)
			if curVL+c > target {
				break
			}
			curVL += c
			curLL++
		}
	} else {
		for curLL > 0 {
			curLL--
			c := m.visualLineCount(curLL)
			curVL -= c
			if curVL <= target {
				break
			}
		}
	}
	logicalLineIndex := curLL
	logicalLineVisualOriginIndex := curVL

	renderingLineStart, renderingLineEnd := m.renderingRange(logicalLineIndex)
	line := p.RenderingTextRange(renderingLineStart, renderingLineEnd)

	// Translate the position into the logical line's local Y so the per-line
	// resolution picks the right visual subline.
	localY := position.Y - int(float64(logicalLineVisualOriginIndex)*p.Style.LineHeight)
	localPos := image.Pt(position.X, localY)
	var pos int
	if p.Style.WrapMode != WrapModeNone {
		if vlStarts, ok := cachedVisualLineStarts(p.Width, line, p.Style.WrapMode, p.Style.Face, p.Style.TabWidth, p.Style.KeepTailingSpace); ok {
			pos = textIndexFromPositionInVisualLines(p.Width, localPos, visualLinesFromStarts(line, slices.Values(vlStarts)), &p.Style)
			return renderingLineStart + pos
		}
	}
	pos = TextIndexFromPositionInLogicalLine(p.Width, localPos, line, &p.Style)
	return renderingLineStart + pos
}

// textIndexFromPositionInVisualLines returns the byte offset within a logical
// line closest to position, given that line's visual lines. The position's Y is
// relative to the top of the logical line.
func textIndexFromPositionInVisualLines(width int, position image.Point, vls iter.Seq[visualLine], style *Style) int {
	// Determine the visual line first.
	padding := textPadding(style.Face.TextFace(), style.LineHeight)
	n := int((float64(position.Y) + padding) / style.LineHeight)

	var pos int
	var vlStr string
	var vlIndex int
	for l := range vls {
		vlStr = l.str
		pos = l.pos
		if vlIndex >= n {
			break
		}
		vlIndex++
	}

	// Determine the index within the visual line.
	left := oneLineLeft(width, vlStr, style.Face.TextFace(), style.HorizontalAlign, style.TabWidth, style.KeepTailingSpace)
	pos += indexFromXInVisualLine(vlStr, float64(position.X)-left, style)
	return pos
}

// textIndexFromPosition is the unrestricted whole-document
// implementation: it walks every visual line in str to find the one
// covering position.Y. O(documentLen) per call and only suitable when
// no precomputed [LineByteOffsets] is available; the public
// [TextIndexFromPosition] uses this as a fallback.
func textIndexFromPosition(width int, position image.Point, str string, style *Style) int {
	// Determine the visual line first.
	padding := textPadding(style.Face.TextFace(), style.LineHeight)
	n := int((float64(position.Y) + padding) / style.LineHeight)

	var pos int
	var vlStr string
	var vlIndex int
	for l := range visualLines(width, str, style.WrapMode, func(str string, indexInBytes int) float64 {
		return advance(str, indexInBytes, style.Face.TextFace(), style.TabWidth, style.KeepTailingSpace)
	}) {
		vlStr = l.str
		pos = l.pos
		if vlIndex >= n {
			break
		}
		vlIndex++
	}

	// Determine the index within the visual line.
	left := oneLineLeft(width, vlStr, style.Face.TextFace(), style.HorizontalAlign, style.TabWidth, style.KeepTailingSpace)
	pos += indexFromXInVisualLine(vlStr, float64(position.X)-left, style)
	return pos
}
