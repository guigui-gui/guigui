// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package draw

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// colorToken represents a themable color as a semantic hue and a lightness for each color mode.
type colorToken struct {
	// semanticColor is the hue the color is derived from.
	semanticColor SemanticColor

	// light is the lightness in the light color mode, in the range [0, 1].
	light float64

	// dark is the lightness in the dark color mode, in the range [0, 1].
	dark float64
}

func (c colorToken) color(colorMode ebiten.ColorMode) color.Color {
	return Color2(colorMode, c.semanticColor, c.light, c.dark)
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
		panic(fmt.Sprintf("draw: invalid border type: %d", borderType))
	}
}

var (
	borderTokens = borderColorTokens{
		regular: colorToken{
			semanticColor: SemanticColorBase,
			light:         0.8,
			dark:          0.1,
		},
		inset1: colorToken{
			semanticColor: SemanticColorBase,
			light:         0.7,
			dark:          0,
		},
		inset2: colorToken{
			semanticColor: SemanticColorBase,
			light:         0.85,
			dark:          0.15,
		},
		outset1: colorToken{
			semanticColor: SemanticColorBase,
			light:         0.85,
			dark:          0.5,
		},
		outset2: colorToken{
			semanticColor: SemanticColorBase,
			light:         0.7,
			dark:          0.2,
		},
	}
	borderAccentTokens = borderColorTokens{
		regular: colorToken{
			semanticColor: SemanticColorAccent,
			light:         0.35,
			dark:          0.35,
		},
		inset1: colorToken{
			semanticColor: SemanticColorAccent,
			light:         0.325,
			dark:          0.2,
		},
		inset2: colorToken{
			semanticColor: SemanticColorAccent,
			light:         0.35,
			dark:          0.35,
		},
		outset1: colorToken{
			semanticColor: SemanticColorAccent,
			light:         0.6,
			dark:          0.8,
		},
		outset2: colorToken{
			semanticColor: SemanticColorAccent,
			light:         0.35,
			dark:          0.35,
		},
	}
	borderAccentSecondaryTokens = borderColorTokens{
		regular: colorToken{
			semanticColor: SemanticColorAccent,
			light:         0.8,
			dark:          0.1,
		},
		inset1: colorToken{
			semanticColor: SemanticColorAccent,
			light:         0.7,
			dark:          0.2,
		},
		inset2: colorToken{
			semanticColor: SemanticColorAccent,
			light:         0.85,
			dark:          0.05,
		},
		outset1: colorToken{
			semanticColor: SemanticColorAccent,
			light:         0.85,
			dark:          0.05,
		},
		outset2: colorToken{
			semanticColor: SemanticColorAccent,
			light:         0.7,
			dark:          0.2,
		},
	}
	borderDangerToken = colorToken{
		semanticColor: SemanticColorDanger,
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
		semanticColor: SemanticColorBase,
		light:         0.1,
		dark:          0.9,
	}
	textDisabledToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.5,
		dark:          0.5,
	}
	textSelectionToken = colorToken{
		semanticColor: SemanticColorAccent,
		light:         0.8,
		dark:          0.35,
	}
	textActiveCompositionToken = colorToken{
		semanticColor: SemanticColorAccent,
		light:         0.4,
		dark:          0.6,
	}
	textInactiveCompositionToken = colorToken{
		semanticColor: SemanticColorAccent,
		light:         0.8,
		dark:          0.2,
	}
	controlEnabledToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         1,
		dark:          0.25,
	}
	controlDisabledToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.9,
		dark:          0.15,
	}
	contentBackgroundEnabledToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         1,
		dark:          0.15,
	}
	contentBackgroundDisabledToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.9,
		dark:          0.1,
	}
	menuBackgroundEnabledToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.95,
		dark:          0.3,
	}
	menuBackgroundDisabledToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.85,
		dark:          0.25,
	}
	controlSecondaryEnabledToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.95,
		dark:          0.3,
	}
	controlSecondaryDisabledToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.85,
		dark:          0.25,
	}
	buttonPressedToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.95,
		dark:          0.3,
	}
	buttonHoveredToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.975,
		dark:          0.275,
	}
	thumbEnabledToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         1,
		dark:          0.6,
	}
	thumbDisabledToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.9,
		dark:          0.55,
	}
	backgroundToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.95,
		dark:          0.05,
	}
	backgroundSecondaryToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.9,
		dark:          0.1,
	}
	popupBackgroundToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         1,
		dark:          0.05,
	}
	accentToken = colorToken{
		semanticColor: SemanticColorAccent,
		light:         0.5,
		dark:          0.5,
	}
	textOnAccentToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.9,
		dark:          0.9,
	}
	itemHighlightedTextToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         1,
		dark:          1,
	}
	itemHighlightedBackgroundToken = colorToken{
		semanticColor: SemanticColorAccent,
		light:         0.6,
		dark:          0.35,
	}
	itemSelectedUnfocusedBackgroundToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.85,
		dark:          0.35,
	}
	itemSelectedDisabledBackgroundToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.8,
		dark:          0.35,
	}
	itemHoveredBackgroundToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.9,
		dark:          0.35,
	}
	dividerToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.8,
		dark:          0.2,
	}
	focusBorderToken = colorToken{
		semanticColor: SemanticColorAccent,
		light:         0.8,
		dark:          0.2,
	}
	textCaretToken = colorToken{
		semanticColor: SemanticColorAccent,
		light:         0.5,
		dark:          0.6,
	}
	trackOffToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.8,
		dark:          0.2,
	}
	trackOffPressedToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.75,
		dark:          0.25,
	}
	thumbHoveredToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.975,
		dark:          0.575,
	}
	thumbPressedToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.95,
		dark:          0.55,
	}
	accentPressedToken = colorToken{
		semanticColor: SemanticColorAccent,
		light:         0.45,
		dark:          0.55,
	}
	accentHoveredToken = colorToken{
		semanticColor: SemanticColorAccent,
		light:         0.475,
		dark:          0.525,
	}
	activeSegmentedControlButtonToken = colorToken{
		semanticColor: SemanticColorAccent,
		light:         0.875,
		dark:          0.5,
	}
	activeSegmentedControlButtonHoveredToken = colorToken{
		semanticColor: SemanticColorAccent,
		light:         0.85,
		dark:          0.525,
	}
	headerSeparatorToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.9,
		dark:          0.4,
	}
	headerSeparatorDisabledToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.8,
		dark:          0.3,
	}
	sliderTickToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.7,
		dark:          0.3,
	}
	popupDarkBackgroundToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0.1,
		dark:          0,
	}
	shadowToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0,
		dark:          0,
	}
	foregroundToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         0,
		dark:          1,
	}
	radioButtonMarkToken = colorToken{
		semanticColor: SemanticColorBase,
		light:         1,
		dark:          1,
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
		semanticColor: semanticColor,
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

// ContentBackgroundColor returns the color for a surface that holds content, like a list or a text input.
func ContentBackgroundColor(colorMode ebiten.ColorMode, enabled bool) color.Color {
	if enabled {
		return contentBackgroundEnabledToken.color(colorMode)
	}
	return contentBackgroundDisabledToken.color(colorMode)
}

// MenuBackgroundColor returns the color for the surface of a menu.
func MenuBackgroundColor(colorMode ebiten.ColorMode, enabled bool) color.Color {
	if enabled {
		return menuBackgroundEnabledToken.color(colorMode)
	}
	return menuBackgroundDisabledToken.color(colorMode)
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
	if pressed {
		c := colorToken{
			semanticColor: semanticColor,
			light:         0.85,
			dark:          0.4,
		}
		return c.color(colorMode)
	}
	if hovered {
		c := colorToken{
			semanticColor: semanticColor,
			light:         0.875,
			dark:          0.375,
		}
		return c.color(colorMode)
	}
	c := colorToken{
		semanticColor: semanticColor,
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
		semanticColor: semanticColor,
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
		semanticColor: semanticColor,
		light:         0.95,
		dark:          0.15,
	}
	return c.color(colorMode)
}

func AccentColor(colorMode ebiten.ColorMode) color.Color {
	return accentToken.color(colorMode)
}

func TextOnAccentColor(colorMode ebiten.ColorMode) color.Color {
	return textOnAccentToken.color(colorMode)
}

func ItemHighlightedTextColor(colorMode ebiten.ColorMode) color.Color {
	return itemHighlightedTextToken.color(colorMode)
}

func ItemHighlightedBackgroundColor(colorMode ebiten.ColorMode) color.Color {
	return itemHighlightedBackgroundToken.color(colorMode)
}

func ItemSelectedUnfocusedBackgroundColor(colorMode ebiten.ColorMode) color.Color {
	return itemSelectedUnfocusedBackgroundToken.color(colorMode)
}

func ItemSelectedDisabledBackgroundColor(colorMode ebiten.ColorMode) color.Color {
	return itemSelectedDisabledBackgroundToken.color(colorMode)
}

func ItemHoveredBackgroundColor(colorMode ebiten.ColorMode) color.Color {
	return itemHoveredBackgroundToken.color(colorMode)
}

func ItemStripeBackgroundColor(colorMode ebiten.ColorMode) color.Color {
	return ScaleAlpha(foregroundToken.color(colorMode), 2.0/32)
}

func DividerColor(colorMode ebiten.ColorMode) color.Color {
	return dividerToken.color(colorMode)
}

func FocusBorderColor(colorMode ebiten.ColorMode) color.Color {
	return focusBorderToken.color(colorMode)
}

func TextCaretColor(colorMode ebiten.ColorMode) color.Color {
	return textCaretToken.color(colorMode)
}

func ScrollBarColor(colorMode ebiten.ColorMode) color.Color {
	return ScaleAlpha(foregroundToken.color(colorMode), 0.8)
}

func TrackColor(colorMode ebiten.ColorMode, on bool, pressed bool) color.Color {
	if on {
		if pressed {
			return accentPressedToken.color(colorMode)
		}
		return accentToken.color(colorMode)
	}
	if pressed {
		return trackOffPressedToken.color(colorMode)
	}
	return trackOffToken.color(colorMode)
}

func ThumbHoveredColor(colorMode ebiten.ColorMode) color.Color {
	return thumbHoveredToken.color(colorMode)
}

func ThumbPressedColor(colorMode ebiten.ColorMode) color.Color {
	return thumbPressedToken.color(colorMode)
}

func CheckableControlBackgroundColor(colorMode ebiten.ColorMode, checked bool, pressed bool, enabled bool) color.Color {
	if !enabled {
		return ControlColor(colorMode, false)
	}
	if checked {
		if pressed {
			return accentPressedToken.color(colorMode)
		}
		return accentToken.color(colorMode)
	}
	if pressed {
		return ControlSecondaryColor(colorMode, true)
	}
	return ControlColor(colorMode, true)
}

func RadioButtonMarkColor(colorMode ebiten.ColorMode, enabled bool) color.Color {
	if enabled {
		return radioButtonMarkToken.color(colorMode)
	}
	return ScaleAlpha(foregroundToken.color(colorMode), 0.5)
}

func PrimaryButtonBackgroundColor(colorMode ebiten.ColorMode, pressed bool, hovered bool) color.Color {
	if pressed {
		return accentPressedToken.color(colorMode)
	}
	if hovered {
		return accentHoveredToken.color(colorMode)
	}
	return accentToken.color(colorMode)
}

func ActiveSegmentedControlButtonBackgroundColor(colorMode ebiten.ColorMode, hovered bool) color.Color {
	if hovered {
		return activeSegmentedControlButtonHoveredToken.color(colorMode)
	}
	return activeSegmentedControlButtonToken.color(colorMode)
}

func HeaderSeparatorColor(colorMode ebiten.ColorMode, enabled bool) color.Color {
	if enabled {
		return headerSeparatorToken.color(colorMode)
	}
	return headerSeparatorDisabledToken.color(colorMode)
}

func SliderTickColor(colorMode ebiten.ColorMode) color.Color {
	return sliderTickToken.color(colorMode)
}

func PopupDarkBackgroundColor(colorMode ebiten.ColorMode, openingRate float64) color.Color {
	alpha := 0.25
	if colorMode == ebiten.ColorModeDark {
		alpha = 0.5
	}
	return ScaleAlpha(popupDarkBackgroundToken.color(colorMode), alpha*openingRate)
}

func ShadowColor(colorMode ebiten.ColorMode) color.Color {
	return shadowToken.color(colorMode)
}

func IconColor(colorMode ebiten.ColorMode) color.Color {
	return foregroundToken.color(colorMode)
}

func OverlayBackgroundColor(colorMode ebiten.ColorMode) color.Color {
	return ScaleAlpha(foregroundToken.color(colorMode), 1.0/32)
}

func OverlayBorderColor(colorMode ebiten.ColorMode) color.Color {
	return ScaleAlpha(foregroundToken.color(colorMode), 2.0/32)
}
