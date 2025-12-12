// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidgetdraw

import (
	"fmt"
	"image/color"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

func BorderColors(colorMode guigui.ColorMode, borderType RoundedRectBorderType, accent bool) (color.Color, color.Color) {
	typ1 := draw.ColorTypeBase
	typ2 := draw.ColorTypeBase
	if accent {
		typ1 = draw.ColorTypeAccent
	}
	switch borderType {
	case RoundedRectBorderTypeRegular:
		return draw.Color2(colorMode, typ1, 0.8, 0.1), draw.Color2(colorMode, typ2, 0.8, 0.1)
	case RoundedRectBorderTypeInset:
		return draw.Color2(colorMode, typ1, 0.7, 0), draw.Color2(colorMode, typ2, 0.85, 0.15)
	case RoundedRectBorderTypeOutset:
		return draw.Color2(colorMode, typ1, 0.85, 0.5), draw.Color2(colorMode, typ2, 0.7, 0.2)
	}
	panic(fmt.Sprintf("basicwidgetdraw: invalid border type: %d", borderType))
}
