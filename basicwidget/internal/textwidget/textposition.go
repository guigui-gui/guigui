// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"image"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

// LineCount returns the number of logical lines (spans between hard line
// breaks) in the value. The empty value has one logical line; a trailing
// line break creates an extra empty line at the end.
func (t *Text) LineCount() int {
	t.ensureLineByteOffsets()
	return t.contentCache.lineByteOffsets.LineCount()
}

// LineStartInBytes returns the byte offset where the lineIndex-th logical
// line begins within the value. lineIndex must be in [0, [Text.LineCount]).
func (t *Text) LineStartInBytes(lineIndex int) int {
	t.ensureLineByteOffsets()
	return t.contentCache.lineByteOffsets.ByteOffsetByLineIndex(lineIndex)
}

// LineIndexFromTextIndexInBytes returns the index of the logical line
// containing textIndexInBytes, clamping out-of-range values to the first or
// last line.
func (t *Text) LineIndexFromTextIndexInBytes(textIndexInBytes int) int {
	t.ensureLineByteOffsets()
	return t.contentCache.lineByteOffsets.LineIndexForByteOffset(textIndexInBytes)
}

// CaretPositionAtTextIndexInBytes returns the on-screen top and bottom
// endpoints of a caret drawn at byte offset textIndexInBytes. ok is false
// when the offset is out of range or the caret's logical line is outside the
// viewport. Available after the layout phase.
func (t *Text) CaretPositionAtTextIndexInBytes(context *guigui.Context, textIndexInBytes int) (top, bottom image.Point, ok bool) {
	if t.widgetBoundsRect.Empty() {
		return image.Point{}, image.Point{}, false
	}
	if textIndexInBytes < 0 || textIndexInBytes > t.store.TextLengthInBytes() {
		return image.Point{}, image.Point{}, false
	}
	if !t.isLogicalLineMaybeVisible(context, t.widgetBoundsRect, textIndexInBytes) {
		return image.Point{}, image.Point{}, false
	}
	pos, ok := t.textPosition(context, t.widgetBoundsRect, textIndexInBytes, false)
	if !ok {
		return image.Point{}, image.Point{}, false
	}
	return image.Pt(int(pos.X), int(pos.Top)), image.Pt(int(pos.X), int(pos.Bottom)), true
}

func (t *Text) caretPosition(context *guigui.Context, textBounds image.Rectangle) (position textutil.TextPosition, ok bool) {
	if !context.IsFocused(t) {
		return textutil.TextPosition{}, false
	}
	if !t.editable {
		return textutil.TextPosition{}, false
	}
	start, end := t.store.Selection()
	if start < 0 {
		return textutil.TextPosition{}, false
	}
	if end < 0 {
		return textutil.TextPosition{}, false
	}
	// A non-empty selection draws as a highlight, not a caret;
	// [textCaret.alpha] returns 0 in that case, so no callers need
	// the position.
	if start != end {
		return textutil.TextPosition{}, false
	}

	// Skip the textPosition walk when the caret's line is off-screen;
	// it can dominate CPU when the user has scrolled far from the caret.
	if !t.isLogicalLineMaybeVisible(context, textBounds, end) {
		return textutil.TextPosition{}, false
	}

	_, e, ok := t.selectionToDraw(context)
	if !ok {
		return textutil.TextPosition{}, false
	}

	return t.textPosition(context, textBounds, e, true)
}

