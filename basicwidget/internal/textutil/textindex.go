// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"image"
)

// TextIndexFromPositionParams describes the inputs for
// [TextIndexFromPosition]. The first group of fields is always
// required; the second group is optional state that enables the
// sidecar-accelerated fast path.
type TextIndexFromPositionParams struct {
	// Position is the (x, y) point in the rendering plane to query.
	// Y is measured from the top of the rendered text.
	Position image.Point

	// RenderingTextRange returns rendering[start:end), where the
	// rendering text is the committed text with any active composition
	// spliced in. RenderingTextLength is the total byte length of the
	// rendering text. Required: all reads of the rendering text — both
	// the fast path and the slow-path fallback — go through this
	// callback so the caller never has to materialize the full
	// document.
	RenderingTextRange  func(start, end int) string
	RenderingTextLength int

	// Width is the rendering width.
	Width int

	// Options carries face, lineHeight, autoWrap, alignment, tab
	// width, etc.
	Options *Options

	// CommittedTextRange returns committed[start:end). Required when
	// CompositionLen > 0; ignored otherwise.
	CommittedTextRange func(start, end int) string

	// LineByteOffsets is the logical-line layout of the committed text.
	// Optional; when nil [TextIndexFromPosition] falls back to an
	// O(documentLen) walk of every visual line.
	LineByteOffsets *LineByteOffsets

	// SelectionStart, SelectionEnd, CompositionLen describe an active
	// IME composition: bytes [SelectionStart, SelectionEnd) in the
	// committed text are replaced with bytes [SelectionStart,
	// SelectionStart+CompositionLen) in the rendering text.
	// CompositionLen == 0 means no active composition; the other
	// fields are ignored in that case.
	SelectionStart int
	SelectionEnd   int
	CompositionLen int

	// LogicalLineIndexHint / VisualLineIndexHint are an optional hint
	// that tells [TextIndexFromPosition] where to start its per-
	// logical-line walk instead of starting from line 0.
	// LogicalLineIndexHint is a logical-line index in committed text;
	// VisualLineIndexHint is the cumulative number of visual lines
	// preceding that logical line in committed text. The walk steps
	// forward (or backward) from the hint measuring one logical line
	// at a time, so a caller that places the hint inside its viewport
	// pays O(visible lines) of typesetting per query instead of
	// walking from the document top.
	//
	// Both fields are optional. The zero value means "start from line
	// 0," equivalent to walking from the top of the document — correct
	// but O(documentLen) for autoWrap when the click is far down.
	// Used only when LineByteOffsets is set and Options.AutoWrap is
	// true.
	LogicalLineIndexHint int
	VisualLineIndexHint  int
}

