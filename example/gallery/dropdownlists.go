// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"image"
	"image/color"
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	"github.com/hajimehoshi/ebiten/v2"
)

type DropdownLists struct {
	guigui.DefaultWidget

	listForm          basicwidget.Form
	dropdown1ListText basicwidget.Text
	dropdown1List     basicwidget.DropdownList[int]
	dropdown2ListText basicwidget.Text
	dropdown2List     basicwidget.DropdownList[int]

	configForm    basicwidget.Form
	enabledText   basicwidget.Text
	enabledToggle basicwidget.Toggle

	dropdown1ListItems       []basicwidget.DropdownListItem[int]
	dropdown2ListItems       []basicwidget.DropdownListItem[int]
	dropdown2ListItemWidgets []dropdownListItem
}

func (d *DropdownLists) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&d.listForm)
	adder.AddChild(&d.configForm)
}

func (d *DropdownLists) Update(context *guigui.Context) error {
	model := context.Model(d, modelKeyModel).(*Model)

	// Dropdown list (Text)
	d.dropdown1ListText.SetValue("Dropdown list")
	d.dropdown1ListItems = slices.Delete(d.dropdown1ListItems, 0, len(d.dropdown1ListItems))
	d.dropdown1ListItems = model.DropdownLists().AppendDropdownListItems(d.dropdown1ListItems)
	d.dropdown1List.SetItems(d.dropdown1ListItems)
	context.SetEnabled(&d.dropdown1List, model.DropdownLists().Enabled())
	if d.dropdown1List.SelectedItemIndex() < 0 {
		d.dropdown1List.SelectItemByIndex(0)
	}

	// Dropdown list (Custom item)
	d.dropdown2ListText.SetValue("Dropdown list with custom widgets")
	// TODO: This logic is quite common. The implementation is basicwidget.adjustSliceSize. Refactor it.
	if len(d.dropdown2ListItemWidgets) < 3 {
		d.dropdown2ListItemWidgets = slices.Grow(d.dropdown2ListItemWidgets, 3-len(d.dropdown2ListItemWidgets))[:3]
	} else if len(d.dropdown2ListItemWidgets) > 3 {
		d.dropdown2ListItemWidgets = slices.Delete(d.dropdown2ListItemWidgets, 3, len(d.dropdown2ListItemWidgets))
	}
	img, err := theImageCache.Get("gopher_left")
	if err != nil {
		return err
	}
	d.dropdown2ListItemWidgets[0].SetImage(img)
	d.dropdown2ListItemWidgets[0].SetText("Left")
	img, err = theImageCache.Get("gopher_center")
	if err != nil {
		return err
	}
	d.dropdown2ListItemWidgets[1].SetImage(img)
	d.dropdown2ListItemWidgets[1].SetText("Center")
	img, err = theImageCache.Get("gopher_right")
	if err != nil {
		return err
	}
	d.dropdown2ListItemWidgets[2].SetImage(img)
	d.dropdown2ListItemWidgets[2].SetText("Right")

	d.dropdown2ListItems = slices.Delete(d.dropdown2ListItems, 0, len(d.dropdown2ListItems))
	for i := range d.dropdown2ListItemWidgets {
		w := &d.dropdown2ListItemWidgets[i]
		d.dropdown2ListItems = append(d.dropdown2ListItems, basicwidget.DropdownListItem[int]{
			Value:   i,
			Content: w,
		})
	}
	d.dropdown2List.SetItems(d.dropdown2ListItems)
	context.SetEnabled(&d.dropdown2List, model.DropdownLists().Enabled())
	if d.dropdown2List.SelectedItemIndex() < 0 {
		d.dropdown2List.SelectItemByIndex(0)
	}
	for i := range d.dropdown2ListItems {
		d.dropdown2ListItemWidgets[i].SetTextColor(d.dropdown2List.ItemTextColor(context, i))
	}

	d.listForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &d.dropdown1ListText,
			SecondaryWidget: &d.dropdown1List,
		},
		{
			PrimaryWidget:   &d.dropdown2ListText,
			SecondaryWidget: &d.dropdown2List,
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

type dropdownListItem struct {
	guigui.DefaultWidget

	image basicwidget.Image
	text  basicwidget.Text
}

func (d *dropdownListItem) SetImage(img *ebiten.Image) {
	d.image.SetImage(img)
}

func (d *dropdownListItem) SetText(s string) {
	d.text.SetValue(s)
}

func (d *dropdownListItem) SetTextColor(clr color.Color) {
	d.text.SetColor(clr)
}

func (d *dropdownListItem) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&d.image)
	adder.AddChild(&d.text)
}

func (d *dropdownListItem) Update(context *guigui.Context) error {
	d.text.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	return nil
}

func (d *dropdownListItem) layout(context *guigui.Context) guigui.LinearLayout {
	u := basicwidget.UnitSize(context)
	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &d.image,
				Size:   guigui.FixedSize(2 * u),
			},
			{
				Widget: &d.text,
			},
		},
		Gap: u / 4,
	}
}

func (d *dropdownListItem) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	d.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (d *dropdownListItem) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return d.layout(context).Measure(context, constraints)
}
