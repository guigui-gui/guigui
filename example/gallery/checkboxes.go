// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Checkboxes struct {
	guigui.DefaultWidget

	checkboxesForm basicwidget.Form

	checkbox1Text basicwidget.Text
	checkbox1     basicwidget.Checkbox
	checkbox2Text basicwidget.Text
	checkbox2     basicwidget.Checkbox
	checkbox3Text basicwidget.Text
	checkbox3     basicwidget.Checkbox

	configForm    basicwidget.Form
	enabledText   basicwidget.Text
	enabledToggle basicwidget.Toggle
}

func (c *Checkboxes) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&c.checkboxesForm)
	adder.AddWidget(&c.configForm)

	model := context.Data(c, modelKeyModel).(*Model)

	c.checkbox1Text.SetValue("Checkbox 1")
	c.checkbox1.OnValueChanged(func(ctx *guigui.Context, value bool) {
		model.Checkboxes().SetValue(0, value)
	})
	c.checkbox1.SetValue(model.Checkboxes().Value(0))
	context.SetEnabled(&c.checkbox1, model.Checkboxes().Enabled())

	c.checkbox2Text.SetValue("Checkbox 2")
	c.checkbox2.OnValueChanged(func(ctx *guigui.Context, value bool) {
		model.Checkboxes().SetValue(1, value)
	})
	c.checkbox2.SetValue(model.Checkboxes().Value(1))
	context.SetEnabled(&c.checkbox2, model.Checkboxes().Enabled())

	c.checkbox3Text.SetValue("Checkbox 3")
	c.checkbox3.OnValueChanged(func(ctx *guigui.Context, value bool) {
		model.Checkboxes().SetValue(2, value)
	})
	c.checkbox3.SetValue(model.Checkboxes().Value(2))
	context.SetEnabled(&c.checkbox3, model.Checkboxes().Enabled())

	c.checkboxesForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &c.checkbox1Text,
			SecondaryWidget: &c.checkbox1,
		},
		{
			PrimaryWidget:   &c.checkbox2Text,
			SecondaryWidget: &c.checkbox2,
		},
		{
			PrimaryWidget:   &c.checkbox3Text,
			SecondaryWidget: &c.checkbox3,
		},
	})

	c.enabledText.SetValue("Enabled")
	c.enabledToggle.OnValueChanged(func(ctx *guigui.Context, enabled bool) {
		model.Checkboxes().SetEnabled(enabled)
	})
	c.enabledToggle.SetValue(model.Checkboxes().Enabled())

	c.configForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &c.enabledText,
			SecondaryWidget: &c.enabledToggle,
		},
	})

	return nil
}

func (c *Checkboxes) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &c.checkboxesForm,
			},
			{
				Size: guigui.FlexibleSize(1),
			},
			{
				Widget: &c.configForm,
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
