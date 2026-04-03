// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Comboboxes struct {
	guigui.DefaultWidget

	listForm      basicwidget.Form
	combobox1Text basicwidget.Text
	combobox1     basicwidget.Combobox
	combobox2Text basicwidget.Text
	combobox2     basicwidget.Combobox

	configForm    basicwidget.Form
	enabledText   basicwidget.Text
	enabledToggle basicwidget.Toggle

	layoutItems []guigui.LinearLayoutItem
}

func (c *Comboboxes) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&c.listForm)
	adder.AddWidget(&c.configForm)

	v, ok := context.Env(c, modelKeyModel)
	if !ok {
		return nil
	}
	model := v.(*Model)

	// Combobox (free input)
	c.combobox1Text.SetValue("Combobox")
	c.combobox1.SetItems(model.Comboboxes().Items())
	c.combobox1.SetAllowFreeInput(true)
	context.SetEnabled(&c.combobox1, model.Comboboxes().Enabled())

	// Combobox (restricted input)
	c.combobox2Text.SetValue("Combobox (restricted)")
	c.combobox2.SetItems(model.Comboboxes().Items())
	c.combobox2.SetAllowFreeInput(false)
	context.SetEnabled(&c.combobox2, model.Comboboxes().Enabled())

	c.listForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &c.combobox1Text,
			SecondaryWidget: &c.combobox1,
		},
		{
			PrimaryWidget:   &c.combobox2Text,
			SecondaryWidget: &c.combobox2,
		},
	})

	// Config form
	c.enabledText.SetValue("Enabled")
	c.enabledToggle.SetValue(model.Comboboxes().Enabled())
	c.enabledToggle.OnValueChanged(func(context *guigui.Context, toggled bool) {
		model.Comboboxes().SetEnabled(toggled)
	})

	c.configForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &c.enabledText,
			SecondaryWidget: &c.enabledToggle,
		},
	})

	return nil
}

func (c *Comboboxes) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	c.layoutItems = slices.Delete(c.layoutItems, 0, len(c.layoutItems))
	c.layoutItems = append(c.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &c.listForm,
		},
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: &c.configForm,
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     c.layoutItems,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}
