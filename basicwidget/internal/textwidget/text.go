// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

// Package textwidget provides the theme-free core of the basicwidget text
// widgets: the editable [Text] widget and the input helpers it is built on.
package textwidget

import (
	"image"
	"image/color"
	"io"
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
	"github.com/guigui-gui/guigui/internal/clipboard"
)

var (
	textEventValueChanged            guigui.EventKey = guigui.GenerateEventKey()
	textEventValueChangedWithoutText guigui.EventKey = guigui.GenerateEventKey()
	textEventScrollDelta             guigui.EventKey = guigui.GenerateEventKey()
	textEventScrollIntoView          guigui.EventKey = guigui.GenerateEventKey()
)

// Text is a theme-free text widget: it owns the value, selection, caret, IME
// composition, input handling, layout, hit-testing, masking, and clipboard
// mechanism, and renders with concrete colors and face inputs set by a
// wrapping widget.
type Text struct {
	guigui.DefaultWidget

	store textStore

	// contentCache holds data derived from the store's content, rebuilt
	// lazily after an edit.
	contentCache textContentCache

	nextTextSet   bool
	nextText      string
	nextSelectAll bool
	textInited    bool

	baseStyle textStyle

	// styleRuns holds the ranged style overrides, with byte offsets into the
	// committed text. Cleared lazily by [Text.ensureStyleRuns] when
	// [textStore.Generation] advances past styleRunsValidGeneration.
	styleRuns                textstyle.Runs
	styleRunsValidGeneration int64

	// faceRunsBuf is the reusable buffer for [Text.appendFaceRunsForStyle]
	// results; each user clears it with a deferred slices.Delete. During
	// [Text.Draw], drawOptions.Style.FaceRuns borrows the same slice and is
	// reset to nil by the deferred cleanup, so the buffer has a single owner.
	faceRunsBuf []textutil.FaceRun

	// lastMetricStyleRunsFingerprint is the metric style overrides'
	// fingerprint at the last size measurement, so cached sizes reset when
	// a metric override changes.
	lastMetricStyleRunsFingerprint uint64

	selectable                  bool
	editable                    bool
	multiline                   bool
	wrapMode                    textutil.WrapMode
	caretStatic                 bool
	keepTailingSpace            bool
	selectionVisibleWhenUnfocus bool
	ellipsisString              string

	// wrapWidth keeps wrapping tied to a viewport even when the widget bounds
	// are widened to cover horizontally overflowing content.
	wrapWidth int

	// maskRune, when non-zero, is drawn in place of every grapheme cluster of
	// the value. The zero value disables masking.
	maskRune rune

	dragState textDragState

	// shiftSelectionSide is the selection endpoint moved by Shift and arrow keys.
	shiftSelectionSide SelectionSide

	caret textCaret

	// widgetBoundsRect is the widget's own bounds rectangle, captured by
	// [Text.Layout] for callers that resolve positions against it. Invalid
	// until [Text.Layout] has run.
	widgetBoundsRect image.Rectangle

	tmpClipboard string

	sizeCache textSizeCache

	// lastFaceAttributes and lastFontFamilyID together fingerprint the face used
	// to size text, so cached sizes reset when either the render attributes or
	// the active font family changes.
	lastFaceAttributes font.Attributes
	lastFontFamilyID   uint64

	lastScale float64

	drawOptions textutil.DrawOptions

	prevStart              int
	prevEnd                int
	paddingForScrollOffset guigui.Padding

	onFocusChanged      func(context *guigui.Context, focused bool)
	onHandleButtonInput func(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult

	// lastDispatchedUncommittedGen is the [textStore.Generation] value at the
	// most recent uncommitted dispatch, used to suppress redundant dispatches
	// (e.g. IME state replays that don't modify the text).
	lastDispatchedUncommittedGen int64

	// lastDispatchedCommittedGen is the [textStore.Generation] value at the
	// most recent committed dispatch. Committed dispatches are suppressed
	// until the store advances past it, filtering focus-loss commits on
	// unchanged buffers.
	lastDispatchedCommittedGen int64

	// firstLogicalLineInViewport is the logical-line index that sits at
	// widget-local Y=0. The zero value means line 0 at the top; virtualizing
	// parents set it via [Text.SetFirstLogicalLineInViewport] so drawing,
	// hit-testing, and caret positioning work relative to the viewport.
	firstLogicalLineInViewport int
}

// OnValueChanged sets the event handler that is called when the text value
// changes. The handler receives the current text and whether the change is
// committed. Dispatch and commit semantics are documented on the wrapping
// widget's OnValueChanged.
func (t *Text) OnValueChanged(f func(context *guigui.Context, text string, committed bool)) {
	guigui.SetEventHandler(t, textEventValueChanged, f)
}

// OnValueChangedWithoutText sets a handler that fires under the same
// conditions as [Text.OnValueChanged] but is not given the current text, so
// the value is not materialized into a string on every change.
func (t *Text) OnValueChangedWithoutText(f func(context *guigui.Context, committed bool)) {
	guigui.SetEventHandler(t, textEventValueChangedWithoutText, f)
}

// dispatchValueChanged dispatches a value-changed event, suppressing it when
// the store's generation hasn't moved past the relevant tracker. Uncommitted
// dispatches are gated on lastDispatchedUncommittedGen (so IME state replays
// at the same generation are filtered); committed dispatches are gated on
// lastDispatchedCommittedGen (so focus-loss commits on unchanged buffers are
// filtered, while still firing the commit that follows a real edit).
//
// force bypasses the committed gate, so an explicit commit gesture (pressing
// Enter) is dispatched even when the value equals the last committed value.
// force is meaningful only for committed dispatches.
func (t *Text) dispatchValueChanged(committed bool, force bool) {
	gen := t.store.Generation()
	if committed {
		if !force && gen == t.lastDispatchedCommittedGen {
			return
		}
		t.lastDispatchedCommittedGen = gen
	} else {
		if gen == t.lastDispatchedUncommittedGen {
			return
		}
		t.lastDispatchedUncommittedGen = gen
	}
	guigui.DispatchEventLazy(t, textEventValueChanged, func() (string, bool) {
		return t.stringValue(), committed
	})
	guigui.DispatchEvent(t, textEventValueChangedWithoutText, committed)
}

func (t *Text) OnHandleButtonInput(f func(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult) {
	t.onHandleButtonInput = f
}

// OnScrollDelta registers a handler invoked when input handling needs the
// containing scrollable area to scroll by a delta in pixels.
func (t *Text) OnScrollDelta(f func(context *guigui.Context, deltaX, deltaY float64)) {
	guigui.SetEventHandler(t, textEventScrollDelta, f)
}

// OnScrollIntoView registers a handler invoked when the selection needs to be
// brought into view. start and end are the selection endpoints (start <= end
// as byte indices); both are equal when the selection has zero width.
func (t *Text) OnScrollIntoView(f func(context *guigui.Context, start, end CaretScrollTarget)) {
	guigui.SetEventHandler(t, textEventScrollIntoView, f)
}

// contentHashForStateKey returns a 128-bit fingerprint of the current field
// contents, including the active IME composition (matching what [Text.Draw]
// and [Text.Measure] see).
func (t *Text) contentHashForStateKey() [16]byte {
	return t.contentCache.contentHash(&t.store)
}

// ensureLineByteOffsets refreshes the cached line byte offsets if the store
// has been mutated since the last call.
func (t *Text) ensureLineByteOffsets() {
	t.contentCache.ensureLineByteOffsets(&t.store)
}

func (t *Text) WriteStateKey(context *guigui.Context, w *guigui.StateKeyWriter) {
	w.WriteUint64(uint64(t.baseStyle.hAlign))
	w.WriteUint64(uint64(t.baseStyle.vAlign))
	writeColor(w, t.baseStyle.textColor)
	writeColor(w, t.baseStyle.selectionColor)
	writeColor(w, t.baseStyle.inactiveCompositionColor)
	writeColor(w, t.baseStyle.activeCompositionColor)
	writeColor(w, t.baseStyle.caretColor)
	w.WriteFloat64(t.baseStyle.scaleMinus1)
	w.WriteBool(t.baseStyle.italic)
	w.WriteInt(len(t.baseStyle.variations))
	for _, v := range t.baseStyle.variations {
		w.WriteUint32(uint32(v.Tag))
		w.WriteFloat64(float64(v.Value))
	}
	w.WriteInt(len(t.baseStyle.features))
	for _, f := range t.baseStyle.features {
		w.WriteUint32(uint32(f.Tag))
		w.WriteUint32(f.Value)
	}
	w.WriteFloat64(t.baseStyle.tabWidth)
	w.WriteFloat64(t.baseStyle.fontSize)
	w.WriteFloat64(t.baseStyle.lineHeight)
	w.WriteString(t.baseStyle.langString)
	w.WriteBool(t.selectable)
	w.WriteBool(t.editable)
	w.WriteBool(t.multiline)
	w.WriteUint64(uint64(t.wrapMode))
	w.WriteBool(t.caretStatic)
	w.WriteBool(t.keepTailingSpace)
	w.WriteBool(t.selectionVisibleWhenUnfocus)
	w.WriteString(t.ellipsisString)
	w.WriteInt(t.wrapWidth)
	w.WriteInt32(t.maskRune)
	writePadding(w, t.paddingForScrollOffset)
	selStart, selEnd := t.store.Selection()
	w.WriteInt(selStart)
	w.WriteInt(selEnd)
	w.WriteBool(t.store.IsFocused())
	w.WriteUint64(t.fontFamilyID())
	ch := t.contentHashForStateKey()
	_, _ = w.Write(ch[:])
	t.ensureStyleRuns().WriteStateKey(w)
}

func (t *Text) resetCachedTextSize() {
	t.sizeCache.reset()
}

// SetWrapWidth sets the width text wraps at, keeping wrapping tied to a
// viewport even when the widget bounds are widened to cover horizontally
// overflowing content. A non-positive width wraps at the widget bounds.
func (t *Text) SetWrapWidth(width int) {
	t.wrapWidth = width
}

// LayoutWidth returns the width used to lay out the text within bounds: the
// wrap width when wrapping is bounded, else the bounds width.
func (t *Text) LayoutWidth(bounds image.Rectangle) int {
	if t.wrapMode != textutil.WrapModeNone && t.wrapWidth > 0 {
		return t.wrapWidth
	}
	return bounds.Dx()
}

func (t *Text) canHaveCaret() bool {
	return t.selectable || t.editable
}

func (t *Text) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if t.canHaveCaret() {
		adder.AddWidget(&t.caret)
	}

	attrs := t.faceAttributes(false)
	fontFamilyID := t.fontFamilyID()
	if t.lastFaceAttributes != attrs || t.lastFontFamilyID != fontFamilyID {
		t.lastFaceAttributes = attrs
		t.lastFontFamilyID = fontFamilyID
		t.resetCachedTextSize()
	}
	if t.lastScale != context.Scale() {
		t.lastScale = context.Scale()
		t.resetCachedTextSize()
	}

	context.SetPassthrough(&t.caret, true)

	if t.selectable || t.editable {
		t.caret.text = t
	}

	if t.onFocusChanged == nil {
		t.onFocusChanged = func(context *guigui.Context, focused bool) {
			if !t.editable {
				return
			}
			if focused {
				t.store.Focus()
				t.caret.resetCounter()
				start, end := t.store.Selection()
				if start < 0 || end < 0 {
					t.doSelectAll()
				}
			} else {
				// End the IME session, committing any in-progress composition
				// so typed-but-uncommitted text is preserved rather than
				// discarded when focus moves away.
				t.store.Blur()
				t.commit(false)
			}
		}
	}
	guigui.OnFocusChanged(t, t.onFocusChanged)

	return nil
}

func (t *Text) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	t.widgetBoundsRect = widgetBounds.Bounds()
	if t.canHaveCaret() {
		layouter.LayoutWidget(&t.caret, t.caretBounds(context, t.widgetBoundsRect))
	}
}

