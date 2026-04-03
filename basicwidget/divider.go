// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

type Divider struct {
	guigui.DefaultWidget
}

func (d *Divider) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	strokeWidth := float32(1 * context.Scale())
	clr := draw.Color(context.ColorMode(), draw.SemanticColorBase, 0.8)
	y := float32(bounds.Min.Y+bounds.Max.Y) / 2
	vector.StrokeLine(dst, float32(bounds.Min.X), y, float32(bounds.Max.X), y, strokeWidth, clr, false)
}

func (d *Divider) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	w, ok := constraints.FixedWidth()
	if !ok {
		w = d.DefaultWidget.Measure(context, constraints).X
	}
	return image.Pt(w, UnitSize(context))
}
