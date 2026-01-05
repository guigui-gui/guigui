// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type RightPanel struct {
	guigui.DefaultWidget

	panel   basicwidget.Panel
	content guigui.WidgetWithSize[*rightPanelContent]
}

func (r *RightPanel) Build(context *guigui.Context, adder *guigui.WidgetAdder) error {
	adder.AddChild(&r.panel)
	r.panel.SetStyle(basicwidget.PanelStyleSide)
	r.panel.SetBorders(basicwidget.PanelBorders{
		Start: true,
	})
	r.panel.SetContent(&r.content)
	return nil
}

func (r *RightPanel) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	r.content.SetFixedSize(widgetBounds.Bounds().Size())
	layouter.LayoutWidget(&r.panel, widgetBounds.Bounds())
}

type rightPanelContent struct {
	guigui.DefaultWidget

	text basicwidget.Text
}

func (r *rightPanelContent) Build(context *guigui.Context, adder *guigui.WidgetAdder) error {
	adder.AddChild(&r.text)
	r.text.SetValue("Right panel: " + dummyText)
	r.text.SetAutoWrap(true)
	r.text.SetSelectable(true)
	return nil
}

func (r *rightPanelContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	layouter.LayoutWidget(&r.text, widgetBounds.Bounds().Inset(u/2))
}
