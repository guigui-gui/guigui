// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"image"
	"slices"

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

	layoutItems []guigui.LinearLayoutItem
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
	b.layoutItems = slices.Delete(b.layoutItems, 0, len(b.layoutItems))
	b.layoutItems = append(b.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &b.form,
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     b.layoutItems,
		Gap:       u / 2,
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

	innerLayouts []guigui.LinearLayout
	innerItems   [3][]guigui.LinearLayoutItem
	outerItems   []guigui.LinearLayoutItem
	outerLayout  guigui.LinearLayout
}

func (i *inlineRadioButtons) buildLayout(context *guigui.Context) {
	u := basicwidget.UnitSize(context)
	i.innerLayouts = slices.Delete(i.innerLayouts, 0, len(i.innerLayouts))
	i.outerItems = slices.Delete(i.outerItems, 0, len(i.outerItems))
	for j := range 3 {
		i.innerItems[j] = slices.Delete(i.innerItems[j], 0, len(i.innerItems[j]))
		i.innerItems[j] = append(i.innerItems[j],
			guigui.LinearLayoutItem{
				Size: guigui.FixedSize(i.group.RadioButton(j).Measure(context, guigui.Constraints{}).X),
			},
			guigui.LinearLayoutItem{
				Widget: &i.texts[j],
			},
		)
		i.innerLayouts = append(i.innerLayouts, guigui.LinearLayout{
			Direction: guigui.LayoutDirectionHorizontal,
			Items:     i.innerItems[j],
			Gap:       u / 4,
		})
		i.outerItems = append(i.outerItems, guigui.LinearLayoutItem{
			// By specifying a widget and a layout at the same time, this item's size becomes the maximum of the two.
			// The region of the radio button is widen to the right to make it easier to click.
			Widget: i.group.RadioButton(j),
			Layout: &i.innerLayouts[j],
		})
	}
	i.outerLayout = guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     i.outerItems,
		Gap:       u,
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
	i.buildLayout(context)
	i.outerLayout.LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (i *inlineRadioButtons) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	i.buildLayout(context)
	return i.outerLayout.Measure(context, constraints)
}
