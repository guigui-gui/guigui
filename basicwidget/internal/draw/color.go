// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package draw

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/iro"

	"github.com/guigui-gui/guigui"
)

func EqualColor(c0, c1 color.Color) bool {
	if c0 == c1 {
		return true
	}
	if c0 == nil || c1 == nil {
		return false
	}
	r0, g0, b0, a0 := c0.RGBA()
	r1, g1, b1, a1 := c1.RGBA()
	return r0 == r1 && g0 == g1 && b0 == b1 && a0 == a1
}

var (
	blue   = iro.ColorFromSRGB(0x00/255.0, 0x5a/255.0, 0xff/255.0, 1)
	green  = iro.ColorFromSRGB(0x03/255.0, 0xaf/255.0, 0x7a/255.0, 1)
	yellow = iro.ColorFromSRGB(0xff/255.0, 0xf1/255.0, 0x00/255.0, 1)
	red    = iro.ColorFromSRGB(0xff/255.0, 0x4b/255.0, 0x00/255.0, 1)
)

var (
	white = iro.ColorFromOKLch(1, 0, 0, 1)
	black = iro.ColorFromOKLch(0.2, 0, 0, 1)
	gray  = iro.ColorFromOKLch(0.6, 0, 0, 1)
)

type ColorType int

const (
	ColorTypeBase ColorType = iota
	ColorTypeAccent
	ColorTypeInfo
	ColorTypeSuccess
	ColorTypeWarning
	ColorTypeDanger
)

func Color(colorMode guigui.ColorMode, typ ColorType, lightnessInLightMode float64) color.Color {
	return Color2(colorMode, typ, lightnessInLightMode, 1-lightnessInLightMode)
}

func Color2(colorMode guigui.ColorMode, typ ColorType, lightnessInLightMode, lightnessInDarkMode float64) color.Color {
	var base iro.Color
	switch typ {
	case ColorTypeBase:
		base = gray
	case ColorTypeAccent:
		base = blue
	case ColorTypeInfo:
		base = blue
	case ColorTypeSuccess:
		base = green
	case ColorTypeWarning:
		base = yellow
	case ColorTypeDanger:
		base = red
	default:
		panic(fmt.Sprintf("draw: invalid color type: %d", typ))
	}
	switch colorMode {
	case guigui.ColorModeLight:
		return getColor(base, lightnessInLightMode, black, white)
	case guigui.ColorModeDark:
		return getColor(base, lightnessInDarkMode, black, white)
	default:
		panic(fmt.Sprintf("draw: invalid color mode: %d", colorMode))
	}
}

func getColor(base iro.Color, lightness float64, back, front iro.Color) color.Color {
	c0l, _, _, _ := back.OKLch()
	c1l, _, _, _ := front.OKLch()
	l, _, _, _ := base.OKLch()
	l = max(min(l, c1l), c0l)
	l2 := c0l*(1-lightness) + c1l*lightness
	if l2 < l {
		rate := (l2 - c0l) / (l - c0l)
		return MixColor(back, base, rate)
	}
	rate := (l2 - l) / (c1l - l)
	return MixColor(base, front, rate)
}

func MixColor(clr0, clr1 iro.Color, rate float64) color.Color {
	if rate == 0 {
		return clr0.SRGBColor()
	}
	if rate == 1 {
		return clr1.SRGBColor()
	}
	l0, a0, b0, alpha0 := clr0.OKLab()
	l1, a1, b1, alpha1 := clr1.OKLab()

	return iro.ColorFromOKLab(
		l0*(1-rate)+l1*rate,
		a0*(1-rate)+a1*rate,
		b0*(1-rate)+b1*rate,
		alpha0*(1-rate)+alpha1*rate,
	).SRGBColor()
}

func ScaleAlpha(clr color.Color, alpha float64) color.Color {
	r, g, b, a := clr.RGBA()
	r = uint32(float64(r) * alpha)
	g = uint32(float64(g) * alpha)
	b = uint32(float64(b) * alpha)
	a = uint32(float64(a) * alpha)
	return color.RGBA64{
		R: uint16(r),
		G: uint16(g),
		B: uint16(b),
		A: uint16(a),
	}
}

