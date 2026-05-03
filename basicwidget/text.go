// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"bytes"
	"image"
	"image/color"
	"io"
	"log/slog"
	"math"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/zeebo/xxh3"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
	"github.com/guigui-gui/guigui/internal/clipboard"
)

type HorizontalAlign int

const (
	HorizontalAlignStart  HorizontalAlign = HorizontalAlign(textutil.HorizontalAlignStart)
	HorizontalAlignCenter HorizontalAlign = HorizontalAlign(textutil.HorizontalAlignCenter)
	HorizontalAlignEnd    HorizontalAlign = HorizontalAlign(textutil.HorizontalAlignEnd)
	HorizontalAlignLeft   HorizontalAlign = HorizontalAlign(textutil.HorizontalAlignLeft)
	HorizontalAlignRight  HorizontalAlign = HorizontalAlign(textutil.HorizontalAlignRight)
)

type VerticalAlign int

const (
	VerticalAlignTop    VerticalAlign = VerticalAlign(textutil.VerticalAlignTop)
	VerticalAlignMiddle VerticalAlign = VerticalAlign(textutil.VerticalAlignMiddle)
	VerticalAlignBottom VerticalAlign = VerticalAlign(textutil.VerticalAlignBottom)
)

var (
	textEventValueChanged            guigui.EventKey = guigui.GenerateEventKey()
	textEventValueChangedWithoutText guigui.EventKey = guigui.GenerateEventKey()
	textEventScrollDelta             guigui.EventKey = guigui.GenerateEventKey()
)

func isMouseButtonRepeating(button ebiten.MouseButton) bool {
	if !ebiten.IsMouseButtonPressed(button) {
		return false
	}
	return repeat(inpututil.MouseButtonPressDuration(button))
}

func isKeyRepeating(key ebiten.Key) bool {
	if !ebiten.IsKeyPressed(key) {
		return false
	}
	return repeat(inpututil.KeyPressDuration(key))
}

func repeat(duration int) bool {
	// duration can be 0 e.g. when pressing Ctrl+A on macOS.
	// A release event might be sent too quickly after the press event.
	if duration <= 1 {
		return true
	}
	delay := ebiten.TPS() * 2 / 5
	if duration < delay {
		return false
	}
	return (duration-delay)%4 == 0
}

