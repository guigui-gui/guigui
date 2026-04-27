// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
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

	style             TextInputStyle
	hasError          bool
	focusBorderHidden bool
	supportTextValue  string
}

// OnValueChanged sets the event handler that is called when the text value changes.
// The handler receives the current text and whether the change is committed.
// A committed change occurs when the user presses Enter (for single-line text) or when the text input loses focus.
// An uncommitted change occurs on every keystroke or text modification during editing.
// Note that the handler might be called even when the text content has not actually changed.
func (t *TextInput) OnValueChanged(f func(context *guigui.Context, text string, committed bool)) {
	t.textInput.OnValueChanged(f)
}

func (t *TextInput) OnHandleButtonInput(f func(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult) {
	t.textInput.OnHandleButtonInput(f)
}

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

func (t *TextInput) ReplaceValueAtSelection(text string) {
	t.textInput.ReplaceValueAtSelection(text)
}

func (t *TextInput) CommitWithCurrentInputValue() {
	t.textInput.CommitWithCurrentInputValue()
}

func (t *TextInput) SetMultiline(multiline bool) {
	t.textInput.SetMultiline(multiline)
}

func (t *TextInput) SetHorizontalAlign(halign HorizontalAlign) {
	t.textInput.SetHorizontalAlign(halign)
}

func (t *TextInput) SetVerticalAlign(valign VerticalAlign) {
	t.textInput.SetVerticalAlign(valign)
}

func (t *TextInput) SetAutoWrap(autoWrap bool) {
	t.textInput.SetAutoWrap(autoWrap)
}

// SetCursorBlinking sets whether the cursor blinks.
// The default value is true.
func (t *TextInput) SetCursorBlinking(cursorBlinking bool) {
	t.textInput.SetCursorBlinking(cursorBlinking)
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

func (t *TextInput) SetTabular(tabular bool) {
	t.textInput.SetTabular(tabular)
}

func (t *TextInput) IsEditable() bool {
	return t.textInput.IsEditable()
}

func (t *TextInput) WriteStateKey(w *guigui.StateKeyWriter) {
	w.WriteUint64(uint64(t.style))
	w.WriteBool(t.hasError)
	w.WriteBool(t.focusBorderHidden)
	w.WriteString(t.supportTextValue)
}

// SetFocusBorderVisible sets whether the focus border is drawn around the
// text input when it has focus. The default is true. The focus border is
// always hidden for [TextInputStyleInline] regardless of this setting.
func (t *TextInput) SetFocusBorderVisible(visible bool) {
	t.focusBorderHidden = !visible
}

func (t *TextInput) SetStyle(style TextInputStyle) {
	if t.style == style {
		return
	}
	t.style = style
	t.textInput.SetStyle(style)
}

func (t *TextInput) SetEditable(editable bool) {
	t.textInput.SetEditable(editable)
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
		t.supportText.SetHorizontalAlign(t.textInput.text.Text().HorizontalAlign())
		if t.hasError {
			t.supportText.SetColor(basicwidgetdraw.TextColorFromSemanticColor(context.ColorMode(), basicwidgetdraw.SemanticColorDanger))
		} else {
			t.supportText.SetColor(basicwidgetdraw.TextColor(context.ColorMode(), false))
		}
	}

	return nil
}

func (t *TextInput) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	bounds := widgetBounds.Bounds()

	inputBounds := bounds
	if t.supportTextValue != "" {
		supportTextSize := t.supportText.Measure(context, guigui.FixedWidthConstraints(bounds.Dx()))
		inputBounds.Max.Y = bounds.Max.Y - supportTextSize.Y - int(2*context.Scale())
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
		supportTextBounds := image.Rectangle{
			Min: image.Pt(inputBounds.Min.X, inputBounds.Max.Y+int(2*context.Scale())),
			Max: image.Pt(inputBounds.Max.X, bounds.Max.Y),
		}
		layouter.LayoutWidget(&t.supportText, supportTextBounds)
	}
}

// measureTextInput returns the size of just the text input area, excluding the support text.
func (t *TextInput) measureTextInput(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.textInput.Measure(context, constraints)
}

func (t *TextInput) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	s := t.measureTextInput(context, constraints)
	if t.supportTextValue != "" {
		supportTextSize := t.supportText.Measure(context, guigui.FixedWidthConstraints(s.X))
		s.Y += supportTextSize.Y + int(2*context.Scale())
	}
	return s
}

