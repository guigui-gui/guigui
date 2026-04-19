// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"
	"math"
	"math/big"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

var (
	sliderEventValueChanged       guigui.EventKey = guigui.GenerateEventKey()
	sliderEventValueChangedBigInt guigui.EventKey = guigui.GenerateEventKey()
	sliderEventValueChangedInt64  guigui.EventKey = guigui.GenerateEventKey()
	sliderEventValueChangedUint64 guigui.EventKey = guigui.GenerateEventKey()
)

type Slider struct {
	guigui.DefaultWidget

	abstractNumberInput abstractNumberInput

	snapOnly bool

	dragging           bool
	draggingStartValue big.Int
	draggingStartX     int

	prevThumbHovered bool

	onValueChanged       func(value int, committed bool)
	onValueChangedBigInt func(value *big.Int, committed bool)
	onValueChangedInt64  func(value int64, committed bool)
	onValueChangedUint64 func(value uint64, committed bool)
}

func (s *Slider) OnValueChanged(f func(context *guigui.Context, value int)) {
	guigui.SetEventHandler(s, sliderEventValueChanged, func(context *guigui.Context, value int, committed bool) {
		f(context, value)
	})
}

func (s *Slider) OnValueChangedBigInt(f func(context *guigui.Context, value *big.Int)) {
	guigui.SetEventHandler(s, sliderEventValueChangedBigInt, func(context *guigui.Context, value *big.Int, committed bool) {
		f(context, value)
	})
}

func (s *Slider) OnValueChangedInt64(f func(context *guigui.Context, value int64)) {
	guigui.SetEventHandler(s, sliderEventValueChangedInt64, func(context *guigui.Context, value int64, committed bool) {
		f(context, value)
	})
}

func (s *Slider) OnValueChangedUint64(f func(context *guigui.Context, value uint64)) {
	guigui.SetEventHandler(s, sliderEventValueChangedUint64, func(context *guigui.Context, value uint64, committed bool) {
		f(context, value)
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
	s.abstractNumberInput.SetValue(value, true)
}

func (s *Slider) SetValueBigInt(value *big.Int) {
	s.abstractNumberInput.SetValueBigInt(value, true)
}

func (s *Slider) SetValueInt64(value int64) {
	s.abstractNumberInput.SetValueInt64(value, true)
}

func (s *Slider) SetValueUint64(value uint64) {
	s.abstractNumberInput.SetValueUint64(value, true)
}

func (s *Slider) MinimumValueBigInt() *big.Int {
	return s.abstractNumberInput.MinimumValueBigInt()
}

func (s *Slider) SetMinimumValue(minimum int) {
	s.abstractNumberInput.SetMinimumValue(minimum)
}

func (s *Slider) SetMinimumValueBigInt(minimum *big.Int) {
	s.abstractNumberInput.SetMinimumValueBigInt(minimum)
}

func (s *Slider) SetMinimumValueInt64(minimum int64) {
	s.abstractNumberInput.SetMinimumValueInt64(minimum)
}

func (s *Slider) SetMinimumValueUint64(minimum uint64) {
	s.abstractNumberInput.SetMinimumValueUint64(minimum)
}

func (s *Slider) MaximumValueBigInt() *big.Int {
	return s.abstractNumberInput.MaximumValueBigInt()
}

func (s *Slider) SetMaximumValue(maximum int) {
	s.abstractNumberInput.SetMaximumValue(maximum)
}

func (s *Slider) SetMaximumValueBigInt(maximum *big.Int) {
	s.abstractNumberInput.SetMaximumValueBigInt(maximum)
}

func (s *Slider) SetMaximumValueInt64(maximum int64) {
	s.abstractNumberInput.SetMaximumValueInt64(maximum)
}

func (s *Slider) SetMaximumValueUint64(maximum uint64) {
	s.abstractNumberInput.SetMaximumValueUint64(maximum)
}

func (s *Slider) SetStep(step int) {
	s.abstractNumberInput.SetStep(step)
}

func (s *Slider) SetStepBigInt(step *big.Int) {
	s.abstractNumberInput.SetStepBigInt(step)
}

func (s *Slider) SetStepInt64(step int64) {
	s.abstractNumberInput.SetStepInt64(step)
}

func (s *Slider) SetStepUint64(step uint64) {
	s.abstractNumberInput.SetStepUint64(step)
}

func (s *Slider) WriteStateKey(w *guigui.StateKeyWriter) {
	s.abstractNumberInput.writeStateKey(w)
	w.WriteBool(s.snapOnly)
	w.WriteBool(s.dragging)
	w.WriteBool(s.prevThumbHovered)
}

func (s *Slider) SetSnapOnly(snapOnly bool) {
	s.snapOnly = snapOnly
}

func (s *Slider) hasSnaps() bool {
	return s.abstractNumberInput.stepSet
}

func (s *Slider) snapValue(v *big.Int) {
	if !s.hasSnaps() {
		return
	}
	min := s.abstractNumberInput.MinimumValueBigInt()
	if min == nil {
		return
	}
	step := &s.abstractNumberInput.step

	// offset = v - min
	var offset big.Int
	offset.Sub(v, min)

	// quotient, remainder = offset / step
	var quotient, remainder big.Int
	quotient.DivMod(&offset, step, &remainder)

	// Round to nearest: if |remainder| > step/2, round away from zero
	var halfStep big.Int
	halfStep.Div(step, big.NewInt(2))

	var absRemainder big.Int
	absRemainder.Abs(&remainder)

	if absRemainder.Cmp(&halfStep) > 0 {
		if offset.Sign() >= 0 {
			quotient.Add(&quotient, big.NewInt(1))
		} else {
			quotient.Sub(&quotient, big.NewInt(1))
		}
	}

	// snapped = min + quotient * step
	v.Mul(&quotient, step)
	v.Add(v, min)
}

func (s *Slider) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if s.onValueChanged == nil {
		s.onValueChanged = func(value int, committed bool) {
			guigui.DispatchEvent(s, sliderEventValueChanged, value, committed)
		}
	}
	s.abstractNumberInput.OnValueChanged(s.onValueChanged)

	if s.onValueChangedBigInt == nil {
		s.onValueChangedBigInt = func(value *big.Int, committed bool) {
			guigui.DispatchEvent(s, sliderEventValueChangedBigInt, value, committed)
		}
	}
	s.abstractNumberInput.OnValueChangedBigInt(s.onValueChangedBigInt)

	if s.onValueChangedInt64 == nil {
		s.onValueChangedInt64 = func(value int64, committed bool) {
			guigui.DispatchEvent(s, sliderEventValueChangedInt64, value, committed)
		}
	}
	s.abstractNumberInput.OnValueChangedInt64(s.onValueChangedInt64)

	if s.onValueChangedUint64 == nil {
		s.onValueChangedUint64 = func(value uint64, committed bool) {
			guigui.DispatchEvent(s, sliderEventValueChangedUint64, value, committed)
		}
	}
	s.abstractNumberInput.OnValueChangedUint64(s.onValueChangedUint64)

	return nil
}

func (s *Slider) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	s.prevThumbHovered = s.isThumbHovered(context, widgetBounds)
	return nil
}

