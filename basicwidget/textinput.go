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

	textInput textInput
	focus     textInputFocus

	style TextInputStyle
}

func (t *TextInput) SetOnValueChanged(f func(context *guigui.Context, text string, committed bool)) {
	t.textInput.SetOnValueChanged(f)
}

func (t *TextInput) SetOnKeyJustPressed(f func(context *guigui.Context, key ebiten.Key)) {
	t.textInput.SetOnKeyJustPressed(f)
}

func (t *TextInput) Value() string {
	return t.textInput.Value()
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

func (t *TextInput) SelectAll() {
	t.textInput.SelectAll()
}

func (t *TextInput) SetTabular(tabular bool) {
	t.textInput.SetTabular(tabular)
}

func (t *TextInput) IsEditable() bool {
	return t.textInput.IsEditable()
}

func (t *TextInput) SetStyle(style TextInputStyle) {
	if t.style == style {
		return
	}
	t.style = style
	t.textInput.SetStyle(style)
	guigui.RequestRebuild(t)
}

func (t *TextInput) SetEditable(editable bool) {
	t.textInput.SetEditable(editable)
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
	adder.AddChild(&t.textInput)
	adder.AddChild(&t.focus)
	context.SetContainer(&t.textInput, true)
	context.SetPassThrough(&t.focus, true)
	context.SetFloat(&t.focus, true)
	context.DelegateFocus(t, &t.textInput.text)
	return nil
}

func (t *TextInput) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&t.textInput, widgetBounds.Bounds())

	bounds := widgetBounds.Bounds()
	w := textInputFocusBorderWidth(context)
	p := bounds.Min.Add(image.Pt(-w, -w))
	s := bounds.Size().Add(image.Pt(2*w, 2*w))
	layouter.LayoutWidget(&t.focus, image.Rectangle{
		Min: p,
		Max: p.Add(s),
	})
}

func (t *TextInput) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.textInput.Measure(context, constraints)
}

func (t *TextInput) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	context.SetVisible(&t.focus, t.style != TextInputStyleInline && context.IsFocused(&t.textInput.text))
	return nil
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
	iconBackground textInputIconBackground
	icon           Image
	frame          textInputFrame
	scrollOverlay  scrollOverlay

	style        TextInputStyle
	readonly     bool
	paddingStart int
	paddingEnd   int

	onTextScroll func(context *guigui.Context, deltaX, deltaY float64)
	scrollDeltaX float64
	scrollDeltaY float64
}

func (t *textInput) SetOnValueChanged(f func(context *guigui.Context, text string, committed bool)) {
	t.text.Text().SetOnValueChanged(f)
}

func (t *textInput) SetOnKeyJustPressed(f func(context *guigui.Context, key ebiten.Key)) {
	t.text.Text().SetOnKeyJustPressed(f)
}

func (t *textInput) Value() string {
	return t.text.Text().Value()
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

func (t *textInput) SelectAll() {
	t.text.Text().selectAll()
}

func (t *textInput) SetTabular(tabular bool) {
	t.text.Text().SetTabular(tabular)
}

func (t *textInput) IsEditable() bool {
	return !t.readonly
}

func (t *textInput) SetStyle(style TextInputStyle) {
	if t.style == style {
		return
	}
	t.style = style
	guigui.RequestRebuild(t)
}

func (t *textInput) SetEditable(editable bool) {
	if t.readonly == !editable {
		return
	}
	t.readonly = !editable
	t.text.Text().SetEditable(editable)
	guigui.RequestRebuild(t)
}

func (t *textInput) setPaddingStart(padding int) {
	if t.paddingStart == padding {
		return
	}
	t.paddingStart = padding
	guigui.RequestRebuild(t)
}

func (t *textInput) setPaddingEnd(padding int) {
	if t.paddingEnd == padding {
		return
	}
	t.paddingEnd = padding
	guigui.RequestRebuild(t)
}

func (t *textInput) SetIcon(icon *ebiten.Image) {
	t.icon.SetImage(icon)
}

func (t *textInput) textInputPaddingInScrollableContent(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.Padding {
	var x, y int
	switch t.style {
	case TextInputStyleNormal:
		x = UnitSize(context) / 2
		y = int(float64(min(widgetBounds.Bounds().Dy(), UnitSize(context)))-LineHeight(context)*t.text.Text().scale()) / 2
	case TextInputStyleInline:
		x = UnitSize(context) / 4
	}
	start := x + t.paddingStart
	if t.icon.HasImage() {
		start += defaultIconSize(context)
	}
	return guigui.Padding{
		Start:  start,
		Top:    y,
		End:    x + t.paddingEnd,
		Bottom: y,
	}
}

func (t *textInput) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&t.background)
	adder.AddChild(&t.text)
	if t.icon.HasImage() {
		adder.AddChild(&t.iconBackground)
		adder.AddChild(&t.icon)
	}
	adder.AddChild(&t.frame)
	adder.AddChild(&t.scrollOverlay)

	t.background.setEditable(!t.readonly)
	t.iconBackground.setEditable(!t.readonly)
	t.text.setEditable(!t.readonly)

	if t.onTextScroll == nil {
		t.onTextScroll = func(context *guigui.Context, deltaX, deltaY float64) {
			t.scrollDeltaX += deltaX
			t.scrollDeltaY += deltaY
		}
	}
	t.text.Text().setOnScroll(t.onTextScroll)

	context.SetVisible(&t.scrollOverlay, t.text.Text().IsMultiline())
	context.SetPassThrough(&t.frame, true)
	context.DelegateFocus(t, t.text.Text())

	return nil
}

