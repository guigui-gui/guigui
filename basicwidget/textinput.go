// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"image"
	"io"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
	"github.com/guigui-gui/guigui/basicwidget/internal/textwidget"
)

type TextInputStyle int

const (
	TextInputStyleNormal TextInputStyle = iota
	TextInputStyleInline
)

type TextInput struct {
	guigui.DefaultWidget

	textInput   textInput
	focus       textInputFocus
	supportText Text

	hasError          bool
	focusBorderHidden bool
	supportTextValue  string
}

// OnValueChanged sets a handler with the same dispatch and commit semantics as
// [Text.OnValueChanged]. The handler receives the current text and whether the
// change is committed.
//
// If the handler does not need the text payload, prefer
// [TextInput.OnValueChangedWithoutText] to avoid materializing the value on
// every change.
func (t *TextInput) OnValueChanged(f func(context *guigui.Context, text string, committed bool)) {
	t.textInput.OnValueChanged(f)
}

// OnValueChangedWithoutText sets a handler that fires under the same
// conditions as [TextInput.OnValueChanged] but is not given the current text.
// Use this when the handler only needs to know that the value changed so the
// underlying value is not materialized into a string on every change.
func (t *TextInput) OnValueChangedWithoutText(f func(context *guigui.Context, committed bool)) {
	t.textInput.OnValueChangedWithoutText(f)
}

