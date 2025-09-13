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
	AppendChildWidgets(context *Context, appender *ChildWidgetAppender)
	Update(context *Context) error
	Layout(context *Context, widget Widget) image.Rectangle
	HandlePointingInput(context *Context) HandleInputResult
	HandleButtonInput(context *Context) HandleInputResult
	Tick(context *Context) error
	CursorShape(context *Context) (ebiten.CursorShapeType, bool)
	Draw(context *Context, dst *ebiten.Image)
	ZDelta() int
	Measure(context *Context, constraints Constraints) image.Point
	PassThrough() bool

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

	widget T

	measure        func(context *Context, constraints Constraints) image.Point
	fixedSizePlus1 image.Point

	initOnce sync.Once
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
	w.initOnce.Do(func() {
		t := reflect.TypeFor[T]()
		if t.Kind() == reflect.Ptr {
			w.widget = reflect.New(t.Elem()).Interface().(T)
		}
	})
	return w.widget
}

func (w *WidgetWithSize[T]) AppendChildWidgets(context *Context, appender *ChildWidgetAppender) {
	appender.AppendChildWidget(w.Widget())
}

func (w *WidgetWithSize[T]) Layout(context *Context, widget Widget) image.Rectangle {
	if widget == Widget(w.Widget()) {
		// WidgetWithSize overwrites Measure, but doesn't overwrite Layout.
		return context.Bounds(w)
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
