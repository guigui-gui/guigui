// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package draw

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
)

var (
	whiteRoundedShadowRects = map[int]*ebiten.Image{}
)

func ensureWhiteRoundedShadowRect(radius int) *ebiten.Image {
	if img, ok := whiteRoundedShadowRects[radius]; ok {
		return img
	}

	s := radius * 3
	img := ebiten.NewImage(s, s)

	pix := make([]byte, 4*s*s)

	easeInQuad := func(x float64) float64 {
		return x * x
	}

	for j := 0; j < radius; j++ {
		for i := 0; i < radius; i++ {
			x := float64(radius - i)
			y := float64(radius - j)
			d := max(0, min(1, math.Hypot(x, y)/float64(radius)))
			a := byte(0xff * easeInQuad(1-d))
			pix[4*(j*s+i)] = a
			pix[4*(j*s+i)+1] = a
			pix[4*(j*s+i)+2] = a
			pix[4*(j*s+i)+3] = a
		}
		for i := radius; i < 2*radius; i++ {
			d := max(0, min(1, float64(radius-j)/float64(radius)))
			a := byte(0xff * easeInQuad(1-d))
			pix[4*(j*s+i)] = a
			pix[4*(j*s+i)+1] = a
			pix[4*(j*s+i)+2] = a
			pix[4*(j*s+i)+3] = a
		}
		for i := 2 * radius; i < 3*radius; i++ {
			x := float64(i - 2*radius)
			y := float64(radius - j)
			d := max(0, min(1, math.Hypot(x, y)/float64(radius)))
			a := byte(0xff * easeInQuad(1-d))
			pix[4*(j*s+i)] = a
			pix[4*(j*s+i)+1] = a
			pix[4*(j*s+i)+2] = a
			pix[4*(j*s+i)+3] = a
		}
	}
	for j := radius; j < 2*radius; j++ {
		for i := 0; i < radius; i++ {
			d := max(0, min(1, float64(radius-i)/float64(radius)))
			a := byte(0xff * easeInQuad(1-d))
			pix[4*(j*s+i)] = a
			pix[4*(j*s+i)+1] = a
			pix[4*(j*s+i)+2] = a
			pix[4*(j*s+i)+3] = a
		}
		for i := radius; i < 2*radius; i++ {
			pix[4*(j*s+i)] = 0xff
			pix[4*(j*s+i)+1] = 0xff
			pix[4*(j*s+i)+2] = 0xff
			pix[4*(j*s+i)+3] = 0xff
		}
		for i := 2 * radius; i < 3*radius; i++ {
			d := max(0, min(1, float64(i-2*radius)/float64(radius)))
			a := byte(0xff * easeInQuad(1-d))
			pix[4*(j*s+i)] = a
			pix[4*(j*s+i)+1] = a
			pix[4*(j*s+i)+2] = a
			pix[4*(j*s+i)+3] = a
		}
	}
	for j := 2 * radius; j < 3*radius; j++ {
		for i := 0; i < radius; i++ {
			x := float64(radius - i)
			y := float64(j - 2*radius)
			d := max(0, min(1, math.Hypot(x, y)/float64(radius)))
			a := byte(0xff * easeInQuad(1-d))
			pix[4*(j*s+i)] = a
			pix[4*(j*s+i)+1] = a
			pix[4*(j*s+i)+2] = a
			pix[4*(j*s+i)+3] = a
		}
		for i := radius; i < 2*radius; i++ {
			d := max(0, min(1, float64(j-2*radius)/float64(radius)))
			a := byte(0xff * easeInQuad(1-d))
			pix[4*(j*s+i)] = a
			pix[4*(j*s+i)+1] = a
			pix[4*(j*s+i)+2] = a
			pix[4*(j*s+i)+3] = a
		}
		for i := 2 * radius; i < 3*radius; i++ {
			x := float64(i - 2*radius)
			y := float64(j - 2*radius)
			d := max(0, min(1, math.Hypot(x, y)/float64(radius)))
			a := byte(0xff * easeInQuad(1-d))
			pix[4*(j*s+i)] = a
			pix[4*(j*s+i)+1] = a
			pix[4*(j*s+i)+2] = a
			pix[4*(j*s+i)+3] = a
		}
	}

	img.WritePixels(pix)

	whiteRoundedShadowRects[radius] = img

	return img
}

func DrawRoundedShadowRect(context *guigui.Context, dst *ebiten.Image, bounds image.Rectangle, clr color.Color, radius int) {
	if !dst.Bounds().Overlaps(bounds) {
		return
	}
	radius = adjustRadius(radius, bounds)
	if radius == 0 {
		return
	}
	DrawNinePatch(dst, bounds, ensureWhiteRoundedShadowRect(radius), clr, clr)
}
