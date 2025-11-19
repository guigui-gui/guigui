// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type DropdownLists struct {
	guigui.DefaultWidget

	listForm         basicwidget.Form
	dropdownListText basicwidget.Text
	dropdownList     basicwidget.DropdownList[int]

	configForm    basicwidget.Form
	enabledText   basicwidget.Text
	enabledToggle basicwidget.Toggle

	dropdownListItems []basicwidget.DropdownListItem[int]
}

func (d *DropdownLists) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&d.listForm)
	adder.AddChild(&d.configForm)
}

func (d *DropdownLists) Update(context *guigui.Context) error {
	model := context.Model(d, modelKeyModel).(*Model)

	// Dropdown list
	d.dropdownListText.SetValue("Dropdown list")
	d.dropdownListItems = slices.Delete(d.dropdownListItems, 0, len(d.dropdownListItems))
	d.dropdownListItems = model.DropdownLists().AppendDropdownListItems(d.dropdownListItems)
	d.dropdownList.SetItems(d.dropdownListItems)
	context.SetEnabled(&d.dropdownList, model.Lists().Enabled())

	d.listForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &d.dropdownListText,
			SecondaryWidget: &d.dropdownList,
		},
	})

	// Config form
	d.enabledText.SetValue("Enabled")
	d.enabledToggle.SetValue(model.DropdownLists().Enabled())
	d.enabledToggle.SetOnValueChanged(func(toggled bool) {
		model.DropdownLists().SetEnabled(toggled)
	})

	d.configForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &d.enabledText,
			SecondaryWidget: &d.enabledToggle,
		},
	})

	return nil
}

func (d *DropdownLists) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &d.listForm,
			},
			{
				Size: guigui.FlexibleSize(1),
			},
			{
				Widget: &d.configForm,
			},
		},
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}
