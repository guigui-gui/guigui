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

	checkboxText basicwidget.Text
	checkbox     basicwidget.Checkbox

	configForm    basicwidget.Form
	enabledText   basicwidget.Text
	enabledToggle basicwidget.Toggle
}

func (c *Checkboxes) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&c.checkboxesForm)
	adder.AddWidget(&c.configForm)

	model := context.Data(c, modelKeyModel).(*Model)

	c.checkboxText.SetValue("Checkbox")
	c.checkbox.OnValueChanged(func(ctx *guigui.Context, value bool) {
		model.Checkboxes().SetValue(value)
	})
	c.checkbox.SetValue(model.Checkboxes().Value())
	context.SetEnabled(&c.checkbox, model.Checkboxes().Enabled())

	c.checkboxesForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &c.checkboxText,
			SecondaryWidget: &c.checkbox,
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