type Text struct {
	guigui.DefaultWidget

	field             textinput.Field
	valueBuilder      bytes.Buffer
	valueEqualChecker stringEqualChecker

	nextTextSet   bool
	nextText      string
	nextSelectAll bool
	textInited    bool

	hAlign        HorizontalAlign
	vAlign        VerticalAlign
	color         color.Color
	semanticColor basicwidgetdraw.SemanticColor
	transparent   float64
	locales       []language.Tag
	scaleMinus1   float64
	bold          bool
	tabular       bool
	tabWidth      float64

	selectable                  bool
	editable                    bool
	multiline                   bool
	autoWrap                    bool
	cursorStatic                bool
	keepTailingSpace            bool
	selectionVisibleWhenUnfocus bool
	ellipsisString              string

	selectionDragStartPlus1 int
	selectionDragEndPlus1   int

	// selectionShiftIndexPlus1 is the index (+1) of the selection that is moved by Shift and arrow keys.
	selectionShiftIndexPlus1 int

	dragging bool

	clickCount         int
	lastClickTick      int64
	lastClickTextIndex int

	cursor textCursor

	tmpClipboard string

	cachedTextWidths      [4][4]cachedTextWidthEntry
	cachedTextHeights     [4][4]cachedTextHeightEntry
	cachedDefaultTabWidth float64
	lastFaceCacheKey      faceCacheKey
	lastScale             float64

	// contentHasher is a reusable xxh3 streaming hasher used by [Text.WriteStateKey]
	// to fingerprint the current field contents without allocating a string.
	contentHasher xxh3.Hasher128

	// contentHashCache memoizes the most recently computed hash, keyed by
	// [textinput.Field.Generation]. While the field has not been mutated, repeated
	// [Text.WriteStateKey] calls return the cached value without re-hashing.
	contentHashCache           xxh3.Uint128
	contentHashFieldGeneration int64

	// lineByteOffsets holds the byte offsets where each logical line begins
	// in the field's committed text. Used by virtualized layout paths that
	// need to walk a window of logical lines without rescanning the whole
	// buffer. Refreshed lazily by ensureLineByteOffsets when
	// [textinput.Field.Generation] advances past
	// lineByteOffsetsFieldGeneration.
	lineByteOffsets                textutil.LineByteOffsets
	lineByteOffsetsFieldGeneration int64

	// cachedLocalesString is a comparable fingerprint of t.locales, refreshed
	// only at [Text.SetLocales] (which has a slices.Equal guard). Included in
	// [Text.WriteStateKey] so locale changes trigger automatic rebuilds without
	// an explicit [guigui.RequestRebuild] call.
	cachedLocalesString string

	drawOptions textutil.DrawOptions

	prevStart              int
	prevEnd                int
	paddingForScrollOffset guigui.Padding

	onFocusChanged      func(context *guigui.Context, focused bool)
	onHandleButtonInput func(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult

	// lastDispatchedUncommittedGen is the [textinput.Field.Generation] value
	// at the most recent uncommitted dispatch. Used to suppress redundant
	// uncommitted dispatches (e.g., IME state replays that don't modify the
	// text).
	lastDispatchedUncommittedGen int64

	// lastDispatchedCommittedGen is the [textinput.Field.Generation] value
	// at the most recent committed dispatch. setText also advances it as if
	// it had dispatched, treating the programmatic value as a self-committed
	// snapshot. Committed dispatches are suppressed when the field hasn't
	// advanced past this point — that filters focus-loss commits on unchanged
	// buffers without swallowing the legitimate commit that follows a user
	// edit at the same generation as the prior uncommitted dispatch.
	lastDispatchedCommittedGen int64

	// firstLogicalLineInViewport is the logical-line index that sits
	// at widget-local Y=0 — the first line drawn at the top of the
	// widget. The default zero value means "line 0 at the top,"
	// matching the historical line-0-anchored coordinate system.
	// Virtualizing parents (e.g. [textInputText.Layout]) set it to
	// the panel's topItemIndex so the inner Y math is taken relative
	// to the visible region, avoiding an O(topIdx) cumulative-Y walk
	// on every Layout. Set via [Text.setFirstLogicalLineInViewport];
	// consumers (Draw, hit-testing, cursor positioning) will read it
	// incrementally as the anchored coordinate system rolls in across
	// phases.
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

type textSizeCacheKey int

func newTextSizeCacheKey(autoWrap, bold bool) textSizeCacheKey {
	var key textSizeCacheKey
	if autoWrap {
		key |= 1 << 0
	}
	if bold {
		key |= 1 << 1
	}
	return key
}

// OnValueChanged sets the event handler that is called when the text value changes.
// The handler receives the current text and whether the change is committed.
// A committed change occurs when the user presses Enter (for single-line text) or when the text input loses focus.
// An uncommitted change occurs on every keystroke or text modification during editing.
//
// The handler fires only when the text content actually advances. Input
// activity that doesn't modify the text — cursor moves, focus changes, IME
// state replays, redundant commit gestures — does not trigger the handler.
//
// If the handler does not need the text payload, prefer
// [Text.OnValueChangedWithoutText] to avoid materializing the value on every
// change.
func (t *Text) OnValueChanged(f func(context *guigui.Context, text string, committed bool)) {
	guigui.SetEventHandler(t, textEventValueChanged, f)
}

// OnValueChangedWithoutText sets a handler that fires under the same conditions
// as [Text.OnValueChanged] but is not given the current text. Use this when the
// handler only needs to know that the value changed (e.g. to mark a document
// dirty) so the underlying value is not materialized into a string on every
// change.
//
// The handler can be registered alongside [Text.OnValueChanged]; both fire on
// the same change.
func (t *Text) OnValueChangedWithoutText(f func(context *guigui.Context, committed bool)) {
	guigui.SetEventHandler(t, textEventValueChangedWithoutText, f)
}

// dispatchValueChanged dispatches a value-changed event, suppressing it when
// the field's generation hasn't moved past the relevant tracker. Uncommitted
// dispatches are gated on lastDispatchedUncommittedGen (so IME state replays
// at the same generation are filtered); committed dispatches are gated on
// lastDispatchedCommittedGen (so focus-loss commits on unchanged buffers are
// filtered, while still firing the commit that follows a real edit).
//
// snapshot only applies when committed is false. It marks the new value as a
// self-committed programmatic snapshot — lastDispatchedCommittedGen is
// advanced alongside the uncommitted tracker so a subsequent focus-loss
// commit at the same generation stays suppressed.
func (t *Text) dispatchValueChanged(committed, snapshot bool) {
	gen := t.field.Generation()
	if committed {
		if gen == t.lastDispatchedCommittedGen {
			return
		}
		t.lastDispatchedCommittedGen = gen
	} else {
		if gen == t.lastDispatchedUncommittedGen {
			return
		}
		t.lastDispatchedUncommittedGen = gen
		if snapshot {
			t.lastDispatchedCommittedGen = gen
		}
	}
	guigui.DispatchEventLazy(t, textEventValueChanged, func() (string, bool) {
		return t.stringValue(), committed
	})
	guigui.DispatchEvent(t, textEventValueChangedWithoutText, committed)
}

func (t *Text) OnHandleButtonInput(f func(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult) {
	t.onHandleButtonInput = f
}

func (t *Text) onScrollDelta(f func(context *guigui.Context, deltaX, deltaY float64)) {
	guigui.SetEventHandler(t, textEventScrollDelta, f)
}

// contentHashForStateKey returns a 128-bit fingerprint of the current field
// contents, including the active IME composition (matching what [Text.Draw]
// and [Text.Measure] see).
func (t *Text) contentHashForStateKey() xxh3.Uint128 {
	generation := t.field.Generation()
	if generation == t.contentHashFieldGeneration {
		return t.contentHashCache
	}
	t.contentHasher.Reset()
	_, _ = t.field.WriteTextForRenderingTo(&t.contentHasher)
	t.contentHashCache = t.contentHasher.Sum128()
	t.contentHashFieldGeneration = generation
	return t.contentHashCache
}

// ensureLineByteOffsets refreshes t.lineByteOffsets if the field has been
// mutated since the last call. The offsets are built from the committed text
// only (no IME composition), matching what [textinput.Field.WriteTextTo]
// returns.
func (t *Text) ensureLineByteOffsets() {
	generation := t.field.Generation()
	if t.lineByteOffsets.LineCount() > 0 && generation == t.lineByteOffsetsFieldGeneration {
		return
	}
	_ = t.lineByteOffsets.Rebuild(func(w io.Writer) error {
		_, err := t.field.WriteTextTo(w)
		return err
	})
	t.lineByteOffsetsFieldGeneration = generation
}

func (t *Text) WriteStateKey(w *guigui.StateKeyWriter) {
	w.WriteUint64(uint64(t.hAlign))
	w.WriteUint64(uint64(t.vAlign))
	hasColor := t.color != nil
	w.WriteBool(hasColor)
	if hasColor {
		r, g, b, a := t.color.RGBA()
		writeRGBA64(w, color.RGBA64{R: uint16(r), G: uint16(g), B: uint16(b), A: uint16(a)})
	}
	w.WriteUint64(uint64(t.semanticColor))
	w.WriteFloat64(t.transparent)
	w.WriteFloat64(t.scaleMinus1)
	w.WriteBool(t.bold)
	w.WriteBool(t.tabular)
	w.WriteFloat64(t.tabWidth)
	w.WriteBool(t.selectable)
	w.WriteBool(t.editable)
	w.WriteBool(t.multiline)
	w.WriteBool(t.autoWrap)
	w.WriteBool(t.cursorStatic)
	w.WriteBool(t.keepTailingSpace)
	w.WriteBool(t.selectionVisibleWhenUnfocus)
	w.WriteString(t.ellipsisString)
	writePadding(w, t.paddingForScrollOffset)
	selStart, selEnd := t.field.Selection()
	w.WriteInt(selStart)
	w.WriteInt(selEnd)
	w.WriteBool(t.field.IsFocused())
	w.WriteString(t.cachedLocalesString)
	ch := t.contentHashForStateKey()
	w.WriteUint64(ch.Lo)
	w.WriteUint64(ch.Hi)
}

func (t *Text) resetCachedTextSize() {
	clear(t.cachedTextWidths[:])
	clear(t.cachedTextHeights[:])
	t.cachedDefaultTabWidth = 0
}

func (t *Text) canHaveCursor() bool {
	return t.selectable || t.editable
}

func (t *Text) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if t.canHaveCursor() {
		adder.AddWidget(&t.cursor)
	}

	if key := t.faceCacheKey(context, false); t.lastFaceCacheKey != key {
		t.lastFaceCacheKey = key
		t.resetCachedTextSize()
	}
	if t.lastScale != context.Scale() {
		t.lastScale = context.Scale()
		t.resetCachedTextSize()
	}

	context.SetPassthrough(&t.cursor, true)

	if t.selectable || t.editable {
		t.cursor.text = t
	}

	if t.onFocusChanged == nil {
		t.onFocusChanged = func(context *guigui.Context, focused bool) {
			if !t.editable {
				return
			}
			if focused {
				t.field.Focus()
				t.cursor.resetCounter()
				start, end := t.field.Selection()
				if start < 0 || end < 0 {
					t.doSelectAll()
				}
			} else {
				t.commit()
			}
		}
	}
	guigui.OnFocusChanged(t, t.onFocusChanged)

	return nil
}

func (t *Text) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	if t.canHaveCursor() {
		b := widgetBounds.Bounds()
		t.cursor.setTextWidgetBounds(b)
		layouter.LayoutWidget(&t.cursor, t.cursorBounds(context, b))
	}
}

func (t *Text) SetSelectable(selectable bool) {
	if t.selectable == selectable {
		return
	}
	t.selectable = selectable
	t.selectionDragStartPlus1 = 0
	t.selectionDragEndPlus1 = 0
	t.selectionShiftIndexPlus1 = 0
	if !t.selectable {
		t.setSelection(0, 0, -1, false)
	}
}

func (t *Text) isEqualToStringValue(text string) bool {
	t.valueEqualChecker.Reset(text)
	_, _ = t.field.WriteTextTo(&t.valueEqualChecker)
	return t.valueEqualChecker.Result()
}

// stringValue returns the field's committed text. The remaining callers
// — value-changed event dispatch, [Text.Value], and the rare fallback
// path of [Text.textToDraw] — fire infrequently enough that the per-
// tick cache the function used to maintain is no longer worth its
// fields; per-tick consumers all read narrower ranges via
// [Text.stringValueWithRange].
func (t *Text) stringValue() string {
	t.valueBuilder.Reset()
	_, _ = t.field.WriteTextTo(&t.valueBuilder)
	return t.valueBuilder.String()
}

func (t *Text) stringValueWithRange(start, end int) string {
	if end < 0 {
		end = t.field.TextLengthInBytes()
	}
	t.valueBuilder.Reset()
	_, _ = t.field.WriteTextRangeTo(&t.valueBuilder, start, end)
	return t.valueBuilder.String()
}

func (t *Text) bytesValueWithRange(start, end int) []byte {
	if end < 0 {
		end = t.field.TextLengthInBytes()
	}
	t.valueBuilder.Reset()
	_, _ = t.field.WriteTextRangeTo(&t.valueBuilder, start, end)
	return t.valueBuilder.Bytes()
}

// stringValueForRenderingRange returns the bytes of the rendering text
// (committed text with the active IME composition spliced in) in
// [start, end). Coordinates are in rendering space; clamped by
// [textinput.Field.WriteTextForRenderingRangeTo].
func (t *Text) stringValueForRenderingRange(start, end int) string {
	t.valueBuilder.Reset()
	_, _ = t.field.WriteTextForRenderingRangeTo(&t.valueBuilder, start, end)
	return t.valueBuilder.String()
}

// stringValueForLineContaining returns the bytes of the logical line that
// contains byteOffset (clamped to the document) along with the line's
// starting byte offset, suitable for translating local↔global byte
// positions. It is used by per-cursor textutil calls (word-boundary
// lookup, grapheme stepping) so they can scan the relevant logical line
// without materializing the whole document.
func (t *Text) stringValueForLineContaining(byteOffset int) (line string, lineStart int) {
	t.ensureLineByteOffsets()
	lineIdx := t.lineByteOffsets.LineIndexForByteOffset(byteOffset)
	lineStart = t.lineByteOffsets.ByteOffsetByLineIndex(lineIdx)
	lineEnd := t.field.TextLengthInBytes()
	if lineIdx+1 < t.lineByteOffsets.LineCount() {
		lineEnd = t.lineByteOffsets.ByteOffsetByLineIndex(lineIdx + 1)
	}
	return t.stringValueWithRange(lineStart, lineEnd), lineStart
}

// LineCount returns the number of logical lines in the value. A logical
// line is a span between hard line breaks; soft-wrapped visual lines are
// not counted. The empty value has one logical line; a trailing line break
// creates an extra empty line at the end, so "abc\n" has 2 lines.
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
// containing textIndexInBytes. textIndexInBytes is clamped: negative values
// map to line 0, values past the end map to the last line.
//
// See [Text.LineCount] for what counts as a logical line.
func (t *Text) LineIndexFromTextIndexInBytes(textIndexInBytes int) int {
	t.ensureLineByteOffsets()
	return t.lineByteOffsets.LineIndexForByteOffset(textIndexInBytes)
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
	if position >= t.field.TextLengthInBytes() {
		return position
	}
	line, lineStart := t.stringValueForLineContaining(position)
	return lineStart + textutil.NextPositionOnGraphemes(line, position-lineStart)
}

func (t *Text) stringValueForRendering() string {
	t.valueBuilder.Reset()
	_, _ = t.field.WriteTextForRenderingTo(&t.valueBuilder)
	return t.valueBuilder.String()
}

// Value returns the current value as a string.
// For large values, prefer [Text.WriteValueTo] to avoid allocating a copy.
func (t *Text) Value() string {
	if t.nextTextSet {
		return t.nextText
	}
	return t.stringValue()
}

// HasValue reports whether the text has a non-empty value.
// This is more efficient than checking Value() != "" as it avoids
// allocating a string.
func (t *Text) HasValue() bool {
	if t.nextTextSet {
		return t.nextText != ""
	}
	return t.hasValueInField()
}

func (t *Text) hasValueInField() bool {
	return t.field.HasText()
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
// written. It is the streaming counterpart of [Text.Value] and avoids
// materializing the full value as a string.
func (t *Text) WriteValueTo(w io.Writer) (int64, error) {
	if t.nextTextSet {
		n, err := io.WriteString(w, t.nextText)
		return int64(n), err
	}
	return t.field.WriteTextTo(w)
}

// WriteValueRangeTo writes the bytes of the current value in
// [startInBytes, endInBytes) to w. startInBytes and endInBytes are clamped
// to [0, len(value)]. If the clamped start is not less than the clamped end,
// nothing is written.
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
	return t.field.WriteTextRangeTo(w, startInBytes, endInBytes)
}

// ReadValueFrom resets the value to the bytes read from r until EOF and
// returns the number of bytes read. It is the streaming counterpart of
// [Text.ForceSetValue]: the change is applied immediately, the undo history
// is cleared, and the selection is reset to (0, 0).
//
// If r returns a non-EOF error, the value is reset to empty and the error
// is returned.
func (t *Text) ReadValueFrom(r io.Reader) (int64, error) {
	n, err := t.field.ReadTextFrom(r)
	t.selectionShiftIndexPlus1 = 0
	t.prevStart = 0
	t.prevEnd = 0
	t.nextText = ""
	t.nextTextSet = false
	t.textInited = true
	t.resetCachedTextSize()
	t.dispatchValueChanged(false, true)
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

func (t *Text) selectAll() {
	if t.nextTextSet {
		t.nextSelectAll = true
		return
	}
	t.doSelectAll()
}

func (t *Text) doSelectAll() {
	t.setSelection(0, t.field.TextLengthInBytes(), -1, false)
}

func (t *Text) Selection() (start, end int) {
	return t.field.Selection()
}

func (t *Text) SetSelection(start, end int) {
	t.setSelection(start, end, -1, true)
}

func (t *Text) setSelection(start, end int, shiftIndex int, adjustScroll bool) bool {
	t.selectionShiftIndexPlus1 = shiftIndex + 1
	if start > end {
		start, end = end, start
	}

	if s, e := t.field.Selection(); s == start && e == end {
		return false
	}
	t.field.SetSelection(start, end)

	if !adjustScroll {
		t.prevStart = start
		t.prevEnd = end
	}

	return true
}

func (t *Text) replaceTextAtSelection(text string) {
	start, end := t.field.Selection()
	t.replaceTextAt(text, start, end)
}

func (t *Text) replaceTextAt(text string, start, end int) {
	if !t.multiline {
		text, start, end = replaceNewLinesWithSpace(text, start, end)
	}

	t.selectionShiftIndexPlus1 = 0
	if start > end {
		start, end = end, start
	}
	if s, e := t.field.Selection(); text == t.stringValueWithRange(start, end) && s == start && e == end {
		return
	}
	t.field.ReplaceText(text, start, end)

	t.resetCachedTextSize()
	t.dispatchValueChanged(false, false)

	t.nextText = ""
	t.nextTextSet = false
}

func (t *Text) setText(text string, selectAll bool) bool {
	if !t.multiline {
		text, _, _ = replaceNewLinesWithSpace(text, 0, 0)
	}

	t.selectionShiftIndexPlus1 = 0

	textChanged := !t.isEqualToStringValue(text)
	if s, e := t.field.Selection(); !textChanged && (!selectAll || s == 0 && e == len(text)) {
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
			t.field.SetTextAndSelection(text, start, end)
		} else {
			// Reset the text so that the undo history's first item is the initial text.
			t.field.ResetText(text)
			t.field.SetSelection(start, end)
		}
		t.resetCachedTextSize()
		t.dispatchValueChanged(false, true)
	} else {
		t.field.SetSelection(0, len(text))
	}

	// Do not adjust scroll.
	t.prevStart = start
	t.prevEnd = end
	t.nextText = ""
	t.nextTextSet = false
	t.textInited = true

	return true
}

func (t *Text) SetLocales(locales []language.Tag) {
	if slices.Equal(t.locales, locales) {
		return
	}

	t.locales = append([]language.Tag(nil), locales...)
	var sb strings.Builder
	for i, tag := range t.locales {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(tag.String())
	}
	t.cachedLocalesString = sb.String()
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
	t.cachedDefaultTabWidth = text.Advance("        ", face)
	return t.cachedDefaultTabWidth
}

func (t *Text) scale() float64 {
	return t.scaleMinus1 + 1
}

func (t *Text) SetScale(scale float64) {
	t.scaleMinus1 = scale - 1
}

func (t *Text) HorizontalAlign() HorizontalAlign {
	return t.hAlign
}

func (t *Text) SetHorizontalAlign(align HorizontalAlign) {
	t.hAlign = align
}

func (t *Text) VerticalAlign() VerticalAlign {
	return t.vAlign
}

func (t *Text) SetVerticalAlign(align VerticalAlign) {
	t.vAlign = align
}

func (t *Text) SetColor(color color.Color) {
	t.color = color
}

func (t *Text) SetSemanticColor(semanticColor basicwidgetdraw.SemanticColor) {
	t.semanticColor = semanticColor
}

func (t *Text) SetOpacity(opacity float64) {
	t.transparent = 1 - opacity
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
		t.selectionShiftIndexPlus1 = 0
	} else if t.field.IsFocused() {
		// Blur immediately so Ebitengine's BeforeUpdate hook stops auto-committing
		// pending input into the field before HandlePointingInput runs next tick.
		t.field.Blur()
	}
	t.editable = editable
}

