// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"image"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Basic struct {
	guigui.DefaultWidget

	form             basicwidget.Form
	buttonText       basicwidget.Text
	button           basicwidget.Button
	toggleText       basicwidget.Text
	toggle           basicwidget.Toggle
	checkboxText     basicwidget.Text
	checkbox         basicwidget.Checkbox
	radioButtonsText basicwidget.Text
	radioButtons     inlineRadioButtons
	textInputText    basicwidget.Text
	textInput        basicwidget.TextInput
	numberInputText  basicwidget.Text
	numberInput      basicwidget.NumberInput
	sliderText       basicwidget.Text
	slider           basicwidget.Slider
	listText         basicwidget.Text
	list             basicwidget.List[int]
}

func (b *Basic) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&b.form)

	b.buttonText.SetValue("Button")
	b.button.SetText("Click me!")
	b.toggleText.SetValue("Toggle")
	b.checkboxText.SetValue("Checkbox")
	b.radioButtonsText.SetValue("Radio buttons")
	b.textInputText.SetValue("Text input")
	b.textInput.SetHorizontalAlign(basicwidget.HorizontalAlignEnd)
	b.numberInputText.SetValue("Number input")
	b.sliderText.SetValue("Slider")
	b.slider.SetMinimumValueInt64(0)
	b.slider.SetMaximumValueInt64(100)
	b.listText.SetValue("Text list")
	b.list.SetItemsByStrings([]string{"Item 1", "Item 2", "Item 3"})

	b.form.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &b.buttonText,
			SecondaryWidget: &b.button,
		},
		{
			PrimaryWidget:   &b.toggleText,
			SecondaryWidget: &b.toggle,
		},
		{
			PrimaryWidget:   &b.checkboxText,
			SecondaryWidget: &b.checkbox,
		},
		{
			PrimaryWidget:   &b.radioButtonsText,
			SecondaryWidget: &b.radioButtons,
		},
		{
			PrimaryWidget:   &b.textInputText,
			SecondaryWidget: &b.textInput,
		},
		{
			PrimaryWidget:   &b.numberInputText,
			SecondaryWidget: &b.numberInput,
		},
		{
			PrimaryWidget:   &b.sliderText,
			SecondaryWidget: &b.slider,
		},
		{
			PrimaryWidget:   &b.listText,
			SecondaryWidget: &b.list,
		},
	})

	return nil
}

func (b *Basic) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &b.form,
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

type inlineRadioButtons struct {
	guigui.DefaultWidget

	group basicwidget.RadioButtonGroup[string]
	texts [3]basicwidget.Text
}

func (i *inlineRadioButtons) layout(context *guigui.Context) guigui.LinearLayout {
	u := basicwidget.UnitSize(context)
	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items: []guigui.LinearLayoutItem{
			{
				// By specifying a widget and a layout at the same time, this item's size becomes the maximum of the two.
				// The region of the radio button is widen to the right to make it easier to click.
				Widget: i.group.RadioButton(0),
				Layout: guigui.LinearLayout{
					Direction: guigui.LayoutDirectionHorizontal,
					Items: []guigui.LinearLayoutItem{
						{
							Size: guigui.FixedSize(i.group.RadioButton(0).Measure(context, guigui.Constraints{}).X),
						},
						{
							Widget: &i.texts[0],
						},
					},
					Gap: u / 4,
				},
			},
			{
				Widget: i.group.RadioButton(1),
				Layout: guigui.LinearLayout{
					Direction: guigui.LayoutDirectionHorizontal,
					Items: []guigui.LinearLayoutItem{
						{
							Size: guigui.FixedSize(i.group.RadioButton(1).Measure(context, guigui.Constraints{}).X),
						},
						{
							Widget: &i.texts[1],
						},
					},
					Gap: u / 4,
				},
			},
			{
				Widget: i.group.RadioButton(2),
				Layout: guigui.LinearLayout{
					Direction: guigui.LayoutDirectionHorizontal,
					Items: []guigui.LinearLayoutItem{
						{
							Size: guigui.FixedSize(i.group.RadioButton(2).Measure(context, guigui.Constraints{}).X),
						},
						{
							Widget: &i.texts[2],
						},
					},
					Gap: u / 4,
				},
			},
		},
		Gap: u,
	}
}

func (i *inlineRadioButtons) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&i.group)

	i.group.SetValues([]string{"Option 1", "Option 2", "Option 3"})
	for j := range len(i.texts) {
		adder.AddWidget(i.group.RadioButton(j))
		adder.AddWidget(&i.texts[j])
	}

	i.texts[0].SetValue("Option 1")
	i.texts[1].SetValue("Option 2")
	i.texts[2].SetValue("Option 3")
	return nil
}

func (i *inlineRadioButtons) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	i.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (i *inlineRadioButtons) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return i.layout(context).Measure(context, constraints)
}