func (t *TextInput) OnHandleButtonInput(f func(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult) {
	t.textInput.OnHandleButtonInput(f)
}

// Value returns the current value as a string.
// For large values, prefer [TextInput.WriteValueTo] to avoid allocating a copy.
func (t *TextInput) Value() string {
	return t.textInput.Value()
}

// HasValue reports whether the text input has a non-empty value.
// This is more efficient than checking Value() != "" as it avoids
// allocating a string.
func (t *TextInput) HasValue() bool {
	return t.textInput.HasValue()
}

func (t *TextInput) SetValue(text string) {
	t.textInput.SetValue(text)
}

func (t *TextInput) ForceSetValue(text string) {
	t.textInput.ForceSetValue(text)
}

// WriteValueTo writes the current value to w and returns the number of bytes
// written. See [Text.WriteValueTo] for details.
func (t *TextInput) WriteValueTo(w io.Writer) (int64, error) {
	return t.textInput.WriteValueTo(w)
}

// WriteValueRangeTo writes the bytes of the current value in
// [startInBytes, endInBytes) to w. See [Text.WriteValueRangeTo] for details.
func (t *TextInput) WriteValueRangeTo(w io.Writer, startInBytes, endInBytes int) (int64, error) {
	return t.textInput.WriteValueRangeTo(w, startInBytes, endInBytes)
}

// LineCount returns the number of logical lines in the value.
// See [Text.LineCount] for details.
func (t *TextInput) LineCount() int {
	return t.textInput.LineCount()
}

// LineStartInBytes returns the byte offset where the lineIndex-th logical
// line begins within the value. See [Text.LineStartInBytes] for details.
func (t *TextInput) LineStartInBytes(lineIndex int) int {
	return t.textInput.LineStartInBytes(lineIndex)
}

// LineIndexFromTextIndexInBytes returns the index of the logical line
// containing textIndexInBytes. See [Text.LineIndexFromTextIndexInBytes] for
// details.
func (t *TextInput) LineIndexFromTextIndexInBytes(textIndexInBytes int) int {
	return t.textInput.LineIndexFromTextIndexInBytes(textIndexInBytes)
}

// CaretPositionAtTextIndexInBytes returns the on-screen top and bottom
// endpoints of a caret drawn at byte offset textIndexInBytes in the text
// value. See [Text.CaretPositionAtTextIndexInBytes] for details.
func (t *TextInput) CaretPositionAtTextIndexInBytes(context *guigui.Context, textIndexInBytes int) (top, bottom image.Point, ok bool) {
	return t.textInput.CaretPositionAtTextIndexInBytes(context, textIndexInBytes)
}

// ReadValueFrom resets the value to the bytes read from r until EOF.
// See [Text.ReadValueFrom] for details.
func (t *TextInput) ReadValueFrom(r io.Reader) (int64, error) {
	return t.textInput.ReadValueFrom(r)
}

func (t *TextInput) ReplaceValueAtSelection(text string) {
	t.textInput.ReplaceValueAtSelection(text)
}

func (t *TextInput) CommitWithCurrentInputValue() {
	t.textInput.CommitWithCurrentInputValue()
}

func (t *TextInput) SetMultiline(multiline bool) {
	t.textInput.SetMultiline(multiline)
}

// SetMaskRune sets the character drawn in place of each grapheme cluster of the
// value, masking it for password-style entry. A non-zero rune also disables
// copy and cut and forces the input to a single line; the zero value restores
// normal text.
func (t *TextInput) SetMaskRune(maskRune rune) {
	t.textInput.SetMaskRune(maskRune)
}

// SetPlaceholder sets the placeholder text shown in a subdued color while the
// value is empty and the text input is editable. The empty string disables the
// placeholder.
func (t *TextInput) SetPlaceholder(placeholder string) {
	t.textInput.SetPlaceholder(placeholder)
}

func (t *TextInput) SetHorizontalAlign(halign HorizontalAlign) {
	t.textInput.SetHorizontalAlign(halign)
}

func (t *TextInput) SetVerticalAlign(valign VerticalAlign) {
	t.textInput.SetVerticalAlign(valign)
}

// WrapMode reports the [WrapMode] currently applied to this text input.
func (t *TextInput) WrapMode() WrapMode {
	return t.textInput.WrapMode()
}

// SetWrapMode sets how visual lines wrap when text exceeds the available
// width. See [WrapMode] for the available modes.
func (t *TextInput) SetWrapMode(wrapMode WrapMode) {
	t.textInput.SetWrapMode(wrapMode)
}

// SetLineHeight sets the line height at scale 1.
// See [Text.SetLineHeight] for details.
func (t *TextInput) SetLineHeight(lineHeight float64) {
	t.textInput.SetLineHeight(lineHeight)
}

// SetLineHeightMode sets how a visual line's height responds to the font
// sizes on it. The default is [LineHeightModeFixed].
func (t *TextInput) SetLineHeightMode(lineHeightMode LineHeightMode) {
	t.textInput.SetLineHeightMode(lineHeightMode)
}

// SetCaretBlinking sets whether the caret blinks.
// The default value is true.
func (t *TextInput) SetCaretBlinking(caretBlinking bool) {
	t.textInput.SetCaretBlinking(caretBlinking)
}

// SetSelectionVisibleWhenUnfocused sets whether the selection range stays
// drawn while the widget is not focused. By default the selection is hidden
// when the widget loses focus.
func (t *TextInput) SetSelectionVisibleWhenUnfocused(visible bool) {
	t.textInput.SetSelectionVisibleWhenUnfocused(visible)
}

func (t *TextInput) SelectAll() {
	t.textInput.SelectAll()
}

func (t *TextInput) Selection() (start, end int) {
	return t.textInput.Selection()
}

func (t *TextInput) SetSelection(start, end int) {
	t.textInput.SetSelection(start, end)
}

func (t *TextInput) IsEditable() bool {
	return t.textInput.IsEditable()
}

func (t *TextInput) WriteStateKey(context *guigui.Context, w *guigui.StateKeyWriter) {
	w.WriteBool(t.hasError)
	w.WriteBool(t.focusBorderHidden)
	w.WriteString(t.supportTextValue)
}

// SetFocusBorderVisible sets whether the focus border is drawn around the
// text input when it has focus. The default is true.
func (t *TextInput) SetFocusBorderVisible(visible bool) {
	t.focusBorderHidden = !visible
}

// SetStyle applies the preset combination of orthogonal properties for the given style.
// SetStyle re-applies every property on each call, so a per-property override takes effect
// only when its setter is called after SetStyle.
func (t *TextInput) SetStyle(style TextInputStyle) {
	switch style {
	case TextInputStyleNormal:
		t.SetFocusBorderVisible(true)
		t.textInput.setCompactPadding(false)
		t.textInput.setIntrinsicSize(false)
	case TextInputStyleInline:
		t.SetFocusBorderVisible(false)
		t.textInput.setCompactPadding(true)
		t.textInput.setIntrinsicSize(true)
	}
}

func (t *TextInput) SetEditable(editable bool) {
	t.textInput.SetEditable(editable)
}

// SetRichTextEditable sets whether pasting applies the ranged styles copied
// along with the text. The default is false: pasting inserts plain text and
// the inserted text adopts the style of the surrounding text.
func (t *TextInput) SetRichTextEditable(richTextEditable bool) {
	t.textInput.SetRichTextEditable(richTextEditable)
}

// ReadBaseStyle replaces style with the base style: the style properties
// applied to the whole value, underneath the ranged style overrides. A
// property is unset when the theme default applies.
func (t *TextInput) ReadBaseStyle(style *TextStyle) {
	t.textInput.text.Text().ReadBaseStyle(style)
}

// SetBaseStyle replaces the base style with style. Unset properties restore
// the theme defaults. The base style holds the font family, the italic face
// selection, OpenType variations and features (including the weight), and
// the text color; the scale and the language are ignored.
func (t *TextInput) SetBaseStyle(style *TextStyle) {
	t.textInput.text.Text().SetBaseStyle(style)
}

// ReadOverrideStyles replaces styles with a copy of the ranged style
// overrides, reflecting the adjustments made for edits since the overrides
// were set.
func (t *TextInput) ReadOverrideStyles(styles *TextStyles) {
	t.textInput.text.Text().ReadOverrideStyles(styles)
}

// SetOverrideStyles replaces the ranged style overrides with styles.
// recorded sets whether the replacement is recorded in the undo history; a
// recorded replacement leaving the overrides unchanged records nothing.
// Pass false when restoring the overrides from a model, such as on every
// build.
func (t *TextInput) SetOverrideStyles(styles *TextStyles, recorded bool) {
	t.textInput.text.Text().SetOverrideStyles(styles, recorded)
}

// ReadOverrideStylesInRange replaces styles with a copy of the ranged style
// overrides in [startInBytes, endInBytes), rebased so that startInBytes maps
// to 0.
func (t *TextInput) ReadOverrideStylesInRange(styles *TextStyles, startInBytes, endInBytes int) {
	t.textInput.text.Text().ReadOverrideStylesInRange(styles, startInBytes, endInBytes)
}

// SetOverrideStylesInRange replaces the ranged style overrides in
// [startInBytes, endInBytes) with styles' overrides in
// [0, endInBytes-startInBytes), shifted so that 0 maps to startInBytes.
// styles' overrides outside [0, endInBytes-startInBytes) are ignored.
// recorded sets whether the replacement is recorded in the undo history; a
// recorded replacement leaving the overrides unchanged records nothing.
func (t *TextInput) SetOverrideStylesInRange(styles *TextStyles, startInBytes, endInBytes int, recorded bool) {
	t.textInput.text.Text().SetOverrideStylesInRange(styles, startInBytes, endInBytes, recorded)
}

// ReadEffectiveStyles replaces styles with the effective styles of the whole
// value. The effective style of a byte is the base style and the rendering
// defaults with the byte's ranged overrides merged on top, so every property
// is set.
func (t *TextInput) ReadEffectiveStyles(styles *TextStyles) {
	t.textInput.text.Text().ReadEffectiveStyles(styles)
}

// ReadEffectiveStylesInRange replaces styles with the effective styles of
// [startInBytes, endInBytes), rebased so that startInBytes maps to 0. The
// effective style of a byte is the base style and the rendering defaults
// with the byte's ranged overrides merged on top, so every property is set.
func (t *TextInput) ReadEffectiveStylesInRange(styles *TextStyles, startInBytes, endInBytes int) {
	t.textInput.text.Text().ReadEffectiveStylesInRange(styles, startInBytes, endInBytes)
}

// ReadEffectiveStyleAt replaces style with the effective style that text
// typed at textIndexInBytes adopts: the base style and the rendering
// defaults with the overrides adopted from the byte right before the index
// merged on top, so every property is set.
func (t *TextInput) ReadEffectiveStyleAt(style *TextStyle, textIndexInBytes int) {
	t.textInput.text.Text().ReadEffectiveStyleAt(style, textIndexInBytes)
}

// SetInsertionStyle replaces the insertion style with style. Its set
// properties are applied as ranged style overrides over the next text
// inserted at the caret, on top of the adopted overrides, and the style is
// then reset. The widget also resets it without applying on other
// interactions, such as a selection change, a deletion, or an undo; neither
// setting nor resetting is recorded in the undo history. SetInsertionStyle
// is typically called on every build with application-owned state, updated
// in the [TextInput.OnInsertionStyleReset] handler so the next build sets
// the cleared style.
func (t *TextInput) SetInsertionStyle(style *TextStyle) {
	t.textInput.text.Text().SetInsertionStyle(style)
}

// OnInsertionStyleReset sets an event handler invoked when the widget resets
// the insertion style: after applying it to inserted text, or when
// discarding it without applying, such as on a selection change. The handler
// is not invoked for [TextInput.SetInsertionStyle].
func (t *TextInput) OnInsertionStyleReset(f func(context *guigui.Context)) {
	t.textInput.text.Text().OnInsertionStyleReset(f)
}

// IsError reports whether the text input is in the error state.
func (t *TextInput) IsError() bool {
	return t.hasError
}

// SetError sets whether the text input is in the error state.
// When the error state is true, the text input border is drawn in a danger color.
func (t *TextInput) SetError(hasError bool) {
	if t.hasError == hasError {
		return
	}
	t.hasError = hasError
	t.textInput.frame.setError(hasError)
}

// SupportText returns the support text displayed below the text input.
func (t *TextInput) SupportText() string {
	return t.supportTextValue
}

// SetSupportText sets the support text displayed below the text input.
// The support text is shown in a subdued color, or in a danger color when the error state is true.
// A line break starts a new line.
func (t *TextInput) SetSupportText(text string) {
	t.supportTextValue = text
}

func (t *TextInput) SetIcon(icon *ebiten.Image) {
	t.textInput.SetIcon(icon)
}

func (t *TextInput) CanCut() bool {
	return t.textInput.CanCut()
}

func (t *TextInput) CanCopy() bool {
	return t.textInput.CanCopy()
}

func (t *TextInput) CanPaste() bool {
	return t.textInput.CanPaste()
}

func (t *TextInput) CanUndo() bool {
	return t.textInput.CanUndo()
}

func (t *TextInput) CanRedo() bool {
	return t.textInput.CanRedo()
}

func (t *TextInput) Cut() bool {
	return t.textInput.Cut()
}

func (t *TextInput) Copy() bool {
	return t.textInput.Copy()
}

func (t *TextInput) Paste() bool {
	return t.textInput.Paste()
}

// PasteWithoutStyles pastes the clipboard text without the ranged styles
// copied along with it, even when the widget is rich text editable. The
// inserted text adopts the style of the surrounding text.
func (t *TextInput) PasteWithoutStyles() bool {
	return t.textInput.PasteWithoutStyles()
}

func (t *TextInput) Undo() bool {
	return t.textInput.Undo()
}

func (t *TextInput) Redo() bool {
	return t.textInput.Redo()
}

func (t *TextInput) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.textInput)
	adder.AddWidget(&t.focus)
	context.SetPassthrough(&t.focus, true)
	context.DelegateFocus(t, &t.textInput.text)

	if t.supportTextValue != "" {
		adder.AddWidget(&t.supportText)
		t.supportText.SetValue(t.supportTextValue)
		t.supportText.SetScale(0.85)
		t.supportText.SetMultiline(true)
		t.supportText.SetHorizontalAlign(t.textInput.text.Text().HorizontalAlign())
		var style TextStyle
		if t.hasError {
			style.SetColor(draw.TextColorFromTint(context.ColorMode(), draw.DangerTintColor()))
		} else {
			style.SetColor(draw.TextColor(context.ColorMode(), false))
		}
		t.supportText.SetBaseStyle(&style)
	}

	return nil
}

