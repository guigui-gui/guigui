// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Basic struct {
	guigui.DefaultWidget

	form            basicwidget.Form
	buttonText      basicwidget.Text
	button          basicwidget.Button
	toggleText      basicwidget.Text
	toggle          basicwidget.Toggle
	textInputText   basicwidget.Text
	textInput       basicwidget.TextInput
	numberInputText basicwidget.Text
	numberInput     basicwidget.NumberInput
	sliderText      basicwidget.Text
	slider          basicwidget.Slider
	listText        basicwidget.Text
	list            basicwidget.List[int]
}

func (b *Basic) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&b.form)
}

func (b *Basic) Update(context *guigui.Context) error {
	b.buttonText.SetValue("Button")
	b.button.SetText("Click me!")
	b.toggleText.SetValue("Toggle")
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

func (b *Basic) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
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
