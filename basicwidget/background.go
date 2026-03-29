// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
)

type Background struct {
	guigui.DefaultWidget

	semanticColor basicwidgetdraw.SemanticColor
}

func (b *Background) SetSemanticColor(semanticColor basicwidgetdraw.SemanticColor) {
	if b.semanticColor == semanticColor {
		return
	}
	b.semanticColor = semanticColor
	guigui.RequestRedraw(b)
}

func (b *Background) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	dst.Fill(basicwidgetdraw.BackgroundColorFromSemanticColor(context.ColorMode(), b.semanticColor))
}
