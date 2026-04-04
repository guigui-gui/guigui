// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"image"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type TooltipAreas struct {
	guigui.DefaultWidget

	button       basicwidget.Button
	text         basicwidget.Text
	tooltipArea1 basicwidget.TooltipArea
	tooltipArea2 basicwidget.TooltipArea
}

func (t *TooltipAreas) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.button)
	adder.AddWidget(&t.tooltipArea1)
	adder.AddWidget(&t.text)
	adder.AddWidget(&t.tooltipArea2)

	t.button.SetText("Hover me")
	t.tooltipArea1.SetText("This is a button tooltip")

	t.text.SetValue("Hover over this text to see a tooltip")
	t.tooltipArea2.SetText("This is a text tooltip")

	return nil
}

func (t *TooltipAreas) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	bounds := widgetBounds.Bounds()
	padding := u / 2
	gap := u / 2

	x := bounds.Min.X + padding
	y := bounds.Min.Y + padding
	w := bounds.Dx() - padding*2

	buttonSize := t.button.Measure(context, guigui.Constraints{})
	buttonBounds := image.Rect(x, y, x+w, y+buttonSize.Y)
	layouter.LayoutWidget(&t.button, buttonBounds)
	layouter.LayoutWidget(&t.tooltipArea1, buttonBounds)

	y = buttonBounds.Max.Y + gap

	textSize := t.text.Measure(context, guigui.Constraints{})
	textBounds := image.Rect(x, y, x+w, y+textSize.Y)
	layouter.LayoutWidget(&t.text, textBounds)
	layouter.LayoutWidget(&t.tooltipArea2, textBounds)
}
