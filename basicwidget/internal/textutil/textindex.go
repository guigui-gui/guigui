// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"image"
	"iter"
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
	// usable as-is. selectionLineHeightDelta carries the height
	// difference between the rendering and committed selection
	// lines (0 for [WrapModeNone] or compositions that don't change the
	// wrap).
	var compInfo CompositionInfo
	var hasComp bool
	var selectionLineHeightDelta float64
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
			CompositionText:           p.RenderingTextRange(p.SelectionStart, p.SelectionStart+p.CompositionLen),
			LineByteOffsets:           p.PrecomputedLineByteOffsets,
			SelectionStart:            p.SelectionStart,
			SelectionEnd:              p.SelectionEnd,
			WrapMode:                  p.Style.WrapMode,
			CommittedSelectionLine:    committedSelectionLine,
			RenderingSelectionLine:    renderingSelectionLine,
			Face:                      p.Style.Face,
			LineHeight:                p.Style.LineHeight,
			TabWidth:                  p.Style.TabWidth,
			KeepTailingSpace:          p.Style.KeepTailingSpace,
			CommittedFaceRuns:         p.CommittedFaceRuns,
			RenderingFaceRuns:         p.Style.FaceRuns,
			SelectionLineStartInBytes: cs,
			WrapWidth:                 p.Width,
		})
		if !ok {
			return textIndexFromPosition(p.Width, position, p.RenderingTextRange(0, p.RenderingTextLength), &p.Style)
		}
		compInfo = info
		hasComp = true

		if p.Style.WrapMode != WrapModeNone {
			committedH := MeasureLogicalLineHeight(p.Width, committedSelectionLine, p.Style.WrapMode, p.Style.Face, p.CommittedFaceRuns, cs, p.Style.LineHeight, p.Style.TabWidth, p.Style.KeepTailingSpace)
			renderingH := MeasureLogicalLineHeight(p.Width, renderingSelectionLine, p.Style.WrapMode, p.Style.Face, p.Style.FaceRuns, cs, p.Style.LineHeight, p.Style.TabWidth, p.Style.KeepTailingSpace)
			selectionLineHeightDelta = renderingH - committedH
		}
	}

	// Target Y in line-box space: a Y just above the hint's first visual
	// line stays below the hint's own Y and drives the backward walk.
	padding := textPadding(p.Style.Face.TextFace(), p.Style.LineHeight)
	targetY := float64(position.Y) + padding

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
		faceRuns:           p.Style.FaceRuns,
		lineHeight:         p.Style.LineHeight,
		composition:        compInfo,
	}

	// Locate the committed logical line whose visual range covers
	// targetY by walking forward (or backward) from the caller-supplied
	// hint, measuring each logical line's height until the running
	// offset crosses targetY. The hint lets the caller scope work
	// to the viewport — without it (zero values) the walk starts from
	// line 0 and degrades to O(documentLen). For [WrapModeNone] each
	// logical line is exactly one visual line so the walk is a simple
	// add/subtract, but it still needs to step from the hint rather
	// than treating targetY as an absolute offset — the caller's
	// coordinate system is whatever the hint says it is.
	hintLL := min(max(p.LogicalLineIndexHint, 0), n-1)
	hintY := p.Style.LineHeight * float64(max(p.VisualLineIndexHint, 0))
	// Translate the committed-text hint into a rendering-text
	// offset by applying the composition delta when the hint sits
	// past the composition's line.
	if hasComp && hintLL > compInfo.LineIndex {
		hintY += selectionLineHeightDelta
	}

	curLL := hintLL
	curY := hintY
	if targetY >= hintY {
		for curLL < n-1 {
			h := m.logicalLineHeight(curLL)
			if curY+h > targetY {
				break
			}
			curY += h
			curLL++
		}
	} else {
		for curLL > 0 {
			curLL--
			curY -= m.logicalLineHeight(curLL)
			if curY <= targetY {
				break
			}
		}
	}
	logicalLineIndex := curLL
	logicalLineOriginY := curY

	renderingLineStart, renderingLineEnd := m.renderingRange(logicalLineIndex)
	line := p.RenderingTextRange(renderingLineStart, renderingLineEnd)

	// Translate the position into the logical line's local Y so the per-line
	// resolution picks the right visual subline.
	localY := position.Y - int(logicalLineOriginY)
	localPos := image.Pt(position.X, localY)
	var pos int
	if p.Style.WrapMode != WrapModeNone && !faceRunsIntersect(p.Style.FaceRuns, renderingLineStart, renderingLineStart+len(line)) {
		if vlStarts, ok := cachedVisualLineStarts(p.Width, line, p.Style.WrapMode, p.Style.Face, p.Style.TabWidth, p.Style.KeepTailingSpace); ok {
			pos = textIndexFromPositionInVisualLines(p.Width, localPos, visualLinesFromStarts(line, vlStarts), renderingLineStart, &p.Style)
			return renderingLineStart + pos
		}
	}
	pos = TextIndexFromPositionInLogicalLine(p.Width, localPos, line, renderingLineStart, &p.Style)
	return renderingLineStart + pos
}

// textIndexFromPositionInVisualLines returns the byte offset within a logical
// line closest to position, given that line's visual lines. The position's Y
// is relative to the top of the logical line. style's face runs use
// whole-text byte offsets; vlsStartInBytes is the whole-text byte offset of
// vls' first byte.
func textIndexFromPositionInVisualLines(width int, position image.Point, vls iter.Seq[visualLine], vlsStartInBytes int, style *Style) int {
	// Determine the visual line first.
	padding := textPadding(style.Face.TextFace(), style.LineHeight)
	targetY := float64(position.Y) + padding

	var pos int
	var vlStr string
	var y float64
	for l := range vls {
		vlStr = l.str
		pos = l.pos
		if y+style.LineHeight > targetY {
			break
		}
		y += style.LineHeight
	}

	// Determine the index within the visual line.
	left := oneLineLeft(width, vlStr, vlsStartInBytes+pos, style)
	pos += indexFromXInVisualLine(vlStr, vlsStartInBytes+pos, float64(position.X)-left, style)
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
	targetY := float64(position.Y) + padding

	var pos int
	var vlStr string
	var y float64
	for l := range visualLines(width, str, style.WrapMode, func(str string, strStartInBytes, endIndexInBytes int) float64 {
		return advanceWithFaces(str, strStartInBytes, endIndexInBytes, style.Face, style.FaceRuns, style.TabWidth, style.KeepTailingSpace)
	}) {
		vlStr = l.str
		pos = l.pos
		if y+style.LineHeight > targetY {
			break
		}
		y += style.LineHeight
	}

	// Determine the index within the visual line.
	left := oneLineLeft(width, vlStr, pos, style)
	pos += indexFromXInVisualLine(vlStr, pos, float64(position.X)-left, style)
	return pos
}
