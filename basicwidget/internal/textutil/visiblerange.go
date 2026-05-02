// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"image"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// CompositionInfoParams describes the inputs for [ComputeCompositionInfo].
type CompositionInfoParams struct {
	// CompositionText is the active composition's bytes — the bytes
	// inserted into the rendering text at SelectionStart, replacing
	// committed[SelectionStart:SelectionEnd].
	CompositionText string

	// LineByteOffsets is the logical-line layout of the committed text.
	LineByteOffsets *LineByteOffsets

	// SelectionStart and SelectionEnd are byte offsets into the
	// committed text describing the range the composition replaces.
	// SelectionStart == SelectionEnd for a pure insertion.
	SelectionStart int
	SelectionEnd   int

	// AutoWrap toggles the visual-Y delta measurement for the
	// selection line. When false, RenderingYShift in the result is
	// always 0 and the fields below are ignored.
	AutoWrap bool

	// CommittedSelectionLine and RenderingSelectionLine are the bytes
	// of the logical line containing the selection (SelectionStart ..
	// SelectionEnd, which always lies within a single logical line —
	// the function rejects multi-line selections), in committed and
	// rendering coordinates respectively. Required when AutoWrap is
	// true; ignored otherwise.
	CommittedSelectionLine string
	RenderingSelectionLine string

	// Face, LineHeight, TabWidth, KeepTailingSpace are passed through
	// to [MeasureLogicalLineHeight] when AutoWrap is true.
	Face             text.Face
	LineHeight       float64
	TabWidth         float64
	KeepTailingSpace bool

	// WrapWidth is the pixel width at which logical lines wrap into
	// visual sublines. Values <= 0 are treated as math.MaxInt (no
	// wrapping).
	WrapWidth int
}

// CompositionInfo describes how an active IME composition shifts the
// document layout for the visible-range slicer. The zero value is safe
// to pass when no composition is active: the shifts are zero, so any
// "past the splice" comparison the slicer makes is harmless.
type CompositionInfo struct {
	// LineIndex is the logical-line index of the selection line.
	// Lines with index > LineIndex are "past the splice" and have
	// RenderingByteShift and RenderingYShift applied.
	LineIndex int

	// RenderingByteShift is added to a past-the-splice line's
	// committed byte offset to get its rendering byte offset. Equals
	// the composition's byte length minus the length of the committed
	// range it replaces, so it can be negative for selection-
	// replacement compositions.
	RenderingByteShift int

	// RenderingYShift is added to a past-the-splice line's committed
	// visual-Y (in pixels, top-of-line) to get its rendering visual-Y.
	// Non-zero only when AutoWrap is on and the composition causes the
	// selection line to wrap into a different number of visual sub-lines.
	RenderingYShift int
}

// ComputeCompositionInfo classifies an active composition and returns
// info for [ComputeVisibleRange]. ok is false when the splice changes
// the logical-line count - a hard line break inside the composition or
// a selection that straddles a logical line boundary - and the caller
// should fall back to drawing the unrestricted text.
func ComputeCompositionInfo(p *CompositionInfoParams) (CompositionInfo, bool) {
	if pos, _ := FirstLineBreakPositionAndLen(p.CompositionText); pos >= 0 {
		return CompositionInfo{}, false
	}
	lineIndex := p.LineByteOffsets.LineIndexForByteOffset(p.SelectionStart)
	if p.SelectionStart != p.SelectionEnd && p.LineByteOffsets.LineIndexForByteOffset(p.SelectionEnd) != lineIndex {
		return CompositionInfo{}, false
	}
	byteDelta := len(p.CompositionText) - (p.SelectionEnd - p.SelectionStart)

	var yDelta int
	if p.AutoWrap {
		// Visual height of the selection line in rendering vs
		// committed: the only line whose wrap layout the composition
		// can change.
		measureWidth := p.WrapWidth
		if measureWidth <= 0 {
			measureWidth = math.MaxInt
		}
		committedH := MeasureLogicalLineHeight(measureWidth, p.CommittedSelectionLine, true, p.Face, p.LineHeight, p.TabWidth, p.KeepTailingSpace)
		renderingH := MeasureLogicalLineHeight(measureWidth, p.RenderingSelectionLine, true, p.Face, p.LineHeight, p.TabWidth, p.KeepTailingSpace)
		yDelta = int(math.Ceil(renderingH)) - int(math.Ceil(committedH))
	}
	return CompositionInfo{
		LineIndex:          lineIndex,
		RenderingByteShift: byteDelta,
		RenderingYShift:    yDelta,
	}, true
}

