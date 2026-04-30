// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"image"
	"sort"
)

// TextIndexFromPositionParams describes the inputs for
// [TextIndexFromPosition]. The first group of fields is always
// required; the second group is optional state that enables the
// sidecar-accelerated fast path.
type TextIndexFromPositionParams struct {
	// Position is the (x, y) point in the rendering plane to query.
	// Y is measured from the top of the rendered text.
	Position image.Point

	// RenderingText is the full text to query: committed text with
	// any active composition spliced in. Required when
	// RenderingTextRange is nil; ignored when it is set.
	RenderingText string

	// RenderingTextRange is an optional callback returning
	// rendering[start:end), and RenderingTextLength is the total
	// byte length of the rendering text (paired with the callback).
	// When set, all fast-path slicing reads through the callback
	// instead of slicing RenderingText, so the caller never has to
	// materialize the full document. The slow-path fallback still
	// uses RenderingText.
	RenderingTextRange  func(start, end int) string
	RenderingTextLength int

	// Width is the rendering width.
	Width int

	// Options carries face, lineHeight, autoWrap, alignment, tab
	// width, etc.
	Options *Options

	// CommittedText is RenderingText without the active composition.
	// Required when CommittedTextRange is nil and CompositionLen > 0;
	// ignored otherwise.
	CommittedText string

	// CommittedTextRange is an optional callback returning
	// committed[start:end). When set, the composition splice line's
	// committed bytes are read through the callback instead of
	// slicing CommittedText.
	CommittedTextRange func(start, end int) string

	// LineByteOffsets is the logical-line layout of CommittedText.
	// Optional; when nil [TextIndexFromPosition] falls back to an
	// O(documentLen) walk of every visual line.
	LineByteOffsets *LineByteOffsets

	// SelectionStart, SelectionEnd, CompositionLen describe an active
	// IME composition: bytes [SelectionStart, SelectionEnd) in
	// CommittedText are replaced with bytes [SelectionStart,
	// SelectionStart+CompositionLen) in RenderingText. CompositionLen
	// == 0 means no active composition; the other fields are ignored
	// in that case.
	SelectionStart int
	SelectionEnd   int
	CompositionLen int

	// PrecedingVisualLineCount returns the cumulative number of
	// visual lines for committed-text logical lines [0, lineIdx).
	// For non-autoWrap text this is just lineIdx; for autoWrap it
	// must be served from a cache to preserve the O(log n + lineLen)
	// per-call cost (otherwise this dominates). Required when
	// LineByteOffsets is set and Options.AutoWrap is true.
	PrecedingVisualLineCount func(lineIdx int) int
}

// readRenderingTextRange returns rendering[start:end), preferring the
// caller-supplied callback over slicing the materialized string.
func (p *TextIndexFromPositionParams) readRenderingTextRange(start, end int) string {
	if p.RenderingTextRange != nil {
		return p.RenderingTextRange(start, end)
	}
	return p.RenderingText[start:end]
}

// readCommittedTextRange returns committed[start:end), preferring the
// caller-supplied callback over slicing the materialized string.
func (p *TextIndexFromPositionParams) readCommittedTextRange(start, end int) string {
	if p.CommittedTextRange != nil {
		return p.CommittedTextRange(start, end)
	}
	return p.CommittedText[start:end]
}

// getRenderingTextLength returns the byte length of the rendering text from
// the explicit field if a callback is set; otherwise from the
// materialized string.
func (p *TextIndexFromPositionParams) getRenderingTextLength() int {
	if p.RenderingTextRange != nil {
		return p.RenderingTextLength
	}
	return len(p.RenderingText)
}

// TextIndexFromPosition returns the byte offset in p.RenderingText
// closest to p.Position. When p.LineByteOffsets and (for autoWrap)
// p.PrecedingVisualLineCount are supplied, the visual-line walk is
// localized to the single logical line covering p.Position.Y - O(log n
// committed-line lookups) instead of the O(documentLen) full scan the
// sidecar-less fallback performs.
//
// When an active IME composition splices into p.RenderingText, the
// committed-text sidecar is reused: byte/visual-line shifts derived
// from [ComputeCompositionInfo] map between committed and rendering
// coordinates without rebuilding the sidecar. Falls back to the
// unrestricted whole-document walk when the composition crosses a
// logical-line boundary, when no sidecar is supplied, or when the
// document is empty. The fallback is observationally equivalent to
// the fast path.
func TextIndexFromPosition(p *TextIndexFromPositionParams) int {
	if p.LineByteOffsets == nil {
		return textIndexFromPosition(p.Width, p.Position, p.RenderingText, p.Options)
	}
	n := p.LineByteOffsets.LineCount()
	if n == 0 {
		return textIndexFromPosition(p.Width, p.Position, p.RenderingText, p.Options)
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
		ce := p.getRenderingTextLength() - byteDelta
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
			committedSelectionLine = p.readCommittedTextRange(cs, ce)
			renderingSelectionLine = p.readRenderingTextRange(cs, ce+byteDelta)
		}

		info, ok := ComputeCompositionInfo(&CompositionInfoParams{
			CompositionText:        p.readRenderingTextRange(p.SelectionStart, p.SelectionStart+p.CompositionLen),
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
			return textIndexFromPosition(p.Width, p.Position, p.RenderingText, p.Options)
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

	// committedVisualOffset returns the rendering visual-line index
	// of the start of committed line idx. For non-autoWrap this is
	// just idx; for autoWrap it consults the preceding-count callback
	// and applies the splice delta.
	committedVisualOffset := func(idx int) int {
		if !p.Options.AutoWrap {
			return idx
		}
		var v int
		if p.PrecedingVisualLineCount != nil {
			v = p.PrecedingVisualLineCount(idx)
		}
		if hasComp && idx > compInfo.LineIndex {
			v += selectionLineVisualCountDelta
		}
		return v
	}

	// Locate the committed logical line whose visual range covers
	// target. For non-autoWrap each logical line is one visual line,
	// so target IS the line index (clamped). For autoWrap, binary-
	// search on committedVisualOffset for the largest idx with
	// committedVisualOffset(idx) <= target.
	var committedLineIdx int
	if !p.Options.AutoWrap {
		committedLineIdx = min(max(target, 0), n-1)
	} else {
		committedLineIdx = max(sort.Search(n, func(i int) bool {
			return committedVisualOffset(i) > target
		})-1, 0)
	}

	committedLineStart := p.LineByteOffsets.ByteOffsetByLineIndex(committedLineIdx)
	committedTextLen := p.getRenderingTextLength()
	if hasComp {
		committedTextLen -= compInfo.RenderingByteShift
	}
	committedLineEnd := committedTextLen
	if committedLineIdx+1 < n {
		committedLineEnd = p.LineByteOffsets.ByteOffsetByLineIndex(committedLineIdx + 1)
	}

	renderingLineStart := committedLineStart
	renderingLineEnd := committedLineEnd
	if hasComp {
		switch {
		case committedLineIdx < compInfo.LineIndex:
			// identity
		case committedLineIdx == compInfo.LineIndex:
			renderingLineEnd += compInfo.RenderingByteShift
		default:
			renderingLineStart += compInfo.RenderingByteShift
			renderingLineEnd += compInfo.RenderingByteShift
		}
	}

	line := p.readRenderingTextRange(renderingLineStart, renderingLineEnd)

	// Translate the position into the logical line's local Y so
	// TextIndexFromPositionInLogicalLine picks the right visual
	// subline.
	visualLineOriginIdx := committedVisualOffset(committedLineIdx)
	localY := p.Position.Y - int(float64(visualLineOriginIdx)*p.Options.LineHeight)
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
