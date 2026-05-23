// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// logicalLineMeasurer maps between committed-text logical-line indices and
// rendering-text byte ranges, reports per-line visual-line counts, and
// resolves a rendering-text byte offset back to its logical line. It applies
// the composition shifts to lines past the splice; the zero CompositionInfo
// represents "no active composition" and shifts every line by zero.
type logicalLineMeasurer struct {
	offsets            *LineByteOffsets
	logicalLineCount   int
	committedTextLen   int
	renderingTextRange func(start, end int) string
	width              int
	face               text.Face
	tabWidth           float64
	keepTailingSpace   bool
	wrapMode           WrapMode
	composition        CompositionInfo

	// compositionRenderingStartPlus1 is one plus the rendering-text byte
	// offset where the active composition begins, or 0 when there is no
	// active composition.
	compositionRenderingStartPlus1 int

	// compositionRenderingEndPlus1 is one plus the rendering-text byte offset
	// just past the active composition, or 0 when there is no active
	// composition.
	compositionRenderingEndPlus1 int
}

// newLogicalLineMeasurer builds the measurement context from the precomputed
// logical-line offsets of p's committed text, resolving any active composition
// once so individual line queries need not redo it. ok is false when the offsets
// are absent or empty, or the composition straddles a logical-line boundary; the
// caller must then fall back to the unrestricted walk.
func newLogicalLineMeasurer(p *TextLayoutParams) (*logicalLineMeasurer, bool) {
	if p.PrecomputedLineByteOffsets == nil {
		return nil, false
	}
	n := p.PrecomputedLineByteOffsets.LineCount()
	if n == 0 {
		return nil, false
	}

	// Resolve composition shifts so the precomputed logical-line offsets are
	// usable without a rebuild. compInfo carries the selection line and the byte
	// shifts applied to lines past the splice.
	var compInfo CompositionInfo
	var compRenderingStartPlus1, compRenderingEndPlus1 int
	if p.CompositionLen > 0 {
		selectionLineIdx := p.PrecomputedLineByteOffsets.LineIndexForByteOffset(p.SelectionStart)
		cs := p.PrecomputedLineByteOffsets.ByteOffsetByLineIndex(selectionLineIdx)
		byteDelta := p.CompositionLen - (p.SelectionEnd - p.SelectionStart)
		ce := p.RenderingTextLength - byteDelta
		if selectionLineIdx+1 < n {
			ce = p.PrecomputedLineByteOffsets.ByteOffsetByLineIndex(selectionLineIdx + 1)
		}
		// Compute the selection-line slices only when the selection lies
		// inside a single logical line; otherwise ce+byteDelta underflows.
		// [ComputeCompositionInfo] rejects the multi-line case before reading them.
		var committedSelectionLine, renderingSelectionLine string
		if p.Options.WrapMode != WrapModeNone && p.PrecomputedLineByteOffsets.LineIndexForByteOffset(p.SelectionEnd) == selectionLineIdx {
			committedSelectionLine = p.CommittedTextRange(cs, ce)
			renderingSelectionLine = p.RenderingTextRange(cs, ce+byteDelta)
		}

		info, ok := ComputeCompositionInfo(&CompositionInfoParams{
			CompositionText:        p.RenderingTextRange(p.SelectionStart, p.SelectionStart+p.CompositionLen),
			LineByteOffsets:        p.PrecomputedLineByteOffsets,
			SelectionStart:         p.SelectionStart,
			SelectionEnd:           p.SelectionEnd,
			WrapMode:               p.Options.WrapMode,
			CommittedSelectionLine: committedSelectionLine,
			RenderingSelectionLine: renderingSelectionLine,
			Face:                   p.Options.Face,
			LineHeight:             p.Options.LineHeight,
			TabWidth:               p.Options.TabWidth,
			KeepTailingSpace:       p.Options.KeepTailingSpace,
			WrapWidth:              p.Width,
		})
		if !ok {
			return nil, false
		}
		compInfo = info
		compRenderingStartPlus1 = p.SelectionStart + 1
		compRenderingEndPlus1 = p.SelectionStart + p.CompositionLen + 1
	}

	return &logicalLineMeasurer{
		offsets:                        p.PrecomputedLineByteOffsets,
		logicalLineCount:               n,
		committedTextLen:               p.RenderingTextLength - compInfo.RenderingByteShift,
		renderingTextRange:             p.RenderingTextRange,
		width:                          p.Width,
		face:                           p.Options.Face,
		tabWidth:                       p.Options.TabWidth,
		keepTailingSpace:               p.Options.KeepTailingSpace,
		wrapMode:                       p.Options.WrapMode,
		composition:                    compInfo,
		compositionRenderingStartPlus1: compRenderingStartPlus1,
		compositionRenderingEndPlus1:   compRenderingEndPlus1,
	}, true
}

// renderingRange returns the [start, end) byte offsets, into the rendering
// text, of the logical line at idx.
func (m *logicalLineMeasurer) renderingRange(idx int) (start, end int) {
	committedStart := m.offsets.ByteOffsetByLineIndex(idx)
	committedEnd := m.committedTextLen
	if idx+1 < m.logicalLineCount {
		committedEnd = m.offsets.ByteOffsetByLineIndex(idx + 1)
	}
	switch {
	case idx < m.composition.LineIndex:
		return committedStart, committedEnd
	case idx == m.composition.LineIndex:
		return committedStart, committedEnd + m.composition.RenderingByteShift
	default:
		return committedStart + m.composition.RenderingByteShift, committedEnd + m.composition.RenderingByteShift
	}
}

// logicalLineIndexForRenderingIndex returns the committed-text logical-line
// index that contains the rendering-text byte offset renderingIndex.
func (m *logicalLineMeasurer) logicalLineIndexForRenderingIndex(renderingIndex int) int {
	// Without an active composition the rendering and committed coordinates
	// coincide.
	if m.compositionRenderingStartPlus1 == 0 {
		return m.offsets.LineIndexForByteOffset(renderingIndex)
	}
	// The composition replaces committed[sStart:sEnd] with rendering bytes
	// [start, end); lines on either side are unaffected other than a constant
	// byte shift past the splice.
	start := m.compositionRenderingStartPlus1 - 1
	end := m.compositionRenderingEndPlus1 - 1
	switch {
	case renderingIndex < start:
		return m.offsets.LineIndexForByteOffset(renderingIndex)
	case renderingIndex <= end:
		return m.composition.LineIndex
	default:
		return m.offsets.LineIndexForByteOffset(renderingIndex - m.composition.RenderingByteShift)
	}
}

// visualLineCount returns the rendering-plane visual-line count of the
// logical line at idx. For [WrapModeNone] text this is always 1; for
// other wrap modes it shapes the line content via
// VisualLineCountForLogicalLine.
func (m *logicalLineMeasurer) visualLineCount(idx int) int {
	if m.wrapMode == WrapModeNone {
		return 1
	}
	s, e := m.renderingRange(idx)
	return VisualLineCountForLogicalLine(m.width, m.renderingTextRange(s, e), m.wrapMode, m.face, m.tabWidth, m.keepTailingSpace)
}