func (t *Text) IsMultiline() bool {
	return t.multiline
}

func (t *Text) SetMultiline(multiline bool) {
	t.multiline = multiline
}

func (t *Text) SetAutoWrap(autoWrap bool) {
	t.autoWrap = autoWrap
}

// SetCursorBlinking sets whether the cursor blinks.
// The default value is true.
func (t *Text) SetCursorBlinking(cursorBlinking bool) {
	t.cursorStatic = !cursorBlinking
}

// SetSelectionVisibleWhenUnfocused sets whether the selection range stays
// drawn while the widget is not focused. By default the selection is hidden
// when the widget loses focus. Enable this when a separate UI (e.g. a Find
// dialog) holds focus but the user still needs to see what was matched.
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

func (t *Text) setKeepTailingSpace(keep bool) {
	t.keepTailingSpace = keep
}

// setFirstLogicalLineInViewport tells the widget which logical line
// should sit at widget-local Y=0 — i.e., the first line drawn at the
// top of the widget. Default 0 means "line 0 at the top," the
// historical behavior; virtualizing parents set this to the topmost
// visible logical line so the widget can position itself without a
// per-line cumulative-Y walk. Subsequent phases plug this value into
// [Text.restrictedTextToDraw], hit testing, and cursor positioning;
// for now writing it has no observable effect.
func (t *Text) setFirstLogicalLineInViewport(idx int) {
	t.firstLogicalLineInViewport = max(idx, 0)
}

