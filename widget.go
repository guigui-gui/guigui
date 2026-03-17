// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package guigui

import (
	"image"
	"reflect"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// Widget is the interface that all UI components must implement.
// Implementations must embed DefaultWidget, as it is the only way to satisfy
// the unexported widgetState method in this interface.
//
// A Widget implementation should work with its zero value.
// In Go, the zero value of a variable is the default value
// (0 for numbers, false for booleans, "" for strings, nil for pointers, etc.).
// This means that the default state of a widget should be reasonable
// without any explicit initialization.
type Widget interface {
	// Env returns an environment value associated with the widget for the given key.
	// [Context.Env] calls this method on the given widget first. If this returns nil,
	// it tries the parent widget, repeating recursively up to the root widget.
	// source provides information about the origin of the [Context.Env] call.
	Env(context *Context, key EnvKey, source *EnvSource) any

	// Build constructs the widget's child widget tree.
	// Use adder to add child widgets that this widget contains.
	// Build is called whenever the widget tree needs to be reconstructed.
	Build(context *Context, adder *ChildAdder) error

	// Layout positions and sizes the widget's children within the widget's bounds.
	// Use layouter to set the bounds of each child widget added in Build.
	Layout(context *Context, widgetBounds *WidgetBounds, layouter *ChildLayouter)

	// HandlePointingInput handles mouse or touch input events for the widget.
	// widgetBounds provides the widget's position and hit-testing information.
	HandlePointingInput(context *Context, widgetBounds *WidgetBounds) HandleInputResult

	// HandleButtonInput handles keyboard and gamepad button input events for the widget.
	// widgetBounds provides the widget's position and hit-testing information.
	// It is invoked when the widget or its ancestor is focused,
	// or when the widget contains a focused child widget.
	HandleButtonInput(context *Context, widgetBounds *WidgetBounds) HandleInputResult

	// Tick is called every tick to update the widget's state.
	// Use this for animations, timers, or other per-tick updates.
	Tick(context *Context, widgetBounds *WidgetBounds) error

	// CursorShape returns the cursor shape to display when the cursor is over this widget.
	// The bool return value indicates whether the widget specifies a cursor shape.
	// If false is returned, the parent widget's cursor shape is used.
	CursorShape(context *Context, widgetBounds *WidgetBounds) (ebiten.CursorShapeType, bool)

	// Draw renders the widget onto dst.
	// dst is a SubImage clipped to the widget's bounds.
	Draw(context *Context, widgetBounds *WidgetBounds, dst *ebiten.Image)

	// Measure returns the preferred size of the widget given the constraints.
	// The returned value is advisory; the parent performing layout is not obligated to use it.
	// The constraints may specify fixed width and/or height that the widget should respect.
	Measure(context *Context, constraints Constraints) image.Point

	widgetState() *widgetState
}

func areWidgetsSame(a Widget, b Widget) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.widgetState() == b.widgetState()
}

type HandleInputResult struct {
	widget  Widget
	aborted bool
}

func HandleInputByWidget(widget Widget) HandleInputResult {
	return HandleInputResult{
		widget: widget,
	}
}

func AbortHandlingInputByWidget(widget Widget) HandleInputResult {
	return HandleInputResult{
		aborted: true,
		widget:  widget,
	}
}

func (r *HandleInputResult) shouldRaise() bool {
	return r.widget != nil || r.aborted
}

type WidgetWithSize[T Widget] struct {
	DefaultWidget

	widget lazyWidget[T]

	measure        func(context *Context, constraints Constraints) image.Point
	fixedSizePlus1 image.Point
}

func (w *WidgetWithSize[T]) SetMeasureFunc(f func(context *Context, constraints Constraints) image.Point) {
	w.measure = f
	w.fixedSizePlus1 = image.Point{}
}

func (w *WidgetWithSize[T]) SetFixedWidth(width int) {
	w.measure = nil
	w.fixedSizePlus1 = image.Point{X: width + 1, Y: 0}
}