func (t *textInput) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	padding := t.textInputPaddingInScrollableContent(context, widgetBounds)
	t.text.setContainerBounds(widgetBounds.Bounds())
	t.text.setPadding(padding)
	s := t.text.Measure(context, guigui.FixedWidthConstraints(widgetBounds.Bounds().Dx()))
	t.scrollOverlay.SetContentSize(context, widgetBounds, s)

	bounds := widgetBounds.Bounds()

	textBounds := image.Rectangle{
		Min: bounds.Min,
		Max: bounds.Min.Add(s),
	}
	offsetX, offsetY := t.scrollOverlay.Offset()
	textBounds = textBounds.Add(image.Pt(int(offsetX), int(offsetY)))

	layouter.LayoutWidget(&t.background, bounds)
	layouter.LayoutWidget(&t.frame, bounds)
	layouter.LayoutWidget(&t.scrollOverlay, bounds)
	layouter.LayoutWidget(&t.text, textBounds)

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
	}
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

func (t *textInput) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if t.scrollDeltaX != 0 || t.scrollDeltaY != 0 {
		s := t.text.Measure(context, guigui.FixedWidthConstraints(widgetBounds.Bounds().Dx()))
		t.scrollOverlay.SetOffsetByDelta(context, widgetBounds, s, t.scrollDeltaX, t.scrollDeltaY)
		t.scrollDeltaX = 0
		t.scrollDeltaY = 0
	}
	return nil
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
}

func (t *textInputFrame) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	clr1, clr2 := basicwidgetdraw.BorderColors(context.ColorMode(), basicwidgetdraw.RoundedRectBorderTypeInset, false)
	basicwidgetdraw.DrawRoundedRectBorder(context, dst, bounds, clr1, clr2, RoundedCornerRadius(context), float32(1*context.Scale()), basicwidgetdraw.RoundedRectBorderTypeInset)
}

type textInputText struct {
	guigui.DefaultWidget

	text Text

	editable        bool
	containerBounds image.Rectangle
	padding         guigui.Padding
}

func (t *textInputText) setEditable(editable bool) {
	t.text.SetEditable(editable)
}

func (t *textInputText) setContainerBounds(bounds image.Rectangle) {
	if t.containerBounds == bounds {
		return
	}
	t.containerBounds = bounds
	guigui.RequestRebuild(t)
}

func (t *textInputText) setPadding(padding guigui.Padding) {
	if t.padding == padding {
		return
	}
	t.padding = padding
	t.text.setPaddingForScrollOffset(padding)
	guigui.RequestRebuild(t)
}

func (t *textInputText) Text() *Text {
	return &t.text
}

func (t *textInputText) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&t.text)

	t.text.SetSelectable(true)
	t.text.SetColor(basicwidgetdraw.TextColor(context.ColorMode(), context.IsEnabled(t)))
	t.text.setKeepTailingSpace(!t.text.autoWrap)

	context.DelegateFocus(t, &t.text)

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

	if draw.OverlapsWithRoundedCorner(t.containerBounds, RoundedCornerRadius(context), bounds) {
		// CustomDraw might be too generic and overkill for this case.
		context.SetCustomDraw(&t.text, func(dst, widgetImage *ebiten.Image, op *ebiten.DrawImageOptions) {
			draw.DrawInRoundedCornerRect(context, dst, t.containerBounds, RoundedCornerRadius(context), widgetImage, op)
		})
	} else {
		context.SetCustomDraw(&t.text, nil)
	}
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
	clr := draw.Color(context.ColorMode(), draw.ColorTypeAccent, 0.8)
	basicwidgetdraw.DrawRoundedRectBorder(context, dst, bounds, clr, clr, w+RoundedCornerRadius(context), float32(w), basicwidgetdraw.RoundedRectBorderTypeRegular)
}
