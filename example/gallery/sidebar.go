// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"image"

	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
)

type Sidebar struct {
	guigui.DefaultWidget

	panel        basicwidget.Panel
	panelContent sidebarContent
}

func (s *Sidebar) AppendChildWidgets(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
	appender.AppendChildWidget(&s.panel)
}

func (s *Sidebar) Update(context *guigui.Context) error {
	s.panel.SetStyle(basicwidget.PanelStyleSide)
	s.panel.SetBorders(basicwidget.PanelBorder{
		End: true,
	})
	s.panelContent.setSize(context.Bounds(s).Size())
	s.panel.SetContent(&s.panelContent)
	return nil
}

func (s *Sidebar) Layout(context *guigui.Context, widget guigui.Widget) image.Rectangle {
	switch widget {
	case &s.panel:
		return context.Bounds(s)
	}
	return image.Rectangle{}
}

type sidebarContent struct {
	guigui.DefaultWidget

	list basicwidget.List[string]

	size image.Point
}

func (s *sidebarContent) setSize(size image.Point) {
	s.size = size
}

func (s *sidebarContent) AppendChildWidgets(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
	appender.AppendChildWidget(&s.list)
}

func (s *sidebarContent) Update(context *guigui.Context) error {
	model := context.Model(s, modelKeyModel).(*Model)

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
	s.list.SetOnItemSelected(func(index int) {
		item, ok := s.list.ItemByIndex(index)
		if !ok {
			model.SetMode("")
			return
		}
		model.SetMode(item.Value)
	})

	return nil
}

func (s *sidebarContent) Layout(context *guigui.Context, widget guigui.Widget) image.Rectangle {
	switch widget {
	case &s.list:
		return context.Bounds(s)
	}
	return image.Rectangle{}
}

func (s *sidebarContent) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return s.size
}
