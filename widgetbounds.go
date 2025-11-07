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
	state := w.widgetState
	if state.hasVisibleBoundsCache {
		return state.visibleBoundsCache
	}

	parent := state.parent
	if parent == nil {
		b := w.context.app.bounds()
		state.hasVisibleBoundsCache = true
		state.visibleBoundsCache = b
		return b
	}
	if w.widgetState.zDelta != 0 {
		b := state.bounds
		state.hasVisibleBoundsCache = true
		state.visibleBoundsCache = b
		return b
	}

	var b image.Rectangle
	parentVB := widgetBoundsFromWidget(w.context, parent.widgetState()).VisibleBounds()
	if !parentVB.Empty() {
		b = parentVB.Intersect(state.bounds)
	}
	state.hasVisibleBoundsCache = true
	state.visibleBoundsCache = b
	return b
}