func (t *TextInput) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	bounds := widgetBounds.Bounds()

	inputBounds := bounds
	var supportTextSize image.Point
	if t.supportTextValue != "" {
		supportTextSize = t.supportText.Measure(context, guigui.Constraints{})
		// The bounds can be shorter than the input box and the support text
		// need. Keep the input box at its own height in that case and let the
		// support text overflow instead of collapsing the box.
		inputHeight := t.measureTextInput(context, guigui.FixedWidthConstraints(bounds.Dx())).Y
		inputBounds.Max.Y = max(bounds.Max.Y-supportTextSize.Y-supportTextGap(context), bounds.Min.Y+inputHeight)
	}

	layouter.LayoutWidget(&t.textInput, inputBounds)

	w := textInputFocusBorderWidth(context)
	p := inputBounds.Min.Add(image.Pt(-w, -w))
	s := inputBounds.Size().Add(image.Pt(2*w, 2*w))
	layouter.LayoutWidget(&t.focus, image.Rectangle{
		Min: p,
		Max: p.Add(s),
	})

	if t.supportTextValue != "" {
		// The support text is not wrapped: one wider than the text input area
		// extends past the widget's bounds instead of narrowing the area or
		// growing the widget.
		sw := max(supportTextSize.X, inputBounds.Dx())
		x := supportTextMinX(t.textInput.text.Text().HorizontalAlign(), inputBounds, sw)
		y := inputBounds.Max.Y + supportTextGap(context)
		layouter.LayoutWidget(&t.supportText, image.Rectangle{
			Min: image.Pt(x, y),
			Max: image.Pt(x+sw, y+supportTextSize.Y),
		})
	}
}

// measureTextInput returns the size of just the text input area, excluding the support text.
func (t *TextInput) measureTextInput(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.textInput.Measure(context, constraints)
}

// supportTextGap returns the vertical gap between the text input area and the
// support text below it.
func supportTextGap(context *guigui.Context) int {
	return int(2 * context.Scale())
}

// supportTextMinX returns the left edge of a support text of the given width
// placed under a text input area occupying inputBounds and aligned as halign.
func supportTextMinX(halign HorizontalAlign, inputBounds image.Rectangle, width int) int {
	switch halign {
	case HorizontalAlignCenter:
		return inputBounds.Min.X - (width-inputBounds.Dx())/2
	case HorizontalAlignEnd:
		return inputBounds.Max.X - width
	default:
		return inputBounds.Min.X
	}
}

func (t *TextInput) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	s := t.measureTextInput(context, constraints)
	if t.supportTextValue != "" {
		// The support text is not wrapped, so its height does not depend on the
		// width the widget is given.
		supportTextSize := t.supportText.Measure(context, guigui.Constraints{})
		s.Y += supportTextSize.Y + supportTextGap(context)
	}
	return s
}