func (t *Text) SetSelectable(selectable bool) {
	if t.selectable == selectable {
		return
	}
	t.selectable = selectable
	t.dragState.startPlus1 = 0
	t.dragState.endPlus1 = 0
	t.shiftSelectionSide = SelectionSideNone
	if !t.selectable {
		t.setSelection(0, 0, SelectionSideNone, false)
	}
}

func (t *Text) isEqualToStringValue(text string) bool {
	return t.contentCache.isEqualToText(&t.store, text)
}

// stringValue returns the store's committed text. The remaining callers
// — value-changed event dispatch, [Text.Value], and the rare fallback
// path of [Text.textToDraw] — fire infrequently enough that the per-
// tick cache the function used to maintain is no longer worth its
// fields; per-tick consumers all read narrower ranges via
// [Text.stringValueWithRange].
func (t *Text) stringValue() string {
	return t.contentCache.text(&t.store)
}

func (t *Text) stringValueWithRange(start, end int) string {
	return t.contentCache.stringWithRange(&t.store, start, end, false)
}

func (t *Text) bytesValueWithRange(start, end int) []byte {
	return t.contentCache.bytesWithRange(&t.store, start, end)
}

// stringValueForRenderingRange returns the bytes of the rendering text
// (committed text with the active IME composition spliced in) in
// [start, end). Coordinates are in rendering space; clamped by
// [textStore.WriteTextForRenderingRangeTo].
func (t *Text) stringValueForRenderingRange(start, end int) string {
	return t.contentCache.stringWithRange(&t.store, start, end, true)
}

