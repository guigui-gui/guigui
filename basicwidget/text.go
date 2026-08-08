// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"
	"io"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
	"github.com/guigui-gui/guigui/basicwidget/internal/textwidget"
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

// WrapMode selects how visual lines wrap when text exceeds the available width.
type WrapMode int

const (
	// WrapModeNone disables automatic wrapping; content extends past the
	// available width and only the explicit hard line breaks in the source
	// text introduce new visual lines.
	WrapModeNone WrapMode = WrapMode(textutil.WrapModeNone)

	// WrapModeNormal wraps at Unicode line break opportunities. These
	// coincide with word boundaries in most cases, but not always.
	WrapModeNormal WrapMode = WrapMode(textutil.WrapModeNormal)

	// WrapModeAnywhere wraps at any grapheme cluster boundary, breaking
	// inside words when needed to fit the available width.
	WrapModeAnywhere WrapMode = WrapMode(textutil.WrapModeAnywhere)
)

// ItemTextStyle bundles the styling attributes applied to the fallback text
// rendered when a widget's Content is nil.
type ItemTextStyle struct {
	Color           color.Color
	HorizontalAlign HorizontalAlign
	VerticalAlign   VerticalAlign
	Bold            bool
	Tabular         bool
	WrapMode        WrapMode
}

func (s ItemTextStyle) writeStateKey(w *guigui.StateKeyWriter) {
	writeColor(w, s.Color)
	w.WriteUint64(uint64(s.HorizontalAlign))
	w.WriteUint64(uint64(s.VerticalAlign))
	w.WriteBool(s.Bold)
	w.WriteBool(s.Tabular)
	w.WriteUint64(uint64(s.WrapMode))
}

// Text is a widget that shows or edits a text value. It wraps the theme-free
// [textwidget.Text] core with theme policy: semantic colors, font family
// resolution, locales, and the placeholder.
type Text struct {
	guigui.DefaultWidget

	core textwidget.Text

	// baseStyle holds the base style set with [Text.SetBaseStyle]. The font
	// family and the text color are resolved against the theme at
	// [Text.Build] and handed to the core widget there.
	baseStyle textstyle.Style

	semanticColor draw.SemanticColor
	transparent   float64
	locales       []language.Tag

	// hotspotRangesBuf is scratch for converting the hotspot ranges handed
	// to the core widget.
	hotspotRangesBuf []textwidget.TextRange

	// cachedLocalesString is a comparable fingerprint of t.locales, refreshed
	// only at [Text.SetLocales] (which has a slices.Equal guard). Included in
	// [Text.WriteStateKey] so locale changes trigger automatic rebuilds without
	// an explicit [guigui.RequestRebuild] call.
	cachedLocalesString string

	// placeholder is drawn in a subdued color when the value is empty and no
	// IME composition is active. An empty string disables it.
	placeholder string

	// fontSize is the font size at scale 1. A non-positive value selects the
	// default font size ([FontSize]).
	fontSize float64

	// lineHeight is the line height at scale 1. A non-positive value selects
	// the default line height ([LineHeight]).
	lineHeight float64
}

// OnValueChanged sets the event handler that is called when the text value changes.
// The handler receives the current text and whether the change is committed.
// A committed change is a finalized value: the user presses Enter (for single-line
// text), the text input loses focus, or the value is set programmatically via
// [Text.ForceSetValue] / [Text.ReadValueFrom] (and equivalent paths on wrapping
// widgets). An uncommitted change is a mid-flight edit such as IME composition.
//
// Pressing Enter on single-line text always dispatches a committed change,
// even when the value equals the last committed value. Every other path fires
// only when the text content actually advances: input activity that doesn't
// modify the text — caret moves, focus changes, IME state replays, a focus-loss
// commit on an unchanged buffer — does not trigger the handler.
//
// An undo or redo also dispatches an uncommitted change when it restores the
// ranged style overrides alone, with the text itself unchanged. The handler
// can read the current overrides back via [Text.ReadOverrideStyles].
//
// If the handler does not need the text payload, prefer
// [Text.OnValueChangedWithoutText] to avoid materializing the value on every
// change.
func (t *Text) OnValueChanged(f func(context *guigui.Context, text string, committed bool)) {
	t.core.OnValueChanged(f)
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
	t.core.OnValueChangedWithoutText(f)
}

