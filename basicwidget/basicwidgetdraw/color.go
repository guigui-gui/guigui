// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidgetdraw

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

func BorderColors(colorMode ebiten.ColorMode, borderType RoundedRectBorderType) (color.Color, color.Color) {
	typ1 := draw.ColorTypeBase
	typ2 := draw.ColorTypeBase
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

func BorderAccentColors(colorMode ebiten.ColorMode, borderType RoundedRectBorderType) (color.Color, color.Color) {
	typ1 := draw.ColorTypeAccent
	typ2 := draw.ColorTypeAccent
	switch borderType {
	case RoundedRectBorderTypeRegular:
		return draw.Color2(colorMode, typ1, 0.35, 0.35), draw.Color2(colorMode, typ2, 0.35, 0.35)
	case RoundedRectBorderTypeInset:
		return draw.Color2(colorMode, typ1, 0.325, 0.2), draw.Color2(colorMode, typ2, 0.35, 0.35)
	case RoundedRectBorderTypeOutset:
		return draw.Color2(colorMode, typ1, 0.6, 0.8), draw.Color2(colorMode, typ2, 0.35, 0.35)
	}
	panic(fmt.Sprintf("basicwidgetdraw: invalid border type: %d", borderType))
}

func BorderAccentSecondaryColors(colorMode ebiten.ColorMode, borderType RoundedRectBorderType) (color.Color, color.Color) {
	typ1 := draw.ColorTypeAccent
	typ2 := draw.ColorTypeAccent
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

var (
	textEnabledLightColor              = draw.Color(ebiten.ColorModeLight, draw.ColorTypeBase, 0.1)
	textEnabledDarkColor               = draw.Color(ebiten.ColorModeDark, draw.ColorTypeBase, 0.1)
	textDisabledLightColor             = draw.Color(ebiten.ColorModeLight, draw.ColorTypeBase, 0.5)
	textDisabledDarkColor              = draw.Color(ebiten.ColorModeDark, draw.ColorTypeBase, 0.5)
	controlEnabledLightColor           = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeBase, 1, 0.3)
	controlEnabledDarkColor            = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeBase, 1, 0.3)
	controlDisabledLightColor          = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeBase, 0.9, 0.1)
	controlDisabledDarkColor           = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeBase, 0.9, 0.1)
	controlSecondaryEnabledLightColor  = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeBase, 0.95, 0.25)
	controlSecondaryEnabledDarkColor   = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeBase, 0.95, 0.25)
	controlSecondaryDisabledLightColor = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeBase, 0.85, 0.05)
	controlSecondaryDisabledDarkColor  = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeBase, 0.85, 0.05)
	thumbEnabledLightColor             = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeBase, 1, 0.6)
	thumbEnabledDarkColor              = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeBase, 1, 0.6)
	thumbDisabledLightColor            = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeBase, 0.9, 0.55)
	thumbDisabledDarkColor             = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeBase, 0.9, 0.55)
	backgroundLightColor               = draw.Color(ebiten.ColorModeLight, draw.ColorTypeBase, 0.95)
	backgroundDarkColor                = draw.Color(ebiten.ColorModeDark, draw.ColorTypeBase, 0.95)
	backgroundSecondaryColorLightColor = draw.Color(ebiten.ColorModeLight, draw.ColorTypeBase, 0.9)
	backgroundSecondaryColorDarkColor  = draw.Color(ebiten.ColorModeDark, draw.ColorTypeBase, 0.9)
)

func TextColor(colorMode ebiten.ColorMode, enabled bool) color.Color {
	switch colorMode {
	case ebiten.ColorModeLight:
		if enabled {
			return textEnabledLightColor
		}
		return textDisabledLightColor
	case ebiten.ColorModeDark:
		if enabled {
			return textEnabledDarkColor
		}
		return textDisabledDarkColor
	default:
		panic(fmt.Sprintf("basicwidgetdraw: invalid color mode: %d", colorMode))
	}
}

func ControlColor(colorMode ebiten.ColorMode, enabled bool) color.Color {
	switch colorMode {
	case ebiten.ColorModeLight:
		if enabled {
			return controlEnabledLightColor
		}
		return controlDisabledLightColor
	case ebiten.ColorModeDark:
		if enabled {
			return controlEnabledDarkColor
		}
		return controlDisabledDarkColor
	default:
		panic(fmt.Sprintf("basicwidgetdraw: invalid color mode: %d", colorMode))
	}
}

func ControlSecondaryColor(colorMode ebiten.ColorMode, enabled bool) color.Color {
	switch colorMode {
	case ebiten.ColorModeLight:
		if enabled {
			return controlSecondaryEnabledLightColor
		}
		return controlSecondaryDisabledLightColor
	case ebiten.ColorModeDark:
		if enabled {
			return controlSecondaryEnabledDarkColor
		}
		return controlSecondaryDisabledDarkColor
	default:
		panic(fmt.Sprintf("basicwidgetdraw: invalid color mode: %d", colorMode))
	}
}

func ThumbColor(colorMode ebiten.ColorMode, enabled bool) color.Color {
	switch colorMode {
	case ebiten.ColorModeLight:
		if enabled {
			return thumbEnabledLightColor
		}
		return thumbDisabledLightColor
	case ebiten.ColorModeDark:
		if enabled {
			return thumbEnabledDarkColor
		}
		return thumbDisabledDarkColor
	default:
		panic(fmt.Sprintf("basicwidgetdraw: invalid color mode: %d", colorMode))
	}
}

func BackgroundColor(colorMode ebiten.ColorMode) color.Color {
	switch colorMode {
	case ebiten.ColorModeLight:
		return backgroundLightColor
	case ebiten.ColorModeDark:
		return backgroundDarkColor
	default:
		panic(fmt.Sprintf("basicwidgetdraw: invalid color mode: %d", colorMode))
	}
}

func BackgroundSecondaryColor(colorMode ebiten.ColorMode) color.Color {
	switch colorMode {
	case ebiten.ColorModeLight:
		return backgroundSecondaryColorLightColor
	case ebiten.ColorModeDark:
		return backgroundSecondaryColorDarkColor
	default:
		panic(fmt.Sprintf("basicwidgetdraw: invalid color mode: %d", colorMode))
	}
}
