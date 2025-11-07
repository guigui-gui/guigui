// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package guigui

import "image"

type WidgetBounds struct {
	context     *Context
	widgetState *widgetState
}

func (w *WidgetBounds) Bounds() image.Rectangle {
	return w.widgetState.bounds
}

func (w *WidgetBounds) VisibleBounds() image.Rectangle {
	return w.context.visibleBounds(w.widgetState)
}
