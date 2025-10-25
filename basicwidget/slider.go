// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"
	"math"
	"math/big"

	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
)

type Slider struct {
	guigui.DefaultWidget

	abstractNumberInput abstractNumberInput

	dragging           bool
	draggingStartValue big.Int
	draggingStartX     int

	prevThumbHovered bool
}

func (s *Slider) SetOnValueChanged(f func(value int)) {
	if f == nil {
		s.abstractNumberInput.SetOnValueChanged(s, nil)
		return
	}
	s.abstractNumberInput.SetOnValueChanged(s, func(value int, committed bool) {
		f(value)
	})
}

func (s *Slider) SetOnValueChangedBigInt(f func(value *big.Int)) {
	if f == nil {
		s.abstractNumberInput.SetOnValueChangedBigInt(s, nil)
		return
	}
	s.abstractNumberInput.SetOnValueChangedBigInt(s, func(value *big.Int, committed bool) {
		f(value)
	})
}

func (s *Slider) SetOnValueChangedInt64(f func(value int64)) {
	if f == nil {
		s.abstractNumberInput.SetOnValueChangedInt64(s, nil)
		return
	}
	s.abstractNumberInput.SetOnValueChangedInt64(s, func(value int64, committed bool) {
		f(value)
	})
}

func (s *Slider) SetOnValueChangedUint64(f func(value uint64)) {
	if f == nil {
		s.abstractNumberInput.SetOnValueChangedUint64(s, nil)
		return
	}
	s.abstractNumberInput.SetOnValueChangedUint64(s, func(value uint64, committed bool) {
		f(value)
	})
}

func (s *Slider) Value() int {
	return s.abstractNumberInput.Value()
}

func (s *Slider) ValueBigInt() *big.Int {
	return s.abstractNumberInput.ValueBigInt()
}

func (s *Slider) ValueInt64() int64 {
	return s.abstractNumberInput.ValueInt64()
}

func (s *Slider) ValueUint64() uint64 {
	return s.abstractNumberInput.ValueUint64()
}

func (s *Slider) SetValue(value int) {
	changed := value != s.abstractNumberInput.Value()
	s.abstractNumberInput.SetValue(s, value, true)
	if changed {
		guigui.RequestRedraw(s)
	}
}

func (s *Slider) SetValueBigInt(value *big.Int) {
	changed := value.Cmp(s.abstractNumberInput.ValueBigInt()) != 0
	s.abstractNumberInput.SetValueBigInt(s, value, true)
	if changed {
		guigui.RequestRedraw(s)
	}
}

func (s *Slider) SetValueInt64(value int64) {
	changed := value != s.abstractNumberInput.ValueInt64()
	s.abstractNumberInput.SetValueInt64(s, value, true)
	if changed {
		guigui.RequestRedraw(s)
	}
}

func (s *Slider) SetValueUint64(value uint64) {
	changed := value != s.abstractNumberInput.ValueUint64()
	s.abstractNumberInput.SetValueUint64(s, value, true)
	if changed {
		guigui.RequestRedraw(s)
	}
}

func (s *Slider) MinimumValueBigInt() *big.Int {
	return s.abstractNumberInput.MinimumValueBigInt()
}

func (s *Slider) SetMinimumValue(minimum int) {
	s.abstractNumberInput.SetMinimumValue(s, minimum)
}

func (s *Slider) SetMinimumValueBigInt(minimum *big.Int) {
	s.abstractNumberInput.SetMinimumValueBigInt(s, minimum)
}

func (s *Slider) SetMinimumValueInt64(minimum int64) {
	s.abstractNumberInput.SetMinimumValueInt64(s, minimum)
}

func (s *Slider) SetMinimumValueUint64(minimum uint64) {
	s.abstractNumberInput.SetMinimumValueUint64(s, minimum)
}

func (s *Slider) MaximumValueBigInt() *big.Int {
	return s.abstractNumberInput.MaximumValueBigInt()
}

