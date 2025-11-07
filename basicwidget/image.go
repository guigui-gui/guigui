// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
)

type Image struct {
	guigui.DefaultWidget

	image *ebiten.Image
}

func (i *Image) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if i.image == nil {
		return
	}

	b := widgetBounds.Bounds()
	imgScale := min(float64(b.Dx())/float64(i.image.Bounds().Dx()), float64(b.Dy())/float64(i.image.Bounds().Dy()))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(imgScale, imgScale)
	op.GeoM.Translate(float64(b.Min.X), float64(b.Min.Y))
	op.GeoM.Translate((float64(b.Dx())-float64(i.image.Bounds().Dx())*imgScale)/2,
		(float64(b.Dy())-float64(i.image.Bounds().Dy())*imgScale)/2)
	if !context.IsEnabled(i) {
		// TODO: Reduce the saturation?
		op.ColorScale.ScaleAlpha(0.25)
	}
	// TODO: Use a better filter.
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(i.image, op)
}

func (i *Image) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	if i.image == nil {
		return image.Point{}
	}
	if fixedWidth, ok := constraints.FixedWidth(); ok {
		h := int(math.Ceil(float64(fixedWidth) * float64(i.image.Bounds().Dy()) / float64(i.image.Bounds().Dx())))
		return image.Pt(fixedWidth, h)
	}
	if fixedHeight, ok := constraints.FixedHeight(); ok {
		w := int(math.Ceil(float64(fixedHeight) * float64(i.image.Bounds().Dx()) / float64(i.image.Bounds().Dy())))
		return image.Pt(w, fixedHeight)
	}
	return i.image.Bounds().Size()
}

func (i *Image) HasImage() bool {
	return i.image != nil
}

func (i *Image) SetImage(image *ebiten.Image) {
	if i.image == image {
		return
	}
	i.image = image
	guigui.RequestRedraw(i)
}
