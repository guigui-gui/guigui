// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

type Background struct {
	guigui.DefaultWidget

	tint color.Color
}

func (b *Background) WriteStateKey(context *guigui.Context, w *guigui.StateKeyWriter) {
	writeColor(w, b.tint)
}

// SetTintColor sets the color the background derives its fill from.
// A nil tint restores the default fill.
func (b *Background) SetTintColor(tint color.Color) {
	b.tint = tint
}

func (b *Background) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	dst.Fill(draw.BackgroundColorFromTint(context.ColorMode(), b.tint))
}
