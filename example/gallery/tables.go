// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Tables struct {
	guigui.DefaultWidget

	table basicwidget.Table[int]

	configForm       basicwidget.Form
	showFooterText   basicwidget.Text
	showFooterToggle basicwidget.Toggle
	movableText      basicwidget.Text
	movableToggle    basicwidget.Toggle
	enabledText      basicwidget.Text
	enabledToggle    basicwidget.Toggle

	tableRows []basicwidget.TableRow[int]
}

func (t *Tables) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&t.table)
	adder.AddChild(&t.configForm)
}

func (t *Tables) Update(context *guigui.Context) error {
	model := context.Model(t, modelKeyModel).(*Model)

	u := basicwidget.UnitSize(context)
	t.table.SetColumns([]basicwidget.TableColumn{
		{
			HeaderText:                "ID",
			HeaderTextHorizontalAlign: basicwidget.HorizontalAlignRight,
			Width:                     guigui.FlexibleSize(1),
			MinWidth:                  2 * u,
		},
		{
			HeaderText: "Name",
			Width:      guigui.FlexibleSize(2),
			MinWidth:   4 * u,
		},
		{
			HeaderText:                "Amount",
			HeaderTextHorizontalAlign: basicwidget.HorizontalAlignRight,
			Width:                     guigui.FlexibleSize(1),
			MinWidth:                  2 * u,
		},
		{
			HeaderText:                "Cost",
			HeaderTextHorizontalAlign: basicwidget.HorizontalAlignRight,
			Width:                     guigui.FlexibleSize(1),
			MinWidth:                  2 * u,
		},
	})

	// Prepare widgets for table rows.
	// Use slices.Grow not to delete cells every frame.
	if newNum := model.Tables().TableItemCount(); len(t.tableRows) < newNum {
		t.tableRows = slices.Grow(t.tableRows, newNum-len(t.tableRows))[:newNum]
	} else {
		t.tableRows = slices.Delete(t.tableRows, newNum, len(t.tableRows))
	}

	const n = 4
	for i, item := range model.Tables().TableItems() {
		t.tableRows[i].Movable = model.Tables().Movable()
		t.tableRows[i].Value = item.ID

		if len(t.tableRows[i].Cells) < n {
			t.tableRows[i].Cells = make([]basicwidget.TableCell, n)
		}

		t.tableRows[i].Cells[0].Text = strconv.Itoa(item.ID)
		t.tableRows[i].Cells[0].TextHorizontalAlign = basicwidget.HorizontalAlignRight
		t.tableRows[i].Cells[0].TextTabular = true

		t.tableRows[i].Cells[1].Text = item.Name

		t.tableRows[i].Cells[2].Text = strconv.Itoa(item.Amount)
		t.tableRows[i].Cells[2].TextHorizontalAlign = basicwidget.HorizontalAlignRight
		t.tableRows[i].Cells[2].TextTabular = true

		t.tableRows[i].Cells[3].Text = fmt.Sprintf("%d.%02d", item.Cost/100, item.Cost%100)
		t.tableRows[i].Cells[3].TextHorizontalAlign = basicwidget.HorizontalAlignRight
		t.tableRows[i].Cells[3].TextTabular = true
	}
	t.table.SetItems(t.tableRows)
	// Set the text colors after setting the items, or ItemTextColor will not work correctly.
	for i := range model.Tables().TableItemCount() {
		clr := t.table.ItemTextColor(context, i)
		for j := range n {
			t.tableRows[i].Cells[j].TextColor = clr
		}
	}
	if model.Tables().IsFooterVisible() {
		t.table.SetFooterHeight(u)
	} else {
		t.table.SetFooterHeight(0)
	}
	context.SetEnabled(&t.table, model.Tables().Enabled())
	t.table.SetOnItemsMoved(func(from, count, to int) {
		idx := model.Tables().MoveTableItems(from, count, to)
		t.table.SelectItemByIndex(idx)
	})

	// Configurations
	t.showFooterText.SetValue("Show footer")
	t.showFooterToggle.SetOnValueChanged(func(value bool) {
		model.Tables().SetFooterVisible(value)
	})
	t.movableText.SetValue("Enable to move items")
	t.movableToggle.SetValue(model.Tables().Movable())
	t.movableToggle.SetOnValueChanged(func(value bool) {
		model.Tables().SetMovable(value)
	})
	t.enabledText.SetValue("Enabled")
	t.enabledToggle.SetOnValueChanged(func(value bool) {
		model.Tables().SetEnabled(value)
	})
	t.enabledToggle.SetValue(model.Tables().Enabled())

	t.configForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &t.showFooterText,
			SecondaryWidget: &t.showFooterToggle,
		},
		{
			PrimaryWidget:   &t.movableText,
			SecondaryWidget: &t.movableToggle,
		},
		{
			PrimaryWidget:   &t.enabledText,
			SecondaryWidget: &t.enabledToggle,
		},
	})

	// layout handled in Layout using LinearLayout

	return nil
}

func (t *Tables) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &t.table, Size: guigui.FixedSize(12 * u),
			},
			{
				Size: guigui.FlexibleSize(1),
			},
			{
				Widget: &t.configForm,
			},
		},
	}).LayoutWidgets(context, widgetBounds.Bounds().Inset(u/2), layouter)
}
