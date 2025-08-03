// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"math"
	"math/big"
	"strings"

	"github.com/hajimehoshi/guigui"
)

const (
	abstractNumberInputEventValueChangedString = "valueChangedString"
	abstractNumberInputEventValueChangedBigInt = "valueChangedBigInt"
	abstractNumberInputEventValueChangedInt64  = "valueChangedInt64"
	abstractNumberInputEventValueChangedUint64 = "valueChangedUint64"
)

type abstractNumberInput struct {
	value   big.Int
	min     big.Int
	minSet  bool
	max     big.Int
	maxSet  bool
	step    big.Int
	stepSet bool
}

func (a *abstractNumberInput) SetOnValueChangedString(widget guigui.Widget, f func(value string, force bool)) {
	guigui.RegisterEventHandler(widget, abstractNumberInputEventValueChangedString, f)
}

func (a *abstractNumberInput) SetOnValueChangedBigInt(widget guigui.Widget, f func(value *big.Int, committed bool)) {
	guigui.RegisterEventHandler(widget, abstractNumberInputEventValueChangedBigInt, f)
}

func (a *abstractNumberInput) SetOnValueChangedInt64(widget guigui.Widget, f func(value int64, committed bool)) {
	guigui.RegisterEventHandler(widget, abstractNumberInputEventValueChangedInt64, f)
}

func (a *abstractNumberInput) SetOnValueChangedUint64(widget guigui.Widget, f func(value uint64, committed bool)) {
	guigui.RegisterEventHandler(widget, abstractNumberInputEventValueChangedUint64, f)
}

func (a *abstractNumberInput) fireValueChangeEvents(widget guigui.Widget, force bool, committed bool) {
	guigui.InvokeEventHandler(widget, abstractNumberInputEventValueChangedString, a.value.String(), force)
	guigui.InvokeEventHandler(widget, abstractNumberInputEventValueChangedBigInt, a.ValueBigInt(), committed)
	guigui.InvokeEventHandler(widget, abstractNumberInputEventValueChangedInt64, a.ValueInt64(), committed)
	guigui.InvokeEventHandler(widget, abstractNumberInputEventValueChangedUint64, a.ValueUint64(), committed)
}

func (a *abstractNumberInput) ValueString() string {
	return a.value.String()
}

func (a *abstractNumberInput) ValueBigInt() *big.Int {
	return (&big.Int{}).Set(&a.value)
}

func (a *abstractNumberInput) ValueInt64() int64 {
	if a.value.IsInt64() {
		return a.value.Int64()
	}
	if a.value.Cmp(&maxInt64) > 0 {
		return math.MaxInt64
	}
	if a.value.Cmp(&minInt64) < 0 {
		return math.MinInt64
	}
	return 0
}

func (a *abstractNumberInput) ValueUint64() uint64 {
	if a.value.IsUint64() {
		return a.value.Uint64()
	}
	if a.value.Cmp(&maxUint64) > 0 {
		return math.MaxUint64
	}
	if a.value.Cmp(big.NewInt(0)) < 0 {
		return 0
	}
	return 0
}

func (a *abstractNumberInput) SetValueBigInt(widget guigui.Widget, value *big.Int, committed bool) {
	a.setValue(widget, value, false, committed)
}

func (a *abstractNumberInput) SetValueInt64(widget guigui.Widget, value int64, committed bool) {
	a.setValue(widget, (&big.Int{}).SetInt64(value), false, committed)
}

func (a *abstractNumberInput) SetValueUint64(widget guigui.Widget, value uint64, committed bool) {
	a.setValue(widget, (&big.Int{}).SetUint64(value), false, committed)
}

func (a *abstractNumberInput) ForceSetValueBigInt(widget guigui.Widget, value *big.Int, committed bool) {
	a.setValue(widget, value, true, committed)
}

func (a *abstractNumberInput) ForceSetValueInt64(widget guigui.Widget, value int64, committed bool) {
	a.setValue(widget, (&big.Int{}).SetInt64(value), true, committed)
}

func (a *abstractNumberInput) ForceSetValueUint64(widget guigui.Widget, value uint64, committed bool) {
	a.setValue(widget, (&big.Int{}).SetUint64(value), true, committed)
}

func (a *abstractNumberInput) setValue(widget guigui.Widget, value *big.Int, force bool, committed bool) {
	a.clamp(value)
	if a.value.Cmp(value) == 0 {
		return
	}
	a.value.Set(value)
	a.fireValueChangeEvents(widget, force, committed)
}

func (a *abstractNumberInput) MinimumValueBigInt() *big.Int {
	if !a.minSet {
		return nil
	}
	return (&big.Int{}).Set(&a.min)
}

func (a *abstractNumberInput) SetMinimumValueBigInt(widget guigui.Widget, minimum *big.Int) {
	if minimum == nil {
		a.min = big.Int{}
		a.minSet = false
		return
	}
	a.min.Set(minimum)
	a.minSet = true
	a.SetValueBigInt(widget, (&big.Int{}).Set(&a.value), true)
}