func (s *Slider) SetMaximumValue(maximum int) {
	s.abstractNumberInput.SetMaximumValue(s, maximum)
}

func (s *Slider) SetMaximumValueBigInt(maximum *big.Int) {
	s.abstractNumberInput.SetMaximumValueBigInt(s, maximum)
}

func (s *Slider) SetMaximumValueInt64(maximum int64) {
	s.abstractNumberInput.SetMaximumValueInt64(s, maximum)
}

func (s *Slider) SetMaximumValueUint64(maximum uint64) {
	s.abstractNumberInput.SetMaximumValueUint64(s, maximum)
}

func (s *Slider) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if hovered := s.isThumbHovered(context, widgetBounds); s.prevThumbHovered != hovered {
		s.prevThumbHovered = hovered
		guigui.RequestRedraw(s)
	}
	return nil
}

func (s *Slider) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	max := s.abstractNumberInput.MaximumValueBigInt()
	min := s.abstractNumberInput.MinimumValueBigInt()
	if max == nil || min == nil {
		return guigui.HandleInputResult{}
	}

	if context.IsEnabled(s) && context.IsWidgetHitAtCursor(s) && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && !s.dragging {
		context.SetFocused(s, true)
		if !s.isThumbHovered(context, widgetBounds) {
			s.setValueFromCursor(context, widgetBounds)
		}
		s.dragging = true
		x, _ := ebiten.CursorPosition()
		s.draggingStartX = x
		s.draggingStartValue.Set(s.abstractNumberInput.ValueBigInt())
		guigui.RequestRedraw(s)
		return guigui.HandleInputByWidget(s)
	}

	if !context.IsEnabled(s) || !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if s.dragging {
			guigui.RequestRedraw(s)
		}
		s.dragging = false
		s.draggingStartX = 0
		s.draggingStartValue = big.Int{}
		return guigui.HandleInputResult{}
	}

	if context.IsEnabled(s) && s.dragging && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		s.setValueFromCursorDelta(context, widgetBounds)
		return guigui.HandleInputByWidget(s)
	}

	return guigui.HandleInputResult{}
}

func (s *Slider) setValueFromCursorDelta(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
	s.setValue(context, widgetBounds, &s.draggingStartValue, s.draggingStartX)
}

func (s *Slider) setValueFromCursor(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
	min := s.abstractNumberInput.MinimumValueBigInt()
	if min == nil {
		return
	}

	b := widgetBounds.Bounds()
	minX := b.Min.X + (b.Dx()-s.barWidth(context, widgetBounds))/2
	s.setValue(context, widgetBounds, min, minX)
}

func (s *Slider) setValue(context *guigui.Context, widgetBounds *guigui.WidgetBounds, originValue *big.Int, originX int) {
	max := s.abstractNumberInput.MaximumValueBigInt()
	min := s.abstractNumberInput.MinimumValueBigInt()
	if max == nil || min == nil {
		return
	}

	c := image.Pt(ebiten.CursorPosition())
	var v big.Int
	v.Sub(max, min)
	v.Mul(&v, (&big.Int{}).SetInt64(int64(c.X-originX)))
	v.Div(&v, (&big.Int{}).SetInt64(int64(s.barWidth(context, widgetBounds))))
	v.Add(&v, originValue)
	changed := v.Cmp(s.abstractNumberInput.ValueBigInt()) != 0
	s.abstractNumberInput.SetValueBigInt(s, &v, true)
	if changed {
		guigui.RequestRedraw(s)
	}
}

func (s *Slider) barWidth(context *guigui.Context, widgetBounds *guigui.WidgetBounds) int {
	w := widgetBounds.Bounds().Dx()
	return w - 2*sliderThumbRadius(context)
}

func sliderThumbRadius(context *guigui.Context) int {
	return int(UnitSize(context) * 7 / 16)
}

