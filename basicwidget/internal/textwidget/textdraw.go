// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"image"
	"image/color"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

func (t *Text) textToDraw(context *guigui.Context, showComposition bool) string {
	if showComposition && t.store.UncommittedTextLengthInBytes() > 0 {
		return t.stringValueForRendering()
	}
	return t.stringValue()
}

// restrictedTextToDraw is [Text.textToDraw] restricted to just the logical
// lines that intersect visibleBounds when conditions allow it. When
// restricted is true the caller shifts textBounds.Min.Y by yShift,
// subtracts byteStart from selection / composition byte offsets, and
// forces [textutil.VerticalAlignTop] before calling [textutil.Draw];
// otherwise txt is the full text and the other returns are zero.
//
// The full rendering text is materialized lazily — only when a fallback
// path needs it or when an active IME composition forces it (because
// [textutil.ComputeCompositionInfo] currently consumes the full string).
// On the happy path with no composition the visible byte range is read
// directly from the store via [Text.stringValueWithRange], so the
// per-frame allocation is bounded by the visible window rather than the
// document length.
//
// committedFaceRuns and renderingFaceRuns are the ranged styles' face runs
// in committed-text and rendering-text byte offsets respectively; without an
// active composition they are the same slice.
func (t *Text) restrictedTextToDraw(context *guigui.Context, textBounds, visibleBounds image.Rectangle, committedFaceRuns, renderingFaceRuns []textutil.FaceRun, insertion textutil.Insertion) (txt string, byteStart int, yShift int, restricted bool) {
	t.ensureLineByteOffsets()
	n := t.contentCache.lineByteOffsets.LineCount()

	hasComp := t.store.UncommittedTextLengthInBytes() > 0
	var fullTxt string
	var fullTxtMaterialized bool
	materializeFull := func() string {
		if !fullTxtMaterialized {
			fullTxt = t.textToDraw(context, true)
			fullTxtMaterialized = true
		}
		return fullTxt
	}

	if n == 0 {
		return materializeFull(), 0, 0, false
	}

	width := t.LayoutWidth(textBounds)

	var compInfo textutil.CompositionInfo
	if hasComp {
		sStart, sEnd := t.store.Selection()
		compLen := t.store.UncommittedTextLengthInBytes()
		byteDelta := compLen - (sEnd - sStart)

		selectionLineIdx := t.contentCache.lineByteOffsets.LineIndexForByteOffset(sStart)
		cs := t.contentCache.lineByteOffsets.ByteOffsetByLineIndex(selectionLineIdx)
		ce := t.store.TextLengthInBytes()
		if selectionLineIdx+1 < n {
			ce = t.contentCache.lineByteOffsets.ByteOffsetByLineIndex(selectionLineIdx + 1)
		}
		// The selection-line slices are only valid when the selection
		// lies inside a single logical line; otherwise ce+byteDelta
		// underflows. When the selection crosses lines we leave them
		// empty — [textutil.ComputeCompositionInfo]'s own multi-line
		// check returns false before reading them, and the caller falls
		// back below.
		var committedSelectionLine, renderingSelectionLine string
		if t.wrapMode != textutil.WrapModeNone && t.contentCache.lineByteOffsets.LineIndexForByteOffset(sEnd) == selectionLineIdx {
			committedSelectionLine = t.stringValueWithRange(cs, ce)
			renderingSelectionLine = t.stringValueForRenderingRange(cs, ce+byteDelta)
		}

		info, ok := textutil.ComputeCompositionInfo(&textutil.CompositionInfoParams{
			CompositionText:           t.stringValueForRenderingRange(sStart, sStart+compLen),
			LineByteOffsets:           &t.contentCache.lineByteOffsets,
			SelectionStart:            sStart,
			SelectionEnd:              sEnd,
			WrapMode:                  t.wrapMode,
			CommittedSelectionLine:    committedSelectionLine,
			RenderingSelectionLine:    renderingSelectionLine,
			Face:                      t.face(context, false),
			LineHeight:                t.LineHeight(),
			LineHeightMode:            t.baseStyle.lineHeightMode,
			TabWidth:                  t.actualTabWidth(context),
			KeepTailingSpace:          t.keepTailingSpace,
			CommittedFaceRuns:         committedFaceRuns,
			RenderingFaceRuns:         renderingFaceRuns,
			SelectionLineStartInBytes: cs,
			WrapWidth:                 width,
		})
		if !ok {
			return materializeFull(), 0, 0, false
		}
		compInfo = info
	}

	lineH := int(math.Ceil(t.LineHeight()))
	if lineH <= 0 {
		return materializeFull(), 0, 0, false
	}

	renderingLength := t.store.TextLengthInBytes()
	if hasComp {
		sStart, sEnd := t.store.Selection()
		renderingLength = renderingLength - (sEnd - sStart) + t.store.UncommittedTextLengthInBytes()
	}

	// vAlign==Top: the walker starts at firstLogicalLineInViewport
	// (the line that the virtualizing parent pinned at widget-local Y=0)
	// and measures only lines from there downward. Other alignments
	// need a totalHeight-based shift; the branch below computes that
	// and walks from line 0.
	if t.baseStyle.vAlign == textutil.VerticalAlignTop {
		readRendering := t.stringValueWithRange
		if hasComp {
			readRendering = t.stringValueForRenderingRange
		}
		r, ok := textutil.VisibleRangeInViewport(&textutil.VisibleRangeInViewportParams{
			FirstLogicalLineInViewport: t.firstLogicalLineInViewport,
			LineByteOffsets:            &t.contentCache.lineByteOffsets,
			RenderingTextRange:         readRendering,
			RenderingTextLength:        renderingLength,
			ViewportSize: image.Pt(
				width,
				visibleBounds.Max.Y-textBounds.Min.Y,
			),
			Face:             t.face(context, false),
			LineHeight:       t.LineHeight(),
			LineHeightMode:   t.baseStyle.lineHeightMode,
			TabWidth:         t.actualTabWidth(context),
			KeepTailingSpace: t.keepTailingSpace,
			FaceRuns:         renderingFaceRuns,
			Insertion:        insertion,
			WrapMode:         t.wrapMode,
			Composition:      compInfo,
		})
		if !ok {
			return materializeFull(), 0, 0, false
		}
		if hasComp {
			return t.stringValueForRenderingRange(r.StartInBytes, r.EndInBytes), r.StartInBytes, r.YShift, true
		}
		return t.stringValueWithRange(r.StartInBytes, r.EndInBytes), r.StartInBytes, r.YShift, true
	}

	// vAlign != Top: standalone Text. The alignment offset shifts the
	// document's drawn-Y from textBounds.Min.Y by alignOffset. Pass
	// that shift through to the caller as yShift; the walker itself
	// stays vAlign-agnostic and just walks from line 0 forward.
	totalHeight := t.textHeight(context, guigui.FixedWidthConstraints(width))
	var alignOffset int
	switch t.baseStyle.vAlign {
	case textutil.VerticalAlignMiddle:
		alignOffset = (textBounds.Dy() - totalHeight) / 2
	case textutil.VerticalAlignBottom:
		alignOffset = textBounds.Dy() - totalHeight
	}

	readRendering := t.stringValueWithRange
	if hasComp {
		readRendering = t.stringValueForRenderingRange
	}
	r, ok := textutil.VisibleRangeInViewport(&textutil.VisibleRangeInViewportParams{
		FirstLogicalLineInViewport: 0,
		LineByteOffsets:            &t.contentCache.lineByteOffsets,
		RenderingTextRange:         readRendering,
		RenderingTextLength:        renderingLength,
		ViewportSize: image.Pt(
			width,
			visibleBounds.Max.Y-textBounds.Min.Y-alignOffset,
		),
		Face:             t.face(context, false),
		LineHeight:       t.LineHeight(),
		LineHeightMode:   t.baseStyle.lineHeightMode,
		TabWidth:         t.actualTabWidth(context),
		KeepTailingSpace: t.keepTailingSpace,
		FaceRuns:         renderingFaceRuns,
		Insertion:        insertion,
		WrapMode:         t.wrapMode,
		Composition:      compInfo,
	})
	if !ok {
		return materializeFull(), 0, 0, false
	}
	if hasComp {
		return t.stringValueForRenderingRange(r.StartInBytes, r.EndInBytes), r.StartInBytes, alignOffset, true
	}
	return t.stringValueWithRange(r.StartInBytes, r.EndInBytes), r.StartInBytes, alignOffset, true
}