func (t *TextInput) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	context.SetVisible(&t.focus, !t.focusBorderHidden && t.textInput.text.Text().isFocused(context))
	return nil
}

func (t *TextInput) setSelection(start, end int) {
	t.textInput.setSelection(start, end)
}

func (t *TextInput) setPaddingEnd(padding int) {
	t.textInput.setPaddingEnd(padding)
}

type textInput struct {
	guigui.DefaultWidget

	background     textInputBackground
	text           textInputText
	panel          virtualScrollPanel
	iconBackground textInputIconBackground
	icon           Image
	frame          textInputFrame

	// compactPadding uses tighter inner padding without vertical centering.
	compactPadding bool
	// intrinsicSize measures to the content size instead of a fixed control size.
	intrinsicSize bool
	readonly      bool
	paddingStart  int
	paddingEnd    int

	onTextScrollDelta    func(context *guigui.Context, deltaX, deltaY float64)
	onTextScrollIntoView func(context *guigui.Context, start, end textwidget.CaretScrollTarget)
}

func (t *textInput) OnValueChanged(f func(context *guigui.Context, text string, committed bool)) {
	t.text.Text().OnValueChanged(f)
}

func (t *textInput) OnValueChangedWithoutText(f func(context *guigui.Context, committed bool)) {
	t.text.Text().OnValueChangedWithoutText(f)
}

func (t *textInput) OnHandleButtonInput(f func(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult) {
	t.text.Text().OnHandleButtonInput(f)
}

func (t *textInput) Value() string {
	return t.text.Text().Value()
}

func (t *textInput) HasValue() bool {
	return t.text.Text().HasValue()
}

func (t *textInput) SetValue(text string) {
	t.text.Text().SetValue(text)
}

func (t *textInput) ForceSetValue(text string) {
	t.text.Text().ForceSetValue(text)
}

func (t *textInput) WriteValueTo(w io.Writer) (int64, error) {
	return t.text.Text().WriteValueTo(w)
}

func (t *textInput) WriteValueRangeTo(w io.Writer, startInBytes, endInBytes int) (int64, error) {
	return t.text.Text().WriteValueRangeTo(w, startInBytes, endInBytes)
}

func (t *textInput) LineCount() int {
	return t.text.Text().LineCount()
}

func (t *textInput) LineStartInBytes(lineIndex int) int {
	return t.text.Text().LineStartInBytes(lineIndex)
}

func (t *textInput) CaretPositionAtTextIndexInBytes(context *guigui.Context, textIndexInBytes int) (top, bottom image.Point, ok bool) {
	return t.text.Text().CaretPositionAtTextIndexInBytes(context, textIndexInBytes)
}

func (t *textInput) LineIndexFromTextIndexInBytes(textIndexInBytes int) int {
	return t.text.Text().LineIndexFromTextIndexInBytes(textIndexInBytes)
}

func (t *textInput) ReadValueFrom(r io.Reader) (int64, error) {
	return t.text.Text().ReadValueFrom(r)
}

func (t *textInput) ReplaceValueAtSelection(text string) {
	t.text.Text().ReplaceValueAtSelection(text)
}

func (t *textInput) CommitWithCurrentInputValue() {
	t.text.Text().CommitWithCurrentInputValue()
}

func (t *textInput) SetMultiline(multiline bool) {
	t.text.Text().SetMultiline(multiline)
}

func (t *textInput) SetMaskRune(maskRune rune) {
	t.text.Text().SetMaskRune(maskRune)
}

func (t *textInput) SetPlaceholder(placeholder string) {
	t.text.Text().SetPlaceholder(placeholder)
}

func (t *textInput) SetHorizontalAlign(halign HorizontalAlign) {
	t.text.Text().SetHorizontalAlign(halign)
}

func (t *textInput) SetVerticalAlign(valign VerticalAlign) {
	t.text.SetVerticalAlign(valign)
}

func (t *textInput) WrapMode() WrapMode {
	return t.text.Text().WrapMode()
}

func (t *textInput) SetWrapMode(wrapMode WrapMode) {
	t.text.Text().SetWrapMode(wrapMode)
}

func (t *textInput) SetLineHeight(lineHeight float64) {
	t.text.Text().SetLineHeight(lineHeight)
}

func (t *textInput) SetLineHeightMode(lineHeightMode LineHeightMode) {
	t.text.Text().SetLineHeightMode(lineHeightMode)
}

func (t *textInput) SetCaretBlinking(caretBlinking bool) {
	t.text.Text().SetCaretBlinking(caretBlinking)
}

func (t *textInput) SetSelectionVisibleWhenUnfocused(visible bool) {
	t.text.Text().SetSelectionVisibleWhenUnfocused(visible)
}

func (t *textInput) SelectAll() {
	t.text.Text().core.SelectAll()
}

func (t *textInput) Selection() (start, end int) {
	return t.text.Text().Selection()
}

func (t *textInput) SetSelection(start, end int) {
	t.text.Text().SetSelection(start, end)
}

func (t *textInput) IsEditable() bool {
	return !t.readonly
}

func (t *textInput) WriteStateKey(context *guigui.Context, w *guigui.StateKeyWriter) {
	w.WriteBool(t.compactPadding)
	w.WriteBool(t.intrinsicSize)
	w.WriteBool(t.readonly)
	w.WriteInt64(int64(t.paddingStart))
	w.WriteInt64(int64(t.paddingEnd))
}

func (t *textInput) setCompactPadding(compact bool) {
	t.compactPadding = compact
}

func (t *textInput) setIntrinsicSize(intrinsic bool) {
	t.intrinsicSize = intrinsic
}

func (t *textInput) SetEditable(editable bool) {
	if t.readonly == !editable {
		return
	}
	t.readonly = !editable
	t.text.Text().SetEditable(editable)
}

func (t *textInput) SetRichTextEditable(richTextEditable bool) {
	t.text.Text().SetRichTextEditable(richTextEditable)
}

func (t *textInput) setSelection(start, end int) {
	t.text.Text().core.SetSelectionWithSide(start, end, -1, false)
}

func (t *textInput) setPaddingEnd(padding int) {
	t.paddingEnd = padding
}

func (t *textInput) SetIcon(icon *ebiten.Image) {
	t.icon.SetImage(icon)
}

func (t *textInput) textInputPaddingInScrollableContent(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.Padding {
	u := UnitSize(context)
	var start, end, y int
	if t.compactPadding {
		start = u / 4
		end = u / 4
	} else {
		start = u / 2
		end = u / 2
		if t.icon.HasImage() {
			start = u / 4
		}
		y = int(float64(min(widgetBounds.Bounds().Dy(), u))-t.text.Text().scaledLineHeight(context)) / 2
	}
	start += t.paddingStart
	end += t.paddingEnd
	return guigui.Padding{
		Start:  start,
		Top:    y,
		End:    end,
		Bottom: y,
	}
}

func (t *textInput) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.background)
	if t.icon.HasImage() {
		adder.AddWidget(&t.iconBackground)
		adder.AddWidget(&t.icon)
	}
	adder.AddWidget(&t.panel)
	adder.AddWidget(&t.frame)

	t.panel.setContent(&t.text)
	t.panel.setHorizontalBarHidden(!t.text.Text().IsMultiline())
	t.text.setPanel(&t.panel)

	t.background.setEditable(!t.readonly)
	t.iconBackground.setEditable(!t.readonly)
	t.text.setEditable(!t.readonly)

	if t.onTextScrollDelta == nil {
		t.onTextScrollDelta = func(context *guigui.Context, deltaX, deltaY float64) {
			t.panel.forceSetScrollOffsetByDelta(deltaX, deltaY)
		}
	}
	t.text.Text().core.OnScrollDelta(t.onTextScrollDelta)

	if t.onTextScrollIntoView == nil {
		t.onTextScrollIntoView = func(context *guigui.Context, start, end textwidget.CaretScrollTarget) {
			t.text.scrollCaretIntoView(context, start, end)
		}
	}
	t.text.Text().core.OnScrollIntoView(t.onTextScrollIntoView)

	context.SetPassthrough(&t.frame, true)
	context.DelegateFocus(t, t.text.Text())

	return nil
}