// stringValueForLineContaining returns the bytes of the logical line that
// contains byteOffset (clamped to the document) along with the line's
// starting byte offset, suitable for translating local↔global byte
// positions. It is used by per-caret textutil calls (word-boundary
// lookup, grapheme stepping) so they can scan the relevant logical line
// without materializing the whole document.
func (t *Text) stringValueForLineContaining(byteOffset int) (line string, lineStart int) {
	t.ensureLineByteOffsets()
	lineIdx := t.contentCache.lineByteOffsets.LineIndexForByteOffset(byteOffset)
	lineStart = t.contentCache.lineByteOffsets.ByteOffsetByLineIndex(lineIdx)
	lineEnd := t.store.TextLengthInBytes()
	if lineIdx+1 < t.contentCache.lineByteOffsets.LineCount() {
		lineEnd = t.contentCache.lineByteOffsets.ByteOffsetByLineIndex(lineIdx + 1)
	}
	return t.stringValueWithRange(lineStart, lineEnd), lineStart
}

func (t *Text) stringValueForRendering() string {
	return t.contentCache.textForRendering(&t.store)
}

// Value returns the current value as a string.
func (t *Text) Value() string {
	if t.nextTextSet {
		return t.nextText
	}
	return t.stringValue()
}

// HasValue reports whether the text has a non-empty value.
func (t *Text) HasValue() bool {
	if t.nextTextSet {
		return t.nextText != ""
	}
	return t.hasValueInField()
}

