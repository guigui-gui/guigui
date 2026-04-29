// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import "slices"

type TextPosition struct {
	X      float64
	Top    float64
	Bottom float64
}

// TextPositionFromIndexParams describes the inputs for
// [TextPositionFromIndex]. The first group of fields is always
// required; the second group is optional state that enables the
// sidecar-accelerated fast path.
type TextPositionFromIndexParams struct {
	// Index is the byte offset in RenderingText to query.
	Index int

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
	// Optional; when nil [TextPositionFromIndex] falls back to an
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
	// LineByteOffsets is set.
	PrecedingVisualLineCount func(lineIdx int) int
}

// TextPositionFromIndex returns the visual position(s) corresponding
// to p.Index in p.RenderingText. When p.LineByteOffsets and
// p.PrecedingVisualLineCount are supplied, the visual-line walk is
// localized to the single logical line containing p.Index — O(log n +
// lineLen) per call instead of the O(documentLen) full scan the
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
func TextPositionFromIndex(p *TextPositionFromIndexParams) (position0, position1 TextPosition, count int) {
	index := p.Index
	if index < 0 || index > len(p.RenderingText) {
		return TextPosition{}, TextPosition{}, 0
	}
	if p.LineByteOffsets == nil || p.PrecedingVisualLineCount == nil {
		return textPositionFromIndex(p.Width, p.RenderingText, nil, index, p.Options)
	}
	n := p.LineByteOffsets.LineCount()
	if n == 0 {
		return textPositionFromIndex(p.Width, p.RenderingText, nil, index, p.Options)
	}

	// Resolve composition shifts so the committed-text sidecar is
	// usable without a rebuild. compInfo carries the splice line and
	// the constant byte/visual-line deltas applied to lines past it;
	// hasComp tracks whether to apply them at all.
	var compInfo CompositionInfo
	var hasComp bool
	var compStart, compRenderingEnd int
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
			// Composition straddles a logical-line boundary: the
			// committed sidecar's logical-line shape doesn't match
			// RenderingText. Fall back to the unrestricted walk.
			return textPositionFromIndex(p.Width, p.RenderingText, nil, index, p.Options)
		}
		compInfo = info
		hasComp = true
		compStart = p.SelectionStart
		compRenderingEnd = p.SelectionStart + p.CompositionLen

		if p.Options.AutoWrap {
			// compInfo.RenderingYShift is in pixels (ceiled); for
			// fractional-Y semantics we need the visual-line-count
			// delta directly. Recompute it from the splice-line
			// shape.
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

	// Map rendering index to a committed byte offset for line lookup.
	// The composition replaces committed[sStart:sEnd] with rendering
	// bytes [compStart, compRenderingEnd); lines on either side are
	// unaffected other than a constant byte shift past the splice.
	var committedLineIdx int
	if hasComp {
		switch {
		case index < compStart:
			committedLineIdx = p.LineByteOffsets.LineIndexForByteOffset(index)
		case index <= compRenderingEnd:
			committedLineIdx = compInfo.LineIndex
		default:
			committedLineIdx = p.LineByteOffsets.LineIndexForByteOffset(index - compInfo.RenderingByteShift)
		}
	} else {
		committedLineIdx = p.LineByteOffsets.LineIndexForByteOffset(index)
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

	// Translate the committed line range into rendering coordinates.
	renderingLineStart := committedLineStart
	renderingLineEnd := committedLineEnd
	if hasComp {
		switch {
		case committedLineIdx < compInfo.LineIndex:
			// Before the splice: identity.
		case committedLineIdx == compInfo.LineIndex:
			renderingLineEnd += compInfo.RenderingByteShift
		default:
			renderingLineStart += compInfo.RenderingByteShift
			renderingLineEnd += compInfo.RenderingByteShift
		}
	}

	line := p.RenderingText[renderingLineStart:renderingLineEnd]
	indexInLine := index - renderingLineStart

	pos0, pos1, c := TextPositionFromIndexInLogicalLine(p.Width, line, indexInLine, p.Options)
	if c == 0 {
		return TextPosition{}, TextPosition{}, 0
	}

	precedingVisualLines := p.PrecedingVisualLineCount(committedLineIdx)
	if hasComp && committedLineIdx > compInfo.LineIndex {
		precedingVisualLines += spliceVisualLineCountDelta
	}
	yOffset := p.Options.LineHeight * float64(precedingVisualLines)

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
	if c == 1 && indexInLine == 0 && committedLineIdx > 0 {
		prevCommittedLineIdx := committedLineIdx - 1
		prevCommittedLineStart := p.LineByteOffsets.ByteOffsetByLineIndex(prevCommittedLineIdx)
		prevCommittedLineEnd := committedLineStart
		prevRenderingLineStart := prevCommittedLineStart
		prevRenderingLineEnd := prevCommittedLineEnd
		if hasComp {
			switch {
			case prevCommittedLineIdx < compInfo.LineIndex:
				// Before the splice: identity.
			case prevCommittedLineIdx == compInfo.LineIndex:
				prevRenderingLineEnd += compInfo.RenderingByteShift
			default:
				prevRenderingLineStart += compInfo.RenderingByteShift
				prevRenderingLineEnd += compInfo.RenderingByteShift
			}
		}
		prevLine := p.RenderingText[prevRenderingLineStart:prevRenderingLineEnd]
		prevPos0, _, prevCount := TextPositionFromIndexInLogicalLine(p.Width, prevLine, len(prevLine), p.Options)
		if prevCount > 0 {
			prevPrecedingVisualLines := p.PrecedingVisualLineCount(prevCommittedLineIdx)
			if hasComp && prevCommittedLineIdx > compInfo.LineIndex {
				prevPrecedingVisualLines += spliceVisualLineCountDelta
			}
			prevYOffset := p.Options.LineHeight * float64(prevPrecedingVisualLines)
			prevPos0.Top += prevYOffset
			prevPos0.Bottom += prevYOffset
			pos1 = pos0
			pos0 = prevPos0
			c = 2
		}
	}
	return pos0, pos1, c
}

// textPositionFromIndex returns the visual position(s) for index in
// str, walking the supplied visual lines vls. When vls is nil it falls
// back to the unrestricted whole-document layout: every visual line in
// str is collected and walked. O(documentLen) in that case and only
// suitable when no [LineByteOffsets] sidecar is available; the public
// [TextPositionFromIndex] uses the nil form as a fallback.
func textPositionFromIndex(width int, str string, vls []visualLine, index int, options *Options) (position0, position1 TextPosition, count int) {
	if index < 0 || index > len(str) {
		return TextPosition{}, TextPosition{}, 0
	}
	if vls == nil {
		vls = slices.Collect(visualLines(width, str, options.AutoWrap, func(str string) float64 {
			return advance(str, options.Face, options.TabWidth, options.KeepTailingSpace)
		}))
	}

	var y, y0, y1 float64
	var indexInLine0, indexInLine1 int
	var line0, line1 string
	var found0, found1 bool
	for _, l := range vls {
		// When auto wrap is on or the string ends with a line break, there can be two positions:
		// one in the tail of the previous line and one in the head of the next line.
		if index == l.pos+len(l.str) {
			if !found0 {
				found0 = true
				line0 = l.str
				indexInLine0 = index - l.pos
				y0 = y
			} else {
				// A previous line already matched as the tail position; this line
				// (typically an empty trailing line for a string ending in a line break)
				// is the head of the next line.
				found1 = true
				line1 = l.str
				indexInLine1 = index - l.pos
				y1 = y
				break
			}
		} else if l.pos <= index && index < l.pos+len(l.str) {
			found1 = true
			line1 = l.str
			indexInLine1 = index - l.pos
			y1 = y
			break
		}
		y += options.LineHeight
	}

	if !found0 && !found1 {
		return TextPosition{}, TextPosition{}, 0
	}

	paddingY := textPadding(options.Face, options.LineHeight)

	var pos0, pos1 TextPosition
	if found0 {
		x0 := oneLineLeft(width, line0, options.Face, options.HorizontalAlign, options.TabWidth, options.KeepTailingSpace)
		x0 += advance(line0[:indexInLine0], options.Face, options.TabWidth, true)
		pos0 = TextPosition{
			X:      x0,
			Top:    y0 + paddingY,
			Bottom: y0 + options.LineHeight - paddingY,
		}
	}
	if found1 {
		x1 := oneLineLeft(width, line1, options.Face, options.HorizontalAlign, options.TabWidth, options.KeepTailingSpace)
		x1 += advance(line1[:indexInLine1], options.Face, options.TabWidth, true)
		pos1 = TextPosition{
			X:      x1,
			Top:    y1 + paddingY,
			Bottom: y1 + options.LineHeight - paddingY,
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
