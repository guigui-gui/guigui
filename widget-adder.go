// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package guigui

type WidgetAdder struct {
	app    *app
	widget Widget
}

func (c *WidgetAdder) AddChild(widget Widget){
	c.AddWidget(widget)
}

func (c *WidgetAdder) AddWidget(widget Widget) {
	widgetState := widget.widgetState()
	widgetState.parent = c.widget
	widgetState.builtAt = c.app.buildCount
	cWidgetState := c.widget.widgetState()
	cWidgetState.children = append(cWidgetState.children, widget)
}