// isLogicalLineMaybeVisible reports whether the logical line containing the
// committed byte offset byteOffset could be inside textBounds. It is
// conservative: a true result means "compute the exact pixel position to know
// for sure"; a false result means "definitely off-screen, no need to walk".
// textBounds is the parent Text's bounds (the rectangle textPosition is
// resolved against), which is also the visible viewport in the
// virtualization-aware layouts that drive the hot path.
func (t *Text) isLogicalLineMaybeVisible(context *guigui.Context, textBounds image.Rectangle, byteOffset int) bool {
	if textBounds.Empty() {
		// No Layout has run yet (or Text is not laid out). Defer to
		// the exact path so behavior matches the pre-short-circuit code.
		return true
	}
	t.ensureLineByteOffsets()
	n := t.contentCache.lineByteOffsets.LineCount()
	if n == 0 {
		return true
	}
	line := t.contentCache.lineByteOffsets.LineIndexForByteOffset(byteOffset)
	first := t.firstLogicalLineInViewport
	if line < first {
		return false
	}
	// The line's top sits at or below
	//   textBounds.Min.Y + (line-first)*lineHeight
	// because each preceding logical line contributes at least one
	// visual line of height lineHeight. If that lower bound is already
	// past the bounds bottom, the actual top is too.
	lh := t.LineHeight()
	minTop := float64(textBounds.Min.Y) + lh*float64(line-first)
	if minTop >= float64(textBounds.Max.Y) {
		return false
	}
	return true
}

func (t *Text) textIndexFromPosition(context *guigui.Context, textBounds image.Rectangle, position image.Point, showComposition bool) int {
	textContentBounds := t.contentBoundsForLayout(context, textBounds)

	if t.masking() {
		m := t.maskMappingForRendering(showComposition)
		s := t.maskStyle(context)
		mi := textutil.TextIndexFromPositionInLogicalLine(textContentBounds.Dx(), position.Sub(textContentBounds.Min), m.maskStr, &s)
		if mi < 0 {
			return -1
		}
		return m.offsetFromMasked(mi)
	}

	// Compute the rendering text's byte length without materializing
	// it. RenderingTextLength = committedLength + composition byte delta
	// when composition is active and visible; otherwise == committedLength.
	renderingLength := t.store.TextLengthInBytes()
	var sStart, sEnd, compLen int
	if showComposition {
		compLen = t.store.UncommittedTextLengthInBytes()
		if compLen > 0 {
			sStart, sEnd = t.store.Selection()
			renderingLength = renderingLength + compLen - (sEnd - sStart)
		}
	}

	width := t.LayoutWidth(textContentBounds)
	t.faceRunsBuf = t.appendFaceRunsForStyle(t.faceRunsBuf, context, false)
	defer func() {
		t.faceRunsBuf = slices.Delete(t.faceRunsBuf, 0, len(t.faceRunsBuf))
	}()
	s := textutil.Style{
		WrapMode:         t.wrapMode,
		Face:             t.face(context, false),
		FaceRuns:         t.faceRunsBuf,
		LineHeight:       t.LineHeight(),
		HorizontalAlign:  t.style.hAlign,
		VerticalAlign:    t.style.vAlign,
		TabWidth:         t.actualTabWidth(context),
		KeepTailingSpace: t.keepTailingSpace,
	}
	position = position.Sub(textContentBounds.Min)

	// Pass the firstLogicalLineInViewport as the textutil walk hint.
	// Virtualizing parents set this to the
	// topmost visible logical line, so the walker only measures
	// O(visible) lines per query instead of walking from line 0.
	// Standalone Text leaves it at 0, which matches the historical
	// "walk from line 0" behavior — fine for small documents.
	t.ensureLineByteOffsets()
	hintLL := t.firstLogicalLineInViewport

	readRendering := t.stringValueWithRange
	if showComposition {
		readRendering = t.stringValueForRenderingRange
	}
	var readCommitted func(start, end int) string
	if compLen > 0 {
		readCommitted = t.stringValueWithRange
	}
	idx := textutil.TextIndexFromPosition(&textutil.TextLayoutParams{
		RenderingTextRange:         readRendering,
		RenderingTextLength:        renderingLength,
		Width:                      width,
		Style:                      s,
		CommittedTextRange:         readCommitted,
		PrecomputedLineByteOffsets: &t.contentCache.lineByteOffsets,
		SelectionStart:             sStart,
		SelectionEnd:               sEnd,
		CompositionLen:             compLen,
		LogicalLineIndexHint:       hintLL,
	}, position)
	if idx < 0 || idx > renderingLength {
		return -1
	}
	return idx
}

