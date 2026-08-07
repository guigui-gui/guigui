// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image"
	"math"
	"math/big"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

// ProgressBar is a horizontal gauge showing how far a task has come.
// The gauge is empty until both the minimum and the maximum value are set.
type ProgressBar struct {
	guigui.DefaultWidget

	abstractNumberInput abstractNumberInput
}

func (p *ProgressBar) Value() int {
	return p.abstractNumberInput.Value()
}

func (p *ProgressBar) ValueBigInt() *big.Int {
	return p.abstractNumberInput.ValueBigInt()
}

func (p *ProgressBar) ValueInt64() int64 {
	return p.abstractNumberInput.ValueInt64()
}

func (p *ProgressBar) ValueUint64() uint64 {
	return p.abstractNumberInput.ValueUint64()
}

func (p *ProgressBar) SetValue(value int) {
	p.abstractNumberInput.SetValue(value, true)
}

func (p *ProgressBar) SetValueBigInt(value *big.Int) {
	p.abstractNumberInput.SetValueBigInt(value, true)
}

func (p *ProgressBar) SetValueInt64(value int64) {
	p.abstractNumberInput.SetValueInt64(value, true)
}

func (p *ProgressBar) SetValueUint64(value uint64) {
	p.abstractNumberInput.SetValueUint64(value, true)
}

func (p *ProgressBar) MinimumValueBigInt() *big.Int {
	return p.abstractNumberInput.MinimumValueBigInt()
}

func (p *ProgressBar) SetMinimumValue(minimum int) {
	p.abstractNumberInput.SetMinimumValue(minimum)
}

func (p *ProgressBar) SetMinimumValueBigInt(minimum *big.Int) {
	p.abstractNumberInput.SetMinimumValueBigInt(minimum)
}

func (p *ProgressBar) SetMinimumValueInt64(minimum int64) {
	p.abstractNumberInput.SetMinimumValueInt64(minimum)
}

func (p *ProgressBar) SetMinimumValueUint64(minimum uint64) {
	p.abstractNumberInput.SetMinimumValueUint64(minimum)
}

func (p *ProgressBar) MaximumValueBigInt() *big.Int {
	return p.abstractNumberInput.MaximumValueBigInt()
}

func (p *ProgressBar) SetMaximumValue(maximum int) {
	p.abstractNumberInput.SetMaximumValue(maximum)
}

func (p *ProgressBar) SetMaximumValueBigInt(maximum *big.Int) {
	p.abstractNumberInput.SetMaximumValueBigInt(maximum)
}

func (p *ProgressBar) SetMaximumValueInt64(maximum int64) {
	p.abstractNumberInput.SetMaximumValueInt64(maximum)
}

func (p *ProgressBar) SetMaximumValueUint64(maximum uint64) {
	p.abstractNumberInput.SetMaximumValueUint64(maximum)
}

func (p *ProgressBar) WriteStateKey(context *guigui.Context, w *guigui.StateKeyWriter) {
	// The value can advance outside any input handler.
	p.abstractNumberInput.writeStateKey(w)
}

func (p *ProgressBar) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	b := widgetBounds.Bounds()
	strokeWidth := int(5 * context.Scale())
	r := strokeWidth / 2
	cy := (b.Min.Y + b.Max.Y) / 2
	barBounds := image.Rect(b.Min.X, cy-r, b.Max.X, cy+r)

	cm := context.ColorMode()
	draw.DrawRoundedRect(context, dst, barBounds, draw.TrackColor(cm, false, false), r)
	borderClr1, borderClr2 := draw.BorderColors(cm, draw.RoundedRectBorderTypeInset)
	draw.DrawRoundedRectBorder(context, dst, barBounds, borderClr1, borderClr2, r, float32(1*context.Scale()), draw.RoundedRectBorderTypeInset)

	rate := p.abstractNumberInput.Rate()
	if math.IsNaN(rate) {
		return
	}
	w := int(math.Round(rate * float64(barBounds.Dx())))
	if w == 0 {
		return
	}
	filledBounds := barBounds
	filledBounds.Max.X = filledBounds.Min.X + w
	draw.DrawRoundedRect(context, dst, filledBounds, draw.TrackFillColor(cm, context.IsEnabled(p)), r)
}

func (p *ProgressBar) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	w, ok := constraints.FixedWidth()
	if !ok {
		w = 6 * UnitSize(context)
	}
	return image.Pt(w, UnitSize(context))
}
