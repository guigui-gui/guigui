// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"image"
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

// verticalStack arranges widgets vertically at their natural heights.
type verticalStack struct {
	guigui.DefaultWidget

	widgets []guigui.Widget

	layoutItems []guigui.LinearLayoutItem
	layout      guigui.LinearLayout
}

// SetWidgets sets the widgets to arrange from top to bottom.
func (v *verticalStack) SetWidgets(widgets []guigui.Widget) {
	v.widgets = slices.Delete(v.widgets, 0, len(v.widgets))
	v.widgets = append(v.widgets, widgets...)
}

func (v *verticalStack) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	for _, widget := range v.widgets {
		adder.AddWidget(widget)
	}
	return nil
}

func (v *verticalStack) buildLayout(context *guigui.Context) {
	u := basicwidget.UnitSize(context)
	v.layoutItems = slices.Delete(v.layoutItems, 0, len(v.layoutItems))
	for _, widget := range v.widgets {
		v.layoutItems = append(v.layoutItems, guigui.LinearLayoutItem{
			Widget: widget,
		})
	}
	v.layout = guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     v.layoutItems,
		Gap:       u / 2,
	}
}

func (v *verticalStack) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	v.buildLayout(context)
	v.layout.LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (v *verticalStack) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	v.buildLayout(context)
	return v.layout.Measure(context, constraints)
}
