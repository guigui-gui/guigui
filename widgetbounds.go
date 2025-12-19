// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package guigui

import "image"

type WidgetBounds struct {
	context     *Context
	widget      Widget
	hitDisabled bool
}

func (w *WidgetBounds) Bounds() image.Rectangle {
	return w.widget.widgetState().bounds
}

func (w *WidgetBounds) VisibleBounds() image.Rectangle {
	return w.context.visibleBounds(w.widget.widgetState())
}

func (w *WidgetBounds) IsHitAtCursor() bool {
	if w.hitDisabled {
		return false
	}
	return w.context.app.isWidgetHitAtCursor(w.widget)
}
