// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidgetdraw

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

type SemanticColor int

const (
	SemanticColorBase    SemanticColor = SemanticColor(draw.SemanticColorBase)
	SemanticColorAccent  SemanticColor = SemanticColor(draw.SemanticColorAccent)
	SemanticColorInfo    SemanticColor = SemanticColor(draw.SemanticColorInfo)
	SemanticColorSuccess SemanticColor = SemanticColor(draw.SemanticColorSuccess)
	SemanticColorWarning SemanticColor = SemanticColor(draw.SemanticColorWarning)
	SemanticColorDanger  SemanticColor = SemanticColor(draw.SemanticColorDanger)
)

var (
	borderRegularLightColor1 = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorBase, 0.8, 0.1)
	borderRegularLightColor2 = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorBase, 0.8, 0.1)
	borderRegularDarkColor1  = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorBase, 0.8, 0.1)
	borderRegularDarkColor2  = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorBase, 0.8, 0.1)
	borderInsetLightColor1   = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorBase, 0.7, 0)
	borderInsetLightColor2   = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorBase, 0.85, 0.15)
	borderInsetDarkColor1    = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorBase, 0.7, 0)
	borderInsetDarkColor2    = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorBase, 0.85, 0.15)
	borderOutsetLightColor1  = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorBase, 0.85, 0.5)
	borderOutsetLightColor2  = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorBase, 0.7, 0.2)
	borderOutsetDarkColor1   = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorBase, 0.85, 0.5)
	borderOutsetDarkColor2   = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorBase, 0.7, 0.2)

	borderAccentRegularLightColor1 = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorAccent, 0.35, 0.35)
	borderAccentRegularLightColor2 = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorAccent, 0.35, 0.35)
	borderAccentRegularDarkColor1  = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorAccent, 0.35, 0.35)
	borderAccentRegularDarkColor2  = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorAccent, 0.35, 0.35)
	borderAccentInsetLightColor1   = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorAccent, 0.325, 0.2)
	borderAccentInsetLightColor2   = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorAccent, 0.35, 0.35)
	borderAccentInsetDarkColor1    = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorAccent, 0.325, 0.2)
	borderAccentInsetDarkColor2    = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorAccent, 0.35, 0.35)
	borderAccentOutsetLightColor1  = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorAccent, 0.6, 0.8)
	borderAccentOutsetLightColor2  = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorAccent, 0.35, 0.35)
	borderAccentOutsetDarkColor1   = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorAccent, 0.6, 0.8)
	borderAccentOutsetDarkColor2   = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorAccent, 0.35, 0.35)

	borderAccentSecondaryRegularLightColor1 = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorAccent, 0.8, 0.1)
	borderAccentSecondaryRegularLightColor2 = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorAccent, 0.8, 0.1)
	borderAccentSecondaryRegularDarkColor1  = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorAccent, 0.8, 0.1)
	borderAccentSecondaryRegularDarkColor2  = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorAccent, 0.8, 0.1)
	borderAccentSecondaryInsetLightColor1   = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorAccent, 0.7, 0.2)
	borderAccentSecondaryInsetLightColor2   = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorAccent, 0.85, 0.05)
	borderAccentSecondaryInsetDarkColor1    = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorAccent, 0.7, 0.2)
	borderAccentSecondaryInsetDarkColor2    = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorAccent, 0.85, 0.05)
	borderAccentSecondaryOutsetLightColor1  = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorAccent, 0.85, 0.05)
	borderAccentSecondaryOutsetLightColor2  = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorAccent, 0.7, 0.2)
	borderAccentSecondaryOutsetDarkColor1   = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorAccent, 0.85, 0.05)
	borderAccentSecondaryOutsetDarkColor2   = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorAccent, 0.7, 0.2)

	borderDangerLightColor1 = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorDanger, 0.4, 0.7)
	borderDangerLightColor2 = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorDanger, 0.4, 0.7)
	borderDangerDarkColor1  = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorDanger, 0.4, 0.7)
	borderDangerDarkColor2  = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorDanger, 0.4, 0.7)
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

func BorderDangerColors(colorMode ebiten.ColorMode) (color.Color, color.Color) {
	switch colorMode {
	case ebiten.ColorModeLight:
		return borderDangerLightColor1, borderDangerLightColor2
	case ebiten.ColorModeDark:
		return borderDangerDarkColor1, borderDangerDarkColor2
	}
	panic(fmt.Sprintf("basicwidgetdraw: invalid color mode: %d", colorMode))
}

