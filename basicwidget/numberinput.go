// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"
	"math"
	"math/big"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
)

type NumberInputEventArgsValueChanged struct {
	Value     int
	Committed bool
}

type NumberInputEventArgsValueChangedBigInt struct {
	Value     *big.Int
	Committed bool
}

type NumberInputEventArgsValueChangedInt64 struct {
	Value     int64
	Committed bool
}

type NumberInputEventArgsValueChangedUint64 struct {
	Value     uint64
	Committed bool
}

type NumberInputEventArgsKeyJustPressed struct {
	Key ebiten.Key
}

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

	onValueChanged       func(value int, committed bool)
	onValueChangedBigInt func(value *big.Int, committed bool)
	onValueChangedInt64  func(value int64, committed bool)
	onValueChangedUint64 func(value uint64, committed bool)
	onValueChangedString func(value string, force bool)
}

func (n *NumberInput) IsEditable() bool {
	return n.textInput.IsEditable()
}

func (n *NumberInput) SetEditable(editable bool) {
	n.textInput.SetEditable(editable)
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
	n.abstractNumberInput.SetValueBigInt(value, true)
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
	n.abstractNumberInput.ForceSetValue(value, true)
}

func (n *NumberInput) ForceSetValueBigInt(value *big.Int) {
	n.abstractNumberInput.ForceSetValueBigInt(value, true)
}

func (n *NumberInput) ForceSetValueInt64(value int64) {
	n.abstractNumberInput.ForceSetValueInt64(value, true)
}

func (n *NumberInput) ForceSetValueUint64(value uint64) {
	n.abstractNumberInput.ForceSetValueUint64(value, true)
}

func (n *NumberInput) MinimumValueBigInt() *big.Int {
	return n.abstractNumberInput.MinimumValueBigInt()
}

func (n *NumberInput) SetMinimumValue(minimum int) {
	n.abstractNumberInput.SetMinimumValue(minimum)
}

func (n *NumberInput) SetMinimumValueBigInt(minimum *big.Int) {
	n.abstractNumberInput.SetMinimumValueBigInt(minimum)
}

func (n *NumberInput) SetMinimumValueInt64(minimum int64) {
	n.abstractNumberInput.SetMinimumValueInt64(minimum)
}

func (n *NumberInput) SetMinimumValueUint64(minimum uint64) {
	n.abstractNumberInput.SetMinimumValueUint64(minimum)
}

func (n *NumberInput) MaximumValueBigInt() *big.Int {
	return n.abstractNumberInput.MaximumValueBigInt()
}

func (n *NumberInput) SetMaximumValue(maximum int) {
	n.abstractNumberInput.SetMaximumValue(maximum)
}

func (n *NumberInput) SetMaximumValueBigInt(maximum *big.Int) {
	n.abstractNumberInput.SetMaximumValueBigInt(maximum)
}

func (n *NumberInput) SetMaximumValueInt64(maximum int64) {
	n.abstractNumberInput.SetMaximumValueInt64(maximum)
}

func (n *NumberInput) SetMaximumValueUint64(maximum uint64) {
	n.abstractNumberInput.SetMaximumValueUint64(maximum)
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

	if n.onValueChanged == nil {
		n.onValueChanged = func(value int, committed bool) {
			guigui.DispatchEvent(n, &NumberInputEventArgsValueChanged{
				Value:     value,
				Committed: committed,
			})
		}
	}
	n.abstractNumberInput.SetOnValueChanged(n.onValueChanged)

	if n.onValueChangedBigInt == nil {
		n.onValueChangedBigInt = func(value *big.Int, committed bool) {
			guigui.DispatchEvent(n, &NumberInputEventArgsValueChangedBigInt{
				Value:     value,
				Committed: committed,
			})
		}
	}
	n.abstractNumberInput.SetOnValueChangedBigInt(n.onValueChangedBigInt)

	if n.onValueChangedInt64 == nil {
		n.onValueChangedInt64 = func(value int64, committed bool) {
			guigui.DispatchEvent(n, &NumberInputEventArgsValueChangedInt64{
				Value:     value,
				Committed: committed,
			})
		}
	}
	n.abstractNumberInput.SetOnValueChangedInt64(n.onValueChangedInt64)

	if n.onValueChangedUint64 == nil {
		n.onValueChangedUint64 = func(value uint64, committed bool) {
			guigui.DispatchEvent(n, &NumberInputEventArgsValueChangedUint64{
				Value:     value,
				Committed: committed,
			})
		}
	}
	n.abstractNumberInput.SetOnValueChangedUint64(n.onValueChangedUint64)

	if n.onValueChangedString == nil {
		n.onValueChangedString = func(value string, force bool) {
			if force {
				n.textInput.ForceSetValue(value)
			} else {
				n.textInput.SetValue(value)
			}
		}
	}
	n.abstractNumberInput.SetOnValueChangedString(n.onValueChangedString)

	n.textInput.SetValue(n.abstractNumberInput.ValueString())
	n.textInput.SetHorizontalAlign(HorizontalAlignRight)
	n.textInput.SetTabular(true)
	n.textInput.setPaddingEnd(UnitSize(context) / 2)

	guigui.AddEventHandler(n, &n.textInput)

	imgUp, err := theResourceImages.Get("keyboard_arrow_up", context.ColorMode())
	if err != nil {
		return err
	}
	imgDown, err := theResourceImages.Get("keyboard_arrow_down", context.ColorMode())
	if err != nil {
		return err
	}

	n.upButton.SetIcon(imgUp)
	n.upButton.setSharpCorners(basicwidgetdraw.Corners{
		BottomStart: true,
		BottomEnd:   true,
	})
	n.upButton.setPairedButton(&n.downButton)
	guigui.AddEventHandler(n, &n.upButton)
	context.SetEnabled(&n.upButton, n.IsEditable() && n.abstractNumberInput.CanIncrement())

	n.downButton.SetIcon(imgDown)
	n.downButton.setSharpCorners(basicwidgetdraw.Corners{
		TopStart: true,
		TopEnd:   true,
	})
	n.downButton.setPairedButton(&n.upButton)
	guigui.AddEventHandler(n, &n.downButton)
	context.SetEnabled(&n.downButton, n.IsEditable() && n.abstractNumberInput.CanDecrement())

	return nil
}

func (n *NumberInput) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
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
	n.abstractNumberInput.Increment()
}

func (n *NumberInput) decrement() {
	if !n.IsEditable() {
		return
	}
	n.CommitWithCurrentInputValue()
	n.abstractNumberInput.Decrement()
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

func (n *NumberInput) HandleEvent(context *guigui.Context, targetWidget guigui.Widget, eventArgs any) {
	switch targetWidget {
	case &n.textInput:
		switch eventArgs := eventArgs.(type) {
		case *TextInputEventArgsValueChanged:
			n.abstractNumberInput.SetString(eventArgs.Value, false, eventArgs.Committed)
		case *TextInputEventArgsKeyJustPressed:
			guigui.DispatchEvent(n, &NumberInputEventArgsKeyJustPressed{
				Key: eventArgs.Key,
			})
		}
	case &n.upButton:
		switch eventArgs.(type) {
		case *ButtonEventArgsDown:
			n.increment()
		case *ButtonEventArgsRepeat:
			n.increment()
		}
	case &n.downButton:
		switch eventArgs.(type) {
		case *ButtonEventArgsDown:
			n.decrement()
		case *ButtonEventArgsRepeat:
			n.decrement()
		}
	}
}