func (t *Text) textContentBounds(context *guigui.Context, bounds image.Rectangle) image.Rectangle {
	b := bounds
	h := t.textHeight(context, guigui.FixedWidthConstraints(b.Dx()))

	switch t.vAlign {
	case VerticalAlignTop:
		b.Max.Y = b.Min.Y + h
	case VerticalAlignMiddle:
		dy := b.Dy()
		b.Min.Y += (dy - h) / 2
		b.Max.Y = b.Min.Y + h
	case VerticalAlignBottom:
		b.Min.Y = b.Max.Y - h
	}

	return b
}

// contentBoundsForLayout returns the bounds for laying out content.
func (t *Text) contentBoundsForLayout(context *guigui.Context, bounds image.Rectangle) image.Rectangle {
	if t.vAlign == VerticalAlignTop {
		// For Top, [Text.textContentBounds] would only tighten Max.Y, which
		// no caller depends on beyond it staying within bounds. Skip it to
		// avoid [Text.textHeight], which walks every logical line for autoWrap.
		return bounds
	}
	return t.textContentBounds(context, bounds)
}

func (t *Text) faceCacheKey(context *guigui.Context, forceBold bool) faceCacheKey {
	size := FontSize(context) * (t.scaleMinus1 + 1)
	weight := text.WeightMedium
	if t.bold || forceBold {
		weight = text.WeightBold
	}

	liga := !t.selectable && !t.editable
	tnum := t.tabular

	var lang language.Tag
	if len(t.locales) > 0 {
		lang = t.locales[0]
	} else {
		lang = context.FirstLocale()
	}
	return faceCacheKey{
		size:   size,
		weight: weight,
		liga:   liga,
		tnum:   tnum,
		lang:   lang,
	}
}

// face must be called after [Text.Build], as it relies on lastFaceCacheKey being set.
func (t *Text) face(context *guigui.Context, forceBold bool) text.Face {
	key := t.lastFaceCacheKey
	if forceBold {
		key.weight = text.WeightBold
	}
	return fontFace(context, key)
}

func (t *Text) lineHeight(context *guigui.Context) float64 {
	return float64(LineHeight(context)) * (t.scaleMinus1 + 1)
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
			if t.setSelection(start, end, -1, true) {
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
		if t.field.IsFocused() {
			t.field.Blur()
		}
		return guigui.HandleInputResult{}
	}
	// The field auto-commits text input via Ebitengine's BeforeUpdate hook whenever
	// it is focused, so only focus it when this widget actually accepts edits.
	if t.editable {
		t.field.Focus()
	} else if t.field.IsFocused() {
		t.field.Blur()
	}

	if !t.editable && !t.selectable {
		return guigui.HandleInputResult{}
	}

	return guigui.HandleInputResult{}
}

func (t *Text) handleClick(context *guigui.Context, textBounds image.Rectangle, cursorPosition image.Point, leftClick bool) {
	idx := t.textIndexFromPosition(context, textBounds, cursorPosition, false)

	if leftClick {
		if ebiten.Tick()-t.lastClickTick < int64(doubleClickLimitInTicks()) && t.lastClickTextIndex == idx {
			t.clickCount++
		} else {
			t.clickCount = 1
		}
	} else {
		t.clickCount = 1
	}

	switch t.clickCount {
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
		if leftClick || !context.IsFocusedOrHasFocusedChild(t) {
			if start, end := t.field.Selection(); start != idx || end != idx {
				t.setSelection(idx, idx, -1, false)
			}
		}
	case 2:
		t.dragging = true
		start, end := t.findWordBoundaries(idx)
		t.selectionDragStartPlus1 = start + 1
		t.selectionDragEndPlus1 = end + 1
		t.setSelection(start, end, -1, false)
	case 3:
		t.doSelectAll()
	}

	context.SetFocused(t, true)

	t.lastClickTick = ebiten.Tick()
	t.lastClickTextIndex = idx
}