func (t *Text) textPosition(context *guigui.Context, bounds image.Rectangle, index int, showComposition bool) (position textutil.TextPosition, ok bool) {
	textBounds := t.contentBoundsForLayout(context, bounds)

	if t.masking() {
		m := t.maskMappingForRendering(showComposition)
		s := t.maskStyle(context)
		pos0, pos1, count := textutil.TextPositionFromIndexInLogicalLine(textBounds.Dx(), m.maskStr, m.offsetToMasked(index), &s)
		if count == 0 {
			return textutil.TextPosition{}, false
		}
		pos := pos0
		if count == 2 {
			pos = pos1
		}
		return textutil.TextPosition{
			X:      pos.X + float64(textBounds.Min.X),
			Top:    pos.Top + float64(textBounds.Min.Y),
			Bottom: pos.Bottom + float64(textBounds.Min.Y),
		}, true
	}

	width := t.LayoutWidth(textBounds)
	t.faceRunsBuf = t.appendFaceRunsForStyle(t.faceRunsBuf, context, false)
	defer func() {
		t.faceRunsBuf = slices.Delete(t.faceRunsBuf, 0, len(t.faceRunsBuf))
	}()
	s := textutil.Style{
		WrapMode:         t.wrapMode,
		Face:             t.face(context, false),
		FaceRuns:         t.faceRunsBuf,
		LineHeight:       t.LineHeight(),
		HorizontalAlign:  t.style.hAlign,
		VerticalAlign:    t.style.vAlign,
		TabWidth:         t.actualTabWidth(context),
		KeepTailingSpace: t.keepTailingSpace,
	}

	// Pass the cached lineByteOffsets and the
	// firstLogicalLineInViewport hint so
	// [textutil.TextPositionFromIndex] walks only the logical lines
	// between the viewport's first line and the caret's line. The
	// fallback without precomputed offsets walks every visual line in the
	// document; for multi-megabyte buffers caretPosition / adjustScrollOffset
	// call this every tick and that fallback dominates CPU.
	t.ensureLineByteOffsets()

	renderingLength := t.store.TextLengthInBytes()
	var sStart, sEnd, compLen int
	if showComposition {
		compLen = t.store.UncommittedTextLengthInBytes()
		if compLen > 0 {
			sStart, sEnd = t.store.Selection()
			renderingLength = renderingLength + compLen - (sEnd - sStart)
		}
	}
	readRendering := t.stringValueWithRange
	if showComposition {
		readRendering = t.stringValueForRenderingRange
	}
	var readCommitted func(start, end int) string
	if compLen > 0 {
		readCommitted = t.stringValueWithRange
	}
	// firstLogicalLineInViewport pins TextPositionFromIndex's Y origin
	// to the line at widget-local Y=0 (the line that
	// virtualizing parent positioned at the panel viewport top); the
	// returned pos.Top is therefore relative to that line, ready to
	// add to textBounds.Min.Y for screen coordinates. The walk is
	// bounded by the logical-line distance between firstLine and the
	// caret's line, which is a viewport's worth of lines for carets
	// visible on screen.
	pos0, pos1, count := textutil.TextPositionFromIndex(&textutil.TextLayoutParams{
		RenderingTextRange:         readRendering,
		RenderingTextLength:        renderingLength,
		Width:                      width,
		Style:                      s,
		CommittedTextRange:         readCommitted,
		PrecomputedLineByteOffsets: &t.contentCache.lineByteOffsets,
		SelectionStart:             sStart,
		SelectionEnd:               sEnd,
		CompositionLen:             compLen,
		LogicalLineIndexHint:       t.firstLogicalLineInViewport,
	}, index)
	if count == 0 {
		return textutil.TextPosition{}, false
	}
	pos := pos0
	if count == 2 {
		pos = pos1
	}
	return textutil.TextPosition{
		X:      pos.X + float64(textBounds.Min.X),
		Top:    pos.Top + float64(textBounds.Min.Y),
		Bottom: pos.Bottom + float64(textBounds.Min.Y),
	}, true
}

