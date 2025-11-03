// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

const (
	textInputEventTextAndSelectionChanged = "textAndSelectionChanged"
)

type TextInputStyle int

const (
	TextInputStyleNormal TextInputStyle = iota
	TextInputStyleInline
)

type TextInput struct {
	guigui.DefaultWidget

	background     textInputBackground
	text           Text
	iconBackground textInputIconBackground
	icon           Image
	frame          textInputFrame
	scrollOverlay  scrollOverlay
	focus          textInputFocus

	style        TextInputStyle
	readonly     bool
	paddingStart int
	paddingEnd   int

	prevFocused bool
}

func (t *TextInput) SetOnValueChanged(f func(text string, committed bool)) {
	t.text.SetOnValueChanged(f)
}

func (t *TextInput) SetOnKeyJustPressed(f func(key ebiten.Key) (handled bool)) {
	t.text.SetOnKeyJustPressed(f)
}

func (t *TextInput) SetOnTextAndSelectionChanged(f func(text string, start, end int)) {
	guigui.RegisterEventHandler(t, textInputEventTextAndSelectionChanged, f)
}

func (t *TextInput) Value() string {
	return t.text.Value()
}

func (t *TextInput) SetValue(text string) {
	t.text.SetValue(text)
}

func (t *TextInput) ForceSetValue(text string) {
	t.text.ForceSetValue(text)
}

func (t *TextInput) ReplaceValueAtSelection(text string) {
	t.text.ReplaceValueAtSelection(text)
}

func (t *TextInput) CommitWithCurrentInputValue() {
	t.text.CommitWithCurrentInputValue()
}

func (t *TextInput) SetMultiline(multiline bool) {
	t.text.SetMultiline(multiline)
}

func (t *TextInput) SetHorizontalAlign(halign HorizontalAlign) {
	t.text.SetHorizontalAlign(halign)
}

func (t *TextInput) SetVerticalAlign(valign VerticalAlign) {
	t.text.SetVerticalAlign(valign)
}

func (t *TextInput) SetAutoWrap(autoWrap bool) {
	t.text.SetAutoWrap(autoWrap)
}

func (t *TextInput) SelectAll() {
	t.text.selectAll()
}

func (t *TextInput) SetTabular(tabular bool) {
	t.text.SetTabular(tabular)
}

func (t *TextInput) IsEditable() bool {
	return !t.readonly
}

func (t *TextInput) SetStyle(style TextInputStyle) {
	if t.style == style {
		return
	}
	t.style = style
	guigui.RequestRedraw(t)
}

func (t *TextInput) SetEditable(editable bool) {
	if t.readonly == !editable {
		return
	}
	t.readonly = !editable
	t.text.SetEditable(editable)
	guigui.RequestRedraw(t)
}

func (t *TextInput) setPaddingStart(padding int) {
	if t.paddingStart == padding {
		return
	}
	t.paddingStart = padding
	guigui.RequestRedraw(t)
}

func (t *TextInput) setPaddingEnd(padding int) {
	if t.paddingEnd == padding {
		return
	}
	t.paddingEnd = padding
	guigui.RequestRedraw(t)
}

func (t *TextInput) SetIcon(icon *ebiten.Image) {
	t.icon.SetImage(icon)
}

func (t *TextInput) textInputPaddingInScrollableContent(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (start, top, end, bottom int) {
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

func (t *TextInput) scrollContentSize(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Point {
	start, top, end, bottom := t.textInputPaddingInScrollableContent(context, widgetBounds)
	w := widgetBounds.Bounds().Dx() - start - end
	return t.text.Measure(context, guigui.FixedWidthConstraints(w)).Add(image.Pt(start+end, top+bottom))
}

func (t *TextInput) isFocused(context *guigui.Context) bool {
	return context.IsFocused(t) || context.IsFocused(&t.text)
}

func (t *TextInput) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&t.background)
	adder.AddChild(&t.text)
	if t.icon.HasImage() {
		adder.AddChild(&t.iconBackground)
		adder.AddChild(&t.icon)
	}
	adder.AddChild(&t.frame)
	adder.AddChild(&t.scrollOverlay)
	if t.style != TextInputStyleInline && (context.IsFocused(t) || context.IsFocused(&t.text)) {
		adder.AddChild(&t.focus)
	}
}

func (t *TextInput) textBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
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

func (t *TextInput) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if t.prevFocused != (context.IsFocused(t) || context.IsFocused(&t.text)) {
		t.prevFocused = (context.IsFocused(t) || context.IsFocused(&t.text))
		guigui.RequestRedraw(t)
	}

	t.scrollOverlay.SetContentSize(context, widgetBounds, t.scrollContentSize(context, widgetBounds))

	t.background.textInput = t

	t.text.SetEditable(!t.readonly)
	t.text.SetSelectable(true)
	t.text.SetColor(draw.TextColor(context.ColorMode(), context.IsEnabled(t)))
	t.text.setKeepTailingSpace(!t.text.autoWrap)

	// TODO: The cursor position might be unstable when the text horizontal align is center or right. Fix this.
	t.adjustScrollOffsetIfNeeded(context, widgetBounds)

	if draw.OverlapsWithRoundedCorner(widgetBounds.Bounds(), RoundedCornerRadius(context), t.textBounds(context, widgetBounds)) {
		// CustomDraw might be too generic and overkill for this case.
		context.SetCustomDraw(&t.text, func(dst, widgetImage *ebiten.Image, op *ebiten.DrawImageOptions) {
			draw.DrawInRoundedCornerRect(context, dst, widgetBounds.Bounds(), RoundedCornerRadius(context), widgetImage, op)
		})
	} else {
		context.SetCustomDraw(&t.text, nil)
	}

	// Focusing the text widget works only after appending it.
	if context.IsFocused(t) {
		context.SetFocused(&t.text, true)
	}

	if t.icon.HasImage() {
		t.iconBackground.textInput = t
	}

	context.SetVisible(&t.scrollOverlay, t.text.IsMultiline())

	if t.style != TextInputStyleInline && (context.IsFocused(t) || context.IsFocused(&t.text)) {
		t.focus.textInput = t
	}

	return nil
}

