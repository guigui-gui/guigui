// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Tooltips struct {
	guigui.DefaultWidget

	tooltip1 basicwidget.WidgetWithTooltip[*basicwidget.Button]
	tooltip2 basicwidget.WidgetWithTooltip[*basicwidget.Text]
}

func (t *Tooltips) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.tooltip1)
	adder.AddWidget(&t.tooltip2)

	t.tooltip1.Widget().SetText("Hover me")
	t.tooltip1.SetTooltipText("This is a button tooltip")

	t.tooltip2.Widget().SetValue("Hover over this text to see a tooltip")
	t.tooltip2.SetTooltipText("This is a text tooltip")

	return nil
}

func (t *Tooltips) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &t.tooltip1,
			},
			{
				Widget: &t.tooltip2,
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
