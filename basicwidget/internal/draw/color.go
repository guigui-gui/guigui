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

func getColor(base iro.Color, lightness float64, black, white iro.Color) color.Color {
	c0l, _, _, _ := black.OKLch()
	c1l, _, _, _ := white.OKLch()
	l, _, _, _ := base.OKLch()
	l = max(min(l, c1l), c0l)
	l2 := c0l*(1-lightness) + c1l*lightness
	if l2 < l {
		rate := (l2 - c0l) / (l - c0l)
		return mixColors(black, base, rate)
	}
	rate := (l2 - l) / (c1l - l)
	return mixColors(base, white, rate)
}

func MixColors(clr0, clr1 color.Color, rate float64) color.Color {
	return mixColors(iro.ColorFromSRGBColor(clr0), iro.ColorFromSRGBColor(clr1), rate)
}

func mixColors(clr0, clr1 iro.Color, rate float64) color.Color {
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