// CaretScrollTarget describes one caret edge for scroll-into-view requests.
type CaretScrollTarget struct {
	// LogicalLineIndex is the caret's committed-text logical-line index.
	LogicalLineIndex int

	// X is the caret's textBounds-relative X coordinate.
	X float64

	// Top is the caret's top Y, measured from the start of the logical line.
	Top float64

	// Bottom is the caret's bottom Y, measured from the start of the logical line.
	Bottom float64
}

// caretPositionWithinLine returns the caret's logical-line index and its
// line-relative position. Costs one logical-line shape regardless of where
// the caret sits in the document.
func (t *Text) caretPositionWithinLine(context *guigui.Context, bounds image.Rectangle, index int, showComposition bool) (target CaretScrollTarget, ok bool) {
	textBounds := t.contentBoundsForLayout(context, bounds)

	if t.masking() {
		m := t.maskMappingForRendering(showComposition)
		s := t.maskStyle(context)
		pos0, pos1, count := textutil.TextPositionFromIndexInLogicalLine(textBounds.Dx(), m.maskStr, m.offsetToMasked(index), &s)
		if count == 0 {
			return CaretScrollTarget{}, false
		}
		pos := pos0
		if count == 2 {
			pos = pos1
		}
		return CaretScrollTarget{
			LogicalLineIndex: 0,
			X:                pos.X + float64(textBounds.Min.X),
			Top:              pos.Top,
			Bottom:           pos.Bottom,
		}, true
	}

	width := t.LayoutWidth(textBounds)
	t.faceRunsBuf = t.appendFaceRunsForStyle(t.faceRunsBuf, context, false)
	defer func() {
		t.faceRunsBuf = slices.Delete(t.faceRunsBuf, 0, len(t.faceRunsBuf))
	}()
	s := textutil.Style{
		WrapMode:         t.wrapMode,
		Face:             t.face(context, false),
		FaceRuns:         t.faceRunsBuf,
		LineHeight:       t.LineHeight(),
		HorizontalAlign:  t.style.hAlign,
		VerticalAlign:    t.style.vAlign,
		TabWidth:         t.actualTabWidth(context),
		KeepTailingSpace: t.keepTailingSpace,
	}
	t.ensureLineByteOffsets()

	renderingLength := t.store.TextLengthInBytes()
	var sStart, sEnd, compLen int
	if showComposition {
		compLen = t.store.UncommittedTextLengthInBytes()
		if compLen > 0 {
			sStart, sEnd = t.store.Selection()
			renderingLength = renderingLength + compLen - (sEnd - sStart)
		}
	}
	readRendering := t.stringValueWithRange
	if showComposition {
		readRendering = t.stringValueForRenderingRange
	}
	var readCommitted func(start, end int) string
	if compLen > 0 {
		readCommitted = t.stringValueWithRange
	}
	li, pos0, pos1, count := textutil.PositionWithinLogicalLine(&textutil.TextLayoutParams{
		RenderingTextRange:         readRendering,
		RenderingTextLength:        renderingLength,
		Width:                      width,
		Style:                      s,
		CommittedTextRange:         readCommitted,
		PrecomputedLineByteOffsets: &t.contentCache.lineByteOffsets,
		SelectionStart:             sStart,
		SelectionEnd:               sEnd,
		CompositionLen:             compLen,
	}, index)
	if count == 0 {
		return CaretScrollTarget{}, false
	}
	pos := pos0
	if count == 2 {
		pos = pos1
	}
	return CaretScrollTarget{
		LogicalLineIndex: li,
		X:                pos.X + float64(textBounds.Min.X),
		Top:              pos.Top,
		Bottom:           pos.Bottom,
	}, true
}

func (t *Text) caretBounds(context *guigui.Context, textBounds image.Rectangle) image.Rectangle {
	pos, ok := t.caretPosition(context, textBounds)
	if !ok {
		return image.Rectangle{}
	}
	w := textCaretWidth(context)
	paddingTop := 2 * t.Scale() * context.Scale()
	paddingBottom := 1 * t.Scale() * context.Scale()
	return image.Rect(int(pos.X)-w/2, int(pos.Top+paddingTop), int(pos.X)+w/2, int(pos.Bottom-paddingBottom))
}

