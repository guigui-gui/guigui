// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image/color"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

// A tint is the color a widget derives the colors it draws from, adjusting
// the lightness for the color mode and for states such as hovering and
// pressing. Any color works as a tint; the functions below return the tints
// of the theme.

// AccentTintColor returns the tint marking the primary action of the theme.
func AccentTintColor() color.Color {
	return draw.AccentTintColor()
}

// InfoTintColor returns the tint marking neutral information.
func InfoTintColor() color.Color {
	return draw.InfoTintColor()
}

// SuccessTintColor returns the tint marking a successful result.
func SuccessTintColor() color.Color {
	return draw.SuccessTintColor()
}

// WarningTintColor returns the tint marking a condition that needs attention.
func WarningTintColor() color.Color {
	return draw.WarningTintColor()
}

// DangerTintColor returns the tint marking a destructive action or an error.
func DangerTintColor() color.Color {
	return draw.DangerTintColor()
}

// TextColorFromTint returns the color to draw text tinted with tint, for use
// with [TextStyle.SetColor]. A nil tint returns the color for enabled text.
func TextColorFromTint(context *guigui.Context, tint color.Color) color.Color {
	return draw.TextColorFromTint(context.ColorMode(), tint)
}
