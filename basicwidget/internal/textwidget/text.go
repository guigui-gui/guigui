// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

// Package textwidget provides the theme-free core of the basicwidget text
// widgets: the editable [Text] widget and the input helpers it is built on.
package textwidget

import (
	"bytes"
	"hash"
	"hash/fnv"
	"image"
	"image/color"
	"io"
	"log/slog"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
	"github.com/guigui-gui/guigui/basicwidget/internal/font"
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

	store             textStore
	valueBuilder      bytes.Buffer
	valueEqualChecker stringEqualChecker

	// stringCache memoizes field substrings keyed by content generation and
	// range with round-robin replacement, reused until the next edit bumps the
	// generation.
	stringCache     [4]stringCacheEntry
	stringCacheNext int

	nextTextSet   bool
	nextText      string
	nextSelectAll bool
	textInited    bool

	hAlign      textutil.HorizontalAlign
	vAlign      textutil.VerticalAlign
	scaleMinus1 float64
	bold        bool
	tabular     bool
	tabWidth    float64

	// fontFamily is the resolved font family used to render the value, or nil
	// to render with the registered face source stack alone.
	fontFamily *font.Family

	// baseFontSize is the font size at scale 1; the rendered size is
	// baseFontSize multiplied by the widget scale.
	baseFontSize float64

	// baseLineHeight is the line height at scale 1; the rendered line height
	// is baseLineHeight multiplied by the widget scale.
	baseLineHeight float64

	// lang is the language used to select the face and its features when
	// shaping the value.
	lang language.Tag

	// langString is lang's string form, cached for [Text.WriteStateKey].
	langString string

	// textColor is the concrete color the value is drawn in.
	textColor color.Color

	// selectionColor is the concrete color of the selection highlight.
	selectionColor color.Color

	// inactiveCompositionColor is the concrete color of the inactive part of
	// an IME composition's underline.
	inactiveCompositionColor color.Color

	// activeCompositionColor is the concrete color of the active part of an
	// IME composition's underline.
	activeCompositionColor color.Color

	// caretColor is the concrete color of the caret.
	caretColor color.Color

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

	selectionDragStartPlus1 int
	selectionDragEndPlus1   int

	// shiftSelectionSide is the selection endpoint moved by Shift and arrow keys.
	shiftSelectionSide SelectionSide

	dragging bool

	clickCounter clickCounter

	caret textCaret

	// widgetBoundsRect is the widget's own bounds rectangle, captured by
	// [Text.Layout] for callers that resolve positions against it. Invalid
	// until [Text.Layout] has run.
	widgetBoundsRect image.Rectangle

	tmpClipboard string

	cachedTextWidths      [8][4]cachedTextWidthEntry
	cachedTextHeights     [8][4]cachedTextHeightEntry
	cachedDefaultTabWidth float64

	// lastFaceAttributes and lastFontFamilyID together fingerprint the face used
	// to size text, so cached sizes reset when either the render attributes or
	// the active font family changes.
	lastFaceAttributes font.Attributes
	lastFontFamilyID   uint64

	lastScale float64

	// contentHasher is a reusable streaming hasher used by [Text.WriteStateKey]
	// to fingerprint the current field contents.
	contentHasher hash.Hash

	// contentHashCache memoizes the most recently computed hash, keyed by
	// [textStore.Generation]. While the store has not been mutated, repeated
	// [Text.WriteStateKey] calls return the cached value without re-hashing.
	contentHashCache           [16]byte
	contentHashFieldGeneration int64

	// lineByteOffsets holds the byte offset of each logical line start in the
	// committed text, refreshed lazily by ensureLineByteOffsets when
	// [textStore.Generation] advances past lineByteOffsetsFieldGeneration.
	lineByteOffsets                textutil.LineByteOffsets
	lineByteOffsetsFieldGeneration int64

	// cachedMaskMapping memoizes the masked rendering of the value, keyed by
	// [textStore.Generation], the mask rune, and whether the active IME
	// composition is included. A zero runeLen means it has not been built yet.
	cachedMaskMapping                maskMapping
	cachedMaskMappingGeneration      int64
	cachedMaskMappingRune            rune
	cachedMaskMappingWithComposition bool

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

type cachedTextWidthEntry struct {
	// 0 indicates that the entry is invalid.
	constraintWidth int

	width int
}

type cachedTextHeightEntry struct {
	// 0 indicates that the entry is invalid.
	constraintWidth int

	height int
}

type stringCacheEntry struct {
	valid        bool
	forRendering bool
	generation   int64
	start        int
	end          int
	str          string
}

type textSizeCacheKey int

func newTextSizeCacheKey(wrapMode textutil.WrapMode, bold bool) textSizeCacheKey {
	key := textSizeCacheKey(wrapMode) & 0x3
	if bold {
		key |= 1 << 2
	}
	return key
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
	generation := t.store.Generation()
	if generation == t.contentHashFieldGeneration {
		return t.contentHashCache
	}
	if t.contentHasher == nil {
		t.contentHasher = fnv.New128a()
	}
	t.contentHasher.Reset()
	_, _ = t.store.WriteTextForRenderingTo(t.contentHasher)
	var ch [16]byte
	t.contentHasher.Sum(ch[:0])
	t.contentHashCache = ch
	t.contentHashFieldGeneration = generation
	return t.contentHashCache
}

// ensureLineByteOffsets refreshes t.lineByteOffsets if the store has been
// mutated since the last call. The offsets are built from the committed text
// only (no IME composition), matching what [textStore.WriteTextTo]
// returns.
func (t *Text) ensureLineByteOffsets() {
	generation := t.store.Generation()
	if t.lineByteOffsets.LineCount() > 0 && generation == t.lineByteOffsetsFieldGeneration {
		return
	}
	_ = t.lineByteOffsets.Rebuild(func(w io.Writer) error {
		_, err := t.store.WriteTextTo(w)
		return err
	})
	t.lineByteOffsetsFieldGeneration = generation
}

func (t *Text) WriteStateKey(context *guigui.Context, w *guigui.StateKeyWriter) {
	w.WriteUint64(uint64(t.hAlign))
	w.WriteUint64(uint64(t.vAlign))
	writeColor(w, t.textColor)
	writeColor(w, t.selectionColor)
	writeColor(w, t.inactiveCompositionColor)
	writeColor(w, t.activeCompositionColor)
	writeColor(w, t.caretColor)
	w.WriteFloat64(t.scaleMinus1)
	w.WriteBool(t.bold)
	w.WriteBool(t.tabular)
	w.WriteFloat64(t.tabWidth)
	w.WriteFloat64(t.baseFontSize)
	w.WriteFloat64(t.baseLineHeight)
	w.WriteString(t.langString)
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
}

func (t *Text) resetCachedTextSize() {
	clear(t.cachedTextWidths[:])
	clear(t.cachedTextHeights[:])
	t.cachedDefaultTabWidth = 0
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
	t.selectionDragStartPlus1 = 0
	t.selectionDragEndPlus1 = 0
	t.shiftSelectionSide = SelectionSideNone
	if !t.selectable {
		t.setSelection(0, 0, SelectionSideNone, false)
	}
}

func (t *Text) isEqualToStringValue(text string) bool {
	t.valueEqualChecker.Reset(text)
	_, _ = t.store.WriteTextTo(&t.valueEqualChecker)
	return t.valueEqualChecker.Result()
}

// stringValue returns the store's committed text. The remaining callers
// — value-changed event dispatch, [Text.Value], and the rare fallback
// path of [Text.textToDraw] — fire infrequently enough that the per-
// tick cache the function used to maintain is no longer worth its
// fields; per-tick consumers all read narrower ranges via
// [Text.stringValueWithRange].
func (t *Text) stringValue() string {
	t.valueBuilder.Reset()
	_, _ = t.store.WriteTextTo(&t.valueBuilder)
	return t.valueBuilder.String()
}

func (t *Text) stringValueWithRange(start, end int) string {
	return t.cachedStringWithRange(start, end, false)
}

// cachedStringWithRange returns the store substring [start, end) — rendering text
// when forRendering, else committed — reusing a cached copy until the next edit.
// A negative end means the text length.
func (t *Text) cachedStringWithRange(start, end int, forRendering bool) string {
	if end < 0 {
		end = t.store.TextLengthInBytes()
	}
	gen := t.store.Generation()
	for i := range t.stringCache {
		if e := &t.stringCache[i]; e.valid && e.generation == gen && e.forRendering == forRendering && e.start == start && e.end == end {
			return e.str
		}
	}
	t.valueBuilder.Reset()
	if forRendering {
		_, _ = t.store.WriteTextForRenderingRangeTo(&t.valueBuilder, start, end)
	} else {
		_, _ = t.store.WriteTextRangeTo(&t.valueBuilder, start, end)
	}
	str := t.valueBuilder.String()
	t.stringCache[t.stringCacheNext] = stringCacheEntry{
		valid:        true,
		forRendering: forRendering,
		generation:   gen,
		start:        start,
		end:          end,
		str:          str,
	}
	t.stringCacheNext = (t.stringCacheNext + 1) % len(t.stringCache)
	return str
}

func (t *Text) bytesValueWithRange(start, end int) []byte {
	if end < 0 {
		end = t.store.TextLengthInBytes()
	}
	t.valueBuilder.Reset()
	_, _ = t.store.WriteTextRangeTo(&t.valueBuilder, start, end)
	return t.valueBuilder.Bytes()
}

// stringValueForRenderingRange returns the bytes of the rendering text
// (committed text with the active IME composition spliced in) in
// [start, end). Coordinates are in rendering space; clamped by
// [textStore.WriteTextForRenderingRangeTo].
func (t *Text) stringValueForRenderingRange(start, end int) string {
	return t.cachedStringWithRange(start, end, true)
}

// stringValueForLineContaining returns the bytes of the logical line that
// contains byteOffset (clamped to the document) along with the line's
// starting byte offset, suitable for translating local↔global byte
// positions. It is used by per-caret textutil calls (word-boundary
// lookup, grapheme stepping) so they can scan the relevant logical line
// without materializing the whole document.
func (t *Text) stringValueForLineContaining(byteOffset int) (line string, lineStart int) {
	t.ensureLineByteOffsets()
	lineIdx := t.lineByteOffsets.LineIndexForByteOffset(byteOffset)
	lineStart = t.lineByteOffsets.ByteOffsetByLineIndex(lineIdx)
	lineEnd := t.store.TextLengthInBytes()
	if lineIdx+1 < t.lineByteOffsets.LineCount() {
		lineEnd = t.lineByteOffsets.ByteOffsetByLineIndex(lineIdx + 1)
	}
	return t.stringValueWithRange(lineStart, lineEnd), lineStart
}

// LineCount returns the number of logical lines (spans between hard line
// breaks) in the value. The empty value has one logical line; a trailing
// line break creates an extra empty line at the end.
func (t *Text) LineCount() int {
	t.ensureLineByteOffsets()
	return t.lineByteOffsets.LineCount()
}

// LineStartInBytes returns the byte offset where the lineIndex-th logical
// line begins within the value. lineIndex must be in [0, [Text.LineCount]).
func (t *Text) LineStartInBytes(lineIndex int) int {
	t.ensureLineByteOffsets()
	return t.lineByteOffsets.ByteOffsetByLineIndex(lineIndex)
}

// LineIndexFromTextIndexInBytes returns the index of the logical line
// containing textIndexInBytes, clamping out-of-range values to the first or
// last line.
func (t *Text) LineIndexFromTextIndexInBytes(textIndexInBytes int) int {
	t.ensureLineByteOffsets()
	return t.lineByteOffsets.LineIndexForByteOffset(textIndexInBytes)
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

// findWordBoundaries returns the byte range of the word containing idx,
// scanning only the logical line containing idx. Word-segmentation rules
// always break at line breaks (UAX #29 WB3a/3b), so a word never spans
// logical lines.
func (t *Text) findWordBoundaries(idx int) (start, end int) {
	line, lineStart := t.stringValueForLineContaining(idx)
	s, e := textutil.FindWordBoundaries(line, idx-lineStart)
	return s + lineStart, e + lineStart
}

// prevPositionOnGraphemes returns the byte offset of the grapheme cluster
// boundary that immediately precedes position. Grapheme breaks always
// exist around line-break characters (UAX #29 GB4/GB5), so the previous
// boundary is always inside the logical line containing position-1.
func (t *Text) prevPositionOnGraphemes(position int) int {
	if position <= 0 {
		return position
	}
	line, lineStart := t.stringValueForLineContaining(position - 1)
	return lineStart + textutil.PrevPositionOnGraphemes(line, position-lineStart)
}

// nextPositionOnGraphemes returns the byte offset of the grapheme cluster
// boundary that immediately follows position. The next boundary is always
// inside the logical line containing position (cf. prevPositionOnGraphemes).
func (t *Text) nextPositionOnGraphemes(position int) int {
	if position >= t.store.TextLengthInBytes() {
		return position
	}
	line, lineStart := t.stringValueForLineContaining(position)
	return lineStart + textutil.NextPositionOnGraphemes(line, position-lineStart)
}

// prevWordStart returns the byte offset of the start of the last word before
// position, or 0 when no earlier word exists.
func (t *Text) prevWordStart(position int) int {
	// Step back over graphemes until the one just before position lies in a
	// word; [Text.findWordBoundaries] then yields that word's start. Both
	// helpers scan only the relevant logical line, so this crosses line breaks
	// without materializing the document.
	for position > 0 {
		prev := t.prevPositionOnGraphemes(position)
		if s, e := t.findWordBoundaries(prev); e > s {
			return s
		}
		position = prev
	}
	return 0
}

// nextWordEnd returns the byte offset of the end of the first word at or after
// position, or the text length when no further word exists.
func (t *Text) nextWordEnd(position int) int {
	// Step forward over graphemes until position lies in a word;
	// [Text.findWordBoundaries] then yields that word's end. Both helpers scan
	// only the relevant logical line, so this crosses line breaks without
	// materializing the document.
	total := t.store.TextLengthInBytes()
	for position < total {
		if _, e := t.findWordBoundaries(position); e > position {
			return e
		}
		next := t.nextPositionOnGraphemes(position)
		if next <= position {
			break
		}
		position = next
	}
	return total
}

// paragraphStart returns the byte offset of the beginning of the logical line
// containing position, or of the previous logical line when position is
// already at a line start.
func (t *Text) paragraphStart(position int) int {
	lineIndex := t.LineIndexFromTextIndexInBytes(position)
	lineStart := t.LineStartInBytes(lineIndex)
	if position > lineStart {
		return lineStart
	}
	if lineIndex > 0 {
		return t.LineStartInBytes(lineIndex - 1)
	}
	return 0
}

// paragraphEnd returns the byte offset of the end of the logical line
// containing position, excluding its trailing line break, or of the next
// logical line when position is already at a line end.
func (t *Text) paragraphEnd(position int) int {
	lineIndex := t.LineIndexFromTextIndexInBytes(position)
	lineEnd := t.logicalLineContentEnd(lineIndex)
	if position < lineEnd {
		return lineEnd
	}
	if lineIndex+1 < t.LineCount() {
		return t.logicalLineContentEnd(lineIndex + 1)
	}
	return t.store.TextLengthInBytes()
}

// logicalLineContentEnd returns the byte offset of the end of the lineIndex-th
// logical line's content, excluding its trailing line break.
func (t *Text) logicalLineContentEnd(lineIndex int) int {
	if lineIndex+1 >= t.LineCount() {
		return t.store.TextLengthInBytes()
	}
	lineEnd := t.LineStartInBytes(lineIndex + 1)
	lineStart := t.LineStartInBytes(lineIndex)
	// The trailing line break is at most a few bytes, so inspect a short
	// suffix rather than materializing the whole line.
	suffix := t.stringValueWithRange(max(lineStart, lineEnd-4), lineEnd)
	if i, l := textutil.LastLineBreakPositionAndLen(suffix); i >= 0 && i+l == len(suffix) {
		return lineEnd - l
	}
	return lineEnd
}

// nextWordStart returns the byte offset of the start of the next word after
// position, or the text length when no later word exists.
func (t *Text) nextWordStart(position int) int {
	total := t.store.TextLengthInBytes()
	// Skip past the word under the caret, then to the first following word
	// start. findWordBoundaries and the grapheme steppers scan only the
	// relevant logical line, so this crosses line breaks without materializing
	// the document.
	if _, e := t.findWordBoundaries(position); e > position {
		position = e
	}
	for position < total {
		if s, e := t.findWordBoundaries(position); s == position && e > position {
			return position
		}
		next := t.nextPositionOnGraphemes(position)
		if next <= position {
			break
		}
		position = next
	}
	return total
}

// visualLineStart returns the byte offset of the first index on the visual
// (wrapped) line holding the caret at position, and whether the caret's line is
// laid out. A far-left probe clamps to that line's start.
func (t *Text) visualLineStart(context *guigui.Context, widgetBounds *guigui.WidgetBounds, position int) (int, bool) {
	pos, ok := t.textPosition(context, widgetBounds.Bounds(), position, false)
	if !ok {
		return 0, false
	}
	y := int((pos.Top + pos.Bottom) / 2)
	idx := t.textIndexFromPosition(context, widgetBounds.Bounds(), image.Pt(math.MinInt32, y), false)
	if idx < 0 {
		return 0, false
	}
	return idx, true
}

// visualLineEnd returns the byte offset of the last index on the visual
// (wrapped) line holding the caret at position, excluding a trailing line
// break, and whether the caret's line is laid out.
func (t *Text) visualLineEnd(context *guigui.Context, widgetBounds *guigui.WidgetBounds, position int) (int, bool) {
	pos, ok := t.textPosition(context, widgetBounds.Bounds(), position, false)
	if !ok {
		return 0, false
	}
	y := int((pos.Top + pos.Bottom) / 2)
	idx := t.textIndexFromPosition(context, widgetBounds.Bounds(), image.Pt(math.MaxInt32, y), false)
	if idx < 0 {
		return 0, false
	}
	return idx, true
}

// navigateBackward moves the caret backward to target(position), or extends the
// selection there under Shift. position is the moving end; target reporting
// ok=false leaves the selection unchanged.
func (t *Text) navigateBackward(shift bool, target func(position int) (int, bool)) {
	start, end := t.store.Selection()
	position := start
	if shift && t.shiftSelectionSide == SelectionSideEnd {
		position = end
	}
	tgt, ok := target(position)
	if !ok {
		return
	}
	switch {
	case !shift:
		t.setSelection(tgt, tgt, SelectionSideNone, true)
	case t.shiftSelectionSide == SelectionSideEnd:
		t.setSelection(start, tgt, SelectionSideEnd, true)
	default:
		t.setSelection(tgt, end, SelectionSideStart, true)
	}
}

// navigateForward mirrors [Text.navigateBackward] in the forward direction.
func (t *Text) navigateForward(shift bool, target func(position int) (int, bool)) {
	start, end := t.store.Selection()
	position := end
	if shift && t.shiftSelectionSide == SelectionSideStart {
		position = start
	}
	tgt, ok := target(position)
	if !ok {
		return
	}
	switch {
	case !shift:
		t.setSelection(tgt, tgt, SelectionSideNone, true)
	case t.shiftSelectionSide == SelectionSideStart:
		t.setSelection(tgt, end, SelectionSideStart, true)
	default:
		t.setSelection(start, tgt, SelectionSideEnd, true)
	}
}

func (t *Text) stringValueForRendering() string {
	t.valueBuilder.Reset()
	_, _ = t.store.WriteTextForRenderingTo(&t.valueBuilder)
	return t.valueBuilder.String()
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
	if t.lineByteOffsets.LineCount() > 0 {
		startCtx := t.stringValueWithRange(max(0, start-2), start)
		endCtxStart := start + len(text)
		endCtxEnd := endCtxStart + 3
		endCtx := t.stringValueWithRange(endCtxStart, endCtxEnd)
		atEOT := endCtxEnd >= t.store.TextLengthInBytes()
		t.lineByteOffsets.Replace(text, start, end, startCtx, endCtx, atEOT)
		t.lineByteOffsetsFieldGeneration = t.store.Generation()
	}

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

func (t *Text) SetBold(bold bool) {
	t.bold = bold
}

func (t *Text) SetTabular(tabular bool) {
	t.tabular = tabular
}

func (t *Text) SetTabWidth(tabWidth float64) {
	if t.tabWidth == tabWidth {
		return
	}
	t.tabWidth = tabWidth
	t.resetCachedTextSize()
}

func (t *Text) actualTabWidth(context *guigui.Context) float64 {
	if t.tabWidth > 0 {
		return t.tabWidth
	}
	if t.cachedDefaultTabWidth > 0 {
		return t.cachedDefaultTabWidth
	}
	face := t.face(context, false)
	const defaultTabSpaces = "        "
	t.cachedDefaultTabWidth = text.AdvanceAt(defaultTabSpaces, len(defaultTabSpaces), face.TextFace())
	return t.cachedDefaultTabWidth
}

// Scale returns the text scale.
func (t *Text) Scale() float64 {
	return t.scaleMinus1 + 1
}

func (t *Text) SetScale(scale float64) {
	t.scaleMinus1 = scale - 1
}

func (t *Text) HorizontalAlign() textutil.HorizontalAlign {
	return t.hAlign
}

func (t *Text) SetHorizontalAlign(align textutil.HorizontalAlign) {
	t.hAlign = align
}

func (t *Text) VerticalAlign() textutil.VerticalAlign {
	return t.vAlign
}

func (t *Text) SetVerticalAlign(align textutil.VerticalAlign) {
	t.vAlign = align
}

func (t *Text) IsEditable() bool {
	return t.editable
}

func (t *Text) SetEditable(editable bool) {
	if t.editable == editable {
		return
	}

	if editable {
		t.selectionDragStartPlus1 = 0
		t.selectionDragEndPlus1 = 0
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

// masking reports whether the value is rendered as the mask rune.
func (t *Text) masking() bool {
	return t.maskRune != 0
}

// maskStyle returns the textutil style used to lay out the masked string. A
// masked value never wraps, so the wrap mode is forced to none.
func (t *Text) maskStyle(context *guigui.Context) textutil.Style {
	return textutil.Style{
		WrapMode:         textutil.WrapModeNone,
		Face:             t.face(context, false),
		LineHeight:       t.LineHeight(),
		HorizontalAlign:  t.hAlign,
		VerticalAlign:    t.vAlign,
		TabWidth:         t.actualTabWidth(context),
		KeepTailingSpace: t.keepTailingSpace,
	}
}

// maskMappingForRendering returns the [maskMapping] for the current rendering
// text. The returned pointer is owned by the receiver and is invalidated by the
// next edit, so callers must not retain it.
func (t *Text) maskMappingForRendering(context *guigui.Context, showComposition bool) *maskMapping {
	withComposition := showComposition && t.store.UncommittedTextLengthInBytes() > 0
	gen := t.store.Generation()
	if t.cachedMaskMapping.runeLen != 0 &&
		t.cachedMaskMappingGeneration == gen &&
		t.cachedMaskMappingRune == t.maskRune &&
		t.cachedMaskMappingWithComposition == withComposition {
		return &t.cachedMaskMapping
	}
	t.cachedMaskMapping.reset(t.textToDraw(context, withComposition), t.maskRune)
	t.cachedMaskMappingGeneration = gen
	t.cachedMaskMappingRune = t.maskRune
	t.cachedMaskMappingWithComposition = withComposition
	return &t.cachedMaskMapping
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

func (t *Text) textContentBounds(context *guigui.Context, bounds image.Rectangle) image.Rectangle {
	b := bounds
	h := t.textHeight(context, guigui.FixedWidthConstraints(t.LayoutWidth(b)))

	switch t.vAlign {
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
	if t.vAlign == textutil.VerticalAlignTop {
		// For Top, [Text.textContentBounds] would only tighten Max.Y, which
		// no caller depends on beyond it staying within bounds. Skip it to
		// avoid [Text.textHeight], which walks every logical line for wrapped text.
		return bounds
	}
	return t.textContentBounds(context, bounds)
}

func (t *Text) fontFamilyID() uint64 {
	if t.fontFamily == nil {
		return 0
	}
	return t.fontFamily.ID()
}

func (t *Text) faceAttributes(forceBold bool) font.Attributes {
	size := t.baseFontSize * t.Scale()
	weight := text.WeightMedium
	if t.bold || forceBold {
		weight = text.WeightBold
	}

	// Disable ligatures for editable, selectable, or masked text so caret
	// positions land on byte boundaries.
	liga := !t.selectable && !t.editable && !t.masking()
	tnum := t.tabular

	return font.Attributes{
		Size:   size,
		Weight: weight,
		Liga:   liga,
		Tnum:   tnum,
		Lang:   t.lang,
	}
}

// face must be called after [Text.Build], as it relies on lastFaceAttributes being set.
func (t *Text) face(context *guigui.Context, forceBold bool) font.Face {
	attrs := t.lastFaceAttributes
	if forceBold {
		attrs.Weight = text.WeightBold
	}
	return font.NewFace(context, t.fontFamily, attrs)
}

// LineHeight returns the line height in pixels, with the widget scale applied.
func (t *Text) LineHeight() float64 {
	return t.baseLineHeight * t.Scale()
}

func (t *Text) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if !t.selectable && !t.editable {
		return guigui.HandleInputResult{}
	}

	cursorPosition := image.Pt(ebiten.CursorPosition())
	if t.dragging {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			idx := t.textIndexFromPosition(context, widgetBounds.Bounds(), cursorPosition, false)
			start, end := idx, idx
			if t.selectionDragStartPlus1-1 >= 0 {
				start = min(start, t.selectionDragStartPlus1-1)
			}
			if t.selectionDragEndPlus1-1 >= 0 {
				end = max(idx, t.selectionDragEndPlus1-1)
			}
			// idx is the dragged-to position; record whichever endpoint it
			// became as the moving end so a subsequent Shift+click or
			// Shift+arrow extends from the opposite, anchored end. While the
			// cursor stays inside a word- or line-selection, idx matches
			// neither endpoint and no moving end is tracked.
			var shiftSide SelectionSide
			switch idx {
			case start:
				shiftSide = SelectionSideStart
			case end:
				shiftSide = SelectionSideEnd
			}
			if t.setSelection(start, end, shiftSide, true) {
				return guigui.HandleInputByWidget(t)
			} else {
				return guigui.AbortHandlingInputByWidget(t)
			}
		}
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			t.dragging = false
			t.selectionDragStartPlus1 = 0
			t.selectionDragEndPlus1 = 0
			return guigui.HandleInputByWidget(t)
		}
		return guigui.AbortHandlingInputByWidget(t)
	}

	left := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	right := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	if left || right {
		if widgetBounds.IsHitAtCursor() {
			t.handleClick(context, widgetBounds.Bounds(), cursorPosition, left)
			if left {
				return guigui.HandleInputByWidget(t)
			}
			return guigui.HandleInputResult{}
		}
		context.SetFocused(t, false)
	}

	if !context.IsFocused(t) {
		if t.store.IsFocused() {
			t.store.Blur()
		}
		return guigui.HandleInputResult{}
	}
	// The field auto-commits text input via Ebitengine's BeforeUpdate hook whenever
	// it is focused, so only focus it when this widget actually accepts edits.
	if t.editable {
		t.store.Focus()
	} else if t.store.IsFocused() {
		t.store.Blur()
	}

	if !t.editable && !t.selectable {
		return guigui.HandleInputResult{}
	}

	return guigui.HandleInputResult{}
}

func (t *Text) handleClick(context *guigui.Context, textBounds image.Rectangle, cursorPosition image.Point, leftClick bool) {
	idx := t.textIndexFromPosition(context, textBounds, cursorPosition, false)

	// Shift+click on a text that already holds a cursor moves one end of the
	// selection to the clicked position and keeps the opposite end anchored.
	// Dragging afterwards keeps extending from the same anchor.
	if leftClick && idx >= 0 && ebiten.IsKeyPressed(ebiten.KeyShift) && context.IsFocusedOrHasFocusedDescendant(t) {
		selStart, selEnd := t.store.Selection()
		anchor := shiftClickAnchor(selStart, selEnd, t.shiftSelectionSide, idx)
		t.dragging = true
		t.selectionDragStartPlus1 = anchor + 1
		t.selectionDragEndPlus1 = anchor + 1
		t.setSelection(anchor, idx, SelectionSideEnd, false)
		context.SetFocused(t, true)
		// Reset the click count so a following plain click is not treated as a
		// double- or triple-click.
		t.clickCounter.reset(ebiten.Tick(), idx)
		return
	}

	clickCount := t.clickCounter.click(ebiten.Tick(), idx, leftClick)

	switch clickCount {
	case 1:
		if leftClick {
			t.dragging = true
			t.selectionDragStartPlus1 = idx + 1
			t.selectionDragEndPlus1 = idx + 1
		} else {
			t.dragging = false
			t.selectionDragStartPlus1 = 0
			t.selectionDragEndPlus1 = 0
		}
		if leftClick || !context.IsFocusedOrHasFocusedDescendant(t) {
			if start, end := t.store.Selection(); start != idx || end != idx {
				t.setSelection(idx, idx, SelectionSideNone, false)
			}
		}
	case 2:
		t.dragging = true
		start, end := t.findWordBoundaries(idx)
		t.selectionDragStartPlus1 = start + 1
		t.selectionDragEndPlus1 = end + 1
		t.setSelection(start, end, SelectionSideNone, false)
	case 3:
		t.doSelectAll()
	}

	context.SetFocused(t, true)
}

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
func (t *Text) restrictedTextToDraw(context *guigui.Context, textBounds, visibleBounds image.Rectangle) (txt string, byteStart int, yShift int, restricted bool) {
	t.ensureLineByteOffsets()
	n := t.lineByteOffsets.LineCount()

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

		selectionLineIdx := t.lineByteOffsets.LineIndexForByteOffset(sStart)
		cs := t.lineByteOffsets.ByteOffsetByLineIndex(selectionLineIdx)
		ce := t.store.TextLengthInBytes()
		if selectionLineIdx+1 < n {
			ce = t.lineByteOffsets.ByteOffsetByLineIndex(selectionLineIdx + 1)
		}
		// The selection-line slices are only valid when the selection
		// lies inside a single logical line; otherwise ce+byteDelta
		// underflows. When the selection crosses lines we leave them
		// empty — [textutil.ComputeCompositionInfo]'s own multi-line
		// check returns false before reading them, and the caller falls
		// back below.
		var committedSelectionLine, renderingSelectionLine string
		if t.wrapMode != textutil.WrapModeNone && t.lineByteOffsets.LineIndexForByteOffset(sEnd) == selectionLineIdx {
			committedSelectionLine = t.stringValueWithRange(cs, ce)
			renderingSelectionLine = t.stringValueForRenderingRange(cs, ce+byteDelta)
		}

		info, ok := textutil.ComputeCompositionInfo(&textutil.CompositionInfoParams{
			CompositionText:        t.stringValueForRenderingRange(sStart, sStart+compLen),
			LineByteOffsets:        &t.lineByteOffsets,
			SelectionStart:         sStart,
			SelectionEnd:           sEnd,
			WrapMode:               t.wrapMode,
			CommittedSelectionLine: committedSelectionLine,
			RenderingSelectionLine: renderingSelectionLine,
			Face:                   t.face(context, false),
			LineHeight:             t.LineHeight(),
			TabWidth:               t.actualTabWidth(context),
			KeepTailingSpace:       t.keepTailingSpace,
			WrapWidth:              width,
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
	if t.vAlign == textutil.VerticalAlignTop {
		readRendering := t.stringValueWithRange
		if hasComp {
			readRendering = t.stringValueForRenderingRange
		}
		r, ok := textutil.VisibleRangeInViewport(&textutil.VisibleRangeInViewportParams{
			FirstLogicalLineInViewport: t.firstLogicalLineInViewport,
			LineByteOffsets:            &t.lineByteOffsets,
			RenderingTextRange:         readRendering,
			RenderingTextLength:        renderingLength,
			ViewportSize: image.Pt(
				width,
				visibleBounds.Max.Y-textBounds.Min.Y,
			),
			Face:             t.face(context, false),
			LineHeight:       t.LineHeight(),
			TabWidth:         t.actualTabWidth(context),
			KeepTailingSpace: t.keepTailingSpace,
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
	switch t.vAlign {
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
		LineByteOffsets:            &t.lineByteOffsets,
		RenderingTextRange:         readRendering,
		RenderingTextLength:        renderingLength,
		ViewportSize: image.Pt(
			width,
			visibleBounds.Max.Y-textBounds.Min.Y-alignOffset,
		),
		Face:             t.face(context, false),
		LineHeight:       t.LineHeight(),
		TabWidth:         t.actualTabWidth(context),
		KeepTailingSpace: t.keepTailingSpace,
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

func (t *Text) HandleButtonInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	r := t.handleButtonInput(context, widgetBounds)
	// Adjust the scroll offset right after handling the input so that
	// the scroll delta is applied during the next Build & Layout pass
	// within the same tick, avoiding a one-tick wobble.
	if r.IsHandled() && (t.selectable || t.editable) {
		if dx, dy := t.adjustScrollOffset(context, widgetBounds); dx != 0 || dy != 0 {
			guigui.DispatchEvent(t, textEventScrollDelta, dx, dy)
		}
	}
	return r
}

// updateIMEComposer pumps the IME composer for one tick and folds a resulting
// composition or commit into the cached text size and the value-changed
// listeners. It reports whether the IME consumed input this tick.
func (t *Text) updateIMEComposer(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	start, _ := t.store.Selection()
	if pos, ok := t.textPosition(context, widgetBounds.Bounds(), start, false); ok {
		t.store.SetBounds(image.Rect(int(pos.X), int(pos.Top), int(pos.X+1), int(pos.Bottom)))
	}
	processed, err := t.store.Update()
	if err != nil {
		slog.Error(err.Error())
	}
	if processed {
		// Reset the cached size before the scroll offset is adjusted so the text size is correct.
		t.resetCachedTextSize()
		t.dispatchValueChanged(false, false)
	}
	return processed
}

func (t *Text) handleButtonInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if t.onHandleButtonInput != nil {
		if r := t.onHandleButtonInput(context, widgetBounds); r.IsHandled() {
			return r
		}
	}

	if !t.selectable && !t.editable {
		return guigui.HandleInputResult{}
	}

	if t.editable {
		if t.updateIMEComposer(context, widgetBounds) {
			return guigui.HandleInputByWidget(t)
		}

		// Do not accept key inputs when compositing.
		if _, _, ok := t.store.CompositionSelection(); ok {
			return guigui.HandleInputByWidget(t)
		}

		// For Windows key binds, see:
		// https://support.microsoft.com/en-us/windows/keyboard-shortcuts-in-windows-dcc61a57-8ff0-cffe-9796-cb9706c75eec#textediting

		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			if t.IsMultiline() {
				t.replaceTextAtSelection("\n")
			} else {
				t.commit(true)
			}
			return guigui.HandleInputByWidget(t)
		case IsKeyRepeating(ebiten.KeyBackspace) ||
			IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyH):
			start, end := t.store.Selection()
			if start != end {
				t.replaceTextAtSelection("")
			} else if start > 0 {
				pos := t.prevPositionOnGraphemes(start)
				t.replaceTextAt("", pos, start)
			}
			return guigui.HandleInputByWidget(t)
		case !IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyD) ||
			IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyD):
			// Delete
			start, end := t.store.Selection()
			if start != end {
				t.replaceTextAtSelection("")
			} else if IsDarwin() && end < t.store.TextLengthInBytes() {
				pos := t.nextPositionOnGraphemes(end)
				t.replaceTextAt("", start, pos)
			}
			return guigui.HandleInputByWidget(t)
		case IsKeyRepeating(ebiten.KeyDelete):
			// Delete one cluster
			if start, end := t.store.Selection(); end < t.store.TextLengthInBytes() {
				pos := t.nextPositionOnGraphemes(end)
				t.replaceTextAt("", start, pos)
			}
			return guigui.HandleInputByWidget(t)
		case !IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyX) ||
			IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && IsKeyRepeating(ebiten.KeyX):
			t.Cut()
			return guigui.HandleInputByWidget(t)
		case !IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyV) ||
			IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && IsKeyRepeating(ebiten.KeyV):
			t.Paste()
			return guigui.HandleInputByWidget(t)
		case !IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyY) ||
			IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && ebiten.IsKeyPressed(ebiten.KeyShift) && IsKeyRepeating(ebiten.KeyZ):
			t.Redo()
			return guigui.HandleInputByWidget(t)
		case !IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyZ) ||
			IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && IsKeyRepeating(ebiten.KeyZ):
			t.Undo()
			return guigui.HandleInputByWidget(t)
		}
	}

	switch {
	// macOS: Command+Arrow moves to a visual-line or document extreme;
	// Option+Arrow moves by word or paragraph. Shift extends the selection.
	case IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && IsKeyRepeating(ebiten.KeyLeft):
		t.navigateBackward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.visualLineStart(context, widgetBounds, position)
		})
		return guigui.HandleInputByWidget(t)
	case IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && IsKeyRepeating(ebiten.KeyRight):
		t.navigateForward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.visualLineEnd(context, widgetBounds, position)
		})
		return guigui.HandleInputByWidget(t)
	case IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && IsKeyRepeating(ebiten.KeyUp):
		t.navigateBackward(ebiten.IsKeyPressed(ebiten.KeyShift), func(int) (int, bool) {
			return 0, true
		})
		return guigui.HandleInputByWidget(t)
	case IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && IsKeyRepeating(ebiten.KeyDown):
		t.navigateForward(ebiten.IsKeyPressed(ebiten.KeyShift), func(int) (int, bool) {
			return t.store.TextLengthInBytes(), true
		})
		return guigui.HandleInputByWidget(t)
	case IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyAlt) && IsKeyRepeating(ebiten.KeyLeft):
		t.navigateBackward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.prevWordStart(position), true
		})
		return guigui.HandleInputByWidget(t)
	case IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyAlt) && IsKeyRepeating(ebiten.KeyRight):
		t.navigateForward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.nextWordEnd(position), true
		})
		return guigui.HandleInputByWidget(t)
	case IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyAlt) && IsKeyRepeating(ebiten.KeyUp):
		t.navigateBackward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.paragraphStart(position), true
		})
		return guigui.HandleInputByWidget(t)
	case IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyAlt) && IsKeyRepeating(ebiten.KeyDown):
		t.navigateForward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.paragraphEnd(position), true
		})
		return guigui.HandleInputByWidget(t)
	// macOS: Shift+Home/End extend the selection to the start/end of the text.
	// Plain Home/End scroll without moving the caret; they are left unhandled
	// here and handled by the virtualizing parent after bubbling up.
	case IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyShift) && IsKeyRepeating(ebiten.KeyHome):
		t.navigateBackward(true, func(int) (int, bool) {
			return 0, true
		})
		return guigui.HandleInputByWidget(t)
	case IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyShift) && IsKeyRepeating(ebiten.KeyEnd):
		t.navigateForward(true, func(int) (int, bool) {
			return t.store.TextLengthInBytes(), true
		})
		return guigui.HandleInputByWidget(t)
	// Windows/Linux: Ctrl+Arrow moves by word, Home/End to line head/tail,
	// Ctrl+Home/End to document head/tail. Shift extends the selection.
	case !IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyLeft):
		t.navigateBackward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.prevWordStart(position), true
		})
		return guigui.HandleInputByWidget(t)
	case !IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyRight):
		t.navigateForward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.nextWordStart(position), true
		})
		return guigui.HandleInputByWidget(t)
	case !IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyHome):
		t.navigateBackward(ebiten.IsKeyPressed(ebiten.KeyShift), func(int) (int, bool) {
			return 0, true
		})
		return guigui.HandleInputByWidget(t)
	case !IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyEnd):
		t.navigateForward(ebiten.IsKeyPressed(ebiten.KeyShift), func(int) (int, bool) {
			return t.store.TextLengthInBytes(), true
		})
		return guigui.HandleInputByWidget(t)
	case !IsDarwin() && IsKeyRepeating(ebiten.KeyHome):
		t.navigateBackward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.visualLineStart(context, widgetBounds, position)
		})
		return guigui.HandleInputByWidget(t)
	case !IsDarwin() && IsKeyRepeating(ebiten.KeyEnd):
		t.navigateForward(ebiten.IsKeyPressed(ebiten.KeyShift), func(position int) (int, bool) {
			return t.visualLineEnd(context, widgetBounds, position)
		})
		return guigui.HandleInputByWidget(t)
	case IsKeyRepeating(ebiten.KeyLeft) ||
		IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyB):
		start, end := t.store.Selection()
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			if t.shiftSelectionSide == SelectionSideEnd {
				pos := t.prevPositionOnGraphemes(end)
				t.setSelection(start, pos, SelectionSideEnd, true)
			} else {
				pos := t.prevPositionOnGraphemes(start)
				t.setSelection(pos, end, SelectionSideStart, true)
			}
		} else {
			if start != end {
				t.setSelection(start, start, SelectionSideNone, true)
			} else if start > 0 {
				pos := t.prevPositionOnGraphemes(start)
				t.setSelection(pos, pos, SelectionSideNone, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case IsKeyRepeating(ebiten.KeyRight) ||
		IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyF):
		start, end := t.store.Selection()
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			if t.shiftSelectionSide == SelectionSideStart {
				pos := t.nextPositionOnGraphemes(start)
				t.setSelection(pos, end, SelectionSideStart, true)
			} else {
				pos := t.nextPositionOnGraphemes(end)
				t.setSelection(start, pos, SelectionSideEnd, true)
			}
		} else {
			if start != end {
				t.setSelection(end, end, SelectionSideNone, true)
			} else if start < t.store.TextLengthInBytes() {
				pos := t.nextPositionOnGraphemes(start)
				t.setSelection(pos, pos, SelectionSideNone, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case IsKeyRepeating(ebiten.KeyUp) ||
		IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyP):
		lh := t.LineHeight()
		shift := ebiten.IsKeyPressed(ebiten.KeyShift)
		var moveEnd bool
		start, end := t.store.Selection()
		idx := start
		if shift && t.shiftSelectionSide == SelectionSideEnd {
			idx = end
			moveEnd = true
		}
		if pos, ok := t.textPosition(context, widgetBounds.Bounds(), idx, false); ok {
			y := (pos.Top+pos.Bottom)/2 - lh
			nextIdx := t.textIndexFromPosition(context, widgetBounds.Bounds(), image.Pt(int(pos.X), int(y)), false)
			// A genuine move to the previous line lands on an earlier byte
			// offset. When the caret is already on the first line, the move is
			// clamped and round-trips to the same offset; move to the head of
			// the text instead.
			if nextIdx >= idx {
				nextIdx = 0
			}
			if shift {
				if moveEnd {
					t.setSelection(start, nextIdx, SelectionSideEnd, true)
				} else {
					t.setSelection(nextIdx, end, SelectionSideStart, true)
				}
			} else {
				t.setSelection(nextIdx, nextIdx, SelectionSideNone, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case IsKeyRepeating(ebiten.KeyDown) ||
		IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyN):
		lh := t.LineHeight()
		shift := ebiten.IsKeyPressed(ebiten.KeyShift)
		var moveStart bool
		start, end := t.store.Selection()
		idx := end
		if shift && t.shiftSelectionSide == SelectionSideStart {
			idx = start
			moveStart = true
		}
		if pos, ok := t.textPosition(context, widgetBounds.Bounds(), idx, false); ok {
			y := (pos.Top+pos.Bottom)/2 + lh
			nextIdx := t.textIndexFromPosition(context, widgetBounds.Bounds(), image.Pt(int(pos.X), int(y)), false)
			// A genuine move to the next line lands on a later byte offset. When
			// the caret is already on the last line, the move is clamped and
			// round-trips to the same offset; move to the tail of the text
			// instead.
			if nextIdx <= idx {
				nextIdx = t.store.TextLengthInBytes()
			}
			if shift {
				if moveStart {
					t.setSelection(nextIdx, end, SelectionSideStart, true)
				} else {
					t.setSelection(start, nextIdx, SelectionSideEnd, true)
				}
			} else {
				t.setSelection(nextIdx, nextIdx, SelectionSideNone, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyA):
		idx := 0
		start, end := t.store.Selection()
		if i, l := textutil.LastLineBreakPositionAndLen(t.stringValueWithRange(0, start)); i >= 0 {
			idx = i + l
		}
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			t.setSelection(idx, end, SelectionSideStart, true)
		} else {
			t.setSelection(idx, idx, SelectionSideNone, true)
		}
		return guigui.HandleInputByWidget(t)
	case IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyE):
		idx := t.store.TextLengthInBytes()
		start, end := t.store.Selection()
		if i, _ := textutil.FirstLineBreakPositionAndLen(t.stringValueWithRange(end, -1)); i >= 0 {
			idx = end + i
		}
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			t.setSelection(start, idx, SelectionSideEnd, true)
		} else {
			t.setSelection(idx, idx, SelectionSideNone, true)
		}
		return guigui.HandleInputByWidget(t)
	case !IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyA) ||
		IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && IsKeyRepeating(ebiten.KeyA):
		t.doSelectAll()
		return guigui.HandleInputByWidget(t)
	case !IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyC) ||
		IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && IsKeyRepeating(ebiten.KeyC):
		// Copy
		t.Copy()
		return guigui.HandleInputByWidget(t)
	case IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyK):
		// 'Kill' the text after the caret or the selection.
		start, end := t.store.Selection()
		if start == end {
			i, l := textutil.FirstLineBreakPositionAndLen(t.stringValueWithRange(start, -1))
			if i < 0 {
				end = t.store.TextLengthInBytes()
			} else if i == 0 {
				end = start + l
			} else {
				end = start + i
			}
		}
		t.tmpClipboard = t.stringValueWithRange(start, end)
		t.replaceTextAt("", start, end)
		return guigui.HandleInputByWidget(t)
	case IsDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && IsKeyRepeating(ebiten.KeyY):
		// 'Yank' the killed text.
		if t.tmpClipboard != "" {
			t.replaceTextAtSelection(t.tmpClipboard)
		}
		return guigui.HandleInputByWidget(t)
	}

	return guigui.HandleInputResult{}
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