func (a *abstractNumberInput) SetMinimumValueInt64(widget guigui.Widget, minimum int64) {
	a.min.SetInt64(minimum)
	a.minSet = true
	a.SetValueBigInt(widget, (&big.Int{}).Set(&a.value), true)
}

func (a *abstractNumberInput) SetMinimumValueUint64(widget guigui.Widget, minimum uint64) {
	a.min.SetUint64(minimum)
	a.minSet = true
	a.SetValueBigInt(widget, (&big.Int{}).Set(&a.value), true)
}

func (a *abstractNumberInput) MaximumValueBigInt() *big.Int {
	if !a.maxSet {
		return nil
	}
	return (&big.Int{}).Set(&a.max)
}

func (a *abstractNumberInput) SetMaximumValueBigInt(widget guigui.Widget, maximum *big.Int) {
	if maximum == nil {
		a.max = big.Int{}
		a.maxSet = false
		return
	}
	a.max.Set(maximum)
	a.maxSet = true
	a.SetValueBigInt(widget, (&big.Int{}).Set(&a.value), true)
}

func (a *abstractNumberInput) SetMaximumValueInt64(widget guigui.Widget, maximum int64) {
	a.max.SetInt64(maximum)
	a.maxSet = true
	a.SetValueBigInt(widget, (&big.Int{}).Set(&a.value), true)
}

func (a *abstractNumberInput) SetMaximumValueUint64(widget guigui.Widget, maximum uint64) {
	a.max.SetUint64(maximum)
	a.maxSet = true
	a.SetValueBigInt(widget, (&big.Int{}).Set(&a.value), true)
}

func (a *abstractNumberInput) SetStepBigInt(step *big.Int) {
	if step == nil {
		a.step = big.Int{}
		a.stepSet = false
		return
	}
	a.step.Set(step)
	a.stepSet = true
}

func (a *abstractNumberInput) SetStepInt64(step int64) {
	a.step.SetInt64(step)
	a.stepSet = true
}

func (a *abstractNumberInput) SetStepUint64(step uint64) {
	a.step.SetUint64(step)
	a.stepSet = true
}

func (a *abstractNumberInput) clamp(value *big.Int) {
	if a.minSet && value.Cmp(&a.min) < 0 {
		value.Set(&a.min)
		return
	}
	if a.maxSet && value.Cmp(&a.max) > 0 {
		value.Set(&a.max)
		return
	}
}

func (a *abstractNumberInput) Rate() float64 {
	if !a.maxSet || !a.minSet {
		return math.NaN()
	}

	numer := (&big.Int{}).Sub(&a.value, &a.min)
	denom := (&big.Int{}).Sub(&a.max, &a.min)
	if denom.Sign() == 0 {
		return math.NaN()
	}

	x, _ := (&big.Rat{}).Quo((&big.Rat{}).SetInt(numer), (&big.Rat{}).SetInt(denom)).Float64()
	return x
}

func (a *abstractNumberInput) SetRate(rate float64) {
}

var numberTextReplacer = strings.NewReplacer(
	"\u2212", "-",
	"\ufe62", "+",
	"\ufe63", "-",
	"\uff0b", "+",
	"\uff0d", "-",
	"\uff10", "0",
	"\uff11", "1",
	"\uff12", "2",
	"\uff13", "3",
	"\uff14", "4",
	"\uff15", "5",
	"\uff16", "6",
	"\uff17", "7",
	"\uff18", "8",
	"\uff19", "9",
)

func (a *abstractNumberInput) SetString(widget guigui.Widget, text string, force bool, committed bool) {
	text = strings.TrimSpace(text)
	text = numberTextReplacer.Replace(text)

	var v big.Int
	if _, ok := v.SetString(text, 10); !ok {
		return
	}
	a.setValue(widget, &v, force, committed)
}

func (n *abstractNumberInput) Increment(widget guigui.Widget) {
	var step big.Int
	if n.stepSet {
		step.Set(&n.step)
	} else {
		step.SetInt64(1)
	}
	n.setValue(widget, (&big.Int{}).Add(&n.value, &step), true, true)
}

func (n *abstractNumberInput) Decrement(widget guigui.Widget) {
	var step big.Int
	if n.stepSet {
		step.Set(&n.step)
	} else {
		step.SetInt64(1)
	}
	n.setValue(widget, (&big.Int{}).Sub(&n.value, &step), true, true)
}

func (n *abstractNumberInput) CanIncrement() bool {
	if !n.maxSet {
		return true
	}
	return n.value.Cmp(&n.max) < 0
}

func (n *abstractNumberInput) CanDecrement() bool {
	if !n.minSet {
		return true
	}
	return n.value.Cmp(&n.min) > 0
}