func (t *Text) hasValueInField() bool {
	return t.store.HasText()
}

func (t *Text) SetValue(text string) {
	if t.nextTextSet && t.nextText == text {
		return
	}
	if !t.nextTextSet && t.isEqualToStringValue(text) {
		return
	}
	if !t.editable {
		t.setText(text, false)
		return
	}

	// Do not call t.setText here. Update the actual value later.
	// For example, when a user is editing, the text should not be changed.
	// Another case is that SetMultiline might be called later.
	t.nextText = text
	t.nextTextSet = true
	t.resetCachedTextSize()
}

func (t *Text) ForceSetValue(text string) {
	t.setText(text, false)
}

// WriteValueTo writes the current value to w and returns the number of bytes
// written.
func (t *Text) WriteValueTo(w io.Writer) (int64, error) {
	if t.nextTextSet {
		n, err := io.WriteString(w, t.nextText)
		return int64(n), err
	}
	return t.store.WriteTextTo(w)
}

// WriteValueRangeTo writes the bytes of the current value in
// [startInBytes, endInBytes), clamped to the value, to w.
func (t *Text) WriteValueRangeTo(w io.Writer, startInBytes, endInBytes int) (int64, error) {
	if t.nextTextSet {
		l := len(t.nextText)
		startInBytes = min(max(startInBytes, 0), l)
		endInBytes = min(max(endInBytes, 0), l)
		if startInBytes >= endInBytes {
			return 0, nil
		}
		n, err := io.WriteString(w, t.nextText[startInBytes:endInBytes])
		return int64(n), err
	}
	return t.store.WriteTextRangeTo(w, startInBytes, endInBytes)
}

// ReadValueFrom resets the value to the bytes read from r until EOF and
// returns the number of bytes read. The undo history is cleared and the
// selection is reset to (0, 0). On a non-EOF error, the value is reset to
// empty and the error is returned.
func (t *Text) ReadValueFrom(r io.Reader) (int64, error) {
	n, err := t.store.ReadTextFrom(r)
	t.shiftSelectionSide = SelectionSideNone
	t.prevStart = 0
	t.prevEnd = 0
	t.nextText = ""
	t.nextTextSet = false
	t.textInited = true
	t.resetCachedTextSize()
	t.dispatchValueChanged(true, false)
	return n, err
}

