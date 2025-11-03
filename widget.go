// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package guigui

import (
	"image"
	"reflect"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

type Widget interface {
	Model(key any) any
	AddChildren(context *Context, adder *ChildAdder)
	Update(context *Context, widgetBounds *WidgetBounds) error
	Layout(context *Context, widgetBounds *WidgetBounds, widget Widget) image.Rectangle
	HandlePointingInput(context *Context, widgetBounds *WidgetBounds) HandleInputResult
	HandleButtonInput(context *Context, widgetBounds *WidgetBounds) HandleInputResult
	Tick(context *Context, widgetBounds *WidgetBounds) error
	CursorShape(context *Context, widgetBounds *WidgetBounds) (ebiten.CursorShapeType, bool)
	Draw(context *Context, widgetBounds *WidgetBounds, dst *ebiten.Image)
	ZDelta() int
	Measure(context *Context, constraints Constraints) image.Point

	widgetState() *widgetState
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

func (w *WidgetWithSize[T]) AddChildren(context *Context, adder *ChildAdder) {
	adder.AddChild(w.Widget())
}

func (w *WidgetWithSize[T]) Layout(context *Context, widgetBounds *WidgetBounds, widget Widget) image.Rectangle {
	if widget == Widget(w.Widget()) {
		// WidgetWithSize overwrites Measure, but doesn't overwrite Layout.
		return widgetBounds.Bounds()
	}
	return image.Rectangle{}
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

func (w *WidgetWithPadding[T]) AddChildren(context *Context, adder *ChildAdder) {
	adder.AddChild(w.Widget())
}

func (w *WidgetWithPadding[T]) Layout(context *Context, widgetBounds *WidgetBounds, widget Widget) image.Rectangle {
	if widget == Widget(w.Widget()) {
		b := widgetBounds.Bounds()
		b.Min.X += w.padding.Start
		b.Min.Y += w.padding.Top
		b.Max.X -= w.padding.End
		b.Max.Y -= w.padding.Bottom
		return b
	}
	return image.Rectangle{}
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
		if t.Kind() == reflect.Ptr {
			l.widget = reflect.New(t.Elem()).Interface().(T)
		}
	})
	return l.widget
}