func BorderColors(colorMode guigui.ColorMode, borderType RoundedRectBorderType, accent bool) (color.Color, color.Color) {
	typ1 := ColorTypeBase
	typ2 := ColorTypeBase
	if accent {
		typ1 = ColorTypeAccent
	}
	switch borderType {
	case RoundedRectBorderTypeRegular:
		return Color2(colorMode, typ1, 0.8, 0.1), Color2(colorMode, typ2, 0.8, 0.1)
	case RoundedRectBorderTypeInset:
		return Color2(colorMode, typ1, 0.7, 0), Color2(colorMode, typ2, 0.85, 0.15)
	case RoundedRectBorderTypeOutset:
		return Color2(colorMode, typ1, 0.85, 0.5), Color2(colorMode, typ2, 0.7, 0.2)
	}
	panic(fmt.Sprintf("draw: invalid border type: %d", borderType))
}

var (
	textEnabledLightColor              = Color(guigui.ColorModeLight, ColorTypeBase, 0.1)
	textEnabledDarkColor               = Color(guigui.ColorModeDark, ColorTypeBase, 0.1)
	textDisabledLightColor             = Color(guigui.ColorModeLight, ColorTypeBase, 0.5)
	textDisabledDarkColor              = Color(guigui.ColorModeDark, ColorTypeBase, 0.5)
	controlEnabledLightColor           = Color2(guigui.ColorModeLight, ColorTypeBase, 1, 0.3)
	controlEnabledDarkColor            = Color2(guigui.ColorModeDark, ColorTypeBase, 1, 0.3)
	controlDisabledLightColor          = Color2(guigui.ColorModeLight, ColorTypeBase, 0.9, 0.1)
	controlDisabledDarkColor           = Color2(guigui.ColorModeDark, ColorTypeBase, 0.9, 0.1)
	secondaryControlEnabledLightColor  = Color2(guigui.ColorModeLight, ColorTypeBase, 0.95, 0.25)
	secondaryControlEnabledDarkColor   = Color2(guigui.ColorModeDark, ColorTypeBase, 0.95, 0.25)
	secondaryControlDisabledLightColor = Color2(guigui.ColorModeLight, ColorTypeBase, 0.85, 0.05)
	secondaryControlDisabledDarkColor  = Color2(guigui.ColorModeDark, ColorTypeBase, 0.85, 0.05)
	thumbEnabledLightColor             = Color2(guigui.ColorModeLight, ColorTypeBase, 1, 0.6)
	thumbEnabledDarkColor              = Color2(guigui.ColorModeDark, ColorTypeBase, 1, 0.6)
	thumbDisabledLightColor            = Color2(guigui.ColorModeLight, ColorTypeBase, 0.9, 0.55)
	thumbDisabledDarkColor             = Color2(guigui.ColorModeDark, ColorTypeBase, 0.9, 0.55)
)

func TextColor(colorMode guigui.ColorMode, enabled bool) color.Color {
	switch colorMode {
	case guigui.ColorModeLight:
		if enabled {
			return textEnabledLightColor
		}
		return textDisabledLightColor
	case guigui.ColorModeDark:
		if enabled {
			return textEnabledDarkColor
		}
		return textDisabledDarkColor
	default:
		panic(fmt.Sprintf("draw: invalid color mode: %d", colorMode))
	}
}

func ControlColor(colorMode guigui.ColorMode, enabled bool) color.Color {
	switch colorMode {
	case guigui.ColorModeLight:
		if enabled {
			return controlEnabledLightColor
		}
		return controlDisabledLightColor
	case guigui.ColorModeDark:
		if enabled {
			return controlEnabledDarkColor
		}
		return controlDisabledDarkColor
	default:
		panic(fmt.Sprintf("draw: invalid color mode: %d", colorMode))
	}
}

func SecondaryControlColor(colorMode guigui.ColorMode, enabled bool) color.Color {
	switch colorMode {
	case guigui.ColorModeLight:
		if enabled {
			return secondaryControlEnabledLightColor
		}
		return secondaryControlDisabledLightColor
	case guigui.ColorModeDark:
		if enabled {
			return secondaryControlEnabledDarkColor
		}
		return secondaryControlDisabledDarkColor
	default:
		panic(fmt.Sprintf("draw: invalid color mode: %d", colorMode))
	}
}

func ThumbColor(colorMode guigui.ColorMode, enabled bool) color.Color {
	switch colorMode {
	case guigui.ColorModeLight:
		if enabled {
			return thumbEnabledLightColor
		}
		return thumbDisabledLightColor
	case guigui.ColorModeDark:
		if enabled {
			return thumbEnabledDarkColor
		}
		return thumbDisabledDarkColor
	default:
		panic(fmt.Sprintf("draw: invalid color mode: %d", colorMode))
	}
}