func (t *textInput) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	padding := t.textInputPaddingInScrollableContent(context, widgetBounds)
	t.text.setPadding(padding)

	bounds := widgetBounds.Bounds()
	layouter.LayoutWidget(&t.background, bounds)
	layouter.LayoutWidget(&t.frame, bounds)

	panelBounds := bounds
	if t.icon.HasImage() {
		iconSize := defaultIconSize(context)
		iconBounds := image.Rectangle{
			Min: bounds.Min.Add(image.Point{
				X: UnitSize(context)/4 + int(0.5*context.Scale()),
				Y: (bounds.Dy() - iconSize) / 2,
			}),
		}
		iconBounds.Max = iconBounds.Min.Add(image.Pt(iconSize, iconSize))
		bgBounds := bounds
		bgBounds.Max.X = iconBounds.Max.X + UnitSize(context)/4
		layouter.LayoutWidget(&t.iconBackground, bgBounds)
		layouter.LayoutWidget(&t.icon, iconBounds)

		panelBounds.Min.X = iconBounds.Max.X
	}
	// Use the panel area (excluding any icon) as the container so that
	// width-related decisions inside textInputText - in particular the
	// horizontal scroll-bar threshold in [textInputText.contentWidth] -
	// are made against the actual scrollable viewport.
	t.text.setContainerBounds(panelBounds)
	layouter.LayoutWidget(&t.panel, panelBounds)
}

func (t *textInput) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	u := UnitSize(context)
	if t.intrinsicSize {
		// WidgetBounds is not needed for intrinsic sizing.
		padding := t.textInputPaddingInScrollableContent(context, nil)
		if fixedWidth, ok := constraints.FixedWidth(); ok {
			constraints = guigui.FixedWidthConstraints(fixedWidth - padding.Start - padding.End)
		}
		s := t.text.Text().Measure(context, constraints)
		w := max(s.X+padding.Start+padding.End, u)
		h := s.Y
		return image.Pt(w, h)
	}
	if t.text.Text().IsMultiline() {
		return image.Pt(6*u, 4*u)
	}
	return image.Pt(6*u, u)
}

func (t *textInput) CanCut() bool {
	return t.text.Text().CanCut()
}

func (t *textInput) CanCopy() bool {
	return t.text.Text().CanCopy()
}

func (t *textInput) CanPaste() bool {
	return t.text.Text().CanPaste()
}

func (t *textInput) CanUndo() bool {
	return t.text.Text().CanUndo()
}

func (t *textInput) CanRedo() bool {
	return t.text.Text().CanRedo()
}

func (t *textInput) Cut() bool {
	return t.text.Text().Cut()
}

func (t *textInput) Copy() bool {
	return t.text.Text().Copy()
}

func (t *textInput) Paste() bool {
	return t.text.Text().Paste()
}

func (t *textInput) PasteWithoutStyles() bool {
	return t.text.Text().PasteWithoutStyles()
}

func (t *textInput) Undo() bool {
	return t.text.Text().Undo()
}

func (t *textInput) Redo() bool {
	return t.text.Text().Redo()
}

type textInputBackground struct {
	guigui.DefaultWidget

	editable bool
}

func (t *textInputBackground) WriteStateKey(context *guigui.Context, w *guigui.StateKeyWriter) {
	w.WriteBool(t.editable)
}

func (t *textInputBackground) setEditable(editable bool) {
	t.editable = editable
}

func (t *textInputBackground) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	clr := draw.ContentBackgroundColor(context.ColorMode(), context.IsEnabled(t) && t.editable)
	draw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
}

type textInputIconBackground struct {
	guigui.DefaultWidget

	editable bool
}

func (t *textInputIconBackground) WriteStateKey(context *guigui.Context, w *guigui.StateKeyWriter) {
	w.WriteBool(t.editable)
}

func (t *textInputIconBackground) setEditable(editable bool) {
	t.editable = editable
}

func (t *textInputIconBackground) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	clr := draw.ContentBackgroundColor(context.ColorMode(), context.IsEnabled(t) && t.editable)
	draw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
}

type textInputFrame struct {
	guigui.DefaultWidget

	hasError bool
}

func (t *textInputFrame) WriteStateKey(context *guigui.Context, w *guigui.StateKeyWriter) {
	w.WriteBool(t.hasError)
}

func (t *textInputFrame) setError(hasError bool) {
	t.hasError = hasError
}