func (t *TextInput) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	context.SetVisible(&t.focus, !t.focusBorderHidden && t.style != TextInputStyleInline && context.IsFocused(t.textInput.text.Text()))
	return nil
}

func (t *TextInput) setSelection(start, end int) {
	t.textInput.setSelection(start, end)
}

func (t *TextInput) setPaddingStart(padding int) {
	t.textInput.setPaddingStart(padding)
}

func (t *TextInput) setPaddingEnd(padding int) {
	t.textInput.setPaddingEnd(padding)
}

type textInput struct {
	guigui.DefaultWidget

	background     textInputBackground
	text           textInputText
	panel          Panel
	iconBackground textInputIconBackground
	icon           Image
	frame          textInputFrame

	style        TextInputStyle
	readonly     bool
	paddingStart int
	paddingEnd   int

	onTextScrollDelta func(context *guigui.Context, deltaX, deltaY float64)
}

func (t *textInput) OnValueChanged(f func(context *guigui.Context, text string, committed bool)) {
	t.text.Text().OnValueChanged(f)
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

func (t *textInput) ReplaceValueAtSelection(text string) {
	t.text.Text().ReplaceValueAtSelection(text)
}

func (t *textInput) CommitWithCurrentInputValue() {
	t.text.Text().CommitWithCurrentInputValue()
}

func (t *textInput) SetMultiline(multiline bool) {
	t.text.Text().SetMultiline(multiline)
}

func (t *textInput) SetHorizontalAlign(halign HorizontalAlign) {
	t.text.Text().SetHorizontalAlign(halign)
}

func (t *textInput) SetVerticalAlign(valign VerticalAlign) {
	t.text.Text().SetVerticalAlign(valign)
}

func (t *textInput) SetAutoWrap(autoWrap bool) {
	t.text.Text().SetAutoWrap(autoWrap)
}

func (t *textInput) SetCursorBlinking(cursorBlinking bool) {
	t.text.Text().SetCursorBlinking(cursorBlinking)
}

func (t *textInput) SetSelectionVisibleWhenUnfocused(visible bool) {
	t.text.Text().SetSelectionVisibleWhenUnfocused(visible)
}

func (t *textInput) SelectAll() {
	t.text.Text().selectAll()
}

func (t *textInput) Selection() (start, end int) {
	return t.text.Text().Selection()
}

func (t *textInput) SetSelection(start, end int) {
	t.text.Text().SetSelection(start, end)
}

func (t *textInput) SetTabular(tabular bool) {
	t.text.Text().SetTabular(tabular)
}

func (t *textInput) IsEditable() bool {
	return !t.readonly
}

func (t *textInput) WriteStateKey(w *guigui.StateKeyWriter) {
	w.WriteUint64(uint64(t.style))
	w.WriteBool(t.readonly)
	w.WriteInt64(int64(t.paddingStart))
	w.WriteInt64(int64(t.paddingEnd))
}

func (t *textInput) SetStyle(style TextInputStyle) {
	t.style = style
}

func (t *textInput) SetEditable(editable bool) {
	if t.readonly == !editable {
		return
	}
	t.readonly = !editable
	t.text.Text().SetEditable(editable)
}

func (t *textInput) setSelection(start, end int) {
	t.text.Text().setSelection(start, end, -1, false)
}

func (t *textInput) setPaddingStart(padding int) {
	t.paddingStart = padding
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
	switch t.style {
	case TextInputStyleNormal:
		start = u / 2
		end = u / 2
		if t.icon.HasImage() {
			start = u / 4
		}
		y = int(float64(min(widgetBounds.Bounds().Dy(), u))-float64(LineHeight(context))*t.text.Text().scale()) / 2
	case TextInputStyleInline:
		start = u / 4
		end = u / 4
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

	t.panel.SetContent(&t.text)
	t.panel.SetContentConstraints(PanelContentConstraintsFixedWidth)

	t.background.setEditable(!t.readonly)
	t.iconBackground.setEditable(!t.readonly)
	t.text.setEditable(!t.readonly)

	if t.onTextScrollDelta == nil {
		t.onTextScrollDelta = func(context *guigui.Context, deltaX, deltaY float64) {
			t.panel.ForceSetScrollOffsetByDelta(deltaX, deltaY)
		}
	}
	t.text.Text().onScrollDelta(t.onTextScrollDelta)

	t.panel.setScrolBarVisible(t.text.Text().IsMultiline())

	context.SetPassthrough(&t.frame, true)
	context.DelegateFocus(t, t.text.Text())

	return nil
}

func (t *textInput) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	padding := t.textInputPaddingInScrollableContent(context, widgetBounds)
	t.text.setContainerBounds(widgetBounds.Bounds())
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
	layouter.LayoutWidget(&t.panel, panelBounds)
}

func (t *textInput) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	u := UnitSize(context)
	if t.style == TextInputStyleInline {
		// WidgetBounds is not needed for inline text input.
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

func (t *textInputBackground) setEditable(editable bool) {
	if t.editable == editable {
		return
	}
	t.editable = editable
	guigui.RequestRedraw(t)
}

func (t *textInputBackground) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	clr := basicwidgetdraw.ControlColor(context.ColorMode(), context.IsEnabled(t) && t.editable)
	basicwidgetdraw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
}

type textInputIconBackground struct {
	guigui.DefaultWidget

	editable bool
}

func (t *textInputIconBackground) setEditable(editable bool) {
	if t.editable == editable {
		return
	}
	t.editable = editable
	guigui.RequestRedraw(t)
}

func (t *textInputIconBackground) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	clr := basicwidgetdraw.ControlColor(context.ColorMode(), context.IsEnabled(t) && t.editable)
	basicwidgetdraw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
}

