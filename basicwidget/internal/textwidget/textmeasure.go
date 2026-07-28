// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"image"
	"math"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

func (t *Text) textContentBounds(context *guigui.Context, bounds image.Rectangle) image.Rectangle {
	b := bounds
	h := t.textHeight(context, guigui.FixedWidthConstraints(t.LayoutWidth(b)))

	switch t.baseStyle.vAlign {
	case textutil.VerticalAlignTop:
		b.Max.Y = b.Min.Y + h
	case textutil.VerticalAlignMiddle:
		dy := b.Dy()
		b.Min.Y += (dy - h) / 2
		b.Max.Y = b.Min.Y + h
	case textutil.VerticalAlignBottom:
		b.Min.Y = b.Max.Y - h
	}

	return b
}

// contentBoundsForLayout returns the bounds for laying out content.
func (t *Text) contentBoundsForLayout(context *guigui.Context, bounds image.Rectangle) image.Rectangle {
	if t.baseStyle.vAlign == textutil.VerticalAlignTop {
		// For Top, [Text.textContentBounds] would only tighten Max.Y, which
		// no caller depends on beyond it staying within bounds. Skip it to
		// avoid [Text.textHeight], which walks every logical line for wrapped text.
		return bounds
	}
	return t.textContentBounds(context, bounds)
}

func (t *Text) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.textSize(context, constraints, false)
}

// MeasureBold returns the size of the text under constraints as if it were
// rendered bold.
func (t *Text) MeasureBold(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.textSize(context, constraints, true)
}

// textHeight returns the height of the rendered text under the given
// constraints, without computing the width. Skipping width avoids per-line
// shaping, which dominates the cost for very long text.
func (t *Text) textHeight(context *guigui.Context, constraints guigui.Constraints) int {
	if t.masking() {
		// A masked value never wraps, so it is always one visual line.
		return int(math.Ceil(t.LineHeight()))
	}

	constraintWidth := math.MaxInt
	if w, ok := constraints.FixedWidth(); ok {
		constraintWidth = w
	}
	if constraintWidth == 0 {
		constraintWidth = 1
	}

	const bold = false
	t.invalidateSizeCacheForMetricStyleRuns()
	key := newTextSizeCacheKey(t.wrapMode, bold)

	if h, ok := t.sizeCache.height(key, constraintWidth); ok {
		return h
	}

	lineH := t.LineHeight()
	var hi int
	if visualCount, ok := t.totalRenderingVisualLineCount(context, constraintWidth, bold); ok {
		hi = int(math.Ceil(lineH * float64(visualCount)))
	} else {
		// Fallback when an active composition contains a hard line break
		// or straddles a logical-line boundary — the rendering text's
		// logical-line shape doesn't match the committed-text logical-line offsets.
		txt := t.textToDraw(context, true)
		_, renderingFaceRuns, mark := t.acquireFaceRuns(context, bold, true)
		defer t.releaseFaceRuns(mark)
		h := textutil.MeasureHeight(constraintWidth, txt, t.wrapMode, t.face(context, bold), renderingFaceRuns, lineH, t.actualTabWidth(context), t.keepTailingSpace)
		hi = int(math.Ceil(h))
	}

	t.sizeCache.setHeight(key, constraintWidth, hi)

	return hi
}