func (s *Slider) thumbBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	rate := s.abstractNumberInput.Rate()
	if math.IsNaN(rate) {
		return image.Rectangle{}
	}
	bounds := widgetBounds.Bounds()
	x := bounds.Min.X + int(rate*float64(s.barWidth(context, widgetBounds)))
	y := bounds.Min.Y + (bounds.Dy()-2*sliderThumbRadius(context))/2
	w := 2 * sliderThumbRadius(context)
	h := 2 * sliderThumbRadius(context)
	return image.Rect(x, y, x+w, y+h)
}

func (s *Slider) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	if s.canPress(context, widgetBounds) || s.dragging {
		return ebiten.CursorShapePointer, true
	}
	return 0, true
}

func (s *Slider) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	rate := s.abstractNumberInput.Rate()

	b := widgetBounds.Bounds()
	x0 := b.Min.X + sliderThumbRadius(context)
	x1 := x0
	if !math.IsNaN(rate) {
		x1 += int(float64(s.barWidth(context, widgetBounds)) * float64(rate))
	}
	x2 := b.Max.X - sliderThumbRadius(context)
	strokeWidth := int(5 * context.Scale())
	r := strokeWidth / 2
	y0 := (b.Min.Y+b.Max.Y)/2 - r
	y1 := (b.Min.Y+b.Max.Y)/2 + r

	bgColorOn := draw.Color(context.ColorMode(), draw.ColorTypeAccent, 0.5)
	bgColorOff := draw.Color(context.ColorMode(), draw.ColorTypeBase, 0.8)
	if !context.IsEnabled(s) {
		bgColorOn = bgColorOff
	}

	if x0 < x1 {
		b := image.Rect(x0, y0, x1, y1)
		draw.DrawRoundedRect(context, dst, b, bgColorOn, r)

		if !context.IsEnabled(s) {
			borderClr1, borderClr2 := draw.BorderColors(context.ColorMode(), draw.RoundedRectBorderTypeInset, false)
			draw.DrawRoundedRectBorder(context, dst, b, borderClr1, borderClr2, r, float32(1*context.Scale()), draw.RoundedRectBorderTypeInset)
		}
	}

	if x1 < x2 {
		b := image.Rect(x1, y0, x2, y1)
		draw.DrawRoundedRect(context, dst, b, bgColorOff, r)

		borderClr1, borderClr2 := draw.BorderColors(context.ColorMode(), draw.RoundedRectBorderTypeInset, false)
		draw.DrawRoundedRectBorder(context, dst, b, borderClr1, borderClr2, r, float32(1*context.Scale()), draw.RoundedRectBorderTypeInset)
	}

	if thumbBounds := s.thumbBounds(context, widgetBounds); !thumbBounds.Empty() {
		cm := context.ColorMode()
		thumbColor := draw.ThumbColor(context.ColorMode(), context.IsEnabled(s))
		if s.isActive(context, widgetBounds) {
			thumbColor = draw.Color2(cm, draw.ColorTypeBase, 0.95, 0.55)
		} else if s.canPress(context, widgetBounds) {
			thumbColor = draw.Color2(cm, draw.ColorTypeBase, 0.975, 0.575)
		}
		thumbClr1, thumbClr2 := draw.BorderColors(context.ColorMode(), draw.RoundedRectBorderTypeOutset, false)
		r := thumbBounds.Dy() / 2
		draw.DrawRoundedRect(context, dst, thumbBounds, thumbColor, r)
		draw.DrawRoundedRectBorder(context, dst, thumbBounds, thumbClr1, thumbClr2, r, float32(1*context.Scale()), draw.RoundedRectBorderTypeOutset)
	}
}

func (s *Slider) canPress(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(s) && s.isThumbHovered(context, widgetBounds) && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && !s.dragging
}

func (s *Slider) isThumbHovered(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsWidgetHitAtCursor(s) && image.Pt(ebiten.CursorPosition()).In(s.thumbBounds(context, widgetBounds))
}

func (s *Slider) isActive(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(s) && s.isThumbHovered(context, widgetBounds) && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && s.dragging
}

func (s *Slider) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return image.Pt(6*UnitSize(context), UnitSize(context))
}
