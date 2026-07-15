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

// colorToken represents a themable color as a semantic hue and a lightness for each color mode.
type colorToken struct {
	// semanticColor is the hue the color is derived from.
	semanticColor draw.SemanticColor

	// light is the lightness in the light color mode, in the range [0, 1].
	light float64

	// dark is the lightness in the dark color mode, in the range [0, 1].
	dark float64
}

func (c colorToken) color(colorMode ebiten.ColorMode) color.Color {
	return draw.Color2(colorMode, c.semanticColor, c.light, c.dark)
}

// borderColorTokens is a set of color tokens for a rounded-rect border, one pair per border type.
type borderColorTokens struct {
	// regular is the color for both edges of a regular border.
	regular colorToken

	// inset1 is the upper-edge color of an inset border.
	inset1 colorToken

	// inset2 is the lower-edge color of an inset border.
	inset2 colorToken

	// outset1 is the upper-edge color of an outset border.
	outset1 colorToken

	// outset2 is the lower-edge color of an outset border.
	outset2 colorToken
}

func (b borderColorTokens) colors(colorMode ebiten.ColorMode, borderType RoundedRectBorderType) (color.Color, color.Color) {
	switch borderType {
	case RoundedRectBorderTypeRegular:
		c := b.regular.color(colorMode)
		return c, c
	case RoundedRectBorderTypeInset:
		return b.inset1.color(colorMode), b.inset2.color(colorMode)
	case RoundedRectBorderTypeOutset:
		return b.outset1.color(colorMode), b.outset2.color(colorMode)
	default:
		panic(fmt.Sprintf("basicwidgetdraw: invalid border type: %d", borderType))
	}
}

var (
	borderTokens = borderColorTokens{
		regular: colorToken{
			semanticColor: draw.SemanticColorBase,
			light:         0.8,
			dark:          0.1,
		},
		inset1: colorToken{
			semanticColor: draw.SemanticColorBase,
			light:         0.7,
			dark:          0,
		},
		inset2: colorToken{
			semanticColor: draw.SemanticColorBase,
			light:         0.85,
			dark:          0.15,
		},
		outset1: colorToken{
			semanticColor: draw.SemanticColorBase,
			light:         0.85,
			dark:          0.5,
		},
		outset2: colorToken{
			semanticColor: draw.SemanticColorBase,
			light:         0.7,
			dark:          0.2,
		},
	}
	borderAccentTokens = borderColorTokens{
		regular: colorToken{
			semanticColor: draw.SemanticColorAccent,
			light:         0.35,
			dark:          0.35,
		},
		inset1: colorToken{
			semanticColor: draw.SemanticColorAccent,
			light:         0.325,
			dark:          0.2,
		},
		inset2: colorToken{
			semanticColor: draw.SemanticColorAccent,
			light:         0.35,
			dark:          0.35,
		},
		outset1: colorToken{
			semanticColor: draw.SemanticColorAccent,
			light:         0.6,
			dark:          0.8,
		},
		outset2: colorToken{
			semanticColor: draw.SemanticColorAccent,
			light:         0.35,
			dark:          0.35,
		},
	}
	borderAccentSecondaryTokens = borderColorTokens{
		regular: colorToken{
			semanticColor: draw.SemanticColorAccent,
			light:         0.8,
			dark:          0.1,
		},
		inset1: colorToken{
			semanticColor: draw.SemanticColorAccent,
			light:         0.7,
			dark:          0.2,
		},
		inset2: colorToken{
			semanticColor: draw.SemanticColorAccent,
			light:         0.85,
			dark:          0.05,
		},
		outset1: colorToken{
			semanticColor: draw.SemanticColorAccent,
			light:         0.85,
			dark:          0.05,
		},
		outset2: colorToken{
			semanticColor: draw.SemanticColorAccent,
			light:         0.7,
			dark:          0.2,
		},
	}
	borderDangerToken = colorToken{
		semanticColor: draw.SemanticColorDanger,
		light:         0.4,
		dark:          0.7,
	}
)

func BorderColors(colorMode ebiten.ColorMode, borderType RoundedRectBorderType) (color.Color, color.Color) {
	return borderTokens.colors(colorMode, borderType)
}

func BorderAccentColors(colorMode ebiten.ColorMode, borderType RoundedRectBorderType) (color.Color, color.Color) {
	return borderAccentTokens.colors(colorMode, borderType)
}

func BorderAccentSecondaryColors(colorMode ebiten.ColorMode, borderType RoundedRectBorderType) (color.Color, color.Color) {
	return borderAccentSecondaryTokens.colors(colorMode, borderType)
}

func BorderDangerColors(colorMode ebiten.ColorMode) (color.Color, color.Color) {
	c := borderDangerToken.color(colorMode)
	return c, c
}

