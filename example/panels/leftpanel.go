// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type LeftPanel struct {
	guigui.DefaultWidget

	panel   basicwidget.Panel
	content guigui.WidgetWithSize[*leftPanelContent]
}

func (l *LeftPanel) Build(context *guigui.Context, adder *guigui.WidgetAdder) error {
	adder.AddChild(&l.panel)
	l.panel.SetStyle(basicwidget.PanelStyleSide)
	l.panel.SetBorders(basicwidget.PanelBorders{
		End: true,
	})
	l.panel.SetContent(&l.content)
	return nil
}

func (l *LeftPanel) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	l.content.SetFixedSize(widgetBounds.Bounds().Size())
	layouter.LayoutWidget(&l.panel, widgetBounds.Bounds())
}

type leftPanelContent struct {
	guigui.DefaultWidget

	text basicwidget.Text
}

func (l *leftPanelContent) Build(context *guigui.Context, adder *guigui.WidgetAdder) error {
	adder.AddChild(&l.text)
	l.text.SetValue("Left panel: " + dummyText)
	l.text.SetAutoWrap(true)
	l.text.SetSelectable(true)
	return nil
}

func (l *leftPanelContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	layouter.LayoutWidget(&l.text, widgetBounds.Bounds().Inset(u/2))
}