func (t *Text) ReplaceValueAtSelection(text string) {
	if text == "" {
		return
	}
	t.replaceTextAtSelection(text)
	t.resetCachedTextSize()
}

func (t *Text) CommitWithCurrentInputValue() {
	t.nextText = ""
	t.nextTextSet = false
	t.dispatchValueChanged(true, false)
}

// SelectAll selects the entire value.
func (t *Text) SelectAll() {
	if t.nextTextSet {
		t.nextSelectAll = true
		return
	}
	t.doSelectAll()
}

func (t *Text) doSelectAll() {
	t.setSelection(0, t.store.TextLengthInBytes(), SelectionSideNone, false)
}

func (t *Text) Selection() (start, end int) {
	return t.store.Selection()
}

func (t *Text) SetSelection(start, end int) {
	t.setSelection(start, end, SelectionSideNone, true)
}

// setSelection sets the selection to the range spanned by start and end and
// records the endpoint moved by Shift and arrow keys. shiftSide names that
// endpoint among the start and end arguments, before they are reordered.
func (t *Text) setSelection(start, end int, shiftSide SelectionSide, adjustScroll bool) bool {
	if start > end {
		start, end = end, start
		switch shiftSide {
		case SelectionSideStart:
			shiftSide = SelectionSideEnd
		case SelectionSideEnd:
			shiftSide = SelectionSideStart
		}
	}
	t.shiftSelectionSide = shiftSide

	if s, e := t.store.Selection(); s == start && e == end {
		return false
	}
	t.store.SetSelection(start, end)

	if !adjustScroll {
		t.prevStart = start
		t.prevEnd = end
	}

	return true
}

func (t *Text) replaceTextAtSelection(text string) {
	start, end := t.store.Selection()
	t.replaceTextAt(text, start, end)
}

func (t *Text) replaceTextAt(text string, start, end int) {
	if !t.IsMultiline() {
		text, start, end = replaceNewLinesWithSpace(text, start, end)
	}

	t.shiftSelectionSide = SelectionSideNone
	if start > end {
		start, end = end, start
	}
	if s, e := t.store.Selection(); text == t.stringValueWithRange(start, end) && s == start && e == end {
		return
	}
	t.store.ReplaceText(text, start, end)
	t.contentCache.applyReplaceToLineByteOffsets(&t.store, text, start, end)

	t.resetCachedTextSize()
	t.dispatchValueChanged(false, false)

	t.nextText = ""
	t.nextTextSet = false
}

func (t *Text) setText(text string, selectAll bool) bool {
	if !t.IsMultiline() {
		text, _, _ = replaceNewLinesWithSpace(text, 0, 0)
	}

	t.shiftSelectionSide = SelectionSideNone

	textChanged := !t.isEqualToStringValue(text)
	if s, e := t.store.Selection(); !textChanged && (!selectAll || s == 0 && e == len(text)) {
		return false
	}

	var start, end int
	if selectAll {
		end = len(text)
	}
	// When selectAll is false, the current selection range might be no longer valid.
	// Reset the selection to (0, 0).

	if textChanged {
		if t.textInited || t.hasValueInField() {
			t.store.SetTextAndSelection(text, start, end)
		} else {
			// Reset the text so that the undo history's first item is the initial text.
			t.store.ResetText(text)
			t.store.SetSelection(start, end)
		}
		t.resetCachedTextSize()
		t.dispatchValueChanged(true, false)
	} else {
		t.store.SetSelection(0, len(text))
	}

	// Do not adjust scroll.
	t.prevStart = start
	t.prevEnd = end
	t.nextText = ""
	t.nextTextSet = false
	t.textInited = true

	return true
}

// SetItalic sets the italic face selection of the base style. Ranged italic
// overrides apply on top.
func (t *Text) SetItalic(italic bool) {
	t.baseStyle.italic = italic
}

// SetVariation sets the OpenType variation axis tag of the base style to
// value. Ranged variation overrides apply on top.
func (t *Text) SetVariation(tag text.Tag, value float32) {
	t.baseStyle.variations = setTagged(t.baseStyle.variations, font.Variation{Tag: tag, Value: value}, func(v font.Variation) text.Tag {
		return v.Tag
	})
}