// totalRenderingVisualLineCount returns the visual-line count of the
// rendering text (committed text with the active composition spliced in)
// at the given width without materializing the full document. Returns
// ok=false when the composition contains a hard line break or the
// composition's selection straddles logical lines; the caller falls
// back to [textutil.MeasureHeight] on the full rendering text in that
// case.
//
// For wrapped text, walks logical lines summing per-line wrap counts via
// [textutil.VisualLineCountForLogicalLine]; reads each line through
// the per-range field methods (committed bytes for unaffected lines,
// rendering bytes for the composition's selection line) so no full-
// document materialization is needed.
func (t *Text) totalRenderingVisualLineCount(context *guigui.Context, width int, bold bool) (int, bool) {
	t.ensureLineByteOffsets()
	n := t.contentCache.lineByteOffsets.LineCount()

	hasComp := t.store.UncommittedTextLengthInBytes() > 0
	var sStart, sEnd, compLen, byteDelta int
	selectionLineIdx := -1
	if hasComp {
		sStart, sEnd = t.store.Selection()
		compLen = t.store.UncommittedTextLengthInBytes()
		byteDelta = compLen - (sEnd - sStart)
		compositionText := t.stringValueForRenderingRange(sStart, sStart+compLen)
		if pos, _ := textutil.FirstLineBreakPositionAndLen(compositionText); pos >= 0 {
			return 0, false
		}
		selectionLineIdx = t.contentCache.lineByteOffsets.LineIndexForByteOffset(sStart)
		if t.contentCache.lineByteOffsets.LineIndexForByteOffset(sEnd) != selectionLineIdx {
			return 0, false
		}
	}

	// textutil.WrapModeNone: each logical line is one visual line; composition
	// can't change that (single-line composition keeps the line count).
	if t.wrapMode == textutil.WrapModeNone {
		return n, true
	}

	// Wrapped text: walk logical lines summing per-line wrap counts.
	// Reads the rendering content for the composition's selection line
	// (so the wrap delta is included naturally) and committed content
	// for everything else.
	face := t.face(context, bold)
	tabW := t.actualTabWidth(context)
	keepTailing := t.keepTailingSpace
	measureWidth := width
	if measureWidth <= 0 {
		measureWidth = math.MaxInt
	}
	committedFaceRuns, renderingFaceRuns, mark := t.acquireFaceRuns(context, bold, true)
	defer t.releaseFaceRuns(mark)
	totalLen := t.store.TextLengthInBytes()
	var count int
	for i := range n {
		cs := t.contentCache.lineByteOffsets.ByteOffsetByLineIndex(i)
		ce := totalLen
		if i+1 < n {
			ce = t.contentCache.lineByteOffsets.ByteOffsetByLineIndex(i + 1)
		}
		var line string
		faceRuns := committedFaceRuns
		if hasComp && i == selectionLineIdx {
			// The splice line is read in rendering coordinates; its start
			// offset cs precedes the splice and is the same in both spaces.
			line = t.stringValueForRenderingRange(cs, ce+byteDelta)
			faceRuns = renderingFaceRuns
		} else {
			line = t.stringValueWithRange(cs, ce)
		}
		count += textutil.VisualLineCountForLogicalLine(measureWidth, line, t.wrapMode, face, faceRuns, cs, tabW, keepTailing)
	}
	return count, true
}