func (t *textInputFrame) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	clr1, clr2 := draw.BorderColors(context.ColorMode(), draw.RoundedRectBorderTypeInset)
	draw.DrawRoundedRectBorder(context, dst, bounds, clr1, clr2, RoundedCornerRadius(context), float32(1*context.Scale()), draw.RoundedRectBorderTypeInset)
	if t.hasError {
		dclr1, dclr2 := draw.BorderDangerColors(context.ColorMode())
		draw.DrawRoundedRectBorder(context, dst, bounds, dclr1, dclr2, RoundedCornerRadius(context), float32(1*context.Scale()), draw.RoundedRectBorderTypeRegular)
	}
}

type textInputText struct {
	guigui.DefaultWidget

	text roundedCornerWidget[*Text]

	editable        bool
	containerBounds image.Rectangle
	padding         guigui.Padding

	// vAlign is the user-set vertical alignment for the TextInput. The inner
	// [*Text] widget is intentionally left at its default ([VerticalAlignTop])
	// so its own per-line shaping (via [Text.textContentBounds] /
	// [Text.textHeight]) doesn't run on every Draw - dominant for wrapped
	// on multi-megabyte buffers. Instead, [textInputText.Layout] applies
	// vAlign as a Min.Y shift on textBounds when the document fits the
	// viewport; when it overflows, vAlign is moot and the panel's scroll
	// state owns vertical positioning.
	vAlign VerticalAlign

	// panel is the [virtualScrollPanel] this content lives inside, set by
	// [textInput.Build]. Layout uses windowed positioning anchored at the
	// panel's topItemIndex/topItemOffset, and the [virtualScrollContent]
	// methods report logical-line counts and heights so the panel can size
	// its scroll bar without measuring the whole document.
	panel *virtualScrollPanel

	// measuredLineHeights caches per-Layout logical-line heights, populated
	// during virtualized layout and consumed by [textInputText.measureItemHeight].
	// Cleared at the start of each virtualized Layout.
	measuredLineHeights map[int]int

	// measuredLineHeightsGeneration is the text field generation the cached
	// heights were measured against. The cache is keyed by line index, so a
	// content edit invalidates it.
	measuredLineHeightsGeneration int64

	// measuredMaxWidth tracks the widest visual line measured during the
	// current Layout. Used by [textInputText.contentWidth] to size the
	// panel's horizontal scroll bar without scanning every logical line.
	//
	// Reset at the start of each [textInputText.Layout]; updated by
	// [textInputText.measureMaxWidthForViewport] for each visible line
	// measured. As a result the H scroll bar reflects the widest line in the
	// current viewport rather than a historical high-water mark - the bar
	// grows and shrinks as the user scrolls past wide regions, but it is
	// never stale after edits or document replacement.
	measuredMaxWidth int

	// measuredMaxWidthWrapMode is the wrap mode measuredMaxWidth was measured
	// under. [textInputText.contentWidth] ignores measuredMaxWidth when the
	// current wrap mode differs.
	measuredMaxWidthWrapMode WrapMode

	// measuredMaxWidthInnerWidth is the wrapping width measuredMaxWidth was
	// measured at. [textInputText.contentWidth] ignores measuredMaxWidth for
	// wrapped text when the current inner width differs (e.g. after a
	// container resize).
	measuredMaxWidthInnerWidth int
}

var _ virtualScrollContent = (*textInputText)(nil)

func (t *textInputText) setEditable(editable bool) {
	t.text.Widget().SetEditable(editable)
}

// SetVerticalAlign records the user-set vertical alignment. The inner
// [*Text] widget is not updated - see the [textInputText.vAlign] field
// comment for why.
func (t *textInputText) SetVerticalAlign(valign VerticalAlign) {
	t.vAlign = valign
}

func (t *textInputText) WriteStateKey(context *guigui.Context, w *guigui.StateKeyWriter) {
	writeRectangle(w, t.containerBounds)
	writePadding(w, t.padding)
	w.WriteUint64(uint64(t.vAlign))
}

func (t *textInputText) setContainerBounds(bounds image.Rectangle) {
	t.containerBounds = bounds
}

func (t *textInputText) setPadding(padding guigui.Padding) {
	if t.padding == padding {
		return
	}
	t.padding = padding
	t.text.Widget().core.SetPaddingForScrollOffset(padding)
}

func (t *textInputText) Text() *Text {
	return t.text.Widget()
}

func (t *textInputText) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.text)

	t.text.Widget().SetSelectable(true)
	t.text.Widget().core.SetKeepTailingSpace(t.text.Widget().WrapMode() == WrapModeNone)

	context.DelegateFocus(t, t.text.Widget())

	return nil
}

func (t *textInputText) setPanel(p *virtualScrollPanel) {
	t.panel = p
}

// contentWidth implements [virtualScrollContent]. For single-line text the
// width is measured on demand (cheap: one line). For multiline text the
// width is taken from the high-water mark recorded during virtualized
// Layout - lines outside the viewport aren't measured, so the bar may
// underestimate until the user has scrolled through wide regions.
//
// The result is clamped up to at least the container width so the *Text
// widget always covers the full viewport horizontally and clicks anywhere
// inside the panel reach it (I-beam mouse pointer, click-to-focus,
// click-to-position-caret).
func (t *textInputText) contentWidth(context *guigui.Context) int {
	txt := t.text.Widget()
	var measured int
	if !txt.IsMultiline() {
		w := txt.Measure(context, guigui.Constraints{}).X
		measured = w + t.padding.Start + t.padding.End
	} else if t.measuredMaxWidthWrapMode == txt.WrapMode() &&
		(txt.WrapMode() == WrapModeNone || t.measuredMaxWidthInnerWidth == t.containerBounds.Dx()-t.padding.Start-t.padding.End) {
		measured = t.measuredMaxWidth
	}
	return max(measured, t.containerBounds.Dx())
}

// itemCount implements [virtualScrollContent]. Each item is one logical
// line of the source text.
func (t *textInputText) itemCount() int {
	return t.text.Widget().LineCount()
}

// viewportPaddingY implements [virtualScrollContent.viewportPaddingY].
func (t *textInputText) viewportPaddingY(_ *guigui.Context) int {
	return t.padding.Top + t.padding.Bottom
}