func (t *TextInput) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, widget guigui.Widget) image.Rectangle {
	switch widget {
	case &t.background:
		return widgetBounds.Bounds()
	case &t.text:
		return t.textBounds(context, widgetBounds)
	case &t.iconBackground, &t.icon:
		b := widgetBounds.Bounds()
		iconSize := defaultIconSize(context)
		var imgBounds image.Rectangle
		imgBounds.Min = b.Min.Add(image.Point{
			X: UnitSize(context)/4 + int(0.5*context.Scale()),
			Y: (b.Dy() - iconSize) / 2,
		})
		imgBounds.Max = imgBounds.Min.Add(image.Pt(iconSize, iconSize))
		if widget == &t.icon {
			return imgBounds
		}

		imgBgBounds := b
		imgBgBounds.Max.X = imgBounds.Max.X + UnitSize(context)/4
		return imgBgBounds
	case &t.frame:
		return widgetBounds.Bounds()
	case &t.scrollOverlay:
		return widgetBounds.Bounds()
	case &t.focus:
		w := textInputFocusBorderWidth(context)
		p := widgetBounds.Bounds().Min.Add(image.Pt(-w, -w))
		s := widgetBounds.Bounds().Size().Add(image.Pt(2*textInputFocusBorderWidth(context), 2*textInputFocusBorderWidth(context)))
		return image.Rectangle{
			Min: p,
			Max: p.Add(s),
		}
	}
	return image.Rectangle{}
}

func (t *TextInput) adjustScrollOffsetIfNeeded(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
	bounds := widgetBounds.Bounds()
	paddingStart, paddingTop, paddingEnd, paddingBottom := t.textInputPaddingInScrollableContent(context, widgetBounds)
	bounds.Max.X -= paddingEnd
	bounds.Min.X += paddingStart
	bounds.Max.Y -= paddingBottom
	bounds.Min.Y += paddingTop

	dx, dy := t.text.adjustScrollOffset(context, bounds, t.textBounds(context, widgetBounds))
	t.scrollOverlay.SetOffsetByDelta(context, widgetBounds, t.scrollContentSize(context, widgetBounds), dx, dy)
}

func (t *TextInput) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	cp := image.Pt(ebiten.CursorPosition())
	if context.IsWidgetHitAtCursor(t) {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			t.text.handleClick(context, t.textBounds(context, widgetBounds), cp)
			return guigui.HandleInputByWidget(t)
		}
	}
	return t.scrollOverlay.handlePointingInput(context, widgetBounds)
}

func (t *TextInput) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	return t.text.CursorShape(context, nil)
}

func (t *TextInput) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
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

type textInputBackground struct {
	guigui.DefaultWidget

	textInput *TextInput
}

func (t *textInputBackground) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	clr := draw.ControlColor(context.ColorMode(), context.IsEnabled(t) && t.textInput.IsEditable())
	draw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
}

type textInputIconBackground struct {
	guigui.DefaultWidget

	textInput *TextInput
}

func (t *textInputIconBackground) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	clr := draw.ControlColor(context.ColorMode(), context.IsEnabled(t) && t.textInput.IsEditable())
	draw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
}

type textInputFrame struct {
	guigui.DefaultWidget
}

func (t *textInputFrame) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	clr1, clr2 := draw.BorderColors(context.ColorMode(), draw.RoundedRectBorderTypeInset, false)
	draw.DrawRoundedRectBorder(context, dst, bounds, clr1, clr2, RoundedCornerRadius(context), float32(1*context.Scale()), draw.RoundedRectBorderTypeInset)
}

func (t *textInputFrame) PassThrough() bool {
	return true
}

func textInputFocusBorderWidth(context *guigui.Context) int {
	return int(4 * context.Scale())
}

type textInputFocus struct {
	guigui.DefaultWidget

	textInput *TextInput
}

func (t *textInputFocus) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	w := textInputFocusBorderWidth(context)
	clr := draw.Color(context.ColorMode(), draw.ColorTypeAccent, 0.8)
	draw.DrawRoundedRectBorder(context, dst, bounds, clr, clr, w+RoundedCornerRadius(context), float32(w), draw.RoundedRectBorderTypeRegular)
}

func (t *textInputFocus) ZDelta() int {
	return 1
}

func (t *textInputFocus) PassThrough() bool {
	return true
}