func (t *Text) textToDraw(context *guigui.Context, showComposition bool) string {
	if showComposition && t.field.UncommittedTextLengthInBytes() > 0 {
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
// directly from the field via [Text.stringValueWithRange], so the
// per-frame allocation is bounded by the visible window rather than the
// document length.
func (t *Text) restrictedTextToDraw(context *guigui.Context, textBounds, visibleBounds image.Rectangle) (txt string, byteStart int, yShift int, restricted bool) {
	t.ensureLineByteOffsets()
	n := t.lineByteOffsets.LineCount()

	hasComp := t.field.UncommittedTextLengthInBytes() > 0
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

	width := textBounds.Dx()

	var compInfo textutil.CompositionInfo
	if hasComp {
		sStart, sEnd := t.field.Selection()
		compLen := t.field.UncommittedTextLengthInBytes()
		byteDelta := compLen - (sEnd - sStart)

		selectionLineIdx := t.lineByteOffsets.LineIndexForByteOffset(sStart)
		cs := t.lineByteOffsets.ByteOffsetByLineIndex(selectionLineIdx)
		ce := t.field.TextLengthInBytes()
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
		if t.autoWrap && t.lineByteOffsets.LineIndexForByteOffset(sEnd) == selectionLineIdx {
			committedSelectionLine = t.stringValueWithRange(cs, ce)
			renderingSelectionLine = t.stringValueForRenderingRange(cs, ce+byteDelta)
		}

		info, ok := textutil.ComputeCompositionInfo(&textutil.CompositionInfoParams{
			CompositionText:        t.stringValueForRenderingRange(sStart, sStart+compLen),
			LineByteOffsets:        &t.lineByteOffsets,
			SelectionStart:         sStart,
			SelectionEnd:           sEnd,
			AutoWrap:               t.autoWrap,
			CommittedSelectionLine: committedSelectionLine,
			RenderingSelectionLine: renderingSelectionLine,
			Face:                   t.face(context, false),
			LineHeight:             t.lineHeight(context),
			TabWidth:               t.actualTabWidth(context),
			KeepTailingSpace:       t.keepTailingSpace,
			WrapWidth:              width,
		})
		if !ok {
			return materializeFull(), 0, 0, false
		}
		compInfo = info
	}

	lineH := int(math.Ceil(t.lineHeight(context)))
	if lineH <= 0 {
		return materializeFull(), 0, 0, false
	}

	renderingLength := t.field.TextLengthInBytes()
	if hasComp {
		sStart, sEnd := t.field.Selection()
		renderingLength = renderingLength - (sEnd - sStart) + t.field.UncommittedTextLengthInBytes()
	}

	// vAlign==Top: the walker starts at firstLogicalLineInViewport
	// (the line that textInputText.Layout pinned at widget-local Y=0)
	// and measures only lines from there downward. Other alignments
	// need a totalHeight-based shift; the branch below computes that
	// and walks from line 0.
	if t.vAlign == VerticalAlignTop {
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
			LineHeight:       t.lineHeight(context),
			TabWidth:         t.actualTabWidth(context),
			KeepTailingSpace: t.keepTailingSpace,
			AutoWrap:         t.autoWrap,
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
	case VerticalAlignMiddle:
		alignOffset = (textBounds.Dy() - totalHeight) / 2
	case VerticalAlignBottom:
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
		LineHeight:       t.lineHeight(context),
		TabWidth:         t.actualTabWidth(context),
		KeepTailingSpace: t.keepTailingSpace,
		AutoWrap:         t.autoWrap,
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
	s, e := t.field.Selection()
	if !t.editable {
		return s, e, true
	}
	if !context.IsFocused(t) {
		return s, e, true
	}
	cs, ce, ok := t.field.CompositionSelection()
	if !ok {
		return s, e, true
	}
	// When cs == ce, the composition already started but any conversion is not done yet.
	// In this case, put the cursor at the end of the composition.
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
	s, _ := t.field.Selection()
	cs, ce, ok := t.field.CompositionSelection()
	if !ok {
		return 0, 0, 0, 0, false
	}
	// When cs == ce, the composition already started but any conversion is not done yet.
	// In this case, assume the entire region is the composition.
	// TODO: This behavior might be macOS specific. Investigate this.
	l := t.field.UncommittedTextLengthInBytes()
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
		start, _ := t.field.Selection()
		var processed bool
		if pos, ok := t.textPosition(context, widgetBounds.Bounds(), start, false); ok {
			t.field.SetBounds(image.Rect(int(pos.X), int(pos.Top), int(pos.X+1), int(pos.Bottom)))
			processed = t.field.Handled()
		}
		if processed {
			// Reset the cache size before adjust the scroll offset in order to get the correct text size.
			t.resetCachedTextSize()
			t.dispatchValueChanged(false, false)
			return guigui.HandleInputByWidget(t)
		}

		// Do not accept key inputs when compositing.
		if _, _, ok := t.field.CompositionSelection(); ok {
			return guigui.HandleInputByWidget(t)
		}

		// For Windows key binds, see:
		// https://support.microsoft.com/en-us/windows/keyboard-shortcuts-in-windows-dcc61a57-8ff0-cffe-9796-cb9706c75eec#textediting

		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			if t.multiline {
				t.replaceTextAtSelection("\n")
			} else {
				t.commit()
			}
			return guigui.HandleInputByWidget(t)
		case isKeyRepeating(ebiten.KeyBackspace) ||
			isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyH):
			start, end := t.field.Selection()
			if start != end {
				t.replaceTextAtSelection("")
			} else if start > 0 {
				pos := t.prevPositionOnGraphemes(start)
				t.replaceTextAt("", pos, start)
			}
			return guigui.HandleInputByWidget(t)
		case !isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyD) ||
			isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyD):
			// Delete
			start, end := t.field.Selection()
			if start != end {
				t.replaceTextAtSelection("")
			} else if isDarwin() && end < t.field.TextLengthInBytes() {
				pos := t.nextPositionOnGraphemes(end)
				t.replaceTextAt("", start, pos)
			}
			return guigui.HandleInputByWidget(t)
		case isKeyRepeating(ebiten.KeyDelete):
			// Delete one cluster
			if _, end := t.field.Selection(); end < t.field.TextLengthInBytes() {
				pos := t.nextPositionOnGraphemes(end)
				t.replaceTextAt("", start, pos)
			}
			return guigui.HandleInputByWidget(t)
		case !isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyX) ||
			isDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && isKeyRepeating(ebiten.KeyX):
			t.Cut()
			return guigui.HandleInputByWidget(t)
		case !isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyV) ||
			isDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && isKeyRepeating(ebiten.KeyV):
			t.Paste()
			return guigui.HandleInputByWidget(t)
		case !isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyY) ||
			isDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && ebiten.IsKeyPressed(ebiten.KeyShift) && isKeyRepeating(ebiten.KeyZ):
			t.Redo()
			return guigui.HandleInputByWidget(t)
		case !isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyZ) ||
			isDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && isKeyRepeating(ebiten.KeyZ):
			t.Undo()
			return guigui.HandleInputByWidget(t)
		}
	}

	switch {
	case ebiten.IsKeyPressed(ebiten.KeyControl) && ebiten.IsKeyPressed(ebiten.KeyShift) && isKeyRepeating(ebiten.KeyLeft):
		idx := 0
		start, end := t.field.Selection()
		if i, l := textutil.LastLineBreakPositionAndLen(t.stringValueWithRange(0, start)); i >= 0 {
			idx = i + l
		}
		t.setSelection(idx, end, idx, true)
		return guigui.HandleInputByWidget(t)
	case ebiten.IsKeyPressed(ebiten.KeyControl) && ebiten.IsKeyPressed(ebiten.KeyShift) && isKeyRepeating(ebiten.KeyRight):
		idx := t.field.TextLengthInBytes()
		start, end := t.field.Selection()
		if i, _ := textutil.FirstLineBreakPositionAndLen(t.stringValueWithRange(end, -1)); i >= 0 {
			idx = end + i
		}
		t.setSelection(start, idx, idx, true)
		return guigui.HandleInputByWidget(t)
	case isKeyRepeating(ebiten.KeyLeft) ||
		isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyB):
		start, end := t.field.Selection()
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			if t.selectionShiftIndexPlus1-1 == end {
				pos := t.prevPositionOnGraphemes(end)
				t.setSelection(start, pos, pos, true)
			} else {
				pos := t.prevPositionOnGraphemes(start)
				t.setSelection(pos, end, pos, true)
			}
		} else {
			if start != end {
				t.setSelection(start, start, -1, true)
			} else if start > 0 {
				pos := t.prevPositionOnGraphemes(start)
				t.setSelection(pos, pos, -1, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case isKeyRepeating(ebiten.KeyRight) ||
		isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyF):
		start, end := t.field.Selection()
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			if t.selectionShiftIndexPlus1-1 == start {
				pos := t.nextPositionOnGraphemes(start)
				t.setSelection(pos, end, pos, true)
			} else {
				pos := t.nextPositionOnGraphemes(end)
				t.setSelection(start, pos, pos, true)
			}
		} else {
			if start != end {
				t.setSelection(end, end, -1, true)
			} else if start < t.field.TextLengthInBytes() {
				pos := t.nextPositionOnGraphemes(start)
				t.setSelection(pos, pos, -1, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case isKeyRepeating(ebiten.KeyUp) ||
		isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyP):
		lh := t.lineHeight(context)
		shift := ebiten.IsKeyPressed(ebiten.KeyShift)
		var moveEnd bool
		start, end := t.field.Selection()
		idx := start
		if shift && t.selectionShiftIndexPlus1-1 == end {
			idx = end
			moveEnd = true
		}
		if pos, ok := t.textPosition(context, widgetBounds.Bounds(), idx, false); ok {
			y := (pos.Top+pos.Bottom)/2 - lh
			idx := t.textIndexFromPosition(context, widgetBounds.Bounds(), image.Pt(int(pos.X), int(y)), false)
			if shift {
				if moveEnd {
					t.setSelection(start, idx, idx, true)
				} else {
					t.setSelection(idx, end, idx, true)
				}
			} else {
				t.setSelection(idx, idx, -1, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case isKeyRepeating(ebiten.KeyDown) ||
		isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyN):
		lh := t.lineHeight(context)
		shift := ebiten.IsKeyPressed(ebiten.KeyShift)
		var moveStart bool
		start, end := t.field.Selection()
		idx := end
		if shift && t.selectionShiftIndexPlus1-1 == start {
			idx = start
			moveStart = true
		}
		if pos, ok := t.textPosition(context, widgetBounds.Bounds(), idx, false); ok {
			y := (pos.Top+pos.Bottom)/2 + lh
			idx := t.textIndexFromPosition(context, widgetBounds.Bounds(), image.Pt(int(pos.X), int(y)), false)
			if shift {
				if moveStart {
					t.setSelection(idx, end, idx, true)
				} else {
					t.setSelection(start, idx, idx, true)
				}
			} else {
				t.setSelection(idx, idx, -1, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyA):
		idx := 0
		start, end := t.field.Selection()
		if i, l := textutil.LastLineBreakPositionAndLen(t.stringValueWithRange(0, start)); i >= 0 {
			idx = i + l
		}
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			t.setSelection(idx, end, idx, true)
		} else {
			t.setSelection(idx, idx, -1, true)
		}
		return guigui.HandleInputByWidget(t)
	case isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyE):
		idx := t.field.TextLengthInBytes()
		start, end := t.field.Selection()
		if i, _ := textutil.FirstLineBreakPositionAndLen(t.stringValueWithRange(end, -1)); i >= 0 {
			idx = end + i
		}
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			t.setSelection(start, idx, idx, true)
		} else {
			t.setSelection(idx, idx, -1, true)
		}
		return guigui.HandleInputByWidget(t)
	case !isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyA) ||
		isDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && isKeyRepeating(ebiten.KeyA):
		t.doSelectAll()
		return guigui.HandleInputByWidget(t)
	case !isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyC) ||
		isDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && isKeyRepeating(ebiten.KeyC):
		// Copy
		t.Copy()
		return guigui.HandleInputByWidget(t)
	case isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyK):
		// 'Kill' the text after the cursor or the selection.
		start, end := t.field.Selection()
		if start == end {
			i, l := textutil.FirstLineBreakPositionAndLen(t.stringValueWithRange(start, -1))
			if i < 0 {
				end = t.field.TextLengthInBytes()
			} else if i == 0 {
				end = start + l
			} else {
				end = start + i
			}
		}
		t.tmpClipboard = t.stringValueWithRange(start, end)
		t.replaceTextAt("", start, end)
		return guigui.HandleInputByWidget(t)
	case isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyY):
		// 'Yank' the killed text.
		if t.tmpClipboard != "" {
			t.replaceTextAtSelection(t.tmpClipboard)
		}
		return guigui.HandleInputByWidget(t)
	}

	return guigui.HandleInputResult{}
}

