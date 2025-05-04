// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"math"
	"math/big"
	"strings"
)

type abstractNumberInput struct {
	value   big.Int
	min     big.Int
	minSet  bool
	max     big.Int
	maxSet  bool
	step    big.Int
	stepSet bool

	onValueChangedString func(value string, force bool)
	onValueChangedBigInt func(value *big.Int)
	onValueChangedInt64  func(value int64)
	onValueChangedUint64 func(value uint64)
}

func (a *abstractNumberInput) SetOnValueChangedString(f func(value string, force bool)) {
	a.onValueChangedString = f
}

func (a *abstractNumberInput) SetOnValueChangedBigInt(f func(value *big.Int)) {
	a.onValueChangedBigInt = f
}

func (a *abstractNumberInput) SetOnValueChangedInt64(f func(value int64)) {
	a.onValueChangedInt64 = f
}

func (a *abstractNumberInput) SetOnValueChangedUint64(f func(value uint64)) {
	a.onValueChangedUint64 = f
}

func (a *abstractNumberInput) fireValueChangeEvents(force bool) {
	if a.onValueChangedString != nil {
		a.onValueChangedString(a.value.String(), force)
	}
	if a.onValueChangedBigInt != nil {
		a.onValueChangedBigInt(a.ValueBigInt())
	}
	if a.onValueChangedInt64 != nil {
		a.onValueChangedInt64(a.ValueInt64())
	}
	if a.onValueChangedUint64 != nil {
		a.onValueChangedUint64(a.ValueUint64())
	}
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

func (a *abstractNumberInput) SetValueBigInt(value *big.Int) {
	a.setValue(value, false)
}

func (a *abstractNumberInput) SetValueInt64(value int64) {
	a.setValue((&big.Int{}).SetInt64(value), false)
}

func (a *abstractNumberInput) SetValueUint64(value uint64) {
	a.setValue((&big.Int{}).SetUint64(value), false)
}

func (a *abstractNumberInput) setValue(value *big.Int, force bool) {
	a.clamp(value)
	if a.value.Cmp(value) == 0 {
		return
	}
	a.value.Set(value)
	a.fireValueChangeEvents(force)
}

func (a *abstractNumberInput) MinimumValueBigInt() *big.Int {
	if !a.minSet {
		return nil
	}
	return (&big.Int{}).Set(&a.min)
}

func (a *abstractNumberInput) SetMinimumValueBigInt(minimum *big.Int) {
	if minimum == nil {
		a.min = big.Int{}
		a.minSet = false
		return
	}
	a.min.Set(minimum)
	a.minSet = true
	a.SetValueBigInt((&big.Int{}).Set(&a.value))
}

func (a *abstractNumberInput) SetMinimumValueInt64(minimum int64) {
	a.min.SetInt64(minimum)
	a.minSet = true
	a.SetValueBigInt((&big.Int{}).Set(&a.value))
}

func (a *abstractNumberInput) SetMinimumValueUint64(minimum uint64) {
	a.min.SetUint64(minimum)
	a.minSet = true
	a.SetValueBigInt((&big.Int{}).Set(&a.value))
}

func (a *abstractNumberInput) MaximumValueBigInt() *big.Int {
	if !a.maxSet {
		return nil
	}
	return (&big.Int{}).Set(&a.max)
}

func (a *abstractNumberInput) SetMaximumValueBigInt(maximum *big.Int) {
	if maximum == nil {
		a.max = big.Int{}
		a.maxSet = false
		return
	}
	a.max.Set(maximum)
	a.maxSet = true
	a.SetValueBigInt((&big.Int{}).Set(&a.value))
}

func (a *abstractNumberInput) SetMaximumValueInt64(maximum int64) {
	a.max.SetInt64(maximum)
	a.maxSet = true
	a.SetValueBigInt((&big.Int{}).Set(&a.value))
}

func (a *abstractNumberInput) SetMaximumValueUint64(maximum uint64) {
	a.max.SetUint64(maximum)
	a.maxSet = true
	a.SetValueBigInt((&big.Int{}).Set(&a.value))
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

func (a *abstractNumberInput) CommitString(text string) {
	text = strings.TrimSpace(text)
	text = numberTextReplacer.Replace(text)

	var v big.Int
	if _, ok := v.SetString(text, 10); !ok {
		return
	}
	a.SetValueBigInt(&v)
	a.fireValueChangeEvents(false)
}

func (n *abstractNumberInput) Increment() {
	var step big.Int
	if n.stepSet {
		step.Set(&n.step)
	} else {
		step.SetInt64(1)
	}
	n.setValue((&big.Int{}).Add(&n.value, &step), true)
}

func (n *abstractNumberInput) Decrement() {
	var step big.Int
	if n.stepSet {
		step.Set(&n.step)
	} else {
		step.SetInt64(1)
	}
	n.setValue((&big.Int{}).Sub(&n.value, &step), true)
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
