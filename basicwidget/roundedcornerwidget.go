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

func (r *roundedCornerWidget[T]) SetCornderRouneded(rounded bool) {
	if r.disabled == !rounded {
		return
	}
	r.disabled = !rounded
	guigui.RequestRebuild(r)
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
	adder.AddChild(r.widget.Widget())
	adder.AddChild(&r.corners)
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

	if r.corners.image != nil {
		if !dst.Bounds().In(r.corners.image.Bounds()) {
			r.corners.image.Deallocate()
			r.corners.image = nil
		}
	}
	if r.corners.image == nil {
		r.corners.image = ebiten.NewImageWithOptions(dst.Bounds(), nil)
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(dst.Bounds().Min.X), float64(dst.Bounds().Min.Y))
	op.Blend = ebiten.BlendCopy
	r.corners.image.SubImage(dst.Bounds()).(*ebiten.Image).DrawImage(dst, op)
}

type roundedCornerWidgetCorners struct {
	guigui.DefaultWidget

	image           *ebiten.Image
	renderingBounds image.Rectangle
}

func (r *roundedCornerWidgetCorners) setRenderingBounds(bounds image.Rectangle) {
	r.renderingBounds = bounds
}

func (r *roundedCornerWidgetCorners) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if r.image == nil {
		return
	}
	if r.renderingBounds.Empty() {
		return
	}
	// TODO: This rendering is not efficient. Improve the performance.
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(dst.Bounds().Min.X), float64(dst.Bounds().Min.Y))
	op.Blend = ebiten.BlendCopy
	draw.DrawRoundedCorners(dst, r.image.SubImage(dst.Bounds()).(*ebiten.Image), r.renderingBounds, RoundedCornerRadius(context), op)
}

type lazyWidget[T guigui.Widget] struct {
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