func (w *WidgetWithSize[T]) SetFixedHeight(height int) {
	w.measure = nil
	w.fixedSizePlus1 = image.Point{X: 0, Y: height + 1}
}

func (w *WidgetWithSize[T]) SetFixedSize(size image.Point) {
	w.measure = nil
	w.fixedSizePlus1 = size.Add(image.Pt(1, 1))
}

func (w *WidgetWithSize[T]) SetIntrinsicSize() {
	w.measure = nil
	w.fixedSizePlus1 = image.Point{}
}

func (w *WidgetWithSize[T]) Widget() T {
	return w.widget.Widget()
}

func (w *WidgetWithSize[T]) Build(context *Context, adder *ChildAdder) error {
	adder.AddWidget(w.Widget())
	context.DelegateFocus(w, w.Widget())
	return nil
}

func (w *WidgetWithSize[T]) Layout(context *Context, widgetBounds *WidgetBounds, layouter *ChildLayouter) {
	layouter.LayoutWidget(w.Widget(), widgetBounds.Bounds())
}

func (w *WidgetWithSize[T]) Measure(context *Context, constraints Constraints) image.Point {
	if w.measure != nil {
		return w.measure(context, constraints)
	}
	if w.fixedSizePlus1.X > 0 && w.fixedSizePlus1.Y > 0 {
		return w.fixedSizePlus1.Sub(image.Pt(1, 1))
	}
	if w.fixedSizePlus1.X > 0 {
		// TODO: Consider constraints.
		s := w.Widget().Measure(context, FixedWidthConstraints(w.fixedSizePlus1.X-1))
		return image.Pt(w.fixedSizePlus1.X-1, s.Y)
	}
	if w.fixedSizePlus1.Y > 0 {
		// TODO: Consider constraints.
		s := w.Widget().Measure(context, FixedHeightConstraints(w.fixedSizePlus1.Y-1))
		return image.Pt(s.X, w.fixedSizePlus1.Y-1)
	}
	return w.Widget().Measure(context, constraints)
}

type WidgetWithPadding[T Widget] struct {
	DefaultWidget

	widget  lazyWidget[T]
	padding Padding
}

func (w *WidgetWithPadding[T]) SetPadding(padding Padding) {
	w.padding = padding
}

func (w *WidgetWithPadding[T]) Widget() T {
	return w.widget.Widget()
}

func (w *WidgetWithPadding[T]) Build(context *Context, adder *ChildAdder) error {
	adder.AddWidget(w.Widget())
	context.DelegateFocus(w, w.Widget())
	return nil
}

func (w *WidgetWithPadding[T]) Layout(context *Context, widgetBounds *WidgetBounds, layouter *ChildLayouter) {
	b := widgetBounds.Bounds()
	b.Min.X += w.padding.Start
	b.Min.Y += w.padding.Top
	b.Max.X -= w.padding.End
	b.Max.Y -= w.padding.Bottom
	layouter.LayoutWidget(w.Widget(), b)
}

func (w *WidgetWithPadding[T]) Measure(context *Context, constraints Constraints) image.Point {
	// TODO: What if constraints can have fixed width and height at the same time?
	if fixedWidth, ok := constraints.FixedWidth(); ok {
		constraints = FixedWidthConstraints(fixedWidth - w.padding.Start - w.padding.End)
	}
	if fixedHeight, ok := constraints.FixedHeight(); ok {
		constraints = FixedHeightConstraints(fixedHeight - w.padding.Top - w.padding.Bottom)
	}
	s := w.Widget().Measure(context, constraints)
	s.X += w.padding.Start + w.padding.End
	s.Y += w.padding.Top + w.padding.Bottom
	return s
}

type lazyWidget[T Widget] struct {
	widget T
	once   sync.Once
}

func (l *lazyWidget[T]) Widget() T {
	l.once.Do(func() {
		t := reflect.TypeFor[T]()
		if t.Kind() == reflect.Pointer {
			l.widget = reflect.New(t.Elem()).Interface().(T)
		}
	})
	return l.widget
}
