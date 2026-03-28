// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"image"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Tooltips struct {
	guigui.DefaultWidget

	button   basicwidget.Button
	text     basicwidget.Text
	tooltip1 basicwidget.Tooltip
	tooltip2 basicwidget.Tooltip
}

func (t *Tooltips) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.button)
	adder.AddWidget(&t.tooltip1)
	adder.AddWidget(&t.text)
	adder.AddWidget(&t.tooltip2)

	t.button.SetText("Hover me")
	t.tooltip1.SetText("This is a button tooltip")

	t.text.SetValue("Hover over this text to see a tooltip")
	t.tooltip2.SetText("This is a text tooltip")

	return nil
}

func (t *Tooltips) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
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
	t.tooltip1.SetHoverBounds(buttonBounds)
	layouter.LayoutWidget(&t.tooltip1, bounds)

	y = buttonBounds.Max.Y + gap

	textSize := t.text.Measure(context, guigui.Constraints{})
	textBounds := image.Rect(x, y, x+w, y+textSize.Y)
	layouter.LayoutWidget(&t.text, textBounds)
	t.tooltip2.SetHoverBounds(textBounds)
	layouter.LayoutWidget(&t.tooltip2, bounds)
}
