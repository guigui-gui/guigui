// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type ContentPanel struct {
	guigui.DefaultWidget

	panel   basicwidget.Panel
	content guigui.WidgetWithSize[*contentPanelContent]
}

func (c *ContentPanel) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&c.panel)
	c.panel.SetContent(&c.content)
	return nil
}

func (c *ContentPanel) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	c.content.SetFixedSize(widgetBounds.Bounds().Size())
	layouter.LayoutWidget(&c.panel, widgetBounds.Bounds())
}

type contentPanelContent struct {
	guigui.DefaultWidget

	text basicwidget.Text
}

func (c *contentPanelContent) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&c.text)
	c.text.SetValue("Content panel: " + dummyText)
	c.text.SetAutoWrap(true)
	c.text.SetSelectable(true)
	return nil
}

func (c *contentPanelContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	layouter.LayoutWidget(&c.text, widgetBounds.Bounds().Inset(u/2))
}