// UnsetVariation removes the OpenType variation axis tag from the base
// style.
func (t *Text) UnsetVariation(tag text.Tag) {
	t.baseStyle.variations = removeTagged(t.baseStyle.variations, tag, func(v font.Variation) text.Tag {
		return v.Tag
	})
}

// SetFeature sets the OpenType feature tag of the base style to value.
// Ranged feature overrides apply on top.
func (t *Text) SetFeature(tag text.Tag, value uint32) {
	t.baseStyle.features = setTagged(t.baseStyle.features, font.Feature{Tag: tag, Value: value}, func(f font.Feature) text.Tag {
		return f.Tag
	})
}

// UnsetFeature removes the OpenType feature tag from the base style.
func (t *Text) UnsetFeature(tag text.Tag) {
	t.baseStyle.features = removeTagged(t.baseStyle.features, tag, func(f font.Feature) text.Tag {
		return f.Tag
	})
}

func (t *Text) SetTabWidth(tabWidth float64) {
	if t.baseStyle.tabWidth == tabWidth {
		return
	}
	t.baseStyle.tabWidth = tabWidth
	t.resetCachedTextSize()
}

func (t *Text) actualTabWidth(context *guigui.Context) float64 {
	if t.baseStyle.tabWidth > 0 {
		return t.baseStyle.tabWidth
	}
	if t.sizeCache.defaultTabWidth > 0 {
		return t.sizeCache.defaultTabWidth
	}
	face := t.face(context, false)
	const defaultTabSpaces = "        "
	t.sizeCache.defaultTabWidth = text.AdvanceAt(defaultTabSpaces, len(defaultTabSpaces), face.TextFace())
	return t.sizeCache.defaultTabWidth
}

// Scale returns the base text scale.
func (t *Text) Scale() float64 {
	return t.baseStyle.scale()
}

// SetScale sets the base text scale, which ranged scale overrides
// multiply.
func (t *Text) SetScale(scale float64) {
	t.baseStyle.scaleMinus1 = scale - 1
}

func (t *Text) HorizontalAlign() textutil.HorizontalAlign {
	return t.baseStyle.hAlign
}

func (t *Text) SetHorizontalAlign(align textutil.HorizontalAlign) {
	t.baseStyle.hAlign = align
}

func (t *Text) VerticalAlign() textutil.VerticalAlign {
	return t.baseStyle.vAlign
}

func (t *Text) SetVerticalAlign(align textutil.VerticalAlign) {
	t.baseStyle.vAlign = align
}

func (t *Text) IsEditable() bool {
	return t.editable
}

func (t *Text) SetEditable(editable bool) {
	if t.editable == editable {
		return
	}

	if editable {
		t.dragState.startPlus1 = 0
		t.dragState.endPlus1 = 0
		t.shiftSelectionSide = SelectionSideNone
	} else if t.store.IsFocused() {
		// Blur immediately so Ebitengine's BeforeUpdate hook stops auto-committing
		// pending input into the store before HandlePointingInput runs next tick.
		t.store.Blur()
	}
	t.editable = editable
}

// IsMultiline reports whether the value may span multiple lines. It is always
// false while masking, which is single-line.
func (t *Text) IsMultiline() bool {
	return t.multiline && !t.masking()
}

func (t *Text) SetMultiline(multiline bool) {
	t.multiline = multiline
}

// WrapMode reports how visual lines wrap when text exceeds the available
// width. The default is [textutil.WrapModeNone].
func (t *Text) WrapMode() textutil.WrapMode {
	return t.wrapMode
}

// SetWrapMode selects how visual lines wrap when text exceeds the available
// width. See [textutil.WrapMode] for the available modes.
func (t *Text) SetWrapMode(wrapMode textutil.WrapMode) {
	t.wrapMode = wrapMode
}

// SetCaretBlinking sets whether the caret blinks.
// The default value is true.
func (t *Text) SetCaretBlinking(caretBlinking bool) {
	t.caretStatic = !caretBlinking
}

// SetSelectionVisibleWhenUnfocused sets whether the selection range stays
// drawn while the widget is not focused. The default is false.
func (t *Text) SetSelectionVisibleWhenUnfocused(visible bool) {
	t.selectionVisibleWhenUnfocus = visible
}

