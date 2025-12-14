// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

type TextInputEventArgsValueChanged struct {
	Value     string
	Committed bool
}

type TextInputEventArgsKeyJustPressed struct {
	Key ebiten.Key
}

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
	guigui.RequestRedraw(t)
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

func (t *TextInput) Cut() bool {
	return t.textInput.Cut()
}

func (t *TextInput) Copy() bool {
	return t.textInput.Copy()
}

func (t *TextInput) Paste() bool {
	return t.textInput.Paste()
}

func (t *TextInput) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&t.textInput)
	adder.AddChild(&t.focus)
	context.SetContainer(&t.textInput, true)
	context.SetPassThrough(&t.focus, true)
	context.SetFloat(&t.focus, true)
	context.DelegateFocus(t, &t.textInput.text)
	guigui.RegisterEventHandler2(t, &t.textInput.text)
	return nil
}

func (t *TextInput) HandleEvent(context *guigui.Context, targetWidget guigui.Widget, eventArgs any) {
	if targetWidget == &t.textInput.text {
		switch eventArgs := eventArgs.(type) {
		case *TextEventArgsValueChanged:
			guigui.DispatchEventHandler2(t, &TextInputEventArgsValueChanged{
				Value:     eventArgs.Value,
				Committed: eventArgs.Committed,
			})
		case *TextEventArgsKeyJustPressed:
			guigui.DispatchEventHandler2(t, &TextInputEventArgsKeyJustPressed{
				Key: eventArgs.Key,
			})
		}
	}
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
	text           Text
	iconBackground textInputIconBackground
	icon           Image
	frame          textInputFrame
	scrollOverlay  scrollOverlay

	style        TextInputStyle
	readonly     bool
	paddingStart int
	paddingEnd   int
}

func (t *textInput) Value() string {
	return t.text.Value()
}

func (t *textInput) SetValue(text string) {
	t.text.SetValue(text)
}

func (t *textInput) ForceSetValue(text string) {
	t.text.ForceSetValue(text)
}

func (t *textInput) ReplaceValueAtSelection(text string) {
	t.text.ReplaceValueAtSelection(text)
}

func (t *textInput) CommitWithCurrentInputValue() {
	t.text.CommitWithCurrentInputValue()
}

func (t *textInput) SetMultiline(multiline bool) {
	t.text.SetMultiline(multiline)
}

func (t *textInput) SetHorizontalAlign(halign HorizontalAlign) {
	t.text.SetHorizontalAlign(halign)
}

func (t *textInput) SetVerticalAlign(valign VerticalAlign) {
	t.text.SetVerticalAlign(valign)
}

func (t *textInput) SetAutoWrap(autoWrap bool) {
	t.text.SetAutoWrap(autoWrap)
}

func (t *textInput) SelectAll() {
	t.text.selectAll()
}

func (t *textInput) SetTabular(tabular bool) {
	t.text.SetTabular(tabular)
}

func (t *textInput) IsEditable() bool {
	return !t.readonly
}

func (t *textInput) SetStyle(style TextInputStyle) {
	if t.style == style {
		return
	}
	t.style = style
	guigui.RequestRedraw(t)
}

func (t *textInput) SetEditable(editable bool) {
	if t.readonly == !editable {
		return
	}
	t.readonly = !editable
	t.text.SetEditable(editable)
	guigui.RequestRedraw(t)
}

func (t *textInput) setPaddingStart(padding int) {
	if t.paddingStart == padding {
		return
	}
	t.paddingStart = padding
	guigui.RequestRedraw(t)
}

func (t *textInput) setPaddingEnd(padding int) {
	if t.paddingEnd == padding {
		return
	}
	t.paddingEnd = padding
	guigui.RequestRedraw(t)
}

func (t *textInput) SetIcon(icon *ebiten.Image) {
	t.icon.SetImage(icon)
}