func (t *Text) selectionToDraw(context *guigui.Context) (start, end int, ok bool) {
	s, e := t.store.Selection()
	if !t.editable {
		return s, e, true
	}
	if !context.IsFocused(t) {
		return s, e, true
	}
	cs, ce, ok := t.store.CompositionSelection()
	if !ok {
		return s, e, true
	}
	// When cs == ce, the composition already started but any conversion is not done yet.
	// In this case, put the caret at the end of the composition.
	// TODO: This behavior might be macOS specific. Investigate this.
	if cs == ce {
		return s + ce, s + ce, true
	}
	return 0, 0, false
}

func (t *Text) compositionSelectionToDraw(context *guigui.Context) (uStart, cStart, cEnd, uEnd int, ok bool) {
	if !t.editable {
		return 0, 0, 0, 0, false
	}
	if !context.IsFocused(t) {
		return 0, 0, 0, 0, false
	}
	s, _ := t.store.Selection()
	cs, ce, ok := t.store.CompositionSelection()
	if !ok {
		return 0, 0, 0, 0, false
	}
	// When cs == ce, the composition already started but any conversion is not done yet.
	// In this case, assume the entire region is the composition.
	// TODO: This behavior might be macOS specific. Investigate this.
	l := t.store.UncommittedTextLengthInBytes()
	if cs == ce {
		return s, s, s + l, s + l, true
	}
	return s, s + cs, s + ce, s + l, true
}

