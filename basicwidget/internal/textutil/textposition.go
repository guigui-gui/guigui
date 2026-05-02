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
	// Index is the byte offset in the rendering text to query.
	Index int

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
	// Optional; when nil [TextPositionFromIndex] falls back to an
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

	// LogicalLineIndexHint / VisualLineIndexHint pin the result's Y
	// coordinate system: the function treats the logical line at
	// LogicalLineIndexHint as starting at visual-line index
	// VisualLineIndexHint, and walks forward (or backward) from there
	// to whichever line contains Index. The returned position's Top
	// is therefore measured in the caller's coordinate system —
	// (0, 0) means "Y is measured from line 0," matching the legacy
	// behavior; (firstLogicalLineInViewport, 0) means "Y is measured
	// from the first visible line's top," used by virtualized text.
	//
	// The walk is bounded by the logical-line distance between the
	// hint and the line containing Index, so a caller that pins the
	// hint inside its viewport pays only O(visible) typesetting per
	// query. Used only when LineByteOffsets is set and Options.AutoWrap
	// is true.
	LogicalLineIndexHint int
	VisualLineIndexHint  int
}

// TextPositionFromIndex returns the visual position(s) corresponding
// to p.Index in the rendering text. When p.LineByteOffsets is supplied,
// the visual-line walk is localized: it starts from
// (p.LogicalLineIndexHint, p.VisualLineIndexHint) and steps forward
// (or backward) one logical line at a time, measuring per-line wrap
// counts, until it reaches the logical line containing p.Index. With
// the hint placed inside the viewport the cost is O(visible logical
// lines) per query, instead of the O(documentLen) full scan the
// sidecar-less fallback performs.
//
// When an active IME composition splices into the rendering text, the
// committed-text sidecar is reused: byte/visual-line shifts derived
// from [ComputeCompositionInfo] map between committed and rendering
// coordinates without rebuilding the sidecar. Falls back to the
// unrestricted whole-document walk when the composition crosses a
// logical-line boundary, when no sidecar is supplied, or when the
// document is empty. The fallback is observationally equivalent to
// the fast path.
func TextPositionFromIndex(p *TextPositionFromIndexParams) (position0, position1 TextPosition, count int) {
	index := p.Index
	if index < 0 || index > p.RenderingTextLength {
		return TextPosition{}, TextPosition{}, 0
	}
	if p.LineByteOffsets == nil {
		return textPositionFromIndex(p.Width, p.RenderingTextRange(0, p.RenderingTextLength), nil, index, p.Options)
	}
	n := p.LineByteOffsets.LineCount()
	if n == 0 {
		return textPositionFromIndex(p.Width, p.RenderingTextRange(0, p.RenderingTextLength), nil, index, p.Options)
	}

	// Resolve composition shifts so the committed-text sidecar is
	// usable without a rebuild. compInfo carries the selection line
	// and the constant byte shifts applied to lines past it; hasComp
	// tracks whether to apply them at all. The visual-line-count delta
	// the old code maintained explicitly is now folded into the
	// per-line walk via lineMeasurer.visualLineCount — measuring the
	// rendering content at compInfo.LineIndex picks up the delta
	// naturally.
	var compInfo CompositionInfo
	var hasComp bool
	var compStart, compRenderingEnd int
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
			// Composition straddles a logical-line boundary: the
			// committed sidecar's logical-line shape doesn't match
			// the rendering text. Fall back to the unrestricted walk.
			return textPositionFromIndex(p.Width, p.RenderingTextRange(0, p.RenderingTextLength), nil, index, p.Options)
		}
		compInfo = info
		hasComp = true
		compStart = p.SelectionStart
		compRenderingEnd = p.SelectionStart + p.CompositionLen
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

	renderingLineStart, renderingLineEnd := m.renderingRange(committedLineIdx)
	line := p.RenderingTextRange(renderingLineStart, renderingLineEnd)
	indexInLine := index - renderingLineStart

	pos0, pos1, c := TextPositionFromIndexInLogicalLine(p.Width, line, indexInLine, p.Options)
	if c == 0 {
		return TextPosition{}, TextPosition{}, 0
	}

	// visualLineIndexAt walks from the caller-supplied hint to
	// targetLine, accumulating per-line wrap counts so the result
	// is the visual-line index where targetLine starts in the
	// caller's coordinate system.
	hintLine := min(max(p.LogicalLineIndexHint, 0), n-1)
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
	precedingVisualLines := visualLineIndexAt(committedLineIdx)
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
		prevRenderingLineStart, prevRenderingLineEnd := m.renderingRange(prevCommittedLineIdx)
		prevLine := p.RenderingTextRange(prevRenderingLineStart, prevRenderingLineEnd)
		prevPos0, _, prevCount := TextPositionFromIndexInLogicalLine(p.Width, prevLine, len(prevLine), p.Options)
		if prevCount > 0 {
			prevYOffset := p.Options.LineHeight * float64(visualLineIndexAt(prevCommittedLineIdx))
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