// VisibleRangeParams describes the inputs for [ComputeVisibleRange].
//
// Deprecated: along with [ComputeVisibleRange]; will be removed once
// every caller has migrated to [VisibleRangeInViewport].
type VisibleRangeParams struct {
	// LineByteOffsets is the logical-line layout of the rendering text
	// (same text that would be passed to [Draw]). The number of logical
	// lines comes from its LineCount.
	LineByteOffsets *LineByteOffsets

	// RenderingTextLength is the total byte length of the rendering text.
	RenderingTextLength int

	// CumulativeYs[i] is the rendered Y of the start of logical line
	// i in the committed text; required when AutoWrap is true and must
	// be extended to at least the last line whose rendered Y meets or
	// exceeds VisibleMaxY.
	CumulativeYs []int

	// LineHeight is ceil(lineHeight); used when AutoWrap is false to
	// derive line indices via integer division.
	LineHeight int

	// AutoWrap toggles between LineHeight*idx (false) and CumulativeYs
	// lookups (true) for finding visible lines.
	AutoWrap bool

	// VerticalAlign and BoundsHeight / TotalHeight together yield the
	// alignment-specific Y offset that VisibleMinY / VisibleMaxY are
	// implicitly measured against.
	VerticalAlign VerticalAlign
	BoundsHeight  int
	TotalHeight   int

	// VisibleMinY and VisibleMaxY are the visible Y range, relative to
	// the bounds origin (i.e. already had bounds.Min.Y subtracted).
	VisibleMinY int
	VisibleMaxY int

	// Composition is the splice info from [ComputeCompositionInfo].
	// The zero value means "no active composition".
	Composition CompositionInfo
}

// VisibleRange is the result of [ComputeVisibleRange] when its ok
// return is true.
type VisibleRange struct {
	// FirstLine and LastLine are the inclusive range of logical-line
	// indices the caller should draw.
	FirstLine, LastLine int

	// StartInBytes and EndInBytes are the byte range of the rendering
	// text the caller should draw: rendering[StartInBytes:EndInBytes].
	StartInBytes, EndInBytes int

	// YShift is added to the drawing-origin Y so the first sliced line
	// lands at its original screen Y. Already includes the alignment-
	// specific portion of the original Y offset, so the caller forces
	// [VerticalAlignTop] when calling [Draw].
	YShift int
}