func (t *Text) OnHandleButtonInput(f func(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult) {
	t.core.OnHandleButtonInput(f)
}

func (t *Text) WriteStateKey(context *guigui.Context, w *guigui.StateKeyWriter) {
	var style TextStyle
	t.ReadBaseStyle(&style)
	style.writeStateKey(w)
	w.WriteUint64(uint64(t.semanticColor))
	w.WriteFloat64(t.transparent)
	w.WriteString(t.placeholder)
	w.WriteString(t.cachedLocalesString)
	w.WriteFloat64(t.fontSize)
	w.WriteFloat64(t.lineHeight)
}

func (t *Text) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.core)
	context.DelegateFocus(t, &t.core)

	family, _ := t.baseStyle.Family()
	t.core.SetFontFamily(family)
	fontSize := FontSize(context)
	if t.fontSize > 0 {
		fontSize = t.fontSize
	}
	t.core.SetFontSize(fontSize)
	lineHeight := float64(LineHeight(context))
	if t.lineHeight > 0 {
		lineHeight = t.lineHeight
	}
	t.core.SetLineHeight(lineHeight)

	var lang language.Tag
	if len(t.locales) > 0 {
		lang = t.locales[0]
	} else {
		lang = context.FirstLocale()
	}
	t.core.SetLang(lang)

	t.core.SetTextColor(t.resolveTextColor(context))
	t.core.SetSelectionColor(draw.TextSelectionColor(context.ColorMode()))
	t.core.SetCompositionColors(
		draw.TextInactiveCompositionColor(context.ColorMode()),
		draw.TextActiveCompositionColor(context.ColorMode()),
	)
	t.core.SetCaretColor(draw.TextCaretColor(context.ColorMode()))

	return nil
}

// resolveTextColor returns the concrete color the value is drawn in: the
// explicitly set color, or the theme color for the semantic color and the
// enabled state, with the opacity applied.
func (t *Text) resolveTextColor(context *guigui.Context) color.Color {
	var clr color.Color
	switch baseColor, _ := t.baseStyle.Color(); {
	case baseColor != nil:
		clr = baseColor
	case t.semanticColor != draw.SemanticColorBase:
		clr = draw.TextColorFromSemanticColor(context.ColorMode(), t.semanticColor)
	default:
		clr = draw.TextColor(context.ColorMode(), context.IsEnabled(t))
	}
	if t.transparent > 0 {
		clr = draw.ScaleAlpha(clr, 1-t.transparent)
	}
	return clr
}

func (t *Text) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&t.core, widgetBounds.Bounds())
}

func (t *Text) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	// The placeholder is drawn beneath the core, which renders nothing while
	// the value is visually empty.
	if t.placeholder == "" || !t.core.IsEditable() || !t.core.IsVisuallyEmpty() {
		return
	}
	clr := draw.TextColor(context.ColorMode(), false)
	if t.transparent > 0 {
		clr = draw.ScaleAlpha(clr, 1-t.transparent)
	}
	t.core.DrawPlainString(context, widgetBounds, dst, t.placeholder, clr)
}

func (t *Text) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.core.Measure(context, constraints)
}

func (t *Text) boldTextSize(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.core.MeasureBold(context, constraints)
}

// isFocused reports whether the text holds the focus, which lands on its core
// widget via focus delegation.
func (t *Text) isFocused(context *guigui.Context) bool {
	return context.IsFocused(&t.core)
}

func (t *Text) SetSelectable(selectable bool) {
	t.core.SetSelectable(selectable)
}

// LineCount returns the number of logical lines in the value. A logical
// line is a span between hard line breaks; soft-wrapped visual lines are
// not counted. The empty value has one logical line; a trailing line break
// creates an extra empty line at the end, so "abc\n" has 2 lines.
func (t *Text) LineCount() int {
	return t.core.LineCount()
}

// LineStartInBytes returns the byte offset where the lineIndex-th logical
// line begins within the value. lineIndex must be in [0, [Text.LineCount]).
func (t *Text) LineStartInBytes(lineIndex int) int {
	return t.core.LineStartInBytes(lineIndex)
}