func (t *Text) commit() {
	t.dispatchValueChanged(true, false)
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

	// Adjust the scroll offset for cases not covered by HandleButtonInput,
	// such as continuous scrolling during drag selection.
	// TODO: The cursor position might be unstable when the text horizontal align is center or right. Fix this.
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

	var textColor color.Color
	if t.color != nil {
		textColor = t.color
	} else if t.semanticColor != basicwidgetdraw.SemanticColorBase {
		textColor = basicwidgetdraw.TextColorFromSemanticColor(context.ColorMode(), t.semanticColor)
	} else {
		textColor = basicwidgetdraw.TextColor(context.ColorMode(), context.IsEnabled(t))
	}
	if t.transparent > 0 {
		textColor = draw.ScaleAlpha(textColor, 1-t.transparent)
	}
	face := t.face(context, false)
	op := &t.drawOptions
	op.Options.AutoWrap = t.autoWrap
	op.Options.Face = face
	op.Options.LineHeight = t.lineHeight(context)
	op.Options.HorizontalAlign = textutil.HorizontalAlign(t.hAlign)
	op.Options.VerticalAlign = textutil.VerticalAlign(t.vAlign)
	op.Options.TabWidth = t.actualTabWidth(context)
	op.Options.KeepTailingSpace = t.keepTailingSpace
	if !t.editable {
		op.Options.EllipsisString = t.ellipsisString
	} else {
		op.Options.EllipsisString = ""
	}
	op.TextColor = textColor
	op.VisibleBounds = widgetBounds.VisibleBounds()
	if start, end, ok := t.selectionToDraw(context); ok {
		if context.IsFocused(t) || (t.selectionVisibleWhenUnfocus && start != end) {
			op.DrawSelection = true
			op.SelectionStart = start
			op.SelectionEnd = end
			op.SelectionColor = basicwidgetdraw.TextSelectionColor(context.ColorMode())
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
		op.InactiveCompositionColor = basicwidgetdraw.TextInactiveCompositionColor(context.ColorMode())
		op.ActiveCompositionColor = basicwidgetdraw.TextActiveCompositionColor(context.ColorMode())
		op.CompositionBorderWidth = float32(textCursorWidth(context))
	} else {
		op.DrawComposition = false
	}

	txt, byteStart, yShift, restricted := t.restrictedTextToDraw(context, textBounds, widgetBounds.VisibleBounds())
	if restricted {
		textBounds.Min.Y += yShift
		// yShift already includes the alignment-specific portion of the
		// textPositionYOffset the inner Draw would have computed; force
		// vAlign=Top so it only adds paddingY rather than re-centering /
		// re-bottom-aligning the restricted text inside the bounds.
		op.Options.VerticalAlign = textutil.VerticalAlignTop
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

func (t *Text) boldTextSize(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.textSize(context, constraints, true)
}

// textHeight returns the height of the rendered text under the given
// constraints, without computing the width. Skipping width avoids per-line
// shaping, which dominates the cost for very long text.
func (t *Text) textHeight(context *guigui.Context, constraints guigui.Constraints) int {
	constraintWidth := math.MaxInt
	if w, ok := constraints.FixedWidth(); ok {
		constraintWidth = w
	}
	if constraintWidth == 0 {
		constraintWidth = 1
	}

	bold := t.bold
	key := newTextSizeCacheKey(t.autoWrap, bold)

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

	lineH := t.lineHeight(context)
	var hi int
	if visualCount, ok := t.totalRenderingVisualLineCount(context, constraintWidth, bold); ok {
		hi = int(math.Ceil(lineH * float64(visualCount)))
	} else {
		// Fallback when an active composition contains a hard line break
		// or straddles a logical-line boundary — the rendering text's
		// logical-line shape doesn't match the committed sidecar.
		txt := t.textToDraw(context, true)
		h := textutil.MeasureHeight(constraintWidth, txt, t.autoWrap, t.face(context, bold), lineH, t.actualTabWidth(context), t.keepTailingSpace)
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
// For autoWrap, walks logical lines summing per-line wrap counts via
// [textutil.VisualLineCountForLogicalLine]; reads each line through
// the per-range field methods (committed bytes for unaffected lines,
// rendering bytes for the composition's selection line) so no full-
// document materialization is needed.
func (t *Text) totalRenderingVisualLineCount(context *guigui.Context, width int, bold bool) (int, bool) {
	t.ensureLineByteOffsets()
	n := t.lineByteOffsets.LineCount()

	hasComp := t.field.UncommittedTextLengthInBytes() > 0
	var sStart, sEnd, compLen, byteDelta int
	selectionLineIdx := -1
	if hasComp {
		sStart, sEnd = t.field.Selection()
		compLen = t.field.UncommittedTextLengthInBytes()
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

	// Non-autoWrap: each logical line is one visual line; composition
	// can't change that (single-line composition keeps the line count).
	if !t.autoWrap {
		return n, true
	}

	// AutoWrap: walk logical lines summing per-line wrap counts.
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
	totalLen := t.field.TextLengthInBytes()
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
		count += textutil.VisualLineCountForLogicalLine(measureWidth, line, true, face, tabW, keepTailing)
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

	hasComp := t.field.UncommittedTextLengthInBytes() > 0
	var sStart, sEnd, compLen, byteDelta int
	selectionLineIdx := -1
	if hasComp {
		sStart, sEnd = t.field.Selection()
		compLen = t.field.UncommittedTextLengthInBytes()
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

	lineH := t.lineHeight(context)
	face := t.face(context, bold)
	tabW := t.actualTabWidth(context)
	keepTailing := t.keepTailingSpace
	measureWidth := width
	if measureWidth <= 0 {
		measureWidth = math.MaxInt
	}
	totalLen := t.field.TextLengthInBytes()

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
		w, h := textutil.MeasureLogicalLine(measureWidth, line, t.autoWrap, face, lineH, tabW, keepTailing, ellipsisString)
		maxWidth = max(maxWidth, w)
		height += h
	}
	return maxWidth, height, true
}

func (t *Text) textSize(context *guigui.Context, constraints guigui.Constraints, forceBold bool) image.Point {
	constraintWidth := math.MaxInt
	if w, ok := constraints.FixedWidth(); ok {
		constraintWidth = w
	}
	if constraintWidth == 0 {
		constraintWidth = 1
	}

	bold := t.bold || forceBold
	key := newTextSizeCacheKey(t.autoWrap, bold)

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
		w, h = textutil.Measure(constraintWidth, txt, t.autoWrap, t.face(context, bold), t.lineHeight(context), t.actualTabWidth(context), t.keepTailingSpace, ellipsisString)
	}
	// If width is 0, the text's bounds and visible bounds are empty, and nothing including its cursor is rendered.
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

func (t *Text) cursorPosition(context *guigui.Context, textBounds image.Rectangle) (position textutil.TextPosition, ok bool) {
	if !context.IsFocused(t) {
		return textutil.TextPosition{}, false
	}
	if !t.editable {
		return textutil.TextPosition{}, false
	}
	start, end := t.field.Selection()
	if start < 0 {
		return textutil.TextPosition{}, false
	}
	if end < 0 {
		return textutil.TextPosition{}, false
	}
	// A non-empty selection draws as a highlight, not a caret;
	// [textCursor.alpha] returns 0 in that case, so no callers need
	// the position.
	if start != end {
		return textutil.TextPosition{}, false
	}

	// Skip the textPosition walk when the cursor's line is off-screen;
	// it can dominate CPU when the user has scrolled far from the cursor.
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
	lh := t.lineHeight(context)
	minTop := float64(textBounds.Min.Y) + lh*float64(line-first)
	if minTop >= float64(textBounds.Max.Y) {
		return false
	}
	return true
}

func (t *Text) textIndexFromPosition(context *guigui.Context, textBounds image.Rectangle, position image.Point, showComposition bool) int {
	textContentBounds := t.contentBoundsForLayout(context, textBounds)

	// Compute the rendering text's byte length without materializing
	// it. RenderingTextLength = committedLength + composition byte delta
	// when composition is active and visible; otherwise == committedLength.
	renderingLength := t.field.TextLengthInBytes()
	var sStart, sEnd, compLen int
	if showComposition {
		compLen = t.field.UncommittedTextLengthInBytes()
		if compLen > 0 {
			sStart, sEnd = t.field.Selection()
			renderingLength = renderingLength + compLen - (sEnd - sStart)
		}
	}

	width := textContentBounds.Dx()
	op := &textutil.Options{
		AutoWrap:         t.autoWrap,
		Face:             t.face(context, false),
		LineHeight:       t.lineHeight(context),
		HorizontalAlign:  textutil.HorizontalAlign(t.hAlign),
		VerticalAlign:    textutil.VerticalAlign(t.vAlign),
		TabWidth:         t.actualTabWidth(context),
		KeepTailingSpace: t.keepTailingSpace,
	}
	position = position.Sub(textContentBounds.Min)

	// Pass the firstLogicalLineInViewport as the textutil walk hint.
	// Virtualizing parents (textInputText.Layout) set this to the
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
	idx := textutil.TextIndexFromPosition(&textutil.TextIndexFromPositionParams{
		Position:             position,
		RenderingTextRange:   readRendering,
		RenderingTextLength:  renderingLength,
		Width:                width,
		Options:              op,
		CommittedTextRange:   readCommitted,
		LineByteOffsets:      &t.lineByteOffsets,
		SelectionStart:       sStart,
		SelectionEnd:         sEnd,
		CompositionLen:       compLen,
		LogicalLineIndexHint: hintLL,
	})
	if idx < 0 || idx > renderingLength {
		return -1
	}
	return idx
}

func (t *Text) textPosition(context *guigui.Context, bounds image.Rectangle, index int, showComposition bool) (position textutil.TextPosition, ok bool) {
	textBounds := t.contentBoundsForLayout(context, bounds)
	width := textBounds.Dx()
	op := &textutil.Options{
		AutoWrap:         t.autoWrap,
		Face:             t.face(context, false),
		LineHeight:       t.lineHeight(context),
		HorizontalAlign:  textutil.HorizontalAlign(t.hAlign),
		VerticalAlign:    textutil.VerticalAlign(t.vAlign),
		TabWidth:         t.actualTabWidth(context),
		KeepTailingSpace: t.keepTailingSpace,
	}

	// Pass the cached lineByteOffsets sidecar and the
	// firstLogicalLineInViewport hint so
	// [textutil.TextPositionFromIndex] walks only the logical lines
	// between the viewport's first line and the cursor's line. The
	// sidecar-less fallback walks every visual line in the document;
	// for multi-megabyte buffers cursorPosition / adjustScrollOffset
	// call this every tick and that fallback dominates CPU.
	t.ensureLineByteOffsets()

	renderingLength := t.field.TextLengthInBytes()
	var sStart, sEnd, compLen int
	if showComposition {
		compLen = t.field.UncommittedTextLengthInBytes()
		if compLen > 0 {
			sStart, sEnd = t.field.Selection()
			renderingLength = renderingLength + compLen - (sEnd - sStart)
		}
	}
	readRendering := func(start, end int) string { return t.stringValueWithRange(start, end) }
	if showComposition {
		readRendering = func(start, end int) string { return t.stringValueForRenderingRange(start, end) }
	}
	var readCommitted func(start, end int) string
	if compLen > 0 {
		readCommitted = func(start, end int) string { return t.stringValueWithRange(start, end) }
	}
	// firstLogicalLineInViewport pins TextPositionFromIndex's Y origin
	// to the line at widget-local Y=0 (the line that
	// textInputText.Layout positioned at the panel viewport top); the
	// returned pos.Top is therefore relative to that line, ready to
	// add to textBounds.Min.Y for screen coordinates. The walk is
	// bounded by the logical-line distance between firstLine and the
	// cursor's line, which is a viewport's worth of lines for cursors
	// visible on screen.
	pos0, pos1, count := textutil.TextPositionFromIndex(&textutil.TextPositionFromIndexParams{
		Index:                index,
		RenderingTextRange:   readRendering,
		RenderingTextLength:  renderingLength,
		Width:                width,
		Options:              op,
		CommittedTextRange:   readCommitted,
		LineByteOffsets:      &t.lineByteOffsets,
		SelectionStart:       sStart,
		SelectionEnd:         sEnd,
		CompositionLen:       compLen,
		LogicalLineIndexHint: t.firstLogicalLineInViewport,
	})
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

func textCursorWidth(context *guigui.Context) int {
	return int(math.Ceil(2 * context.Scale()))
}

func (t *Text) cursorBounds(context *guigui.Context, textBounds image.Rectangle) image.Rectangle {
	pos, ok := t.cursorPosition(context, textBounds)
	if !ok {
		return image.Rectangle{}
	}
	w := textCursorWidth(context)
	paddingTop := 2 * t.scale() * context.Scale()
	paddingBottom := 1 * t.scale() * context.Scale()
	return image.Rect(int(pos.X)-w/2, int(pos.Top+paddingTop), int(pos.X)+w/2, int(pos.Bottom-paddingBottom))
}

func (t *Text) setPaddingForScrollOffset(padding guigui.Padding) {
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

	cx, cy := ebiten.CursorPosition()
	if pos, ok := t.textPosition(context, textBounds, end, true); ok {
		var deltaX, deltaY float64
		if t.dragging {
			deltaX = float64(textVisibleBounds.Max.X) - float64(cx) - float64(t.paddingForScrollOffset.End)
			deltaY = float64(textVisibleBounds.Max.Y) - float64(cy) - float64(t.paddingForScrollOffset.Bottom)
			if cx > textVisibleBounds.Max.X {
				deltaX /= 4
			} else {
				deltaX = 0
			}
			if cy > textVisibleBounds.Max.Y {
				deltaY /= 4
			} else {
				deltaY = 0
			}
		} else {
			deltaX = float64(textVisibleBounds.Max.X) - pos.X - float64(t.paddingForScrollOffset.End)
			deltaY = float64(textVisibleBounds.Max.Y) - pos.Bottom - float64(t.paddingForScrollOffset.Bottom)
		}
		deltaX = min(deltaX, 0)
		deltaY = min(deltaY, 0)
		dx += deltaX
		dy += deltaY
	}
	if pos, ok := t.textPosition(context, textBounds, start, true); ok {
		var deltaX, deltaY float64
		if t.dragging {
			deltaX = float64(textVisibleBounds.Min.X) - float64(cx) + float64(t.paddingForScrollOffset.Start)
			deltaY = float64(textVisibleBounds.Min.Y) - float64(cy) + float64(t.paddingForScrollOffset.Top)
			if cx < textVisibleBounds.Min.X {
				deltaX /= 4
			} else {
				deltaX = 0
			}
			if cy < textVisibleBounds.Min.Y {
				deltaY /= 4
			} else {
				deltaY = 0
			}
		} else {
			deltaX = float64(textVisibleBounds.Min.X) - pos.X + float64(t.paddingForScrollOffset.Start)
			deltaY = float64(textVisibleBounds.Min.Y) - pos.Top + float64(t.paddingForScrollOffset.Top)
		}
		deltaX = max(deltaX, 0)
		deltaY = max(deltaY, 0)
		dx += deltaX
		dy += deltaY
	}
	return dx, dy
}

func (t *Text) CanCut() bool {
	if !t.editable {
		return false
	}
	start, end := t.field.Selection()
	return start != end
}

func (t *Text) CanCopy() bool {
	start, end := t.field.Selection()
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
	return t.field.CanUndo()
}

func (t *Text) CanRedo() bool {
	if !t.editable {
		return false
	}
	return t.field.CanRedo()
}

func (t *Text) Cut() bool {
	start, end := t.field.Selection()
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
	start, end := t.field.Selection()
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
	if !t.field.CanUndo() {
		return false
	}
	t.field.Undo()
	t.resetCachedTextSize()
	return true
}

func (t *Text) Redo() bool {
	if !t.field.CanRedo() {
		return false
	}
	t.field.Redo()
	t.resetCachedTextSize()
	return true
}

type textCursor struct {
	guigui.DefaultWidget

	text *Text

	// textWidgetBoundsRect is the parent Text's bounds rectangle, captured by
	// [Text.Layout]. cursorPosition is resolved against this rectangle, not the
	// cursor's own caret-sized widgetBounds.
	//
	// The value is invalid and unavailable during the Build phase, as it is only
	// populated once [Text.Layout] runs.
	textWidgetBoundsRect image.Rectangle

	counter   int
	prevAlpha float64
	prevPos   textutil.TextPosition
	prevOK    bool
}

func (t *textCursor) setTextWidgetBounds(rect image.Rectangle) {
	t.textWidgetBoundsRect = rect
}

func (t *textCursor) resetCounter() {
	t.counter = 0
}

func (t *textCursor) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	pos, ok := t.text.cursorPosition(context, t.textWidgetBoundsRect)
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

func (t *textCursor) alpha(context *guigui.Context) float64 {
	if _, ok := t.text.cursorPosition(context, t.textWidgetBoundsRect); !ok {
		return 0
	}
	s, e, ok := t.text.selectionToDraw(context)
	if !ok {
		return 0
	}
	if s != e {
		return 0
	}
	if t.text.cursorStatic {
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

func (t *textCursor) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	alpha := t.alpha(context)
	if alpha == 0 {
		return
	}
	b := widgetBounds.Bounds()
	clr := draw.ScaleAlpha(draw.Color2(context.ColorMode(), draw.SemanticColorAccent, 0.5, 0.6), alpha)
	basicwidgetdraw.DrawRoundedRect(context, dst, b, clr, b.Dx()/2)
}

func replaceNewLinesWithSpace(text string, start, end int) (string, int, int) {
	var buf strings.Builder
	for {
		pos, len := textutil.FirstLineBreakPositionAndLen(text)
		if len == 0 {
			buf.WriteString(text)
			break
		}
		buf.WriteString(text[:pos])
		origLen := buf.Len()
		buf.WriteString(" ")
		if diff := len - 1; diff > 0 {
			if origLen < start {
				if start >= origLen+len {
					start -= diff
				} else {
					// This is a very rare case, e.g. the position is in between '\r' and '\n'.
					start = origLen + 1
				}
			}
			if origLen < end {
				if end >= origLen+len {
					end -= diff
				} else {
					end = origLen + 1
				}
			}
		}
		text = text[pos+len:]
	}
	text = buf.String()

	return text, start, end
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