// TextIndexFromPosition returns the byte offset in the rendering text
// closest to p.Position. When p.LineByteOffsets is supplied, the
// visual-line walk is localized: it starts from
// (p.LogicalLineIndexHint, p.VisualLineIndexHint) and steps forward
// (or backward) one logical line at a time until the line covering
// p.Position.Y is found. With the hint placed inside the viewport
// this costs O(visible lines) of typesetting per query, instead of
// the O(documentLen) full scan the sidecar-less fallback performs.
//
// When an active IME composition splices into the rendering text, the
// committed-text sidecar is reused: byte/visual-line shifts derived
// from [ComputeCompositionInfo] map between committed and rendering
// coordinates without rebuilding the sidecar. Falls back to the
// unrestricted whole-document walk when the composition crosses a
// logical-line boundary, when no sidecar is supplied, or when the
// document is empty. The fallback is observationally equivalent to
// the fast path.
func TextIndexFromPosition(p *TextIndexFromPositionParams) int {
	if p.LineByteOffsets == nil {
		return textIndexFromPosition(p.Width, p.Position, p.RenderingTextRange(0, p.RenderingTextLength), p.Options)
	}
	n := p.LineByteOffsets.LineCount()
	if n == 0 {
		return textIndexFromPosition(p.Width, p.Position, p.RenderingTextRange(0, p.RenderingTextLength), p.Options)
	}

	// Resolve composition shifts so the committed-text sidecar is
	// usable as-is. selectionLineVisualCountDelta carries the wrap-
	// count difference between the rendering and committed selection
	// lines (0 for non-autoWrap or compositions that don't change the
	// wrap).
	var compInfo CompositionInfo
	var hasComp bool
	var selectionLineVisualCountDelta int
	if p.CompositionLen > 0 {
		selectionLineIdx := p.LineByteOffsets.LineIndexForByteOffset(p.SelectionStart)
		cs := p.LineByteOffsets.ByteOffsetByLineIndex(selectionLineIdx)
		byteDelta := p.CompositionLen - (p.SelectionEnd - p.SelectionStart)
		ce := p.RenderingTextLength - byteDelta
		if selectionLineIdx+1 < n {
			ce = p.LineByteOffsets.ByteOffsetByLineIndex(selectionLineIdx + 1)
		}
		// The selection-line slices are only valid when the selection
		// lies inside a single logical line; otherwise ce+byteDelta
		// underflows. When the selection crosses lines we leave them
		// empty — [ComputeCompositionInfo]'s own multi-line check
		// returns false before reading them, and the caller falls back
		// below.
		var committedSelectionLine, renderingSelectionLine string
		if p.Options.AutoWrap && p.LineByteOffsets.LineIndexForByteOffset(p.SelectionEnd) == selectionLineIdx {
			committedSelectionLine = p.CommittedTextRange(cs, ce)
			renderingSelectionLine = p.RenderingTextRange(cs, ce+byteDelta)
		}

		info, ok := ComputeCompositionInfo(&CompositionInfoParams{
			CompositionText:        p.RenderingTextRange(p.SelectionStart, p.SelectionStart+p.CompositionLen),
			LineByteOffsets:        p.LineByteOffsets,
			SelectionStart:         p.SelectionStart,
			SelectionEnd:           p.SelectionEnd,
			AutoWrap:               p.Options.AutoWrap,
			CommittedSelectionLine: committedSelectionLine,
			RenderingSelectionLine: renderingSelectionLine,
			Face:                   p.Options.Face,
			LineHeight:             p.Options.LineHeight,
			TabWidth:               p.Options.TabWidth,
			KeepTailingSpace:       p.Options.KeepTailingSpace,
			WrapWidth:              p.Width,
		})
		if !ok {
			return textIndexFromPosition(p.Width, p.Position, p.RenderingTextRange(0, p.RenderingTextLength), p.Options)
		}
		compInfo = info
		hasComp = true

		if p.Options.AutoWrap {
			committedCount := VisualLineCountForLogicalLine(p.Width, committedSelectionLine, true, p.Options.Face, p.Options.TabWidth, p.Options.KeepTailingSpace)
			renderingCount := VisualLineCountForLogicalLine(p.Width, renderingSelectionLine, true, p.Options.Face, p.Options.TabWidth, p.Options.KeepTailingSpace)
			selectionLineVisualCountDelta = renderingCount - committedCount
		}
	}

	// Target visual-line index from position.Y. Mirrors the slow
	// path's lineHeight-based integer divide.
	padding := textPadding(p.Options.Face, p.Options.LineHeight)
	target := max(int((float64(p.Position.Y)+padding)/p.Options.LineHeight), 0)

	committedTextLen := p.RenderingTextLength
	if hasComp {
		committedTextLen -= compInfo.RenderingByteShift
	}

	m := &lineMeasurer{
		offsets:            p.LineByteOffsets,
		logicalLineCount:   n,
		committedTextLen:   committedTextLen,
		renderingTextRange: p.RenderingTextRange,
		width:              p.Width,
		face:               p.Options.Face,
		tabWidth:           p.Options.TabWidth,
		keepTailingSpace:   p.Options.KeepTailingSpace,
		autoWrap:           p.Options.AutoWrap,
		composition:        compInfo,
	}

	// Locate the committed logical line whose visual range covers
	// target. For non-autoWrap each logical line is one visual line,
	// so target IS the line index (clamped). For autoWrap, walk
	// forward (or backward) from the caller-supplied hint measuring
	// each logical line's wrap count until the running visual offset
	// crosses target. The hint lets the caller scope work to the
	// viewport — without it (zero values) the walk starts from line 0
	// and degrades to O(documentLen).
	var logicalLineIndex int
	var logicalLineVisualOriginIndex int
	if !p.Options.AutoWrap {
		logicalLineIndex = min(max(target, 0), n-1)
		logicalLineVisualOriginIndex = logicalLineIndex
	} else {
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
			if curVL < 0 {
				curVL = 0
			}
		}
		logicalLineIndex = curLL
		logicalLineVisualOriginIndex = curVL
	}

	renderingLineStart, renderingLineEnd := m.renderingRange(logicalLineIndex)
	line := p.RenderingTextRange(renderingLineStart, renderingLineEnd)

	// Translate the position into the logical line's local Y so
	// TextIndexFromPositionInLogicalLine picks the right visual
	// subline.
	localY := p.Position.Y - int(float64(logicalLineVisualOriginIndex)*p.Options.LineHeight)
	pos := TextIndexFromPositionInLogicalLine(p.Width, image.Pt(p.Position.X, localY), line, p.Options)
	return renderingLineStart + pos
}

// textIndexFromPosition is the unrestricted whole-document
// implementation: it walks every visual line in str to find the one
// covering position.Y. O(documentLen) per call and only suitable when
// no [LineByteOffsets] sidecar is available; the public
// [TextIndexFromPosition] uses this as a fallback.
func textIndexFromPosition(width int, position image.Point, str string, options *Options) int {
	// Determine the visual line first.
	padding := textPadding(options.Face, options.LineHeight)
	n := int((float64(position.Y) + padding) / options.LineHeight)

	var pos int
	var vlStr string
	var vlIndex int
	for l := range visualLines(width, str, options.AutoWrap, func(str string) float64 {
		return advance(str, options.Face, options.TabWidth, options.KeepTailingSpace)
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