// LineIndexFromTextIndexInBytes returns the index of the logical line
// containing textIndexInBytes. textIndexInBytes is clamped: negative values
// map to line 0, values past the end map to the last line.
//
// See [Text.LineCount] for what counts as a logical line.
func (t *Text) LineIndexFromTextIndexInBytes(textIndexInBytes int) int {
	return t.core.LineIndexFromTextIndexInBytes(textIndexInBytes)
}

// CaretPositionAtTextIndexInBytes returns the on-screen top and bottom
// endpoints of a caret drawn at byte offset textIndexInBytes in the text
// value. ok is false when textIndexInBytes is out of range, or when the
// caret's logical line is outside the current viewport.
//
// CaretPositionAtTextIndexInBytes is available after the layout phase.
func (t *Text) CaretPositionAtTextIndexInBytes(context *guigui.Context, textIndexInBytes int) (top, bottom image.Point, ok bool) {
	return t.core.CaretPositionAtTextIndexInBytes(context, textIndexInBytes)
}

// Value returns the current value as a string.
// For large values, prefer [Text.WriteValueTo] to avoid allocating a copy.
func (t *Text) Value() string {
	return t.core.Value()
}

// HasValue reports whether the text has a non-empty value.
// This is more efficient than checking Value() != "" as it avoids
// allocating a string.
func (t *Text) HasValue() bool {
	return t.core.HasValue()
}

func (t *Text) SetValue(text string) {
	t.core.SetValue(text)
}

func (t *Text) ForceSetValue(text string) {
	t.core.ForceSetValue(text)
}

// WriteValueTo writes the current value to w and returns the number of bytes
// written. It is the streaming counterpart of [Text.Value] and avoids
// materializing the full value as a string.
func (t *Text) WriteValueTo(w io.Writer) (int64, error) {
	return t.core.WriteValueTo(w)
}

// WriteValueRangeTo writes the bytes of the current value in
// [startInBytes, endInBytes) to w. startInBytes and endInBytes are clamped
// to [0, len(value)]. If the clamped start is not less than the clamped end,
// nothing is written.
func (t *Text) WriteValueRangeTo(w io.Writer, startInBytes, endInBytes int) (int64, error) {
	return t.core.WriteValueRangeTo(w, startInBytes, endInBytes)
}

// ReadValueFrom resets the value to the bytes read from r until EOF and
// returns the number of bytes read. It is the streaming counterpart of
// [Text.ForceSetValue]: the change is applied immediately, the undo history
// is cleared, and the selection is reset to (0, 0).
//
// If r returns a non-EOF error, the value is reset to empty and the error
// is returned.
func (t *Text) ReadValueFrom(r io.Reader) (int64, error) {
	return t.core.ReadValueFrom(r)
}

func (t *Text) ReplaceValueAtSelection(text string) {
	t.core.ReplaceValueAtSelection(text)
}

func (t *Text) CommitWithCurrentInputValue() {
	t.core.CommitWithCurrentInputValue()
}

func (t *Text) Selection() (start, end int) {
	return t.core.Selection()
}

