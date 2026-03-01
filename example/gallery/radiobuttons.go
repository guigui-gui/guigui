// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type RadioButtons struct {
	guigui.DefaultWidget

	radioButtonsForm1 basicwidget.Form
	radioButtonsForm2 basicwidget.Form

	radioButtonGroup1 basicwidget.RadioButtonGroup[int]
	radioButtonTexts1 [3]basicwidget.Text
	radioButtons1     [3]basicwidget.RadioButton[int]

	radioButtonGroup2 basicwidget.RadioButtonGroup[string]
	radioButtonTexts2 [2]basicwidget.Text
	radioButtons2     [2]basicwidget.RadioButton[string]

	configForm    basicwidget.Form
	enabledText   basicwidget.Text
	enabledToggle basicwidget.Toggle
}

func (r *RadioButtons) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.radioButtonsForm1)
	adder.AddWidget(&r.radioButtonGroup1)

	adder.AddWidget(&r.radioButtonsForm2)
	adder.AddWidget(&r.radioButtonGroup2)

	adder.AddWidget(&r.configForm)

	model := context.Data(r, modelKeyModel).(*Model)

	r.radioButtonGroup1.SetValues([]int{1, 2, 3})
	r.radioButtonGroup1.OnItemSelected(func(ctx *guigui.Context, index int) {
		val, ok := r.radioButtonGroup1.SelectedValue()
		if !ok {
			return
		}
		model.RadioButtons().SetValue1(val)
	})
	r.radioButtonGroup1.SelectItemByValue(model.RadioButtons().Value1())

	r.radioButtonGroup2.SetValues([]string{"cats", "dogs"})
	r.radioButtonGroup2.OnItemSelected(func(ctx *guigui.Context, index int) {
		val, ok := r.radioButtonGroup2.SelectedValue()
		if !ok {
			return
		}
		model.RadioButtons().SetValue2(val)
	})
	r.radioButtonGroup2.SelectItemByValue(model.RadioButtons().Value2())

	r.radioButtonTexts1[0].SetValue("Option 1")
	r.radioButtonTexts1[1].SetValue("Option 2")
	r.radioButtonTexts1[2].SetValue("Option 3")

	r.radioButtonTexts2[0].SetValue("Cats")
	r.radioButtonTexts2[1].SetValue("Dogs")

	for i := range len(r.radioButtons1) {
		context.SetEnabled(r.radioButtonGroup1.RadioButton(i), model.RadioButtons().Enabled())
	}
	for i := range len(r.radioButtons2) {
		context.SetEnabled(r.radioButtonGroup2.RadioButton(i), model.RadioButtons().Enabled())
	}

	r.radioButtonsForm1.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &r.radioButtonTexts1[0],
			SecondaryWidget: r.radioButtonGroup1.RadioButton(0),
		},
		{
			PrimaryWidget:   &r.radioButtonTexts1[1],
			SecondaryWidget: r.radioButtonGroup1.RadioButton(1),
		},
		{
			PrimaryWidget:   &r.radioButtonTexts1[2],
			SecondaryWidget: r.radioButtonGroup1.RadioButton(2),
		},
	})

	r.radioButtonsForm2.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &r.radioButtonTexts2[0],
			SecondaryWidget: r.radioButtonGroup2.RadioButton(0),
		},
		{
			PrimaryWidget:   &r.radioButtonTexts2[1],
			SecondaryWidget: r.radioButtonGroup2.RadioButton(1),
		},
	})

	r.enabledText.SetValue("Enabled")
	r.enabledToggle.OnValueChanged(func(ctx *guigui.Context, enabled bool) {
		model.RadioButtons().SetEnabled(enabled)
	})
	r.enabledToggle.SetValue(model.RadioButtons().Enabled())

	r.configForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &r.enabledText,
			SecondaryWidget: &r.enabledToggle,
		},
	})

	return nil
}

func (r *RadioButtons) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &r.radioButtonsForm1,
			},
			{
				Widget: &r.radioButtonsForm2,
			},
			{
				Size: guigui.FlexibleSize(1),
			},
			{
				Widget: &r.configForm,
			},
		},
		Gap: u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}
