// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package guigui

type ChildAdder struct {
	app    *app
	widget Widget
}

// TODO: Rename this to AddWidget.
func (c *ChildAdder) AddChild(widget Widget) {
	widgetState := widget.widgetState()
	widgetState.parent = c.widget
	widgetState.builtAt = c.app.buildCount
	cWidgetState := c.widget.widgetState()
	cWidgetState.children = append(cWidgetState.children, widget)
}
