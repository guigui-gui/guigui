// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image"
	"reflect"
	"sync"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

type roundedCornerWidget[T guigui.Widget] struct {
	guigui.DefaultWidget

	widget  lazyWidget[T]
	corners roundedCornerWidgetCorners

	disabled bool
}

func (r *roundedCornerWidget[T]) WriteStateKey(context *guigui.Context, w *guigui.StateKeyWriter) {
	w.WriteBool(r.disabled)
}

func (r *roundedCornerWidget[T]) SetCornderRouneded(rounded bool) {
	r.disabled = !rounded
}

func (r *roundedCornerWidget[T]) needsToRenderCorners(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	if r.disabled {
		return false
	}
	radius := RoundedCornerRadius(context)
	return draw.OverlapsWithRoundedCorner(r.corners.renderingBounds, radius, widgetBounds.Bounds())
}

func (r *roundedCornerWidget[T]) Widget() T {
	return r.widget.Widget()
}

func (r *roundedCornerWidget[T]) SetRenderingBounds(bounds image.Rectangle) {
	r.corners.setRenderingBounds(bounds)
}

func (r *roundedCornerWidget[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(r.widget.Widget())
	adder.AddWidget(&r.corners)
	context.DelegateFocus(r, r.widget.Widget())
	return nil
}

func (r *roundedCornerWidget[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(r.widget.Widget(), widgetBounds.Bounds())
	if r.needsToRenderCorners(context, widgetBounds) {
		layouter.LayoutWidget(&r.corners, widgetBounds.Bounds())
	}
}

func (r *roundedCornerWidget[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return r.widget.Widget().Measure(context, constraints)
}

func (r *roundedCornerWidget[T]) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if !r.needsToRenderCorners(context, widgetBounds) {
		return
	}
	r.corners.copyCorners(context, dst)
}

type roundedCornerWidgetCorners struct {
	guigui.DefaultWidget

	image           *ebiten.Image
	renderingBounds image.Rectangle
}

func (r *roundedCornerWidgetCorners) setRenderingBounds(bounds image.Rectangle) {
	r.renderingBounds = bounds
}

// copyCorners saves the pixels of src at the corners of the rendering bounds for Draw to restore.
func (r *roundedCornerWidgetCorners) copyCorners(context *guigui.Context, src *ebiten.Image) {
	radius := RoundedCornerRadius(context)
	size := image.Pt(2*radius, 2*radius)
	if r.image != nil && r.image.Bounds().Size() != size {
		r.image.Deallocate()
		r.image = nil
	}
	if r.image == nil {
		r.image = ebiten.NewImage(size.X, size.Y)
	}
	draw.CopyRoundedCorners(r.image, src, r.renderingBounds, radius)
}

func (r *roundedCornerWidgetCorners) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if r.image == nil {
		return
	}
	if r.renderingBounds.Empty() {
		return
	}
	draw.DrawRoundedCorners(dst, r.image, r.renderingBounds, RoundedCornerRadius(context))
}

type lazyWidget[T guigui.Widget] struct {
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
