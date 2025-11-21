// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"
	"math"
	"math/big"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

var (
	minInt    big.Int
	maxInt    big.Int
	minInt64  big.Int
	maxInt64  big.Int
	maxUint64 big.Int
)

func init() {
	minInt.SetInt64(math.MinInt)
	maxInt.SetInt64(math.MaxInt)
	minInt64.SetInt64(math.MinInt64)
	maxInt64.SetInt64(math.MaxInt64)
	maxUint64.SetUint64(math.MaxUint64)
}

type NumberInput struct {
	guigui.DefaultWidget

	textInput  TextInput
	upButton   Button
	downButton Button

	abstractNumberInput abstractNumberInput

	onTextInputValueChanged func(value string, committed bool)
	onUpButtonDown          func()
	onDownButtonDown        func()
}

func (n *NumberInput) IsEditable() bool {
	return n.textInput.IsEditable()
}

func (n *NumberInput) SetEditable(editable bool) {
	n.textInput.SetEditable(editable)
}

func (n *NumberInput) SetOnValueChanged(f func(value int, committed bool)) {
	n.abstractNumberInput.SetOnValueChanged(n, f)
}

func (n *NumberInput) SetOnValueChangedBigInt(f func(value *big.Int, committed bool)) {
	n.abstractNumberInput.SetOnValueChangedBigInt(n, f)
}

func (n *NumberInput) SetOnValueChangedInt64(f func(value int64, committed bool)) {
	n.abstractNumberInput.SetOnValueChangedInt64(n, f)
}

func (n *NumberInput) SetOnValueChangedUint64(f func(value uint64, committed bool)) {
	n.abstractNumberInput.SetOnValueChangedUint64(n, f)
}

func (n *NumberInput) SetOnKeyJustPressed(f func(key ebiten.Key) (handled bool)) {
	n.textInput.SetOnKeyJustPressed(f)
}

func (n *NumberInput) Value() int {
	return n.abstractNumberInput.Value()
}

func (n *NumberInput) ValueBigInt() *big.Int {
	return n.abstractNumberInput.ValueBigInt()
}

func (n *NumberInput) ValueInt64() int64 {
	return n.abstractNumberInput.ValueInt64()
}

func (n *NumberInput) ValueUint64() uint64 {
	return n.abstractNumberInput.ValueUint64()
}

func (n *NumberInput) SetValueBigInt(value *big.Int) {
	n.abstractNumberInput.SetValueBigInt(n, value, true)
	/*if n.nextValue != nil && n.nextValue.Cmp(value) == 0 {
		return
	}
	if n.nextValue == nil {
		n.nextValue = &big.Int{}
	}
	n.nextValue.Set(value)*/
}

func (n *NumberInput) SetValue(value int) {
	n.SetValueBigInt((&big.Int{}).SetInt64(int64(value)))
}

func (n *NumberInput) SetValueInt64(value int64) {
	n.SetValueBigInt((&big.Int{}).SetInt64(value))
}

func (n *NumberInput) SetValueUint64(value uint64) {
	n.SetValueBigInt((&big.Int{}).SetUint64(value))
}

func (n *NumberInput) ForceSetValue(value int) {
	n.abstractNumberInput.ForceSetValue(n, value, true)
}

func (n *NumberInput) ForceSetValueBigInt(value *big.Int) {
	n.abstractNumberInput.ForceSetValueBigInt(n, value, true)
}

func (n *NumberInput) ForceSetValueInt64(value int64) {
	n.abstractNumberInput.ForceSetValueInt64(n, value, true)
}

func (n *NumberInput) ForceSetValueUint64(value uint64) {
	n.abstractNumberInput.ForceSetValueUint64(n, value, true)
}

func (n *NumberInput) MinimumValueBigInt() *big.Int {
	return n.abstractNumberInput.MinimumValueBigInt()
}

func (n *NumberInput) SetMinimumValue(minimum int) {
	n.abstractNumberInput.SetMinimumValue(n, minimum)
}

func (n *NumberInput) SetMinimumValueBigInt(minimum *big.Int) {
	n.abstractNumberInput.SetMinimumValueBigInt(n, minimum)
}

func (n *NumberInput) SetMinimumValueInt64(minimum int64) {
	n.abstractNumberInput.SetMinimumValueInt64(n, minimum)
}

func (n *NumberInput) SetMinimumValueUint64(minimum uint64) {
	n.abstractNumberInput.SetMinimumValueUint64(n, minimum)
}

func (n *NumberInput) MaximumValueBigInt() *big.Int {
	return n.abstractNumberInput.MaximumValueBigInt()
}

func (n *NumberInput) SetMaximumValue(maximum int) {
	n.abstractNumberInput.SetMaximumValue(n, maximum)
}

func (n *NumberInput) SetMaximumValueBigInt(maximum *big.Int) {
	n.abstractNumberInput.SetMaximumValueBigInt(n, maximum)
}

func (n *NumberInput) SetMaximumValueInt64(maximum int64) {
	n.abstractNumberInput.SetMaximumValueInt64(n, maximum)
}