func (t *textInput) textInputPaddingInScrollableContent(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (start, top, end, bottom int) {
	var x, y int
	switch t.style {
	case TextInputStyleNormal:
		x = UnitSize(context) / 2
		y = int(float64(min(widgetBounds.Bounds().Dy(), UnitSize(context)))-LineHeight(context)*(t.text.scaleMinus1+1)) / 2
	case TextInputStyleInline:
		x = UnitSize(context) / 4
	}
	start = x + t.paddingStart
	if t.icon.HasImage() {
		start += defaultIconSize(context)
	}
	top = y
	end = x + t.paddingEnd
	bottom = y
	return
}

func (t *textInput) scrollContentSize(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Point {
	start, top, end, bottom := t.textInputPaddingInScrollableContent(context, widgetBounds)
	w := widgetBounds.Bounds().Dx() - start - end
	return t.text.Measure(context, guigui.FixedWidthConstraints(w)).Add(image.Pt(start+end, top+bottom))
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

	t.background.textInput = t

	t.text.SetEditable(!t.readonly)
	t.text.SetSelectable(true)
	t.text.SetColor(basicwidgetdraw.TextColor(context.ColorMode(), context.IsEnabled(t)))
	t.text.setKeepTailingSpace(!t.text.autoWrap)

	if t.icon.HasImage() {
		t.iconBackground.textInput = t
	}

	context.SetVisible(&t.scrollOverlay, t.text.IsMultiline())
	context.SetPassThrough(&t.frame, true)
	context.DelegateFocus(t, &t.text)

	return nil
}

func (t *textInput) textBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	paddingStart, paddingTop, paddingEnd, paddingBottom := t.textInputPaddingInScrollableContent(context, widgetBounds)
	bt := widgetBounds.Bounds()
	pt := bt.Min
	s := t.text.Measure(context, guigui.FixedWidthConstraints(bt.Dx()-paddingStart-paddingEnd))
	s.X = max(s.X, bt.Dx()-paddingStart-paddingEnd)
	s.Y = max(s.Y, bt.Dy()-paddingTop-paddingBottom)
	b := image.Rectangle{
		Min: pt,
		Max: pt.Add(s),
	}
	b = b.Add(image.Pt(paddingStart, paddingTop))

	// As the text is rendered in an inset box, shift the text bounds down by 0.5 pixel.
	b = b.Add(image.Pt(0, int(0.5*context.Scale())))

	offsetX, offsetY := t.scrollOverlay.Offset()
	b = b.Add(image.Pt(int(offsetX), int(offsetY)))

	return b
}

func (t *textInput) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	t.scrollOverlay.SetContentSize(context, widgetBounds, t.scrollContentSize(context, widgetBounds))
	// TODO: The cursor position might be unstable when the text horizontal align is center or right. Fix this.
	t.adjustScrollOffsetIfNeeded(context, widgetBounds)

	bounds := widgetBounds.Bounds()

	if draw.OverlapsWithRoundedCorner(bounds, RoundedCornerRadius(context), t.textBounds(context, widgetBounds)) {
		// CustomDraw might be too generic and overkill for this case.
		context.SetCustomDraw(&t.text, func(dst, widgetImage *ebiten.Image, op *ebiten.DrawImageOptions) {
			draw.DrawInRoundedCornerRect(context, dst, widgetBounds.Bounds(), RoundedCornerRadius(context), widgetImage, op)
		})
	} else {
		context.SetCustomDraw(&t.text, nil)
	}

	layouter.LayoutWidget(&t.background, bounds)
	layouter.LayoutWidget(&t.frame, bounds)
	layouter.LayoutWidget(&t.scrollOverlay, bounds)
	layouter.LayoutWidget(&t.text, t.textBounds(context, widgetBounds))

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

func (t *textInput) adjustScrollOffsetIfNeeded(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
	bounds := widgetBounds.Bounds()
	paddingStart, paddingTop, paddingEnd, paddingBottom := t.textInputPaddingInScrollableContent(context, widgetBounds)
	bounds.Max.X -= paddingEnd
	bounds.Min.X += paddingStart
	bounds.Max.Y -= paddingBottom
	bounds.Min.Y += paddingTop

	dx, dy := t.text.adjustScrollOffset(context, bounds, t.textBounds(context, widgetBounds))
	t.scrollOverlay.SetOffsetByDelta(context, widgetBounds, t.scrollContentSize(context, widgetBounds), dx, dy)
}

func (t *textInput) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if widgetBounds.IsHitAtCursor() {
		left := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
		right := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
		if left || right {
			t.text.handleClick(context, t.textBounds(context, widgetBounds), image.Pt(ebiten.CursorPosition()), left)
			if left {
				return guigui.HandleInputByWidget(t)
			}
			return guigui.HandleInputResult{}
		}
	}
	return t.scrollOverlay.handlePointingInput(context, widgetBounds)
}

func (t *textInput) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	return t.text.CursorShape(context, nil)
}

func (t *textInput) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	if t.style == TextInputStyleInline {
		// WidgetBounds is not needed for inline text input.
		start, _, end, _ := t.textInputPaddingInScrollableContent(context, nil)
		if fixedWidth, ok := constraints.FixedWidth(); ok {
			constraints = guigui.FixedWidthConstraints(fixedWidth - start - end)
		}
		s := t.text.Measure(context, constraints)
		w := max(s.X+start+end, UnitSize(context))
		h := s.Y
		return image.Pt(w, h)
	}
	if t.text.IsMultiline() {
		return image.Pt(6*UnitSize(context), 4*UnitSize(context))
	}
	return image.Pt(6*UnitSize(context), UnitSize(context))
}

func (t *textInput) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	return nil
}

func (t *textInput) CanCut() bool {
	return t.text.CanCut()
}

func (t *textInput) CanCopy() bool {
	return t.text.CanCopy()
}

func (t *textInput) CanPaste() bool {
	return t.text.CanPaste()
}

func (t *textInput) Cut() bool {
	return t.text.Cut()
}

func (t *textInput) Copy() bool {
	return t.text.Copy()
}

func (t *textInput) Paste() bool {
	return t.text.Paste()
}

type textInputBackground struct {
	guigui.DefaultWidget

	textInput *textInput
}

func (t *textInputBackground) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	clr := basicwidgetdraw.ControlColor(context.ColorMode(), context.IsEnabled(t) && t.textInput.IsEditable())
	basicwidgetdraw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
}

type textInputIconBackground struct {
	guigui.DefaultWidget

	textInput *textInput
}

func (t *textInputIconBackground) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	clr := basicwidgetdraw.ControlColor(context.ColorMode(), context.IsEnabled(t) && t.textInput.IsEditable())
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
