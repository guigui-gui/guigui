// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package guigui

import "image"

// WidgetBounds provides position and hit-testing information for a widget.
// It is passed to widget methods such as [Widget.Layout], [Widget.Draw], and input handlers.
type WidgetBounds struct {
	context *Context
	widget  Widget
}

// Bounds returns the widget's bounding rectangle in screen coordinates.
func (w *WidgetBounds) Bounds() image.Rectangle {
	return w.widget.widgetState().bounds
}

// VisibleBounds returns the portion of the widget's bounds that is actually visible on screen.
// This is the intersection of the widget's bounds with all ancestor clipping regions.
func (w *WidgetBounds) VisibleBounds() image.Rectangle {
	return w.context.visibleBounds(w.widget.widgetState())
}

// IsHitAtCursor reports whether the cursor is over this widget
// and no higher-layer widget is obscuring it at the cursor position.
func (w *WidgetBounds) IsHitAtCursor() bool {
	return w.context.app.isWidgetHitAtCursor(w.widget)
}
