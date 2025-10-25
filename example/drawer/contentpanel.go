// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"image"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type ContentPanel struct {
	guigui.DefaultWidget

	panel   basicwidget.Panel
	content guigui.WidgetWithSize[*contentPanelContent]
}

func (c *ContentPanel) AddChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, adder *guigui.ChildAdder) {
	adder.AddChild(&c.panel)
}

func (c *ContentPanel) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	c.content.SetFixedSize(widgetBounds.Bounds().Size())
	c.panel.SetContent(&c.content)
	return nil
}

func (c *ContentPanel) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, widget guigui.Widget) image.Rectangle {
	switch widget {
	case &c.panel:
		return widgetBounds.Bounds()
	}
	return image.Rectangle{}
}

type contentPanelContent struct {
	guigui.DefaultWidget

	text basicwidget.Text
}

func (c *contentPanelContent) AddChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, adder *guigui.ChildAdder) {
	adder.AddChild(&c.text)
}

func (c *contentPanelContent) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	c.text.SetValue("Content panel: " + dummyText)
	c.text.SetAutoWrap(true)
	c.text.SetSelectable(true)
	return nil
}

func (c *contentPanelContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, widget guigui.Widget) image.Rectangle {
	switch widget {
	case &c.text:
		u := basicwidget.UnitSize(context)
		return widgetBounds.Bounds().Inset(u / 2)
	}
	return image.Rectangle{}
}
