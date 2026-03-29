// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidgetdraw

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

var (
	borderRegularLightColor1 = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeBase, 0.8, 0.1)
	borderRegularLightColor2 = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeBase, 0.8, 0.1)
	borderRegularDarkColor1  = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeBase, 0.8, 0.1)
	borderRegularDarkColor2  = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeBase, 0.8, 0.1)
	borderInsetLightColor1   = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeBase, 0.7, 0)
	borderInsetLightColor2   = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeBase, 0.85, 0.15)
	borderInsetDarkColor1    = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeBase, 0.7, 0)
	borderInsetDarkColor2    = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeBase, 0.85, 0.15)
	borderOutsetLightColor1  = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeBase, 0.85, 0.5)
	borderOutsetLightColor2  = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeBase, 0.7, 0.2)
	borderOutsetDarkColor1   = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeBase, 0.85, 0.5)
	borderOutsetDarkColor2   = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeBase, 0.7, 0.2)

	borderAccentRegularLightColor1 = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeAccent, 0.35, 0.35)
	borderAccentRegularLightColor2 = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeAccent, 0.35, 0.35)
	borderAccentRegularDarkColor1  = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeAccent, 0.35, 0.35)
	borderAccentRegularDarkColor2  = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeAccent, 0.35, 0.35)
	borderAccentInsetLightColor1   = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeAccent, 0.325, 0.2)
	borderAccentInsetLightColor2   = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeAccent, 0.35, 0.35)
	borderAccentInsetDarkColor1    = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeAccent, 0.325, 0.2)
	borderAccentInsetDarkColor2    = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeAccent, 0.35, 0.35)
	borderAccentOutsetLightColor1  = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeAccent, 0.6, 0.8)
	borderAccentOutsetLightColor2  = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeAccent, 0.35, 0.35)
	borderAccentOutsetDarkColor1   = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeAccent, 0.6, 0.8)
	borderAccentOutsetDarkColor2   = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeAccent, 0.35, 0.35)

	borderAccentSecondaryRegularLightColor1 = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeAccent, 0.8, 0.1)
	borderAccentSecondaryRegularLightColor2 = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeAccent, 0.8, 0.1)
	borderAccentSecondaryRegularDarkColor1  = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeAccent, 0.8, 0.1)
	borderAccentSecondaryRegularDarkColor2  = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeAccent, 0.8, 0.1)
	borderAccentSecondaryInsetLightColor1   = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeAccent, 0.7, 0.2)
	borderAccentSecondaryInsetLightColor2   = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeAccent, 0.85, 0.05)
	borderAccentSecondaryInsetDarkColor1    = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeAccent, 0.7, 0.2)
	borderAccentSecondaryInsetDarkColor2    = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeAccent, 0.85, 0.05)
	borderAccentSecondaryOutsetLightColor1  = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeAccent, 0.85, 0.05)
	borderAccentSecondaryOutsetLightColor2  = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeAccent, 0.7, 0.2)
	borderAccentSecondaryOutsetDarkColor1   = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeAccent, 0.85, 0.05)
	borderAccentSecondaryOutsetDarkColor2   = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeAccent, 0.7, 0.2)
)

func BorderColors(colorMode ebiten.ColorMode, borderType RoundedRectBorderType) (color.Color, color.Color) {
	switch colorMode {
	case ebiten.ColorModeLight:
		switch borderType {
		case RoundedRectBorderTypeRegular:
			return borderRegularLightColor1, borderRegularLightColor2
		case RoundedRectBorderTypeInset:
			return borderInsetLightColor1, borderInsetLightColor2
		case RoundedRectBorderTypeOutset:
			return borderOutsetLightColor1, borderOutsetLightColor2
		}
	case ebiten.ColorModeDark:
		switch borderType {
		case RoundedRectBorderTypeRegular:
			return borderRegularDarkColor1, borderRegularDarkColor2
		case RoundedRectBorderTypeInset:
			return borderInsetDarkColor1, borderInsetDarkColor2
		case RoundedRectBorderTypeOutset:
			return borderOutsetDarkColor1, borderOutsetDarkColor2
		}
	}
	panic(fmt.Sprintf("basicwidgetdraw: invalid color mode or border type: %d, %d", colorMode, borderType))
}

func BorderAccentColors(colorMode ebiten.ColorMode, borderType RoundedRectBorderType) (color.Color, color.Color) {
	switch colorMode {
	case ebiten.ColorModeLight:
		switch borderType {
		case RoundedRectBorderTypeRegular:
			return borderAccentRegularLightColor1, borderAccentRegularLightColor2
		case RoundedRectBorderTypeInset:
			return borderAccentInsetLightColor1, borderAccentInsetLightColor2
		case RoundedRectBorderTypeOutset:
			return borderAccentOutsetLightColor1, borderAccentOutsetLightColor2
		}
	case ebiten.ColorModeDark:
		switch borderType {
		case RoundedRectBorderTypeRegular:
			return borderAccentRegularDarkColor1, borderAccentRegularDarkColor2
		case RoundedRectBorderTypeInset:
			return borderAccentInsetDarkColor1, borderAccentInsetDarkColor2
		case RoundedRectBorderTypeOutset:
			return borderAccentOutsetDarkColor1, borderAccentOutsetDarkColor2
		}
	}
	panic(fmt.Sprintf("basicwidgetdraw: invalid color mode or border type: %d, %d", colorMode, borderType))
}

