// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"image"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type LeftPanel struct {
	guigui.DefaultWidget

	panel   basicwidget.Panel
	content guigui.WidgetWithSize[*leftPanelContent]
}

func (l *LeftPanel) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&l.panel)
}

func (l *LeftPanel) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	l.panel.SetStyle(basicwidget.PanelStyleSide)
	l.panel.SetBorders(basicwidget.PanelBorder{
		End: true,
	})
	l.content.SetFixedSize(widgetBounds.Bounds().Size())
	l.panel.SetContent(&l.content)

	return nil
}

func (l *LeftPanel) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, widget guigui.Widget) image.Rectangle {
	switch widget {
	case &l.panel:
		return widgetBounds.Bounds()
	}
	return image.Rectangle{}
}

type leftPanelContent struct {
	guigui.DefaultWidget

	text basicwidget.Text
}

func (l *leftPanelContent) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&l.text)
}

func (l *leftPanelContent) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	l.text.SetValue("Left panel: " + dummyText)
	l.text.SetAutoWrap(true)
	l.text.SetSelectable(true)
	return nil
}

func (l *leftPanelContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, widget guigui.Widget) image.Rectangle {
	switch widget {
	case &l.text:
		u := basicwidget.UnitSize(context)
		return widgetBounds.Bounds().Inset(u / 2)
	}
	return image.Rectangle{}
}