var (
	textEnabledToken = colorToken{
		semanticColor: draw.SemanticColorBase,
		light:         0.1,
		dark:          0.9,
	}
	textDisabledToken = colorToken{
		semanticColor: draw.SemanticColorBase,
		light:         0.5,
		dark:          0.5,
	}
	textSelectionToken = colorToken{
		semanticColor: draw.SemanticColorAccent,
		light:         0.8,
		dark:          0.35,
	}
	textActiveCompositionToken = colorToken{
		semanticColor: draw.SemanticColorAccent,
		light:         0.4,
		dark:          0.6,
	}
	textInactiveCompositionToken = colorToken{
		semanticColor: draw.SemanticColorAccent,
		light:         0.8,
		dark:          0.2,
	}
	controlEnabledToken = colorToken{
		semanticColor: draw.SemanticColorBase,
		light:         1,
		dark:          0.25,
	}
	controlDisabledToken = colorToken{
		semanticColor: draw.SemanticColorBase,
		light:         0.9,
		dark:          0.15,
	}
	controlSecondaryEnabledToken = colorToken{
		semanticColor: draw.SemanticColorBase,
		light:         0.95,
		dark:          0.3,
	}
	controlSecondaryDisabledToken = colorToken{
		semanticColor: draw.SemanticColorBase,
		light:         0.85,
		dark:          0.25,
	}
	buttonPressedToken = colorToken{
		semanticColor: draw.SemanticColorBase,
		light:         0.95,
		dark:          0.3,
	}
	buttonHoveredToken = colorToken{
		semanticColor: draw.SemanticColorBase,
		light:         0.975,
		dark:          0.275,
	}
	thumbEnabledToken = colorToken{
		semanticColor: draw.SemanticColorBase,
		light:         1,
		dark:          0.6,
	}
	thumbDisabledToken = colorToken{
		semanticColor: draw.SemanticColorBase,
		light:         0.9,
		dark:          0.55,
	}
	backgroundToken = colorToken{
		semanticColor: draw.SemanticColorBase,
		light:         0.95,
		dark:          0.05,
	}
	backgroundSecondaryToken = colorToken{
		semanticColor: draw.SemanticColorBase,
		light:         0.9,
		dark:          0.1,
	}
	popupBackgroundToken = colorToken{
		semanticColor: draw.SemanticColorBase,
		light:         1,
		dark:          0.05,
	}
)

func TextColor(colorMode ebiten.ColorMode, enabled bool) color.Color {
	if enabled {
		return textEnabledToken.color(colorMode)
	}
	return textDisabledToken.color(colorMode)
}

func TextColorFromSemanticColor(colorMode ebiten.ColorMode, semanticColor SemanticColor) color.Color {
	if semanticColor == SemanticColorBase {
		return TextColor(colorMode, true)
	}
	c := colorToken{
		semanticColor: draw.SemanticColor(semanticColor),
		light:         0.3,
		dark:          0.8,
	}
	return c.color(colorMode)
}

func TextSelectionColor(colorMode ebiten.ColorMode) color.Color {
	return textSelectionToken.color(colorMode)
}

func TextActiveCompositionColor(colorMode ebiten.ColorMode) color.Color {
	return textActiveCompositionToken.color(colorMode)
}

func TextInactiveCompositionColor(colorMode ebiten.ColorMode) color.Color {
	return textInactiveCompositionToken.color(colorMode)
}

func ControlColor(colorMode ebiten.ColorMode, enabled bool) color.Color {
	if enabled {
		return controlEnabledToken.color(colorMode)
	}
	return controlDisabledToken.color(colorMode)
}

func ControlSecondaryColor(colorMode ebiten.ColorMode, enabled bool) color.Color {
	if enabled {
		return controlSecondaryEnabledToken.color(colorMode)
	}
	return controlSecondaryDisabledToken.color(colorMode)
}

func ButtonBackgroundColorFromSemanticColor(colorMode ebiten.ColorMode, semanticColor SemanticColor, pressed bool, hovered bool) color.Color {
	if semanticColor == SemanticColorBase {
		if pressed {
			return buttonPressedToken.color(colorMode)
		}
		if hovered {
			return buttonHoveredToken.color(colorMode)
		}
		return ControlColor(colorMode, true)
	}
	sc := draw.SemanticColor(semanticColor)
	if pressed {
		c := colorToken{
			semanticColor: sc,
			light:         0.85,
			dark:          0.4,
		}
		return c.color(colorMode)
	}
	if hovered {
		c := colorToken{
			semanticColor: sc,
			light:         0.875,
			dark:          0.375,
		}
		return c.color(colorMode)
	}
	c := colorToken{
		semanticColor: sc,
		light:         0.9,
		dark:          0.35,
	}
	return c.color(colorMode)
}

func ThumbColor(colorMode ebiten.ColorMode, enabled bool) color.Color {
	if enabled {
		return thumbEnabledToken.color(colorMode)
	}
	return thumbDisabledToken.color(colorMode)
}

func BackgroundColor(colorMode ebiten.ColorMode) color.Color {
	return backgroundToken.color(colorMode)
}

func BackgroundSecondaryColor(colorMode ebiten.ColorMode) color.Color {
	return backgroundSecondaryToken.color(colorMode)
}

func PopupBackgroundColor(colorMode ebiten.ColorMode) color.Color {
	return popupBackgroundToken.color(colorMode)
}

func PopupBackgroundColorFromSemanticColor(colorMode ebiten.ColorMode, semanticColor SemanticColor) color.Color {
	if semanticColor == SemanticColorBase {
		return PopupBackgroundColor(colorMode)
	}
	c := colorToken{
		semanticColor: draw.SemanticColor(semanticColor),
		light:         0.95,
		dark:          0.1,
	}
	return c.color(colorMode)
}

func BackgroundColorFromSemanticColor(colorMode ebiten.ColorMode, semanticColor SemanticColor) color.Color {
	if semanticColor == SemanticColorBase {
		return BackgroundColor(colorMode)
	}
	c := colorToken{
		semanticColor: draw.SemanticColor(semanticColor),
		light:         0.95,
		dark:          0.15,
	}
	return c.color(colorMode)
}
