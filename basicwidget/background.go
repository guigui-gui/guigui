// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image/color"

	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
)

type Background struct {
	guigui.DefaultWidget

	clr color.Color
}

func (b *Background) SetColor(clr color.Color) {
	if draw.EqualColor(b.clr, clr) {
		return
	}
	b.clr = clr
	guigui.RequestRedraw(b)
}

func (b *Background) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	clr := b.clr
	if clr == nil {
		clr = basicwidgetdraw.BackgroundColor(context.ColorMode())
	}
	dst.Fill(clr)
}