func (s *Slider) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	max := s.abstractNumberInput.MaximumValueBigInt()
	min := s.abstractNumberInput.MinimumValueBigInt()
	if max == nil || min == nil {
		return guigui.HandleInputResult{}
	}

	if context.IsEnabled(s) && widgetBounds.IsHitAtCursor() && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && !s.dragging {
		context.SetFocused(s, true)
		if !s.isThumbHovered(context, widgetBounds) {
			s.setValueFromCursor(context, widgetBounds)
		}
		s.dragging = true
		x, _ := ebiten.CursorPosition()
		s.draggingStartX = x
		s.draggingStartValue.Set(s.abstractNumberInput.ValueBigInt())
		return guigui.HandleInputByWidget(s)
	}

	if !context.IsEnabled(s) || !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
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
	if s.snapOnly {
		s.snapValue(&v)
	}
	s.abstractNumberInput.SetValueBigInt(&v, true)
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
	radius := sliderThumbRadius(context)

	if s.hasSnaps() {
		w := radius
		h := 2 * radius
		x := bounds.Min.X + int(rate*float64(s.barWidth(context, widgetBounds))) + radius - w/2
		y := bounds.Min.Y + (bounds.Dy()-h)/2
		return image.Rect(x, y, x+w, y+h)
	}

	x := bounds.Min.X + int(rate*float64(s.barWidth(context, widgetBounds)))
	y := bounds.Min.Y + (bounds.Dy()-2*radius)/2
	w := 2 * radius
	h := 2 * radius
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
	strokeWidth := int(5 * context.Scale())
	r := strokeWidth / 2
	barY0 := (b.Min.Y+b.Max.Y)/2 - r
	barY1 := (b.Min.Y+b.Max.Y)/2 + r

	barX0 := b.Min.X + sliderThumbRadius(context)*3/4
	barX1 := barX0
	if !math.IsNaN(rate) {
		barX1 = b.Min.X + sliderThumbRadius(context) + int(float64(s.barWidth(context, widgetBounds))*float64(rate))
	}
	barX2 := b.Max.X - sliderThumbRadius(context)*3/4

	bgColorOn := draw.Color(context.ColorMode(), draw.SemanticColorAccent, 0.5)
	bgColorOff := draw.Color(context.ColorMode(), draw.SemanticColorBase, 0.8)
	if !context.IsEnabled(s) {
		bgColorOn = bgColorOff
	}

	if barX0 < barX1 {
		b := image.Rect(barX0, barY0, barX1, barY1)
		basicwidgetdraw.DrawRoundedRect(context, dst, b, bgColorOn, r)

		if !context.IsEnabled(s) {
			borderClr1, borderClr2 := basicwidgetdraw.BorderColors(context.ColorMode(), basicwidgetdraw.RoundedRectBorderTypeInset)
			basicwidgetdraw.DrawRoundedRectBorder(context, dst, b, borderClr1, borderClr2, r, float32(1*context.Scale()), basicwidgetdraw.RoundedRectBorderTypeInset)
		}
	}

	if barX1 < barX2 {
		b := image.Rect(barX1, barY0, barX2, barY1)
		basicwidgetdraw.DrawRoundedRect(context, dst, b, bgColorOff, r)

		borderClr1, borderClr2 := basicwidgetdraw.BorderColors(context.ColorMode(), basicwidgetdraw.RoundedRectBorderTypeInset)
		basicwidgetdraw.DrawRoundedRectBorder(context, dst, b, borderClr1, borderClr2, r, float32(1*context.Scale()), basicwidgetdraw.RoundedRectBorderTypeInset)
	}

	// Draw gauge marks at snap positions.
	if s.hasSnaps() && s.abstractNumberInput.minSet && s.abstractNumberInput.maxSet {
		step := &s.abstractNumberInput.step
		min := &s.abstractNumberInput.min
		max := &s.abstractNumberInput.max

		barW := s.barWidth(context, widgetBounds)
		barStartX := b.Min.X + sliderThumbRadius(context)
		radius := sliderThumbRadius(context)
		gap := float32(2 * context.Scale())

		barTop := float32(barY0)
		barBottom := float32(barY1)

		tickColor := draw.Color(context.ColorMode(), draw.SemanticColorBase, 0.7)
		tickWidth := float32(2 * context.Scale())
		tickHeight := float32(radius) / 2

		var denom big.Int
		denom.Sub(max, min)
		if denom.Sign() > 0 {
			var pos big.Int
			for pos.Set(min); pos.Cmp(max) <= 0; pos.Add(&pos, step) {
				var numer big.Int
				numer.Sub(&pos, min)
				tickRate, _ := (&big.Rat{}).Quo(
					(&big.Rat{}).SetInt(&numer),
					(&big.Rat{}).SetInt(&denom),
				).Float64()

				tx := float32(barStartX) + float32(float64(barW)*tickRate)
				vector.StrokeLine(dst,
					tx, barTop-tickHeight,
					tx, barTop-gap,
					tickWidth, tickColor, true)
				vector.StrokeLine(dst,
					tx, barBottom+gap,
					tx, barBottom+tickHeight,
					tickWidth, tickColor, true)
			}
		}
	}

	if thumbBounds := s.thumbBounds(context, widgetBounds); !thumbBounds.Empty() {
		cm := context.ColorMode()
		thumbColor := basicwidgetdraw.ThumbColor(context.ColorMode(), context.IsEnabled(s))
		if s.isActive(context, widgetBounds) {
			thumbColor = draw.Color2(cm, draw.SemanticColorBase, 0.95, 0.55)
		} else if s.canPress(context, widgetBounds) {
			thumbColor = draw.Color2(cm, draw.SemanticColorBase, 0.975, 0.575)
		}
		thumbClr1, thumbClr2 := basicwidgetdraw.BorderColors(context.ColorMode(), basicwidgetdraw.RoundedRectBorderTypeOutset)
		r := thumbBounds.Dy() / 2
		if s.hasSnaps() {
			r = thumbBounds.Dx() / 2
		}
		basicwidgetdraw.DrawRoundedRect(context, dst, thumbBounds, thumbColor, r)
		basicwidgetdraw.DrawRoundedRectBorder(context, dst, thumbBounds, thumbClr1, thumbClr2, r, float32(1*context.Scale()), basicwidgetdraw.RoundedRectBorderTypeOutset)
	}
}

func (s *Slider) canPress(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(s) && s.isThumbHovered(context, widgetBounds) && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && !s.dragging
}

func (s *Slider) isThumbHovered(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return widgetBounds.IsHitAtCursor() && image.Pt(ebiten.CursorPosition()).In(s.thumbBounds(context, widgetBounds))
}

func (s *Slider) isActive(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(s) && s.isThumbHovered(context, widgetBounds) && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && s.dragging
}

func (s *Slider) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return image.Pt(6*UnitSize(context), UnitSize(context))
}