// ComputeVisibleRange returns the byte range and rendered Y offset that
// covers just the logical lines whose visible-Y region intersects
// [VisibleMinY, VisibleMaxY], plus one line of slack on each side. The
// slack absorbs per-line padding that [Draw] adds internally and any
// integer rounding; the inner Y clip in [Draw] drops lines that turn out
// to be off-screen anyway.
//
// ok is false when the document is empty or the visible range falls
// entirely outside the document; the caller should draw the unrestricted
// text.
//
// CumulativeYs must be extended (when AutoWrap is true) to at least the
// last line whose rendered Y meets or exceeds VisibleMaxY. The function
// does not extend the cache.
//
// Deprecated: use [VisibleRangeInViewport]. ComputeVisibleRange relies
// on a CumulativeYs prefix-sum cache populated up to the visible
// region, which costs O(top of visible region) of typesetting on cold
// cache and dominates scroll CPU on multi-megabyte buffers.
// [VisibleRangeInViewport] walks forward from a caller-supplied
// FirstLogicalLineInViewport instead, paying only O(visible) per
// query.
// ComputeVisibleRange will be removed once Text and textInputText
// migrate.
func ComputeVisibleRange(p *VisibleRangeParams) (VisibleRange, bool) {
	n := p.LineByteOffsets.LineCount()
	if n == 0 {
		return VisibleRange{}, false
	}

	alignOffset := alignOffsetFor(p.VerticalAlign, p.BoundsHeight, p.TotalHeight)
	relMinY := p.VisibleMinY - alignOffset
	relMaxY := p.VisibleMaxY - alignOffset

	var firstLine, lastLine int
	if !p.AutoWrap {
		if p.LineHeight <= 0 {
			return VisibleRange{}, false
		}
		firstLine = max(0, relMinY/p.LineHeight-1)
		lastLine = min(n-1, relMaxY/p.LineHeight+1)
	} else {
		renderingYAt := func(i int) int {
			y := p.CumulativeYs[i]
			if i > p.Composition.LineIndex {
				y += p.Composition.RenderingYShift
			}
			return y
		}
		firstLine = max(0, sort.Search(len(p.CumulativeYs), func(i int) bool {
			return renderingYAt(i) > relMinY
		})-2)
		lastLine = min(n-1, sort.Search(len(p.CumulativeYs), func(i int) bool {
			return renderingYAt(i) >= relMaxY
		}))
	}
	if firstLine > lastLine {
		return VisibleRange{}, false
	}

	startInBytes := p.LineByteOffsets.ByteOffsetByLineIndex(firstLine)
	if firstLine > p.Composition.LineIndex {
		startInBytes += p.Composition.RenderingByteShift
	}
	endInBytes := p.RenderingTextLength
	if lastLine+1 < n {
		endInBytes = p.LineByteOffsets.ByteOffsetByLineIndex(lastLine + 1)
		if lastLine+1 > p.Composition.LineIndex {
			endInBytes += p.Composition.RenderingByteShift
		}
	}

	var lineY int
	if !p.AutoWrap {
		lineY = firstLine * p.LineHeight
	} else {
		lineY = p.CumulativeYs[firstLine]
		if firstLine > p.Composition.LineIndex {
			lineY += p.Composition.RenderingYShift
		}
	}

	return VisibleRange{
		FirstLine:    firstLine,
		LastLine:     lastLine,
		StartInBytes: startInBytes,
		EndInBytes:   endInBytes,
		YShift:       alignOffset + lineY,
	}, true
}

func alignOffsetFor(vAlign VerticalAlign, boundsHeight, totalHeight int) int {
	switch vAlign {
	case VerticalAlignMiddle:
		return (boundsHeight - totalHeight) / 2
	case VerticalAlignBottom:
		return boundsHeight - totalHeight
	}
	return 0
}

// VisibleRangeInViewportParams describes the inputs for
// [VisibleRangeInViewport]. The walk steps forward from
// FirstLogicalLineInViewport measuring per-line heights via
// [VisualLineCountForLogicalLine] until cumulative height covers
// Height, so the cost is O(visible logical lines) — the prefix
// [0, FirstLogicalLineInViewport) is never measured.
type VisibleRangeInViewportParams struct {
	// FirstLogicalLineInViewport is the logical line whose top sits
	// at the widget-local origin (Y=0). The caller's bounds-positioning
	// places this line at the top of the rendered output, so the
	// returned VisibleRange.FirstLine is always this index (clamped to
	// the document) and YShift is always 0.
	FirstLogicalLineInViewport int

	// LineByteOffsets is the logical-line layout of the committed
	// text. The number of logical lines comes from its LineCount.
	LineByteOffsets *LineByteOffsets

	// RenderingTextRange returns rendering[start:end). The walker
	// reads each measured line through this callback so the caller
	// never has to materialize the full rendering text. Required when
	// AutoWrap is true (so the walker can shape per-line content); for
	// AutoWrap=false only RenderingTextLength is consulted.
	RenderingTextRange func(start, end int) string

	// RenderingTextLength is the total byte length of the rendering
	// text.
	RenderingTextLength int

	// ViewportSize describes the rendering box the walker operates
	// against: X is the wrap width passed through to
	// [VisualLineCountForLogicalLine] when AutoWrap is true, and Y is
	// the distance below FirstLogicalLineInViewport's top that the
	// visible region extends downward. The walk stops once cumulative
	// line heights exceed Y, leaving one line of slack so the
	// caller's inner Y clip can handle off-by-one rounding.
	ViewportSize image.Point

	// Face, LineHeight, TabWidth, KeepTailingSpace are passed through
	// to [VisualLineCountForLogicalLine] when AutoWrap is true.
	Face             text.Face
	LineHeight       float64
	TabWidth         float64
	KeepTailingSpace bool

	// AutoWrap toggles between a per-line shaping walk (true) and a
	// flat LineHeight*idx arithmetic (false).
	AutoWrap bool

	// Composition is the splice info from [ComputeCompositionInfo].
	// The zero value means "no active composition".
	Composition CompositionInfo
}