var (
	textEnabledLightColor              = draw.Color(ebiten.ColorModeLight, draw.SemanticColorBase, 0.1)
	textEnabledDarkColor               = draw.Color(ebiten.ColorModeDark, draw.SemanticColorBase, 0.1)
	textDisabledLightColor             = draw.Color(ebiten.ColorModeLight, draw.SemanticColorBase, 0.5)
	textDisabledDarkColor              = draw.Color(ebiten.ColorModeDark, draw.SemanticColorBase, 0.5)
	textSelectionLightColor            = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorAccent, 0.8, 0.35)
	textSelectionDarkColor             = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorAccent, 0.8, 0.35)
	textActiveCompositionLightColor    = draw.Color(ebiten.ColorModeLight, draw.SemanticColorAccent, 0.4)
	textActiveCompositionDarkColor     = draw.Color(ebiten.ColorModeDark, draw.SemanticColorAccent, 0.4)
	textInactiveCompositionLightColor  = draw.Color(ebiten.ColorModeLight, draw.SemanticColorAccent, 0.8)
	textInactiveCompositionDarkColor   = draw.Color(ebiten.ColorModeDark, draw.SemanticColorAccent, 0.8)
	controlEnabledLightColor           = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorBase, 1, 0.25)
	controlEnabledDarkColor            = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorBase, 1, 0.25)
	controlDisabledLightColor          = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorBase, 0.9, 0.15)
	controlDisabledDarkColor           = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorBase, 0.9, 0.15)
	controlSecondaryEnabledLightColor  = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorBase, 0.95, 0.3)
	controlSecondaryEnabledDarkColor   = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorBase, 0.95, 0.3)
	controlSecondaryDisabledLightColor = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorBase, 0.85, 0.25)
	controlSecondaryDisabledDarkColor  = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorBase, 0.85, 0.25)
	thumbEnabledLightColor             = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorBase, 1, 0.6)
	thumbEnabledDarkColor              = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorBase, 1, 0.6)
	thumbDisabledLightColor            = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorBase, 0.9, 0.55)
	thumbDisabledDarkColor             = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorBase, 0.9, 0.55)
	backgroundLightColor               = draw.Color(ebiten.ColorModeLight, draw.SemanticColorBase, 0.95)
	backgroundDarkColor                = draw.Color(ebiten.ColorModeDark, draw.SemanticColorBase, 0.95)
	backgroundSecondaryColorLightColor = draw.Color(ebiten.ColorModeLight, draw.SemanticColorBase, 0.9)
	backgroundSecondaryColorDarkColor  = draw.Color(ebiten.ColorModeDark, draw.SemanticColorBase, 0.9)
	popupBackgroundLightColor          = draw.Color2(ebiten.ColorModeLight, draw.SemanticColorBase, 1, 0.05)
	popupBackgroundDarkColor           = draw.Color2(ebiten.ColorModeDark, draw.SemanticColorBase, 1, 0.05)
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

func TextColorFromSemanticColor(colorMode ebiten.ColorMode, semanticColor SemanticColor) color.Color {
	if semanticColor == SemanticColorBase {
		return TextColor(colorMode, true)
	}
	return draw.Color2(colorMode, draw.SemanticColor(semanticColor), 0.3, 0.8)
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

func ButtonBackgroundColorFromSemanticColor(colorMode ebiten.ColorMode, semanticColor SemanticColor, pressed bool, hovered bool) color.Color {
	if semanticColor == SemanticColorBase {
		if pressed {
			return draw.Color2(colorMode, draw.SemanticColorBase, 0.95, 0.3)
		}
		if hovered {
			return draw.Color2(colorMode, draw.SemanticColorBase, 0.975, 0.275)
		}
		return ControlColor(colorMode, true)
	}
	sc := draw.SemanticColor(semanticColor)
	if pressed {
		return draw.Color2(colorMode, sc, 0.85, 0.4)
	}
	if hovered {
		return draw.Color2(colorMode, sc, 0.875, 0.375)
	}
	return draw.Color2(colorMode, sc, 0.9, 0.35)
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

func PopupBackgroundColor(colorMode ebiten.ColorMode) color.Color {
	switch colorMode {
	case ebiten.ColorModeLight:
		return popupBackgroundLightColor
	case ebiten.ColorModeDark:
		return popupBackgroundDarkColor
	default:
		panic(fmt.Sprintf("basicwidgetdraw: invalid color mode: %d", colorMode))
	}
}

func PopupBackgroundColorFromSemanticColor(colorMode ebiten.ColorMode, semanticColor SemanticColor) color.Color {
	if semanticColor == SemanticColorBase {
		return PopupBackgroundColor(colorMode)
	}
	return draw.Color2(colorMode, draw.SemanticColor(semanticColor), 0.95, 0.1)
}

func BackgroundColorFromSemanticColor(colorMode ebiten.ColorMode, semanticColor SemanticColor) color.Color {
	if semanticColor == SemanticColorBase {
		return BackgroundColor(colorMode)
	}
	return draw.Color2(colorMode, draw.SemanticColor(semanticColor), 0.95, 0.15)
}