func (t *Text) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	textBounds := t.contentBoundsForLayout(context, widgetBounds.Bounds())
	if !textBounds.Overlaps(widgetBounds.VisibleBounds()) {
		return
	}

	face := t.face(context, false)
	op := &t.drawOptions
	op.Style.WrapMode = t.wrapMode
	op.Style.Face = face
	op.Style.LineHeight = t.LineHeight()
	op.Style.HorizontalAlign = t.hAlign
	op.Style.VerticalAlign = t.vAlign
	op.Style.TabWidth = t.actualTabWidth(context)
	op.Style.KeepTailingSpace = t.keepTailingSpace
	op.LayoutWidth = t.LayoutWidth(textBounds)
	if !t.editable {
		op.Style.EllipsisString = t.ellipsisString
	} else {
		op.Style.EllipsisString = ""
	}
	op.TextColor = t.textColor
	op.VisibleBounds = widgetBounds.VisibleBounds()
	if start, end, ok := t.selectionToDraw(context); ok {
		if context.IsFocused(t) || (t.selectionVisibleWhenUnfocus && start != end) {
			op.DrawSelection = true
			op.SelectionStart = start
			op.SelectionEnd = end
			op.SelectionColor = t.selectionColor
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
		op.InactiveCompositionColor = t.inactiveCompositionColor
		op.ActiveCompositionColor = t.activeCompositionColor
		op.CompositionBorderWidth = float32(textCaretWidth(context))
	} else {
		op.DrawComposition = false
	}

	if t.masking() {
		// A masked value is single-line, uniform, and short, so it bypasses
		// the virtualized restriction path: draw the whole masked string and
		// remap the selection/composition offsets into masked space.
		m := t.maskMappingForRendering(context, true)
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

	txt, byteStart, yShift, restricted := t.restrictedTextToDraw(context, textBounds, widgetBounds.VisibleBounds())
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
	textutil.Draw(textBounds, dst, txt, op)
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

	bold := t.bold
	key := newTextSizeCacheKey(t.wrapMode, bold)

	for i := range t.cachedTextHeights[key] {
		entry := &t.cachedTextHeights[key][i]
		if entry.constraintWidth == 0 {
			continue
		}
		if entry.constraintWidth != constraintWidth {
			continue
		}
		if i == 0 {
			return entry.height
		}
		e := *entry
		copy(t.cachedTextHeights[key][1:i+1], t.cachedTextHeights[key][:i])
		t.cachedTextHeights[key][0] = e
		return e.height
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
		h := textutil.MeasureHeight(constraintWidth, txt, t.wrapMode, t.face(context, bold), lineH, t.actualTabWidth(context), t.keepTailingSpace)
		hi = int(math.Ceil(h))
	}

	copy(t.cachedTextHeights[key][1:], t.cachedTextHeights[key][:])
	t.cachedTextHeights[key][0] = cachedTextHeightEntry{
		constraintWidth: constraintWidth,
		height:          hi,
	}

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
	n := t.lineByteOffsets.LineCount()

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
		selectionLineIdx = t.lineByteOffsets.LineIndexForByteOffset(sStart)
		if t.lineByteOffsets.LineIndexForByteOffset(sEnd) != selectionLineIdx {
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
	totalLen := t.store.TextLengthInBytes()
	var count int
	for i := range n {
		cs := t.lineByteOffsets.ByteOffsetByLineIndex(i)
		ce := totalLen
		if i+1 < n {
			ce = t.lineByteOffsets.ByteOffsetByLineIndex(i + 1)
		}
		var line string
		if hasComp && i == selectionLineIdx {
			line = t.stringValueForRenderingRange(cs, ce+byteDelta)
		} else {
			line = t.stringValueWithRange(cs, ce)
		}
		count += textutil.VisualLineCountForLogicalLine(measureWidth, line, t.wrapMode, face, tabW, keepTailing)
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
	n := t.lineByteOffsets.LineCount()

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
		selectionLineIdx = t.lineByteOffsets.LineIndexForByteOffset(sStart)
		if t.lineByteOffsets.LineIndexForByteOffset(sEnd) != selectionLineIdx {
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
	totalLen := t.store.TextLengthInBytes()

	var maxWidth, height float64
	for i := range n {
		cs := t.lineByteOffsets.ByteOffsetByLineIndex(i)
		ce := totalLen
		if i+1 < n {
			ce = t.lineByteOffsets.ByteOffsetByLineIndex(i + 1)
		}
		var line string
		if hasComp && i == selectionLineIdx {
			line = t.stringValueForRenderingRange(cs, ce+byteDelta)
		} else {
			line = t.stringValueWithRange(cs, ce)
		}
		w, h := textutil.MeasureLogicalLine(measureWidth, line, t.wrapMode, face, lineH, tabW, keepTailing, ellipsisString)
		maxWidth = max(maxWidth, w)
		height += h
	}
	return maxWidth, height, true
}

func (t *Text) textSize(context *guigui.Context, constraints guigui.Constraints, forceBold bool) image.Point {
	bold := t.bold || forceBold

	if t.masking() {
		// A masked value is a single uniform line; measure it directly rather
		// than through the cache, which is populated from the real text.
		m := t.maskMappingForRendering(context, true)
		w, h := textutil.Measure(math.MaxInt, m.maskStr, textutil.WrapModeNone, t.face(context, bold), t.LineHeight(), t.actualTabWidth(context), t.keepTailingSpace, "")
		return image.Pt(max(int(math.Ceil(w)), 1), int(math.Ceil(h)))
	}

	constraintWidth := math.MaxInt
	if w, ok := constraints.FixedWidth(); ok {
		constraintWidth = w
	}
	if constraintWidth == 0 {
		constraintWidth = 1
	}

	key := newTextSizeCacheKey(t.wrapMode, bold)

	var width, height int
	var hasWidth, hasHeight bool

	for i := range t.cachedTextWidths[key] {
		entry := &t.cachedTextWidths[key][i]
		if entry.constraintWidth == 0 {
			continue
		}
		if entry.constraintWidth != constraintWidth {
			continue
		}
		width = entry.width
		hasWidth = true
		if i != 0 {
			e := *entry
			copy(t.cachedTextWidths[key][1:i+1], t.cachedTextWidths[key][:i])
			t.cachedTextWidths[key][0] = e
		}
		break
	}

	for i := range t.cachedTextHeights[key] {
		entry := &t.cachedTextHeights[key][i]
		if entry.constraintWidth == 0 {
			continue
		}
		if entry.constraintWidth != constraintWidth {
			continue
		}
		height = entry.height
		hasHeight = true
		if i != 0 {
			e := *entry
			copy(t.cachedTextHeights[key][1:i+1], t.cachedTextHeights[key][:i])
			t.cachedTextHeights[key][0] = e
		}
		break
	}

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
		w, h = textutil.Measure(constraintWidth, txt, t.wrapMode, t.face(context, bold), t.LineHeight(), t.actualTabWidth(context), t.keepTailingSpace, ellipsisString)
	}
	// If width is 0, the text's bounds and visible bounds are empty, and nothing including its caret is rendered.
	// Force to set a positive number as the width.
	w = max(w, 1)

	if !hasWidth {
		width = int(math.Ceil(w))
		copy(t.cachedTextWidths[key][1:], t.cachedTextWidths[key][:])
		t.cachedTextWidths[key][0] = cachedTextWidthEntry{
			constraintWidth: constraintWidth,
			width:           width,
		}
	}
	if !hasHeight {
		height = int(math.Ceil(h))
		copy(t.cachedTextHeights[key][1:], t.cachedTextHeights[key][:])
		t.cachedTextHeights[key][0] = cachedTextHeightEntry{
			constraintWidth: constraintWidth,
			height:          height,
		}
	}

	return image.Pt(width, height)
}

func (t *Text) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	if t.selectable || t.editable {
		return ebiten.CursorShapeText, true
	}
	return 0, false
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
	n := t.lineByteOffsets.LineCount()
	if n == 0 {
		return true
	}
	line := t.lineByteOffsets.LineIndexForByteOffset(byteOffset)
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
		m := t.maskMappingForRendering(context, showComposition)
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
	s := textutil.Style{
		WrapMode:         t.wrapMode,
		Face:             t.face(context, false),
		LineHeight:       t.LineHeight(),
		HorizontalAlign:  t.hAlign,
		VerticalAlign:    t.vAlign,
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
		PrecomputedLineByteOffsets: &t.lineByteOffsets,
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
		m := t.maskMappingForRendering(context, showComposition)
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
	s := textutil.Style{
		WrapMode:         t.wrapMode,
		Face:             t.face(context, false),
		LineHeight:       t.LineHeight(),
		HorizontalAlign:  t.hAlign,
		VerticalAlign:    t.vAlign,
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
		PrecomputedLineByteOffsets: &t.lineByteOffsets,
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
		m := t.maskMappingForRendering(context, showComposition)
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
	s := textutil.Style{
		WrapMode:         t.wrapMode,
		Face:             t.face(context, false),
		LineHeight:       t.LineHeight(),
		HorizontalAlign:  t.hAlign,
		VerticalAlign:    t.vAlign,
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
		PrecomputedLineByteOffsets: &t.lineByteOffsets,
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

func textCaretWidth(context *guigui.Context) int {
	return int(math.Ceil(2 * context.Scale()))
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

// SetPaddingForScrollOffset sets the padding kept between the caret and the
// viewport edges when scrolling the selection into view.
func (t *Text) SetPaddingForScrollOffset(padding guigui.Padding) {
	t.paddingForScrollOffset = padding
}

func (t *Text) adjustScrollOffset(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (dx, dy float64) {
	start, end, ok := t.selectionToDraw(context)
	if !ok {
		return
	}
	if t.prevStart == start && t.prevEnd == end && !t.dragging {
		return
	}
	t.prevStart = start
	t.prevEnd = end

	textBounds := widgetBounds.Bounds()
	textVisibleBounds := widgetBounds.VisibleBounds()

	if t.dragging {
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

type textCaret struct {
	guigui.DefaultWidget

	text *Text

	counter   int
	prevAlpha float64
	prevPos   textutil.TextPosition
	prevOK    bool
}

func (t *textCaret) resetCounter() {
	t.counter = 0
}

func (t *textCaret) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	pos, ok := t.text.caretPosition(context, t.text.widgetBoundsRect)
	if t.prevPos != pos {
		t.resetCounter()
	}
	t.prevPos = pos
	t.prevOK = ok

	t.counter++
	if a := t.alpha(context); t.prevAlpha != a {
		t.prevAlpha = a
		guigui.RequestRedraw(t)
	}
	return nil
}

func (t *textCaret) alpha(context *guigui.Context) float64 {
	// prevOK reflects the current tick: Tick refreshes it before alpha
	// is called, and Draw runs after Tick in the same tick.
	if !t.prevOK {
		return 0
	}
	s, e, ok := t.text.selectionToDraw(context)
	if !ok {
		return 0
	}
	if s != e {
		return 0
	}
	if t.text.caretStatic {
		return 1
	}
	offset := ebiten.TPS() / 2
	if t.counter <= offset {
		return 1
	}
	interval := ebiten.TPS()
	c := (t.counter - offset) % interval
	if c < interval/5 {
		return 1 - float64(c)/float64(interval/5)
	}
	if c < interval*2/5 {
		return 0
	}
	if c < interval*3/5 {
		return float64(c-interval*2/5) / float64(interval/5)
	}
	return 1
}

func (t *textCaret) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	alpha := t.alpha(context)
	if alpha == 0 {
		return
	}
	b := widgetBounds.Bounds()
	if b.Empty() {
		return
	}
	w := textCaretWidth(context)
	region := t.text.widgetBoundsRect
	region.Min.X -= w
	region.Max.X += w
	if !b.In(region) {
		return
	}
	clr := draw.ScaleAlpha(t.text.caretColor, alpha)
	draw.DrawRoundedRect(context, dst, b, clr, b.Dx()/2)
}

type stringEqualChecker struct {
	str    string
	pos    int
	result bool
}

func (s *stringEqualChecker) Reset(str string) {
	s.str = str
	s.pos = 0
	s.result = true
}

func (s *stringEqualChecker) Result() bool {
	if s.pos != len(s.str) {
		return false
	}
	return s.result
}

func (s *stringEqualChecker) Write(b []byte) (int, error) {
	if s.pos+len(b) > len(s.str) {
		s.result = false
		return 0, io.EOF
	}
	if !bytes.Equal([]byte(s.str[s.pos:s.pos+len(b)]), b) {
		s.result = false
		return 0, io.EOF
	}
	s.pos += len(b)
	return len(b), nil
}

// SetFontFamily sets the resolved font family used to render the value. A nil
// family renders with the registered face source stack alone.
func (t *Text) SetFontFamily(fontFamily *font.Family) {
	t.fontFamily = fontFamily
}

// SetBaseFontSize sets the font size at scale 1. The rendered size is the
// base size multiplied by the scale set via [Text.SetScale].
func (t *Text) SetBaseFontSize(size float64) {
	t.baseFontSize = size
}

// SetBaseLineHeight sets the line height at scale 1. The rendered line height
// is the base line height multiplied by the scale set via [Text.SetScale].
func (t *Text) SetBaseLineHeight(lineHeight float64) {
	if t.baseLineHeight == lineHeight {
		return
	}
	t.baseLineHeight = lineHeight
	t.resetCachedTextSize()
}

// SetLang sets the language used to select the face and its features when
// shaping the value.
func (t *Text) SetLang(lang language.Tag) {
	if t.lang == lang {
		return
	}
	t.lang = lang
	t.langString = lang.String()
}

// SetTextColor sets the concrete color the value is drawn in.
func (t *Text) SetTextColor(clr color.Color) {
	t.textColor = clr
}

// SetSelectionColor sets the concrete color of the selection highlight.
func (t *Text) SetSelectionColor(clr color.Color) {
	t.selectionColor = clr
}

// SetCompositionColors sets the concrete colors of the underlines drawn below
// the inactive and active parts of an IME composition.
func (t *Text) SetCompositionColors(inactive, active color.Color) {
	t.inactiveCompositionColor = inactive
	t.activeCompositionColor = active
}

// SetCaretColor sets the concrete color of the caret.
func (t *Text) SetCaretColor(clr color.Color) {
	t.caretColor = clr
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
	op.Style.HorizontalAlign = t.hAlign
	op.Style.VerticalAlign = t.vAlign
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
	textutil.Draw(textBounds, dst, str, op)
}

// VisualLineCountOfLogicalLine returns the number of visual lines the
// lineIndex-th logical line occupies when wrapped at wrapWidth. lineIndex must
// be in [0, [Text.LineCount]).
func (t *Text) VisualLineCountOfLogicalLine(context *guigui.Context, lineIndex, wrapWidth int) int {
	t.ensureLineByteOffsets()
	n := t.lineByteOffsets.LineCount()
	start := t.lineByteOffsets.ByteOffsetByLineIndex(lineIndex)
	end := t.store.TextLengthInBytes()
	if lineIndex+1 < n {
		end = t.lineByteOffsets.ByteOffsetByLineIndex(lineIndex + 1)
	}
	line := t.stringValueWithRange(start, end)
	return textutil.CachedVisualLineCount(wrapWidth, line, t.wrapMode, t.face(context, false), t.actualTabWidth(context), t.keepTailingSpace)
}

// MaxCaretXOfLogicalLine returns the maximum caret X coordinate over the
// visual lines of the lineIndex-th logical line when wrapped at wrapWidth.
// lineIndex must be in [0, [Text.LineCount]).
func (t *Text) MaxCaretXOfLogicalLine(context *guigui.Context, lineIndex, wrapWidth int) float64 {
	t.ensureLineByteOffsets()
	n := t.lineByteOffsets.LineCount()
	start := t.lineByteOffsets.ByteOffsetByLineIndex(lineIndex)
	end := t.store.TextLengthInBytes()
	if lineIndex+1 < n {
		end = t.lineByteOffsets.ByteOffsetByLineIndex(lineIndex + 1)
	}
	line := t.stringValueWithRange(start, end)
	return textutil.CachedVisualLineMaxCaretX(wrapWidth, line, t.wrapMode, t.face(context, false), t.actualTabWidth(context), t.keepTailingSpace)
}