type textInputFrame struct {
	guigui.DefaultWidget

	hasError bool
}

func (t *textInputFrame) setError(hasError bool) {
	if t.hasError == hasError {
		return
	}
	t.hasError = hasError
	guigui.RequestRedraw(t)
}

func (t *textInputFrame) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	clr1, clr2 := basicwidgetdraw.BorderColors(context.ColorMode(), basicwidgetdraw.RoundedRectBorderTypeInset)
	basicwidgetdraw.DrawRoundedRectBorder(context, dst, bounds, clr1, clr2, RoundedCornerRadius(context), float32(1*context.Scale()), basicwidgetdraw.RoundedRectBorderTypeInset)
	if t.hasError {
		dclr1, dclr2 := basicwidgetdraw.BorderDangerColors(context.ColorMode())
		basicwidgetdraw.DrawRoundedRectBorder(context, dst, bounds, dclr1, dclr2, RoundedCornerRadius(context), float32(1*context.Scale()), basicwidgetdraw.RoundedRectBorderTypeRegular)
	}
}

type textInputText struct {
	guigui.DefaultWidget

	text roundedCornerWidget[*Text]

	editable        bool
	containerBounds image.Rectangle
	padding         guigui.Padding
}

func (t *textInputText) setEditable(editable bool) {
	t.text.Widget().SetEditable(editable)
}

func (t *textInputText) WriteStateKey(w *guigui.StateKeyWriter) {
	writeRectangle(w, t.containerBounds)
	writePadding(w, t.padding)
}

func (t *textInputText) setContainerBounds(bounds image.Rectangle) {
	t.containerBounds = bounds
}

func (t *textInputText) setPadding(padding guigui.Padding) {
	if t.padding == padding {
		return
	}
	t.padding = padding
	t.text.Widget().setPaddingForScrollOffset(padding)
}

func (t *textInputText) Text() *Text {
	return t.text.Widget()
}

func (t *textInputText) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.text)

	t.text.Widget().SetSelectable(true)
	t.text.Widget().SetColor(basicwidgetdraw.TextColor(context.ColorMode(), context.IsEnabled(t)))
	t.text.Widget().setKeepTailingSpace(!t.text.Widget().autoWrap)

	context.DelegateFocus(t, t.text.Widget())

	return nil
}

func (t *textInputText) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	// guigui.LinearLayout cannot treat auto-wrapping texts very well.
	// Calculate the layout directly here.
	bounds := widgetBounds.Bounds()
	bounds.Min.X += t.padding.Start
	bounds.Min.Y += t.padding.Top
	bounds.Max.X -= t.padding.End
	bounds.Max.Y -= t.padding.Bottom

	// As the text is rendered in an inset box, shift the text bounds down by 0.5 pixel.
	bounds = bounds.Add(image.Pt(0, int(0.5*context.Scale())))
	layouter.LayoutWidget(&t.text, bounds)

	t.text.SetRenderingBounds(t.containerBounds)
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
	clr := draw.Color(context.ColorMode(), draw.SemanticColorAccent, 0.8)
	basicwidgetdraw.DrawRoundedRectBorder(context, dst, bounds, clr, clr, w+RoundedCornerRadius(context), float32(w), basicwidgetdraw.RoundedRectBorderTypeRegular)
}