// VisibleRangeInViewport returns the byte range and logical-line
// indices that cover the visible region when the widget is positioned
// so FirstLogicalLineInViewport sits at widget-local Y=0. Compared to
// [ComputeVisibleRange], this variant requires no precomputed
// CumulativeYs — it walks logical lines forward from
// FirstLogicalLineInViewport and measures each as it goes — so a
// caller pinned to the topmost visible line pays only O(visible)
// typesetting per query. Composition splices on lines past the
// splice are handled the same way [ComputeVisibleRange] does.
//
// ok is false when the document is empty.
//
// VerticalAlign is intentionally not part of the input: when the
// caller pins the viewport at a non-zero logical line, the document
// is assumed to overflow the viewport (the case where alignment
// matters), so YShift is always 0 and the caller's bounds positioning
// carries any needed offset itself.
func VisibleRangeInViewport(p *VisibleRangeInViewportParams) (VisibleRange, bool) {
	n := p.LineByteOffsets.LineCount()
	if n == 0 {
		return VisibleRange{}, false
	}
	first := min(max(p.FirstLogicalLineInViewport, 0), n-1)

	// renderingRangeForLogicalLine returns the [start, end) byte
	// offsets, into the rendering text, of the committed-text logical
	// line at idx. Mirrors the equivalent helper in
	// [TextIndexFromPosition].
	committedTextLen := p.RenderingTextLength - p.Composition.RenderingByteShift
	renderingRangeForLogicalLine := func(idx int) (start, end int) {
		committedStart := p.LineByteOffsets.ByteOffsetByLineIndex(idx)
		committedEnd := committedTextLen
		if idx+1 < n {
			committedEnd = p.LineByteOffsets.ByteOffsetByLineIndex(idx + 1)
		}
		switch {
		case idx < p.Composition.LineIndex:
			return committedStart, committedEnd
		case idx == p.Composition.LineIndex:
			return committedStart, committedEnd + p.Composition.RenderingByteShift
		default:
			return committedStart + p.Composition.RenderingByteShift, committedEnd + p.Composition.RenderingByteShift
		}
	}

	var lastLine int
	if !p.AutoWrap {
		lh := int(math.Ceil(p.LineHeight))
		if lh <= 0 {
			return VisibleRange{}, false
		}
		// One line of slack at the bottom to absorb per-line padding
		// and integer rounding.
		count := p.ViewportSize.Y/lh + 2
		lastLine = min(n-1, first+count-1)
	} else {
		cur := first
		accY := 0
		for cur < n-1 && accY <= p.ViewportSize.Y {
			s, e := renderingRangeForLogicalLine(cur)
			c := VisualLineCountForLogicalLine(p.ViewportSize.X, p.RenderingTextRange(s, e), true, p.Face, p.TabWidth, p.KeepTailingSpace)
			accY += int(math.Ceil(p.LineHeight * float64(c)))
			cur++
		}
		lastLine = cur
	}
	if lastLine < first {
		lastLine = first
	}

	startInBytes, _ := renderingRangeForLogicalLine(first)
	endInBytes := p.RenderingTextLength
	if lastLine+1 < n {
		_, endInBytes = renderingRangeForLogicalLine(lastLine)
		// renderingRangeForLogicalLine(lastLine).end equals the start
		// of lastLine+1 in rendering coordinates, which is what we
		// want for the upper bound of the slice.
	}

	return VisibleRange{
		FirstLine:    first,
		LastLine:     lastLine,
		StartInBytes: startInBytes,
		EndInBytes:   endInBytes,
		YShift:       0,
	}, true
}