// measureItemHeight implements [virtualScrollContent]. Returns the rendered
// height of one logical line at the panel's current content width, cached
// for the lifetime of the current virtualized Layout.
//
// For [WrapModeNone] text every logical line is exactly one visual line, so
// the height is constant and shaping is skipped entirely; this keeps dense
// walks (e.g. dragging the V scroll thumb across a multi-million-line
// document) O(N) trivial. The horizontal scroll bar's [textInputText.measuredMaxWidth]
// is populated by [textInputText.measureMaxWidthForViewport] over the
// viewport lines that Layout has already touched.
func (t *textInputText) measureItemHeight(context *guigui.Context, lineIndex int) int {
	txt := t.text.Widget()

	// The cache is keyed by line index, so a content edit that shifts lines
	// makes the cached heights stale. Drop them when the store has advanced;
	// scrollEdgeIntoView reads heights from Tick before the next Layout
	// repopulates the cache.
	if gen := txt.core.Generation(); gen > t.measuredLineHeightsGeneration {
		clear(t.measuredLineHeights)
		t.measuredLineHeightsGeneration = gen
	}

	if h, ok := t.measuredLineHeights[lineIndex]; ok {
		return h
	}

	n := txt.LineCount()
	if lineIndex < 0 || lineIndex >= n {
		return -1
	}

	width := t.containerBounds.Dx() - t.padding.Start - t.padding.End
	if width <= 0 {
		width = math.MaxInt
	}

	// Only the height is needed here; the viewport lines' widths are
	// measured afterwards in a separate pass
	// ([textInputText.measureMaxWidthForViewport]) that hits the same
	// cache entry. The height comes from the content-keyed layout cache
	// rather than from re-packing every Layout.
	height := int(math.Ceil(txt.core.LogicalLineHeight(context, lineIndex, width)))

	if t.measuredLineHeights == nil {
		t.measuredLineHeights = map[int]int{}
	}
	t.measuredLineHeights[lineIndex] = height

	return height
}

func (t *textInputText) HandleButtonInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	// On macOS, Home/End scroll to the text edges without moving the caret. The
	// focused Text handles every selection-changing key (including
	// Shift+Home/End) and declines plain Home/End, which then bubble up here.
	if context.KeyBindingMode() != guigui.KeyBindingModeCommand || ebiten.IsKeyPressed(ebiten.KeyShift) || t.panel == nil {
		return guigui.HandleInputResult{}
	}
	switch {
	case isKeyRepeating(ebiten.KeyHome):
		t.panel.setTopItem(0, 0)
		return guigui.HandleInputByWidget(t)
	case isKeyRepeating(ebiten.KeyEnd):
		// Setting the last item as the top item over-scrolls; the panel clamps
		// it to a bottom-aligned position.
		if n := t.itemCount(); n > 0 {
			t.panel.setTopItem(n-1, 0)
		}
		return guigui.HandleInputByWidget(t)
	}
	return guigui.HandleInputResult{}
}

// scrollCaretIntoView scrolls the panel to bring the selection into view.
// start and end are the selection endpoints (start <= end as byte indices),
// equal when the selection has zero width. The moving end — the endpoint that
// Shift+key extends — has priority: if it isn't fully visible, scroll for it;
// otherwise scroll for the anchor when it is off-viewport. When the selection
// is taller than the viewport, the moving end wins.
//
// The X axis accumulates contributions from both endpoints, matching the
// legacy textEventScrollDelta semantics.
func (t *textInputText) scrollCaretIntoView(context *guigui.Context, start, end textwidget.CaretScrollTarget) {
	if t.panel == nil {
		return
	}
	// Follow the moving end so upward/leftward extension scrolls toward the
	// start; only the start side moves while shiftSelectionSide is Start.
	primary, secondary := end, start
	if t.text.Widget().core.ShiftSelectionSide() == textwidget.SelectionSideStart {
		primary, secondary = start, end
	}
	if !t.scrollEdgeIntoView(context, primary) && primary != secondary {
		t.scrollEdgeIntoView(context, secondary)
	}

	bounds := t.containerBounds
	dxEnd := min(float64(bounds.Max.X)-end.X-float64(t.padding.End), 0)
	dxStart := max(float64(bounds.Min.X)-start.X+float64(t.padding.Start), 0)
	if dx := dxEnd + dxStart; dx != 0 {
		t.panel.forceSetScrollOffsetByDelta(dx, 0)
	}
}

// scrollEdgeIntoView scrolls the panel so target is visible, returning true
// when a scroll was applied. Walks at most one viewport's worth of items.
func (t *textInputText) scrollEdgeIntoView(context *guigui.Context, target textwidget.CaretScrollTarget) bool {
	n := t.itemCount()
	if n == 0 {
		return false
	}
	lineIdx := max(target.LogicalLineIndex, 0)
	if lineIdx >= n {
		lineIdx = n - 1
	}

	bounds := t.containerBounds
	paddingTop := float64(t.padding.Top)
	paddingBottom := float64(t.padding.Bottom)
	viewportTop := paddingTop
	viewportBottom := float64(bounds.Dy()) - paddingBottom

	topIdx, topOff := t.panel.topItem()

	if lineIdx < topIdx || (lineIdx == topIdx && target.Top < float64(-topOff)) {
		t.panel.setTopItem(lineIdx, -int(math.Floor(target.Top)))
		return true
	}

	// y is the panel-local Y of the current iter's line top.
	y := paddingTop + float64(topOff)
	for idx := topIdx; idx < n; idx++ {
		h := t.measureItemHeight(context, idx)
		if h < 0 {
			return false
		}
		if idx == lineIdx {
			if caretBottomY := y + target.Bottom; caretBottomY > viewportBottom {
				diff := int(math.Ceil(caretBottomY - viewportBottom))
				t.panel.setTopItem(topIdx, topOff-diff)
				return true
			}
			return false
		}
		y += float64(h)
		if y >= viewportBottom {
			break
		}
	}

	// Below viewport: walk UP from lineIdx, fitting predecessors into the
	// available content height so target.Bottom lands at the viewport bottom.
	remaining := (viewportBottom - target.Bottom) - viewportTop
	newTop := lineIdx
	newOff := 0
	for newTop > 0 && remaining > 0 {
		prevH := t.measureItemHeight(context, newTop-1)
		if prevH < 0 {
			break
		}
		if remaining >= float64(prevH) {
			newTop--
			remaining -= float64(prevH)
			continue
		}
		newOff = -int(math.Ceil(float64(prevH) - remaining))
		newTop--
		break
	}
	t.panel.setTopItem(newTop, newOff)
	return true
}