func (t *Text) adjustScrollOffset(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (dx, dy float64) {
	start, end, ok := t.selectionToDraw(context)
	if !ok {
		return
	}
	if t.prevStart == start && t.prevEnd == end && !t.dragState.dragging {
		return
	}
	t.prevStart = start
	t.prevEnd = end

	textBounds := widgetBounds.Bounds()
	textVisibleBounds := widgetBounds.VisibleBounds()

	if t.dragState.dragging {
		// Drag autoscroll tracks the mouse, not the caret.
		cx, cy := ebiten.CursorPosition()
		exEnd := float64(textVisibleBounds.Max.X) - float64(cx) - float64(t.paddingForScrollOffset.End)
		eyEnd := float64(textVisibleBounds.Max.Y) - float64(cy) - float64(t.paddingForScrollOffset.Bottom)
		if cx > textVisibleBounds.Max.X {
			exEnd /= 4
		} else {
			exEnd = 0
		}
		if cy > textVisibleBounds.Max.Y {
			eyEnd /= 4
		} else {
			eyEnd = 0
		}
		dx += min(exEnd, 0)
		dy += min(eyEnd, 0)
		exStart := float64(textVisibleBounds.Min.X) - float64(cx) + float64(t.paddingForScrollOffset.Start)
		eyStart := float64(textVisibleBounds.Min.Y) - float64(cy) + float64(t.paddingForScrollOffset.Top)
		if cx < textVisibleBounds.Min.X {
			exStart /= 4
		} else {
			exStart = 0
		}
		if cy < textVisibleBounds.Min.Y {
			eyStart /= 4
		} else {
			eyStart = 0
		}
		dx += max(exStart, 0)
		dy += max(eyStart, 0)
		return dx, dy
	}

	endTarget, ok := t.caretPositionWithinLine(context, textBounds, end, true)
	if !ok {
		return 0, 0
	}
	startTarget := endTarget
	if start != end {
		if st, ok := t.caretPositionWithinLine(context, textBounds, start, true); ok {
			startTarget = st
		}
	}
	guigui.DispatchEvent(t, textEventScrollIntoView, startTarget, endTarget)
	return 0, 0
}

// VisualLineCountOfLogicalLine returns the number of visual lines the
// lineIndex-th logical line occupies when wrapped at wrapWidth. lineIndex must
// be in [0, [Text.LineCount]).
func (t *Text) VisualLineCountOfLogicalLine(context *guigui.Context, lineIndex, wrapWidth int) int {
	t.ensureLineByteOffsets()
	n := t.contentCache.lineByteOffsets.LineCount()
	start := t.contentCache.lineByteOffsets.ByteOffsetByLineIndex(lineIndex)
	end := t.store.TextLengthInBytes()
	if lineIndex+1 < n {
		end = t.contentCache.lineByteOffsets.ByteOffsetByLineIndex(lineIndex + 1)
	}
	line := t.stringValueWithRange(start, end)
	return textutil.CachedVisualLineCount(wrapWidth, line, t.wrapMode, t.face(context, false), t.actualTabWidth(context), t.keepTailingSpace)
}

// MaxCaretXOfLogicalLine returns the maximum caret X coordinate over the
// visual lines of the lineIndex-th logical line when wrapped at wrapWidth.
// lineIndex must be in [0, [Text.LineCount]).
func (t *Text) MaxCaretXOfLogicalLine(context *guigui.Context, lineIndex, wrapWidth int) float64 {
	t.ensureLineByteOffsets()
	n := t.contentCache.lineByteOffsets.LineCount()
	start := t.contentCache.lineByteOffsets.ByteOffsetByLineIndex(lineIndex)
	end := t.store.TextLengthInBytes()
	if lineIndex+1 < n {
		end = t.contentCache.lineByteOffsets.ByteOffsetByLineIndex(lineIndex + 1)
	}
	line := t.stringValueWithRange(start, end)
	return textutil.CachedVisualLineMaxCaretX(wrapWidth, line, t.wrapMode, t.face(context, false), t.actualTabWidth(context), t.keepTailingSpace)
}