func (t *Text) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	textBounds := t.contentBoundsForLayout(context, widgetBounds.Bounds())
	if !textBounds.Overlaps(widgetBounds.VisibleBounds()) {
		return
	}

	face := t.face(context, false)
	op := &t.drawOptions
	op.Style.WrapMode = t.wrapMode
	op.Style.Face = face
	committedFaceRuns, renderingFaceRuns, mark := t.acquireFaceRuns(context, false, true)
	defer func() {
		op.Style.FaceRuns = slices.Delete(op.Style.FaceRuns, 0, len(op.Style.FaceRuns))
		t.releaseFaceRuns(mark)
	}()
	op.Style.LineHeight = t.LineHeight()
	op.Style.LineHeightMode = t.baseStyle.lineHeightMode
	op.Style.HorizontalAlign = t.baseStyle.hAlign
	op.Style.VerticalAlign = t.baseStyle.vAlign
	op.Style.TabWidth = t.actualTabWidth(context)
	op.Style.KeepTailingSpace = t.keepTailingSpace
	insertion := t.insertion(context, false)
	op.Style.Insertion = insertion
	op.LayoutWidth = t.LayoutWidth(textBounds)
	if !t.editable {
		op.Style.EllipsisString = t.ellipsisString
	} else {
		op.Style.EllipsisString = ""
	}
	textColor, _ := t.baseStyle.style.Color()
	op.TextColor = textColor
	op.VisibleBounds = widgetBounds.VisibleBounds()
	if start, end, ok := t.selectionToDraw(context); ok {
		if context.IsFocused(t) || (t.selectionVisibleWhenUnfocus && start != end) {
			op.DrawSelection = true
			op.SelectionStart = start
			op.SelectionEnd = end
			op.SelectionColor = t.baseStyle.selectionColor
		} else {
			op.DrawSelection = false
		}
	}
	if uStart, cStart, cEnd, uEnd, ok := t.compositionSelectionToDraw(context); ok {
		op.DrawComposition = true
		op.CompositionStart = uStart
		op.CompositionEnd = uEnd
		op.CompositionActiveStart = cStart
		op.CompositionActiveEnd = cEnd
		op.InactiveCompositionColor = t.baseStyle.inactiveCompositionColor
		op.ActiveCompositionColor = t.baseStyle.activeCompositionColor
		op.CompositionBorderWidth = float32(textCaretWidth(context))
	} else {
		op.DrawComposition = false
	}

	op.StyleRuns = slices.Delete(op.StyleRuns, 0, len(op.StyleRuns))

	if t.masking() {
		// A masked value is single-line, uniform, and short, so it bypasses
		// the virtualized restriction path: draw the whole masked string and
		// remap the selection/composition offsets into masked space. Ranged
		// styles are not drawn on a masked value.
		m := t.maskMappingForRendering(true)
		op.Style.WrapMode = textutil.WrapModeNone
		op.Style.EllipsisString = ""
		if op.DrawSelection {
			op.SelectionStart = m.offsetToMasked(op.SelectionStart)
			op.SelectionEnd = m.offsetToMasked(op.SelectionEnd)
		}
		if op.DrawComposition {
			op.CompositionStart = m.offsetToMasked(op.CompositionStart)
			op.CompositionEnd = m.offsetToMasked(op.CompositionEnd)
			op.CompositionActiveStart = m.offsetToMasked(op.CompositionActiveStart)
			op.CompositionActiveEnd = m.offsetToMasked(op.CompositionActiveEnd)
		}
		textutil.Draw(textBounds, dst, m.maskStr, op)
		return
	}

	txt, byteStart, yShift, restricted := t.restrictedTextToDraw(context, textBounds, widgetBounds.VisibleBounds(), committedFaceRuns, renderingFaceRuns, insertion)
	if restricted {
		textBounds.Min.Y += yShift
		// yShift already includes the alignment-specific portion of the
		// textPositionYOffset the inner Draw would have computed; force
		// vAlign=Top so it only adds paddingY rather than re-centering /
		// re-bottom-aligning the restricted text inside the bounds.
		op.Style.VerticalAlign = textutil.VerticalAlignTop
		if op.DrawSelection {
			op.SelectionStart -= byteStart
			op.SelectionEnd -= byteStart
		}
		if op.DrawComposition {
			op.CompositionStart -= byteStart
			op.CompositionEnd -= byteStart
			op.CompositionActiveStart -= byteStart
			op.CompositionActiveEnd -= byteStart
		}
	}
	op.Style.Insertion = insertionToDraw(insertion, byteStart, len(txt))
	t.appendFaceRunsToDraw(op, renderingFaceRuns, byteStart, len(txt))
	t.appendStyleRunsToDraw(op, byteStart, len(txt))
	textutil.Draw(textBounds, dst, txt, op)
}

