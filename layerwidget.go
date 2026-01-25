// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package guigui

import "image"

// LayerWidget is a widget that can be in a different layer from its parent.
// LayerWidget is on the same layer as its parent by default.
type LayerWidget[T Widget] struct {
	DefaultWidget

	widget lazyWidget[T]
}

// Widget returns the content widget.
func (p *LayerWidget[T]) Widget() T {
	return p.widget.Widget()
}

// BringToFrontLayer brings the widget to the front layer.
// After this call, the widget will be in a different layer from its parent.
//
// Lyaers affect the order of rendering and input handling.
// Usually, a widget's visible bounds are constrained by its parent's visible bounds,
// which means a widget cannot be rendered outside of its parent's visible bounds.
// If a widget is in a different layer from its parent,
// the widget can be rendered regardless of its parent's visible bounds.
//
// Input is handled in the order of layers from top to bottom.
// Also, layers affect the result of [WidgetBounds.IsCursorHitAt].
func (p *LayerWidget[T]) BringToFrontLayer(context *Context) {
	context.bringToFrontLayer(p)
}

// Build implements [Widget.Build].
func (p *LayerWidget[T]) Build(context *Context, adder *ChildAdder) error {
	adder.AddChild(p.widget.Widget())
	return nil
}

// Layout implements [Widget.Layout].
func (p *LayerWidget[T]) Layout(context *Context, widgetBounds *WidgetBounds, layouter *ChildLayouter) {
	layouter.LayoutWidget(p.widget.Widget(), widgetBounds.Bounds())
}

// Measure implements [Widget.Measure].
func (p *LayerWidget[T]) Measure(context *Context, constraints Constraints) image.Point {
	return p.widget.Widget().Measure(context, constraints)
}