// measureMaxWidthForViewport runs after [textInputText.Layout]'s height-only
// walks and records the widest viewport caret extent into
// [textInputText.measuredMaxWidth] so the horizontal scroll bar can reach every
// editable position. Wrapped text normally stays at the viewport width, but an
// unbreakable segment can extend beyond it and require horizontal scrolling.
func (t *textInputText) measureMaxWidthForViewport(context *guigui.Context) {
	txt := t.text.Widget()
	if !txt.IsMultiline() || len(t.measuredLineHeights) == 0 {
		return
	}
	n := txt.LineCount()
	measureWidth := t.containerBounds.Dx() - t.padding.Start - t.padding.End
	if txt.WrapMode() == WrapModeNone || measureWidth <= 0 {
		measureWidth = math.MaxInt
	}
	for lineIdx := range t.measuredLineHeights {
		if lineIdx < 0 || lineIdx >= n {
			continue
		}
		w := txt.core.MaxCaretXOfLogicalLine(context, lineIdx, measureWidth)
		if mw := int(math.Ceil(w)) + t.padding.Start + t.padding.End; mw > t.measuredMaxWidth {
			t.measuredMaxWidth = mw
		}
	}
}

// Layout normalizes the panel's (topItemIndex, topItemOffset) using real
// measured line heights, then positions the [*Text] child so the top
// visible logical line lands at the panel viewport.
func (t *textInputText) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	clear(t.measuredLineHeights)
	t.measuredMaxWidth = 0

	bounds := widgetBounds.Bounds()
	txt := t.text.Widget()
	innerWidth := t.containerBounds.Dx() - t.padding.Start - t.padding.End
	t.measuredMaxWidthWrapMode = txt.WrapMode()
	t.measuredMaxWidthInnerWidth = innerWidth
	txt.core.SetWrapWidth(innerWidth)
	lh := int(math.Ceil(txt.core.LineHeight()))

	viewportInner := bounds.Dy() - t.padding.Top - t.padding.Bottom
	topIdx, topOff := t.panel.layoutTopItem(context, viewportInner,
		func(ai int) int { return t.measureItemHeight(context, ai) })
	n := t.itemCount()

	// Position the *Text widget so logical line topIdx lands at the
	// panel viewport top, shifted by topOff. The inner *Text takes
	// topIdx as its coordinate-system origin via
	// setFirstLogicalLineInViewport, so positioning here is O(1) and
	// never walks the document prefix.
	t.text.Widget().core.SetFirstLogicalLineInViewport(topIdx)

	textBounds := bounds
	textBounds.Min.X += t.padding.Start
	textBounds.Min.Y += topOff + t.padding.Top
	textBounds.Max.X -= t.padding.End

	// Apply the user-set vertical alignment as a Min.Y shift, but only when
	// the document fits the viewport. When it overflows, vAlign is moot -
	// the panel's scroll state owns vertical positioning. The cheap upper-
	// bound predicate n*lh >= viewportInner short-circuits the texteditor
	// case (huge n) without walking any lines; in the may-fit branch n is
	// bounded by viewportInner/lh so the walk is O(viewport).
	if t.vAlign != VerticalAlignTop && n*lh <= viewportInner {
		var sum int
		for i := range n {
			sum += t.measureItemHeight(context, i)
			if sum > viewportInner {
				break
			}
		}
		if sum <= viewportInner {
			var alignOffset int
			switch t.vAlign {
			case VerticalAlignMiddle:
				alignOffset = (viewportInner - sum) / 2
			case VerticalAlignBottom:
				alignOffset = viewportInner - sum
			}
			textBounds.Min.Y += alignOffset
		}
	}

	// The *Text widget only needs to cover the viewport for hit testing -
	// clicks past the viewport can't reach it because the panel clips.
	// Inside Text, the firstLogicalLineInViewport anchor sits at
	// textBounds.Min.Y and lines below extend downward from there;
	// [Text.textContentBounds] recomputes the content extent itself, so
	// the input Max.Y doesn't propagate into Text's content layout. So
	// Max.Y just needs to bottom out at the panel viewport.
	textBounds.Max.Y = bounds.Max.Y - t.padding.Bottom

	textBounds = textBounds.Add(image.Pt(0, int(0.5*context.Scale())))
	layouter.LayoutWidget(&t.text, textBounds)

	t.text.SetRenderingBounds(t.containerBounds)

	// Now that the viewport's logical lines have been touched (and so are
	// in [textInputText.measuredLineHeights]), measure their widths once
	// to size the horizontal scroll bar. Done as a separate pass so that
	// [textInputText.measureItemHeight] can stay shaping-free for non-
	// wrapped text on dense walks.
	t.measureMaxWidthForViewport(context)

}

func (t *textInputText) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	// guigui.LinearLayout cannot treat auto-wrapping texts very well.
	// Calculate the size directly here.
	s := t.measureText(context, constraints)
	s.X += t.padding.Start + t.padding.End
	s.Y += t.padding.Top + t.padding.Bottom
	s.X = max(s.X, t.containerBounds.Dx())
	s.Y = max(s.Y, t.containerBounds.Dy())
	return s
}

func (t *textInputText) measureText(context *guigui.Context, constraints guigui.Constraints) image.Point {
	if fixedWidth, ok := constraints.FixedWidth(); ok {
		constraints = guigui.FixedWidthConstraints(fixedWidth - t.padding.Start - t.padding.End)
	}
	if fixedHeight, ok := constraints.FixedHeight(); ok {
		constraints = guigui.FixedHeightConstraints(fixedHeight - t.padding.Top - t.padding.Bottom)
	}
	return t.text.Measure(context, constraints)
}

func textInputFocusBorderWidth(context *guigui.Context) int {
	return int(4 * context.Scale())
}

type textInputFocus struct {
	guigui.DefaultWidget
}

func (t *textInputFocus) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	w := textInputFocusBorderWidth(context)
	clr := draw.FocusBorderColor(context.ColorMode())
	draw.DrawRoundedRectBorder(context, dst, bounds, clr, clr, w+RoundedCornerRadius(context), float32(w), draw.RoundedRectBorderTypeRegular)
}