func (n *NumberInput) SetMaximumValueUint64(maximum uint64) {
	n.abstractNumberInput.SetMaximumValueUint64(n, maximum)
}

func (n *NumberInput) SetStep(step int) {
	n.abstractNumberInput.SetStep(step)
}

func (n *NumberInput) SetStepBigInt(step *big.Int) {
	n.abstractNumberInput.SetStepBigInt(step)
}

func (n *NumberInput) SetStepInt64(step int64) {
	n.abstractNumberInput.SetStepInt64(step)
}

func (n *NumberInput) SetStepUint64(step uint64) {
	n.abstractNumberInput.SetStepUint64(step)
}

func (n *NumberInput) CommitWithCurrentInputValue() {
	n.textInput.CommitWithCurrentInputValue()
}

func (n *NumberInput) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&n.textInput)
	adder.AddChild(&n.upButton)
	adder.AddChild(&n.downButton)

	n.abstractNumberInput.SetOnValueChangedString(n, func(text string, force bool) {
		if force {
			n.textInput.ForceSetValue(text)
		} else {
			n.textInput.SetValue(text)
		}
	})

	n.textInput.SetValue(n.abstractNumberInput.ValueString())
	n.textInput.SetHorizontalAlign(HorizontalAlignRight)
	n.textInput.SetTabular(true)
	n.textInput.setPaddingEnd(UnitSize(context) / 2)
	if n.onTextInputValueChanged == nil {
		n.onTextInputValueChanged = func(text string, committed bool) {
			n.abstractNumberInput.SetString(n, text, false, committed)
		}
	}
	n.textInput.SetOnValueChanged(n.onTextInputValueChanged)

	imgUp, err := theResourceImages.Get("keyboard_arrow_up", context.ColorMode())
	if err != nil {
		return err
	}
	imgDown, err := theResourceImages.Get("keyboard_arrow_down", context.ColorMode())
	if err != nil {
		return err
	}

	n.upButton.SetIcon(imgUp)
	n.upButton.setSharpenCorners(draw.SharpenCorners{
		LowerStart: true,
		LowerEnd:   true,
	})
	n.upButton.setPairedButton(&n.downButton)
	if n.onUpButtonDown == nil {
		n.onUpButtonDown = func() {
			n.increment()
		}
	}
	n.upButton.setOnRepeat(n.onUpButtonDown)
	context.SetEnabled(&n.upButton, n.IsEditable() && n.abstractNumberInput.CanIncrement())

	n.downButton.SetIcon(imgDown)
	n.downButton.setSharpenCorners(draw.SharpenCorners{
		UpperStart: true,
		UpperEnd:   true,
	})
	n.downButton.setPairedButton(&n.upButton)
	if n.onDownButtonDown == nil {
		n.onDownButtonDown = func() {
			n.decrement()
		}
	}
	n.downButton.setOnRepeat(n.onDownButtonDown)
	context.SetEnabled(&n.downButton, n.IsEditable() && n.abstractNumberInput.CanDecrement())

	return nil
}

func (n *NumberInput) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	b := widgetBounds.Bounds()
	layouter.LayoutWidget(&n.textInput, b)
	layouter.LayoutWidget(&n.upButton, image.Rectangle{
		Min: image.Point{
			X: b.Max.X - UnitSize(context)*3/4,
			Y: b.Min.Y,
		},
		Max: image.Point{
			X: b.Max.X,
			Y: b.Min.Y + b.Dy()/2,
		},
	})
	layouter.LayoutWidget(&n.downButton, image.Rectangle{
		Min: image.Point{
			X: b.Max.X - UnitSize(context)*3/4,
			Y: b.Min.Y + b.Dy()/2,
		},
		Max: b.Max,
	})
}

func (n *NumberInput) HandleButtonInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if isKeyRepeating(ebiten.KeyUp) {
		n.increment()
		return guigui.HandleInputByWidget(n)
	}
	if isKeyRepeating(ebiten.KeyDown) {
		n.decrement()
		return guigui.HandleInputByWidget(n)
	}
	return guigui.HandleInputResult{}
}

func (n *NumberInput) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return n.textInput.Measure(context, constraints)
}

func (n *NumberInput) increment() {
	if !n.IsEditable() {
		return
	}
	n.CommitWithCurrentInputValue()
	n.abstractNumberInput.Increment(n)
}

func (n *NumberInput) decrement() {
	if !n.IsEditable() {
		return
	}
	n.CommitWithCurrentInputValue()
	n.abstractNumberInput.Decrement(n)
}

func (n *NumberInput) CanCut() bool {
	return n.textInput.CanCut()
}

func (n *NumberInput) CanCopy() bool {
	return n.textInput.CanCopy()
}

func (n *NumberInput) CanPaste() bool {
	return n.textInput.CanPaste()
}

func (n *NumberInput) Cut() bool {
	return n.textInput.Cut()
}

func (n *NumberInput) Copy() bool {
	return n.textInput.Copy()
}

func (n *NumberInput) Paste() bool {
	return n.textInput.Paste()
}
