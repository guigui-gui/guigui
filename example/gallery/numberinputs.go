// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"image"
	"math"
	"math/big"

	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	"github.com/hajimehoshi/guigui/layout"
)

type NumberInputs struct {
	guigui.DefaultWidget

	numberInputForm       basicwidget.Form
	numberInput1Text      basicwidget.Text
	numberInput1          guigui.WidgetWithSize[*basicwidget.NumberInput]
	numberInput2Text      basicwidget.Text
	numberInput2          guigui.WidgetWithSize[*basicwidget.NumberInput]
	numberInput3Text      basicwidget.Text
	numberInput3          guigui.WidgetWithSize[*basicwidget.NumberInput]
	sliderText            basicwidget.Text
	slider                guigui.WidgetWithSize[*basicwidget.Slider]
	slierWithoutRangeText basicwidget.Text
	sliderWithoutRange    guigui.WidgetWithSize[*basicwidget.Slider]

	configForm     basicwidget.Form
	editableText   basicwidget.Text
	editableToggle basicwidget.Toggle
	enabledText    basicwidget.Text
	enabledToggle  basicwidget.Toggle

	layout layout.GridLayout
}

func (n *NumberInputs) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&n.numberInputForm)
	adder.AddChild(&n.configForm)
}

func (n *NumberInputs) Update(context *guigui.Context) error {
	model := context.Model(n, modelKeyModel).(*Model)

	u := basicwidget.UnitSize(context)

	// Number Inputs
	width := 12 * u

	n.numberInput1Text.SetValue("Number input (BigInt)")
	n.numberInput1.Widget().SetOnValueChangedBigInt(func(value *big.Int, committed bool) {
		if !committed {
			return
		}
		model.NumberInputs().SetNumberInputValue1(value)
	})
	n.numberInput1.Widget().SetValueBigInt(model.NumberInputs().NumberInputValue1())
	n.numberInput1.Widget().SetEditable(model.NumberInputs().Editable())
	context.SetEnabled(&n.numberInput1, model.NumberInputs().Enabled())
	n.numberInput1.SetFixedWidth(width)

	n.numberInput2Text.SetValue("Number input (uint64)")
	n.numberInput2.Widget().SetOnValueChangedUint64(func(value uint64, committed bool) {
		if !committed {
			return
		}
		model.NumberInputs().SetNumberInputValue2(value)
	})
	n.numberInput2.Widget().SetMinimumValueUint64(0)
	n.numberInput2.Widget().SetMaximumValueUint64(math.MaxUint64)
	n.numberInput2.Widget().SetValueUint64(model.NumberInputs().NumberInputValue2())
	n.numberInput2.Widget().SetEditable(model.NumberInputs().Editable())
	context.SetEnabled(&n.numberInput2, model.NumberInputs().Enabled())
	n.numberInput2.SetFixedWidth(width)

	n.numberInput3Text.SetValue("Number input (Range: [-100, 100], Step: 5)")
	n.numberInput3.Widget().SetOnValueChangedInt64(func(value int64, committed bool) {
		if !committed {
			return
		}
		model.NumberInputs().SetNumberInputValue3(int(value))
	})
	n.numberInput3.Widget().SetMinimumValue(-100)
	n.numberInput3.Widget().SetMaximumValue(100)
	n.numberInput3.Widget().SetStep(5)
	n.numberInput3.Widget().SetValue(model.NumberInputs().NumberInputValue3())
	n.numberInput3.Widget().SetEditable(model.NumberInputs().Editable())
	context.SetEnabled(&n.numberInput3, model.NumberInputs().Enabled())
	n.numberInput3.SetFixedWidth(width)

	n.sliderText.SetValue("Slider (Range: [-100, 100])")
	n.slider.Widget().SetOnValueChanged(func(value int) {
		model.NumberInputs().SetNumberInputValue3(value)
	})
	n.slider.Widget().SetMinimumValue(-100)
	n.slider.Widget().SetMaximumValue(100)
	n.slider.Widget().SetValue(model.NumberInputs().NumberInputValue3())
	context.SetEnabled(&n.slider, model.NumberInputs().Enabled())
	n.slider.SetFixedWidth(width)

	n.slierWithoutRangeText.SetValue("Slider w/o range")
	context.SetEnabled(&n.sliderWithoutRange, model.NumberInputs().Enabled())
	n.sliderWithoutRange.SetFixedWidth(width)

	n.numberInputForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &n.numberInput1Text,
			SecondaryWidget: &n.numberInput1,
		},
		{
			PrimaryWidget:   &n.numberInput2Text,
			SecondaryWidget: &n.numberInput2,
		},
		{
			PrimaryWidget:   &n.numberInput3Text,
			SecondaryWidget: &n.numberInput3,
		},
		{
			PrimaryWidget:   &n.sliderText,
			SecondaryWidget: &n.slider,
		},
		{
			PrimaryWidget:   &n.slierWithoutRangeText,
			SecondaryWidget: &n.sliderWithoutRange,
		},
	})

	// Configurations
	n.editableText.SetValue("Editable (for number inputs)")
	n.editableToggle.SetOnValueChanged(func(value bool) {
		model.NumberInputs().SetEditable(value)
	})
	n.editableToggle.SetValue(model.NumberInputs().Editable())

	n.enabledText.SetValue("Enabled")
	n.enabledToggle.SetOnValueChanged(func(value bool) {
		model.NumberInputs().SetEnabled(value)
	})
	n.enabledToggle.SetValue(model.NumberInputs().Enabled())

	n.configForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &n.editableText,
			SecondaryWidget: &n.editableToggle,
		},
		{
			PrimaryWidget:   &n.enabledText,
			SecondaryWidget: &n.enabledToggle,
		},
	})

	n.layout = layout.GridLayout{
		Bounds: context.Bounds(n).Inset(u / 2),
		Heights: []layout.Size{
			layout.FixedSize(n.numberInputForm.Measure(context, guigui.FixedWidthConstraints(context.Bounds(n).Dx()-u)).Y),
			layout.FlexibleSize(1),
			layout.FixedSize(n.configForm.Measure(context, guigui.FixedWidthConstraints(context.Bounds(n).Dx()-u)).Y),
		},
		RowGap: u / 2,
	}

	return nil
}

func (n *NumberInputs) Layout(context *guigui.Context, widget guigui.Widget) image.Rectangle {
	switch widget {
	case &n.numberInputForm:
		return n.layout.CellBounds(0, 0)
	case &n.configForm:
		return n.layout.CellBounds(0, 2)
	}
	return image.Rectangle{}
}
