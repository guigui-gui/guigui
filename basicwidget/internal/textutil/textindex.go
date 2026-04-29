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
	// any active composition spliced in.
	RenderingText string

	// Width is the rendering width.
	Width int

	// Options carries face, lineHeight, autoWrap, alignment, tab
	// width, etc.
	Options *Options

	// CommittedText is RenderingText without the active composition.
	// When CompositionLen == 0, ignored.
	CommittedText string

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
	// usable as-is. spliceVisualLineCountDelta carries the wrap-count
	// difference between the rendering and committed splice lines (0
	// for non-autoWrap or compositions that don't change the wrap).
	var compInfo CompositionInfo
	var hasComp bool
	var spliceVisualLineCountDelta int
	if p.CompositionLen > 0 {
		info, ok := ComputeCompositionInfo(&CompositionInfoParams{
			RenderingText:    p.RenderingText,
			CommittedText:    p.CommittedText,
			LineByteOffsets:  p.LineByteOffsets,
			SelectionStart:   p.SelectionStart,
			SelectionEnd:     p.SelectionEnd,
			CompositionLen:   p.CompositionLen,
			AutoWrap:         p.Options.AutoWrap,
			Face:             p.Options.Face,
			LineHeight:       p.Options.LineHeight,
			TabWidth:         p.Options.TabWidth,
			KeepTailingSpace: p.Options.KeepTailingSpace,
			WrapWidth:        p.Width,
		})
		if !ok {
			return textIndexFromPosition(p.Width, p.Position, p.RenderingText, p.Options)
		}
		compInfo = info
		hasComp = true

		if p.Options.AutoWrap {
			cs := p.LineByteOffsets.ByteOffsetByLineIndex(compInfo.LineIndex)
			ce := len(p.CommittedText)
			if compInfo.LineIndex+1 < n {
				ce = p.LineByteOffsets.ByteOffsetByLineIndex(compInfo.LineIndex + 1)
			}
			committedCount := VisualLineCountForLogicalLine(p.Width, p.CommittedText[cs:ce], true, p.Options.Face, p.Options.TabWidth, p.Options.KeepTailingSpace)
			renderingCount := VisualLineCountForLogicalLine(p.Width, p.RenderingText[cs:ce+compInfo.RenderingByteShift], true, p.Options.Face, p.Options.TabWidth, p.Options.KeepTailingSpace)
			spliceVisualLineCountDelta = renderingCount - committedCount
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
			v += spliceVisualLineCountDelta
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
	committedTextLen := len(p.RenderingText)
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

	line := p.RenderingText[renderingLineStart:renderingLineEnd]

	// Translate the position into the logical line's local Y so
	// TextIndexFromPositionInLogicalLine picks the right visual
	// subline.
	visualLineOriginIdx := committedVisualOffset(committedLineIdx)
	localY := p.Position.Y - int(float64(visualLineOriginIdx)*p.Options.LineHeight)
	pos := TextIndexFromPositionInLogicalLine(p.Width, image.Pt(p.Position.X, localY), line, p.Options)
	return renderingLineStart + pos
}