func (t *Text) SetEllipsisString(str string) {
	if t.ellipsisString == str {
		return
	}

	t.ellipsisString = str
	t.resetCachedTextSize()
}

// SetMaskRune sets the character drawn in place of each grapheme cluster of the
// value. A non-zero rune masks the text and forces it to a single line; the
// zero value renders it normally.
func (t *Text) SetMaskRune(maskRune rune) {
	if t.maskRune == maskRune {
		return
	}
	t.maskRune = maskRune
	t.resetCachedTextSize()
}

// SetKeepTailingSpace sets whether spaces at the end of a visual line keep
// their advance instead of collapsing.
func (t *Text) SetKeepTailingSpace(keep bool) {
	t.keepTailingSpace = keep
}

// SetFirstLogicalLineInViewport sets the logical line that sits at
// widget-local Y=0. The default 0 means line 0 at the top; virtualizing
// parents set the topmost visible logical line so drawing, hit testing, and
// caret positioning need not walk the document prefix.
func (t *Text) SetFirstLogicalLineInViewport(idx int) {
	t.firstLogicalLineInViewport = max(idx, 0)
}

func (t *Text) fontFamilyID() uint64 {
	return t.baseStyle.fontFamilyID()
}

func (t *Text) faceAttributes(forceBold bool) font.Attributes {
	// Disable ligatures for editable, selectable, or masked text so caret
	// positions land on byte boundaries.
	liga := !t.selectable && !t.editable && !t.masking()
	return t.baseStyle.faceAttributes(forceBold, liga)
}

// face must be called after [Text.Build], as it relies on lastFaceAttributes being set.
func (t *Text) face(context *guigui.Context, forceBold bool) font.Face {
	attrs := t.lastFaceAttributes
	if forceBold {
		attrs = attrs.WithVariation(font.TagWght, float32(text.WeightBold))
	}
	return font.NewFace(context, t.baseStyle.fontFamily, attrs)
}

// LineHeight returns the line height in pixels, with the widget scale applied.
func (t *Text) LineHeight() float64 {
	return t.baseStyle.scaledLineHeight()
}

func (t *Text) commit(force bool) {
	t.dispatchValueChanged(true, force)
	t.nextText = ""
	t.nextTextSet = false
}

func (t *Text) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	// Fast path: skip Tick entirely for non-selectable, non-editable text
	// that is already initialized and has no pending text update.
	if !t.selectable && !t.editable && t.textInited && !t.nextTextSet {
		return nil
	}

	// Once a text is input, it is regarded as initialized.
	if !t.textInited && t.hasValueInField() {
		t.textInited = true
	}
	if (!t.editable || !context.IsFocused(t)) && t.nextTextSet {
		t.setText(t.nextText, t.nextSelectAll)
		t.nextSelectAll = false
	}

	// Pump the IME composer every tick while focused so a composition the OS
	// reports without a key event is drained and rendered. HandleButtonInput
	// only runs on ticks with key activity, which an IME owning the keyboard
	// suppresses.
	if t.editable && t.store.IsFocused() {
		t.updateIMEComposer(context, widgetBounds)
	}

	// Adjust the scroll offset for cases not covered by HandleButtonInput,
	// such as continuous scrolling during drag selection.
	// TODO: The caret position might be unstable when the text horizontal align is center or right. Fix this.
	if t.selectable || t.editable {
		if dx, dy := t.adjustScrollOffset(context, widgetBounds); dx != 0 || dy != 0 {
			guigui.DispatchEvent(t, textEventScrollDelta, dx, dy)
		}
	}

	return nil
}

// SetPaddingForScrollOffset sets the padding kept between the caret and the
// viewport edges when scrolling the selection into view.
func (t *Text) SetPaddingForScrollOffset(padding guigui.Padding) {
	t.paddingForScrollOffset = padding
}

func (t *Text) CanCut() bool {
	if !t.editable {
		return false
	}
	if t.masking() {
		return false
	}
	start, end := t.store.Selection()
	return start != end
}

func (t *Text) CanCopy() bool {
	if t.masking() {
		return false
	}
	start, end := t.store.Selection()
	return start != end
}

func (t *Text) CanPaste() bool {
	if !t.editable {
		return false
	}
	ct, err := clipboard.ReadAll()
	if err != nil {
		slog.Error(err.Error())
		return false
	}
	return len(ct) > 0
}

