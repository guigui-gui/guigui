// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"image"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Sidebar struct {
	guigui.DefaultWidget

	panel        basicwidget.Panel
	panelContent sidebarContent
}

func (s *Sidebar) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&s.panel)
	s.panel.SetStyle(basicwidget.PanelStyleSide)
	s.panel.SetBorders(basicwidget.PanelBorders{
		End: true,
	})
	s.panel.SetContent(&s.panelContent)
	return nil
}

func (s *Sidebar) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	s.panelContent.setSize(widgetBounds.Bounds().Size())
	layouter.LayoutWidget(&s.panel, widgetBounds.Bounds())
}

type sidebarContent struct {
	guigui.DefaultWidget

	list basicwidget.List[string]

	size image.Point
}

func (s *sidebarContent) setSize(size image.Point) {
	s.size = size
}

func (s *sidebarContent) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&s.list)

	model := context.Data(s, modelKeyModel).(*Model)

	s.list.SetStyle(basicwidget.ListStyleSidebar)

	items := []basicwidget.ListItem[string]{
		{
			Text:  "Settings",
			Value: "settings",
		},
		{
			Text:  "Basic",
			Value: "basic",
		},
		{
			Text:  "Buttons",
			Value: "buttons",
		},
		{
			Text:  "Checkboxes",
			Value: "checkboxes",
		},
		{
			Text:  "Radio Buttons",
			Value: "radiobuttons",
		},
		{
			Text:  "Texts",
			Value: "texts",
		},
		{
			Text:  "Text Inputs",
			Value: "textinputs",
		},
		{
			Text:  "Number Inputs",
			Value: "numberinputs",
		},
		{
			Text:  "Lists",
			Value: "lists",
		},
		{
			Text:  "Selects",
			Value: "selects",
		},
		{
			Text:  "Tables",
			Value: "tables",
		},
		{
			Text:  "Popups",
			Value: "popups",
		},
	}

	s.list.SetItems(items)
	s.list.SelectItemByValue(model.Mode())
	s.list.SetItemHeight(basicwidget.UnitSize(context))
	s.list.OnItemSelected(func(context *guigui.Context, index int) {
		item, ok := s.list.ItemByIndex(index)
		if !ok {
			model.SetMode("")
			return
		}
		model.SetMode(item.Value)
	})

	return nil
}

func (s *sidebarContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&s.list, widgetBounds.Bounds())
}

func (s *sidebarContent) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return s.size
}