// totalRenderingMeasurement returns the rendered width and height of the
// rendering text (committed text with the active composition spliced in)
// at the given width without materializing the full document. Walks
// logical lines via [Text.lineByteOffsets], reading each via the per-
// range field methods (committed line bytes for unaffected lines, the
// rendering line bytes for the selection line under composition), and
// shapes each line with [textutil.MeasureLogicalLine] using
// [Text.face](context, bold) — so bold and tabular settings are picked
// up directly from the requested face, no cache to mismatch.
//
// Returns ok=false when the composition contains a hard line break or
// when the composition's selection straddles logical lines; the caller
// falls back to [textutil.Measure] on the full rendering text.
func (t *Text) totalRenderingMeasurement(context *guigui.Context, width int, bold bool, ellipsisString string) (float64, float64, bool) {
	t.ensureLineByteOffsets()
	n := t.contentCache.lineByteOffsets.LineCount()

	hasComp := t.store.UncommittedTextLengthInBytes() > 0
	var sStart, sEnd, compLen, byteDelta int
	selectionLineIdx := -1
	if hasComp {
		sStart, sEnd = t.store.Selection()
		compLen = t.store.UncommittedTextLengthInBytes()
		byteDelta = compLen - (sEnd - sStart)
		compositionText := t.stringValueForRenderingRange(sStart, sStart+compLen)
		if pos, _ := textutil.FirstLineBreakPositionAndLen(compositionText); pos >= 0 {
			return 0, 0, false
		}
		selectionLineIdx = t.contentCache.lineByteOffsets.LineIndexForByteOffset(sStart)
		if t.contentCache.lineByteOffsets.LineIndexForByteOffset(sEnd) != selectionLineIdx {
			return 0, 0, false
		}
	}

	lineH := t.LineHeight()
	face := t.face(context, bold)
	tabW := t.actualTabWidth(context)
	keepTailing := t.keepTailingSpace
	measureWidth := width
	if measureWidth <= 0 {
		measureWidth = math.MaxInt
	}
	committedFaceRuns, renderingFaceRuns, mark := t.acquireFaceRuns(context, bold, true)
	defer t.releaseFaceRuns(mark)
	totalLen := t.store.TextLengthInBytes()

	var maxWidth, height float64
	for i := range n {
		cs := t.contentCache.lineByteOffsets.ByteOffsetByLineIndex(i)
		ce := totalLen
		if i+1 < n {
			ce = t.contentCache.lineByteOffsets.ByteOffsetByLineIndex(i + 1)
		}
		var line string
		faceRuns := committedFaceRuns
		if hasComp && i == selectionLineIdx {
			// The splice line is read in rendering coordinates; its start
			// offset cs precedes the splice and is the same in both spaces.
			line = t.stringValueForRenderingRange(cs, ce+byteDelta)
			faceRuns = renderingFaceRuns
		} else {
			line = t.stringValueWithRange(cs, ce)
		}
		w, h := textutil.MeasureLogicalLine(measureWidth, line, t.wrapMode, face, faceRuns, cs, lineH, tabW, keepTailing, ellipsisString)
		maxWidth = max(maxWidth, w)
		height += h
	}
	return maxWidth, height, true
}

func (t *Text) textSize(context *guigui.Context, constraints guigui.Constraints, forceBold bool) image.Point {
	bold := forceBold

	if t.masking() {
		// A masked value is a single uniform line; measure it directly rather
		// than through the cache, which is populated from the real text.
		m := t.maskMappingForRendering(true)
		w, h := textutil.Measure(math.MaxInt, m.maskStr, textutil.WrapModeNone, t.face(context, bold), nil, t.LineHeight(), t.actualTabWidth(context), t.keepTailingSpace, "")
		return image.Pt(max(int(math.Ceil(w)), 1), int(math.Ceil(h)))
	}

	constraintWidth := math.MaxInt
	if w, ok := constraints.FixedWidth(); ok {
		constraintWidth = w
	}
	if constraintWidth == 0 {
		constraintWidth = 1
	}

	t.invalidateSizeCacheForMetricStyleRuns()
	key := newTextSizeCacheKey(t.wrapMode, bold)

	width, hasWidth := t.sizeCache.width(key, constraintWidth)
	height, hasHeight := t.sizeCache.height(key, constraintWidth)
	if hasWidth && hasHeight {
		return image.Pt(width, height)
	}

	ellipsisString := t.ellipsisString
	if t.editable {
		ellipsisString = ""
	}
	var w, h float64
	if mw, mh, ok := t.totalRenderingMeasurement(context, constraintWidth, bold, ellipsisString); ok {
		w, h = mw, mh
	} else {
		// Fallback when the composition contains a hard line break or
		// straddles logical lines.
		txt := t.textToDraw(context, true)
		_, renderingFaceRuns, mark := t.acquireFaceRuns(context, bold, true)
		defer t.releaseFaceRuns(mark)
		w, h = textutil.Measure(constraintWidth, txt, t.wrapMode, t.face(context, bold), renderingFaceRuns, t.LineHeight(), t.actualTabWidth(context), t.keepTailingSpace, ellipsisString)
	}
	// If width is 0, the text's bounds and visible bounds are empty, and nothing including its caret is rendered.
	// Force to set a positive number as the width.
	w = max(w, 1)

	if !hasWidth {
		width = int(math.Ceil(w))
		t.sizeCache.setWidth(key, constraintWidth, width)
	}
	if !hasHeight {
		height = int(math.Ceil(h))
		t.sizeCache.setHeight(key, constraintWidth, height)
	}

	return image.Pt(width, height)
}
