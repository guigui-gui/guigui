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

func (t *Tables) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&t.table)
	adder.AddChild(&t.configForm)

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
	guigui.RegisterEventHandler2(t, &t.table)

	// Configurations
	t.showFooterText.SetValue("Show footer")
	guigui.RegisterEventHandler2(t, &t.showFooterToggle)

	t.movableText.SetValue("Enable to move items")
	t.movableToggle.SetValue(model.Tables().Movable())
	guigui.RegisterEventHandler2(t, &t.movableToggle)

	t.enabledText.SetValue("Enabled")
	guigui.RegisterEventHandler2(t, &t.enabledToggle)
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

	return nil
}

func (t *Tables) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
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
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (t *Tables) HandleEvent(context *guigui.Context, targetWidget guigui.Widget, eventArgs any) {
	model := context.Model(t, modelKeyModel).(*Model)
	switch targetWidget {
	case &t.table:
		switch eventArgs := eventArgs.(type) {
		case *basicwidget.TableEventArgsItemsMoved:
			idx := model.Tables().MoveTableItems(eventArgs.From, eventArgs.Count, eventArgs.To)
			t.table.SelectItemByIndex(idx)
		}
	case &t.showFooterToggle:
		switch eventArgs := eventArgs.(type) {
		case *basicwidget.ToggleEventArgsValueChanged:
			model.Tables().SetFooterVisible(eventArgs.Value)
		}
	case &t.movableToggle:
		switch eventArgs := eventArgs.(type) {
		case *basicwidget.ToggleEventArgsValueChanged:
			model.Tables().SetMovable(eventArgs.Value)
		}
	case &t.enabledToggle:
		switch eventArgs := eventArgs.(type) {
		case *basicwidget.ToggleEventArgsValueChanged:
			model.Tables().SetEnabled(eventArgs.Value)
		}
	}
}
