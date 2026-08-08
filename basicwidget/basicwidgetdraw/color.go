// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidgetdraw

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

func BorderColors(colorMode ebiten.ColorMode, borderType RoundedRectBorderType) (color.Color, color.Color) {
	return draw.BorderColors(colorMode, draw.RoundedRectBorderType(borderType))
}

func BorderAccentColors(colorMode ebiten.ColorMode, borderType RoundedRectBorderType) (color.Color, color.Color) {
	return draw.BorderAccentColors(colorMode, draw.RoundedRectBorderType(borderType))
}

func BorderAccentSecondaryColors(colorMode ebiten.ColorMode, borderType RoundedRectBorderType) (color.Color, color.Color) {
	return draw.BorderAccentSecondaryColors(colorMode, draw.RoundedRectBorderType(borderType))
}

func BorderDangerColors(colorMode ebiten.ColorMode) (color.Color, color.Color) {
	return draw.BorderDangerColors(colorMode)
}

func TextColor(colorMode ebiten.ColorMode, enabled bool) color.Color {
	return draw.TextColor(colorMode, enabled)
}

func TextColorFromTint(colorMode ebiten.ColorMode, tint color.Color) color.Color {
	return draw.TextColorFromTint(colorMode, tint)
}

func TextSelectionColor(colorMode ebiten.ColorMode) color.Color {
	return draw.TextSelectionColor(colorMode)
}

func TextActiveCompositionColor(colorMode ebiten.ColorMode) color.Color {
	return draw.TextActiveCompositionColor(colorMode)
}

func TextInactiveCompositionColor(colorMode ebiten.ColorMode) color.Color {
	return draw.TextInactiveCompositionColor(colorMode)
}

func ControlColor(colorMode ebiten.ColorMode, enabled bool) color.Color {
	return draw.ControlColor(colorMode, enabled)
}

func ControlSecondaryColor(colorMode ebiten.ColorMode, enabled bool) color.Color {
	return draw.ControlSecondaryColor(colorMode, enabled)
}

func ThumbColor(colorMode ebiten.ColorMode, enabled bool) color.Color {
	return draw.ThumbColor(colorMode, enabled)
}

func BackgroundColor(colorMode ebiten.ColorMode) color.Color {
	return draw.BackgroundColor(colorMode)
}

func BackgroundSecondaryColor(colorMode ebiten.ColorMode) color.Color {
	return draw.BackgroundSecondaryColor(colorMode)
}

func PopupBackgroundColor(colorMode ebiten.ColorMode) color.Color {
	return draw.PopupBackgroundColor(colorMode)
}