// appendFaceRunsToDraw appends the face runs that intersect the drawn byte
// window [byteStart, byteStart+txtLen) to op.Style.FaceRuns, rebased to the
// window. faceRuns uses the drawn text's byte offsets.
func (t *Text) appendFaceRunsToDraw(op *textutil.DrawOptions, faceRuns []textutil.FaceRun, byteStart, txtLen int) {
	for _, run := range faceRuns {
		start := run.Start - byteStart
		end := run.End - byteStart
		if end <= 0 || start >= txtLen {
			continue
		}
		op.Style.FaceRuns = append(op.Style.FaceRuns, textutil.FaceRun{
			Start: start,
			End:   end,
			Face:  run.Face,
		})
	}
}

// insertionToDraw returns insertion rebased to the drawn byte window
// [byteStart, byteStart+txtLen], or the zero value when it sits outside.
func insertionToDraw(insertion textutil.Insertion, byteStart, txtLen int) textutil.Insertion {
	insertion.IndexInBytes -= byteStart
	if insertion.IndexInBytes < 0 || insertion.IndexInBytes > txtLen {
		return textutil.Insertion{}
	}
	return insertion
}

// appendStyleRunsToDraw appends the ranged style overrides that intersect
// the drawn byte window [byteStart, byteStart+txtLen) to op.StyleRuns,
// rebased to the window. While an IME composition is active, the overrides
// are the rendering-text ones, so the composition draws with the style it
// will carry once committed.
func (t *Text) appendStyleRunsToDraw(op *textutil.DrawOptions, byteStart, txtLen int) {
	styleRuns := t.ensureOverrideStyleRuns()
	if t.store.UncommittedTextLengthInBytes() > 0 {
		styleRuns = t.renderingStyleRuns()
	}
	for run := range styleRuns.All() {
		start := run.Start - byteStart
		end := run.End - byteStart
		if end <= 0 || start >= txtLen {
			continue
		}
		styleRun := textutil.StyleRun{
			Start: start,
			End:   end,
		}
		if clr, ok := run.Style.Color(); ok {
			styleRun.Color = clr
		}
		if clr, ok := run.Style.BackgroundColor(); ok {
			styleRun.BackgroundColor = clr
		}
		if underline, ok := run.Style.Underline(); ok {
			styleRun.Underline = underline
		}
		if strikethrough, ok := run.Style.Strikethrough(); ok {
			styleRun.Strikethrough = strikethrough
		}
		op.StyleRuns = append(op.StyleRuns, styleRun)
	}
}

// DrawPlainString draws str in clr with the widget's current text style, laid
// out as the value would be, without selection or composition decorations.
func (t *Text) DrawPlainString(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image, str string, clr color.Color) {
	textBounds := t.contentBoundsForLayout(context, widgetBounds.Bounds())
	if !textBounds.Overlaps(widgetBounds.VisibleBounds()) {
		return
	}

	op := &t.drawOptions
	op.Style.WrapMode = t.wrapMode
	op.Style.Face = t.face(context, false)
	op.Style.LineHeight = t.LineHeight()
	op.Style.LineHeightMode = t.baseStyle.lineHeightMode
	op.Style.HorizontalAlign = t.baseStyle.hAlign
	op.Style.VerticalAlign = t.baseStyle.vAlign
	op.Style.TabWidth = t.actualTabWidth(context)
	op.Style.KeepTailingSpace = t.keepTailingSpace
	op.LayoutWidth = t.LayoutWidth(textBounds)
	if !t.editable {
		op.Style.EllipsisString = t.ellipsisString
	} else {
		op.Style.EllipsisString = ""
	}
	op.TextColor = clr
	op.VisibleBounds = widgetBounds.VisibleBounds()
	op.DrawSelection = false
	op.DrawComposition = false
	op.StyleRuns = slices.Delete(op.StyleRuns, 0, len(op.StyleRuns))
	op.Style.FaceRuns = nil
	op.Style.Insertion = textutil.Insertion{}
	textutil.Draw(textBounds, dst, str, op)
}
