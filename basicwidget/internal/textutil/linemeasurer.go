// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// lineMeasurer maps committed-text logical-line indices to rendering-
// text byte ranges and per-line visual-line counts. It applies the
// composition shifts to lines past the splice; the zero CompositionInfo
// represents "no active composition" and shifts every line by zero.
type lineMeasurer struct {
	offsets            *LineByteOffsets
	logicalLineCount   int
	committedTextLen   int
	renderingTextRange func(start, end int) string
	width              int
	face               text.Face
	tabWidth           float64
	keepTailingSpace   bool
	autoWrap           bool
	composition        CompositionInfo
}

// renderingRange returns the [start, end) byte offsets, into the
// rendering text, of the committed-text logical line at idx. Lines
// before composition.LineIndex coincide with the committed range; the
// line at composition.LineIndex extends its end by RenderingByteShift
// to include the spliced bytes; lines after the composition shift both
// endpoints by the same amount.
func (m *lineMeasurer) renderingRange(idx int) (start, end int) {
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

// visualLineCount returns the rendering-plane visual-line count of the
// logical line at idx. For non-autoWrap text this is always 1; for
// autoWrap it shapes the line content via VisualLineCountForLogicalLine.
func (m *lineMeasurer) visualLineCount(idx int) int {
	if !m.autoWrap {
		return 1
	}
	s, e := m.renderingRange(idx)
	return VisualLineCountForLogicalLine(m.width, m.renderingTextRange(s, e), true, m.face, m.tabWidth, m.keepTailingSpace)
}
