// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

type Background struct {
	guigui.DefaultWidget

	semanticColor draw.SemanticColor
}

func (b *Background) WriteStateKey(context *guigui.Context, w *guigui.StateKeyWriter) {
	w.WriteUint64(uint64(b.semanticColor))
}

func (b *Background) SetSemanticColor(semanticColor basicwidgetdraw.SemanticColor) {
	b.semanticColor = draw.SemanticColor(semanticColor)
}

func (b *Background) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	dst.Fill(draw.BackgroundColorFromSemanticColor(context.ColorMode(), b.semanticColor))
}
