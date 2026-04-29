// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// CompositionInfoParams describes the inputs for [ComputeCompositionInfo].
type CompositionInfoParams struct {
	// RenderingText is the field text with the composition spliced in
	// at SelectionStart; the composition itself is
	// RenderingText[SelectionStart : SelectionStart+CompositionLen].
	RenderingText string

	// CommittedText is the field text without the composition. Only
	// read when AutoWrap is true, to measure the visual-Y delta of the
	// splice line.
	CommittedText string

	// LineByteOffsets is the logical-line layout of CommittedText.
	LineByteOffsets *LineByteOffsets

	// SelectionStart and SelectionEnd are byte offsets into
	// CommittedText describing the range the composition replaces.
	// SelectionStart == SelectionEnd for a pure insertion.
	SelectionStart int
	SelectionEnd   int

	// CompositionLen is the byte length of the composition in
	// RenderingText.
	CompositionLen int

	// AutoWrap toggles the visual-Y delta measurement for the splice
	// line. When false, RenderingYShift in the result is always 0 and
	// the fields below are ignored.
	AutoWrap bool

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
	// LineIndex is the logical-line index of the splice line. Lines
	// with index > LineIndex are "past the splice" and have
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
	// splice line to wrap into a different number of visual sub-lines.
	RenderingYShift int
}

// ComputeCompositionInfo classifies an active composition and returns
// info for [ComputeVisibleRange]. ok is false when the splice changes
// the logical-line count - a hard line break inside the composition or
// a selection that straddles a logical line boundary - and the caller
// should fall back to drawing the unrestricted text.
func ComputeCompositionInfo(p *CompositionInfoParams) (CompositionInfo, bool) {
	if pos, _ := FirstLineBreakPositionAndLen(p.RenderingText[p.SelectionStart : p.SelectionStart+p.CompositionLen]); pos >= 0 {
		return CompositionInfo{}, false
	}
	lineIndex := p.LineByteOffsets.LineIndexForByteOffset(p.SelectionStart)
	if p.SelectionStart != p.SelectionEnd && p.LineByteOffsets.LineIndexForByteOffset(p.SelectionEnd) != lineIndex {
		return CompositionInfo{}, false
	}
	byteDelta := p.CompositionLen - (p.SelectionEnd - p.SelectionStart)

	var yDelta int
	if p.AutoWrap {
		// Visual height of the splice line in rendering vs committed:
		// the only line whose wrap layout the composition can change.
		n := p.LineByteOffsets.LineCount()
		cs := p.LineByteOffsets.ByteOffsetByLineIndex(lineIndex)
		ce := len(p.CommittedText)
		if lineIndex+1 < n {
			ce = p.LineByteOffsets.ByteOffsetByLineIndex(lineIndex + 1)
		}
		measureWidth := p.WrapWidth
		if measureWidth <= 0 {
			measureWidth = math.MaxInt
		}
		committedH := MeasureLogicalLineHeight(measureWidth, p.CommittedText[cs:ce], true, p.Face, p.LineHeight, p.TabWidth, p.KeepTailingSpace)
		renderingH := MeasureLogicalLineHeight(measureWidth, p.RenderingText[cs:ce+byteDelta], true, p.Face, p.LineHeight, p.TabWidth, p.KeepTailingSpace)
		yDelta = int(math.Ceil(renderingH)) - int(math.Ceil(committedH))
	}
	return CompositionInfo{
		LineIndex:          lineIndex,
		RenderingByteShift: byteDelta,
		RenderingYShift:    yDelta,
	}, true
}

// VisibleRangeParams describes the inputs for [ComputeVisibleRange].
type VisibleRangeParams struct {
	// LineByteOffsets is the logical-line layout of the rendering text
	// (same text that would be passed to [Draw]). The number of logical
	// lines comes from its LineCount.
	LineByteOffsets *LineByteOffsets

	// RenderingLength is the total byte length of the rendering text.
	RenderingLength int

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
	endInBytes := p.RenderingLength
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