func (t *Text) SetSelection(start, end int) {
	t.core.SetSelection(start, end)
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

// SetFontSize sets the font size at scale 1. A non-positive size selects the
// default font size ([FontSize]). Unlike the default, the specified size is
// not multiplied by [guigui.Context.Scale], so it can express a size
// independent of the app scale; multiply it explicitly when it should follow
// the app scale.
func (t *Text) SetFontSize(size float64) {
	t.fontSize = size
}

// SetLineHeight sets the line height at scale 1. A non-positive height
// selects the default line height ([LineHeight]). Unlike the default, the
// specified height is not multiplied by [guigui.Context.Scale], so it can
// express a height independent of the app scale; multiply it explicitly when
// it should follow the app scale.
func (t *Text) SetLineHeight(lineHeight float64) {
	t.lineHeight = lineHeight
}

func (t *Text) SetTabWidth(tabWidth float64) {
	t.core.SetTabWidth(tabWidth)
}

func (t *Text) SetScale(scale float64) {
	t.core.SetScale(scale)
}

func (t *Text) HorizontalAlign() HorizontalAlign {
	return HorizontalAlign(t.core.HorizontalAlign())
}

func (t *Text) SetHorizontalAlign(align HorizontalAlign) {
	t.core.SetHorizontalAlign(textutil.HorizontalAlign(align))
}

func (t *Text) VerticalAlign() VerticalAlign {
	return VerticalAlign(t.core.VerticalAlign())
}

func (t *Text) SetVerticalAlign(align VerticalAlign) {
	t.core.SetVerticalAlign(textutil.VerticalAlign(align))
}

func (t *Text) SetSemanticColor(semanticColor basicwidgetdraw.SemanticColor) {
	t.semanticColor = draw.SemanticColor(semanticColor)
}

func (t *Text) SetOpacity(opacity float64) {
	t.transparent = 1 - opacity
}

// ReadBaseStyle replaces style with the base style: the style properties
// applied to the whole value, underneath the ranged style overrides. A
// property is unset when the theme default applies.
func (t *Text) ReadBaseStyle(style *TextStyle) {
	style.s = t.baseStyle
}

// SetBaseStyle replaces the base style with style. Unset properties restore
// the theme defaults. The base style holds the font family, the italic face
// selection, OpenType variations and features (including the weight), and
// the text color; the scale and the language are ignored, as they come from
// [Text.SetScale] and [Text.SetLocales].
func (t *Text) SetBaseStyle(style *TextStyle) {
	t.baseStyle = style.s.WithoutScale().WithoutLang()
	t.core.SetBaseStyle(t.baseStyle)
}

// ReadOverrideStyles replaces styles with a copy of the ranged style
// overrides, reflecting the adjustments made for edits since the overrides
// were set.
func (t *Text) ReadOverrideStyles(styles *TextStyles) {
	t.core.ReadOverrideStyleRuns(&styles.runs)
}

// SetOverrideStyles replaces the ranged style overrides with styles.
// recorded sets whether the replacement is recorded in the undo history; a
// recorded replacement leaving the overrides unchanged records nothing.
// Pass false when restoring the overrides from a model, such as on every
// build.
func (t *Text) SetOverrideStyles(styles *TextStyles, recorded bool) {
	t.core.CopyOverrideStyleRunsFrom(&styles.runs, recorded)
}

// ReadOverrideStylesInRange replaces styles with a copy of the ranged style
// overrides in [startInBytes, endInBytes), rebased so that startInBytes maps
// to 0.
func (t *Text) ReadOverrideStylesInRange(styles *TextStyles, startInBytes, endInBytes int) {
	t.core.ReadOverrideStyleRunsInRange(&styles.runs, startInBytes, endInBytes)
}

// SetOverrideStylesInRange replaces the ranged style overrides in
// [startInBytes, endInBytes) with styles' overrides in
// [0, endInBytes-startInBytes), shifted so that 0 maps to startInBytes.
// styles' overrides outside [0, endInBytes-startInBytes) are ignored.
// recorded sets whether the replacement is recorded in the undo history; a
// recorded replacement leaving the overrides unchanged records nothing.
func (t *Text) SetOverrideStylesInRange(styles *TextStyles, startInBytes, endInBytes int, recorded bool) {
	t.core.ReplaceOverrideStyleRunsInRange(&styles.runs, startInBytes, endInBytes, recorded)
}

// ReadEffectiveStyles replaces styles with the effective styles of the whole
// value. The effective style of a byte is the base style and the rendering
// defaults with the byte's ranged overrides merged on top, so every property
// is set.
func (t *Text) ReadEffectiveStyles(styles *TextStyles) {
	t.core.ReadEffectiveStyleRuns(&styles.runs)
}

// ReadEffectiveStylesInRange replaces styles with the effective styles of
// [startInBytes, endInBytes), rebased so that startInBytes maps to 0. The
// effective style of a byte is the base style and the rendering defaults
// with the byte's ranged overrides merged on top, so every property is set.
func (t *Text) ReadEffectiveStylesInRange(styles *TextStyles, startInBytes, endInBytes int) {
	t.core.ReadEffectiveStyleRunsInRange(&styles.runs, startInBytes, endInBytes)
}

// ReadEffectiveStyleAt replaces style with the effective style that text
// typed at textIndexInBytes adopts: the base style and the rendering
// defaults with the overrides adopted from the byte right before the index
// merged on top, so every property is set.
func (t *Text) ReadEffectiveStyleAt(style *TextStyle, textIndexInBytes int) {
	style.s = t.core.EffectiveStyleAt(textIndexInBytes)
}

// SetInsertionStyle replaces the insertion style with style. Its set
// properties are applied as ranged style overrides over the next text
// inserted at the caret, on top of the adopted overrides, and the style is
// then reset. The widget also resets it without applying on other
// interactions, such as a selection change, a deletion, or an undo; neither
// setting nor resetting is recorded in the undo history. SetInsertionStyle
// is typically called on every build with application-owned state, updated
// in the [Text.OnInsertionStyleReset] handler so the next build sets the
// cleared style.
func (t *Text) SetInsertionStyle(style *TextStyle) {
	t.core.SetInsertionStyle(style.s)
}

// OnInsertionStyleReset sets an event handler invoked when the widget resets
// the insertion style: after applying it to inserted text, or when
// discarding it without applying, such as on a selection change. The handler
// is not invoked for [Text.SetInsertionStyle].
func (t *Text) OnInsertionStyleReset(f func(context *guigui.Context)) {
	t.core.OnInsertionStyleReset(f)
}

// AppendBoundsOfTextRange appends the bounding rectangles covering the text
// range [startInBytes, endInBytes) to dst and returns the extended slice,
// one rectangle per crossed visual line, in order. bounds is the widget's
// bounds; the rectangles are in the same coordinate space. A masked value
// appends nothing. Endpoints outside the text are clamped.
func (t *Text) AppendBoundsOfTextRange(dst []image.Rectangle, context *guigui.Context, bounds image.Rectangle, startInBytes, endInBytes int) []image.Rectangle {
	return t.core.AppendBoundsOfTextRange(dst, context, bounds, startInBytes, endInBytes)
}

// TextRange is a byte range of a text value.
type TextRange struct {
	// StartInBytes is the inclusive start of the range in bytes.
	StartInBytes int

	// EndInBytes is the exclusive end of the range in bytes.
	EndInBytes int
}

// SetHotspotRanges sets the hotspot ranges: over their rectangles the cursor
// turns into a pointer, and mouse presses and releases fire the hotspot down
// and up events. The ranges follow the text through edits like the ranged
// styles. While the value is editable, the hotspots are inert.
func (t *Text) SetHotspotRanges(ranges []TextRange) {
	t.hotspotRangesBuf = slices.Delete(t.hotspotRangesBuf, 0, len(t.hotspotRangesBuf))
	for _, r := range ranges {
		t.hotspotRangesBuf = append(t.hotspotRangesBuf, textwidget.TextRange{
			StartInBytes: r.StartInBytes,
			EndInBytes:   r.EndInBytes,
		})
	}
	t.core.SetHotspotRanges(t.hotspotRangesBuf)
}

// AppendHotspotRanges appends the current hotspot ranges to dst and returns
// the extended slice, reflecting the adjustments made for edits since
// [Text.SetHotspotRanges].
func (t *Text) AppendHotspotRanges(dst []TextRange) []TextRange {
	t.hotspotRangesBuf = t.core.AppendHotspotRanges(t.hotspotRangesBuf[:0])
	defer func() {
		t.hotspotRangesBuf = slices.Delete(t.hotspotRangesBuf, 0, len(t.hotspotRangesBuf))
	}()
	for _, r := range t.hotspotRangesBuf {
		dst = append(dst, TextRange{
			StartInBytes: r.StartInBytes,
			EndInBytes:   r.EndInBytes,
		})
	}
	return dst
}

// OnHotspotDown sets the event handler that is called when the left mouse
// button is pressed on a hotspot range. The handler is given the pressed
// range.
func (t *Text) OnHotspotDown(f func(context *guigui.Context, textRange TextRange)) {
	t.core.OnHotspotDown(func(context *guigui.Context, textRange textwidget.TextRange) {
		f(context, TextRange{
			StartInBytes: textRange.StartInBytes,
			EndInBytes:   textRange.EndInBytes,
		})
	})
}

// OnHotspotUp sets the event handler that is called when the left mouse
// button is released on the hotspot range it was pressed on. For selectable
// text, the click is canceled when the cursor moved or a selection was made
// between the press and the release: a drag selects, it does not click. The
// handler is given the released range.
func (t *Text) OnHotspotUp(f func(context *guigui.Context, textRange TextRange)) {
	t.core.OnHotspotUp(func(context *guigui.Context, textRange textwidget.TextRange) {
		f(context, TextRange{
			StartInBytes: textRange.StartInBytes,
			EndInBytes:   textRange.EndInBytes,
		})
	})
}

func (t *Text) IsEditable() bool {
	return t.core.IsEditable()
}

func (t *Text) SetEditable(editable bool) {
	t.core.SetEditable(editable)
}

// SetRichTextEditable sets whether pasting applies the ranged styles copied
// along with the text. The default is false: pasting inserts plain text and
// the inserted text adopts the style of the surrounding text.
func (t *Text) SetRichTextEditable(richTextEditable bool) {
	t.core.SetRichTextEditable(richTextEditable)
}

// IsMultiline reports whether the value may span multiple lines. It is always
// false while masking, which is single-line.
func (t *Text) IsMultiline() bool {
	return t.core.IsMultiline()
}

func (t *Text) SetMultiline(multiline bool) {
	t.core.SetMultiline(multiline)
}

// WrapMode reports how visual lines wrap when text exceeds the available
// width. The default is [WrapModeNone].
func (t *Text) WrapMode() WrapMode {
	return WrapMode(t.core.WrapMode())
}

// SetWrapMode selects how visual lines wrap when text exceeds the available
// width. See [WrapMode] for the available modes.
func (t *Text) SetWrapMode(wrapMode WrapMode) {
	t.core.SetWrapMode(textutil.WrapMode(wrapMode))
}

// SetCaretBlinking sets whether the caret blinks.
// The default value is true.
func (t *Text) SetCaretBlinking(caretBlinking bool) {
	t.core.SetCaretBlinking(caretBlinking)
}

// SetSelectionVisibleWhenUnfocused sets whether the selection range stays
// drawn while the widget is not focused. By default the selection is hidden
// when the widget loses focus. Enable this when a separate UI (e.g. a Find
// dialog) holds focus but the user still needs to see what was matched.
func (t *Text) SetSelectionVisibleWhenUnfocused(visible bool) {
	t.core.SetSelectionVisibleWhenUnfocused(visible)
}

func (t *Text) SetEllipsisString(str string) {
	t.core.SetEllipsisString(str)
}

// SetPlaceholder sets the placeholder text shown in a subdued color while the
// value is empty and the text is editable. The empty string disables the
// placeholder.
func (t *Text) SetPlaceholder(placeholder string) {
	t.placeholder = placeholder
}

// SetMaskRune sets the character drawn in place of each grapheme cluster of the
// value. A non-zero rune masks the text and forces it to a single line; the
// zero value renders it normally.
func (t *Text) SetMaskRune(maskRune rune) {
	t.core.SetMaskRune(maskRune)
}

func (t *Text) CanCut() bool {
	return t.core.CanCut()
}

func (t *Text) CanCopy() bool {
	return t.core.CanCopy()
}

func (t *Text) CanPaste() bool {
	return t.core.CanPaste()
}

func (t *Text) CanUndo() bool {
	return t.core.CanUndo()
}

func (t *Text) CanRedo() bool {
	return t.core.CanRedo()
}

func (t *Text) Cut() bool {
	return t.core.Cut()
}

func (t *Text) Copy() bool {
	return t.core.Copy()
}

func (t *Text) Paste() bool {
	return t.core.Paste()
}

// PasteWithoutStyles pastes the clipboard text without the ranged styles
// copied along with it, even when the widget is rich text editable. The
// inserted text adopts the style of the surrounding text.
func (t *Text) PasteWithoutStyles() bool {
	return t.core.PasteWithoutStyles()
}

func (t *Text) Undo() bool {
	return t.core.Undo()
}

func (t *Text) Redo() bool {
	return t.core.Redo()
}