func BorderAccentSecondaryColors(colorMode ebiten.ColorMode, borderType RoundedRectBorderType) (color.Color, color.Color) {
	switch colorMode {
	case ebiten.ColorModeLight:
		switch borderType {
		case RoundedRectBorderTypeRegular:
			return borderAccentSecondaryRegularLightColor1, borderAccentSecondaryRegularLightColor2
		case RoundedRectBorderTypeInset:
			return borderAccentSecondaryInsetLightColor1, borderAccentSecondaryInsetLightColor2
		case RoundedRectBorderTypeOutset:
			return borderAccentSecondaryOutsetLightColor1, borderAccentSecondaryOutsetLightColor2
		}
	case ebiten.ColorModeDark:
		switch borderType {
		case RoundedRectBorderTypeRegular:
			return borderAccentSecondaryRegularDarkColor1, borderAccentSecondaryRegularDarkColor2
		case RoundedRectBorderTypeInset:
			return borderAccentSecondaryInsetDarkColor1, borderAccentSecondaryInsetDarkColor2
		case RoundedRectBorderTypeOutset:
			return borderAccentSecondaryOutsetDarkColor1, borderAccentSecondaryOutsetDarkColor2
		}
	}
	panic(fmt.Sprintf("basicwidgetdraw: invalid color mode or border type: %d, %d", colorMode, borderType))
}

var (
	textEnabledLightColor              = draw.Color(ebiten.ColorModeLight, draw.ColorTypeBase, 0.1)
	textEnabledDarkColor               = draw.Color(ebiten.ColorModeDark, draw.ColorTypeBase, 0.1)
	textDisabledLightColor             = draw.Color(ebiten.ColorModeLight, draw.ColorTypeBase, 0.5)
	textDisabledDarkColor              = draw.Color(ebiten.ColorModeDark, draw.ColorTypeBase, 0.5)
	textSelectionLightColor            = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeAccent, 0.8, 0.35)
	textSelectionDarkColor             = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeAccent, 0.8, 0.35)
	textActiveCompositionLightColor    = draw.Color(ebiten.ColorModeLight, draw.ColorTypeAccent, 0.4)
	textActiveCompositionDarkColor     = draw.Color(ebiten.ColorModeDark, draw.ColorTypeAccent, 0.4)
	textInactiveCompositionLightColor  = draw.Color(ebiten.ColorModeLight, draw.ColorTypeAccent, 0.8)
	textInactiveCompositionDarkColor   = draw.Color(ebiten.ColorModeDark, draw.ColorTypeAccent, 0.8)
	controlEnabledLightColor           = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeBase, 1, 0.2)
	controlEnabledDarkColor            = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeBase, 1, 0.2)
	controlDisabledLightColor          = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeBase, 0.9, 0.1)
	controlDisabledDarkColor           = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeBase, 0.9, 0.1)
	controlSecondaryEnabledLightColor  = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeBase, 0.95, 0.25)
	controlSecondaryEnabledDarkColor   = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeBase, 0.95, 0.25)
	controlSecondaryDisabledLightColor = draw.Color2(ebiten.ColorModeLight, draw.ColorTypeBase, 0.85, 0.15)
	controlSecondaryDisabledDarkColor  = draw.Color2(ebiten.ColorModeDark, draw.ColorTypeBase, 0.85, 0.15)
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

func TextSelectionColor(colorMode ebiten.ColorMode) color.Color {
	switch colorMode {
	case ebiten.ColorModeLight:
		return textSelectionLightColor
	case ebiten.ColorModeDark:
		return textSelectionDarkColor
	default:
		panic(fmt.Sprintf("basicwidgetdraw: invalid color mode: %d", colorMode))
	}
}

func TextActiveCompositionColor(colorMode ebiten.ColorMode) color.Color {
	switch colorMode {
	case ebiten.ColorModeLight:
		return textActiveCompositionLightColor
	case ebiten.ColorModeDark:
		return textActiveCompositionDarkColor
	default:
		panic(fmt.Sprintf("basicwidgetdraw: invalid color mode: %d", colorMode))
	}
}

func TextInactiveCompositionColor(colorMode ebiten.ColorMode) color.Color {
	switch colorMode {
	case ebiten.ColorModeLight:
		return textInactiveCompositionLightColor
	case ebiten.ColorModeDark:
		return textInactiveCompositionDarkColor
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