func (t *Text) CanUndo() bool {
	if !t.editable {
		return false
	}
	return t.store.CanUndo()
}

func (t *Text) CanRedo() bool {
	if !t.editable {
		return false
	}
	return t.store.CanRedo()
}

func (t *Text) Cut() bool {
	if t.masking() {
		return false
	}
	start, end := t.store.Selection()
	if start == end {
		return false
	}
	if err := clipboard.WriteAll(t.bytesValueWithRange(start, end)); err != nil {
		slog.Error(err.Error())
		return false
	}
	t.replaceTextAtSelection("")
	return true
}

func (t *Text) Copy() bool {
	if t.masking() {
		return false
	}
	start, end := t.store.Selection()
	if start == end {
		return false
	}
	if err := clipboard.WriteAll(t.bytesValueWithRange(start, end)); err != nil {
		slog.Error(err.Error())
		return false
	}
	return true
}

func (t *Text) Paste() bool {
	ct, err := clipboard.ReadAll()
	if err != nil {
		slog.Error(err.Error())
		return false
	}
	t.replaceTextAtSelection(string(ct))
	return true
}

func (t *Text) Undo() bool {
	if !t.store.CanUndo() {
		return false
	}
	t.store.Undo()
	t.resetCachedTextSize()
	return true
}

func (t *Text) Redo() bool {
	if !t.store.CanRedo() {
		return false
	}
	t.store.Redo()
	t.resetCachedTextSize()
	return true
}

// SetFontFamily sets the resolved font family used to render the value. A nil
// family renders with the registered face source stack alone.
func (t *Text) SetFontFamily(fontFamily *font.Family) {
	t.baseStyle.fontFamily = fontFamily
}

// SetFontSize sets the font size at scale 1. The rendered size is the
// base size multiplied by the scale set via [Text.SetScale].
func (t *Text) SetFontSize(size float64) {
	t.baseStyle.fontSize = size
}

// SetLineHeight sets the line height at scale 1. The rendered line height
// is the base line height multiplied by the scale set via
// [Text.SetScale].
func (t *Text) SetLineHeight(lineHeight float64) {
	if t.baseStyle.lineHeight == lineHeight {
		return
	}
	t.baseStyle.lineHeight = lineHeight
	t.resetCachedTextSize()
}

// SetLang sets the language used to select the face and its features when
// shaping the value.
func (t *Text) SetLang(lang language.Tag) {
	if t.baseStyle.lang == lang {
		return
	}
	t.baseStyle.lang = lang
	t.baseStyle.langString = lang.String()
}

// SetTextColor sets the concrete color the value is drawn in.
func (t *Text) SetTextColor(clr color.Color) {
	t.baseStyle.textColor = clr
}

// SetSelectionColor sets the concrete color of the selection highlight.
func (t *Text) SetSelectionColor(clr color.Color) {
	t.baseStyle.selectionColor = clr
}

// SetCompositionColors sets the concrete colors of the underlines drawn below
// the inactive and active parts of an IME composition.
func (t *Text) SetCompositionColors(inactive, active color.Color) {
	t.baseStyle.inactiveCompositionColor = inactive
	t.baseStyle.activeCompositionColor = active
}

// SetCaretColor sets the concrete color of the caret.
func (t *Text) SetCaretColor(clr color.Color) {
	t.baseStyle.caretColor = clr
}

// SetSelectionWithSide sets the selection to the range spanned by start and
// end and records shiftSide as the endpoint moved by Shift and arrow keys.
func (t *Text) SetSelectionWithSide(start, end int, shiftSide SelectionSide, adjustScroll bool) {
	t.setSelection(start, end, shiftSide, adjustScroll)
}

// Generation returns the store's content generation. The generation advances
// on every content mutation, so an unchanged generation means unchanged text.
func (t *Text) Generation() int64 {
	return t.store.Generation()
}

// ShiftSelectionSide returns the selection endpoint moved by Shift and arrow
// keys.
func (t *Text) ShiftSelectionSide() SelectionSide {
	return t.shiftSelectionSide
}

// IsVisuallyEmpty reports whether nothing of the value would be rendered: the
// committed text is empty and no IME composition is in progress.
func (t *Text) IsVisuallyEmpty() bool {
	return !t.store.HasText() && t.store.UncommittedTextLengthInBytes() == 0
}
