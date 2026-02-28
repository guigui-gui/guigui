// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidgetdraw

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

type RoundedRectBorderType int

const (
	RoundedRectBorderTypeRegular RoundedRectBorderType = RoundedRectBorderType(draw.RoundedRectBorderTypeRegular)
	RoundedRectBorderTypeInset   RoundedRectBorderType = RoundedRectBorderType(draw.RoundedRectBorderTypeInset)
	RoundedRectBorderTypeOutset  RoundedRectBorderType = RoundedRectBorderType(draw.RoundedRectBorderTypeOutset)
)

var (
	whiteImage = ebiten.NewImage(3, 3)
)

func init() {
	b := whiteImage.Bounds()
	pix := make([]byte, 4*b.Dx()*b.Dy())
	for i := range pix {
		pix[i] = 0xff
	}
	// This is hacky, but WritePixels is better than Fill in term of automatic texture packing.
	whiteImage.WritePixels(pix)
}

var (
	theNinePatchVertices []ebiten.Vertex
	theNinePatchIndices  []uint32
)

func appendRectVectorPath(path *vector.Path, x0, y0, x1, y1 float32, radius float32) {
	path.MoveTo(x0, y0)
	path.LineTo(x1, y0)
	path.LineTo(x1, y1)
	path.LineTo(x0, y1)
	path.LineTo(x0, y0)
}

func appendRoundedRectVectorPath(path *vector.Path, rx0, ry0, rx1, ry1 float32, radius float32) {
	x0 := rx0
	x1 := rx0 + radius
	x2 := rx1 - radius
	x3 := rx1
	y0 := ry0
	y1 := ry0 + radius
	y2 := ry1 - radius
	y3 := ry1

	path.MoveTo(x1, y0)
	path.LineTo(x2, y0)
	path.ArcTo(x3, y0, x3, y1, radius)
	path.LineTo(x3, y2)
	path.ArcTo(x3, y3, x2, y3, radius)
	path.LineTo(x1, y3)
	path.ArcTo(x0, y3, x0, y2, radius)
	path.LineTo(x0, y1)
	path.ArcTo(x0, y0, x1, y0, radius)
}

type imageKey struct {
	radius      int
	borderWidth float32
	borderType  RoundedRectBorderType
	colorMode   ebiten.ColorMode
}

var (
	whiteRoundedRects       = map[imageKey]*ebiten.Image{}
	whiteRectBorders        = map[imageKey]*ebiten.Image{}
	whiteRoundedRectBorders = map[imageKey]*ebiten.Image{}
)

func ensureWhiteRoundedRect(radius int) *ebiten.Image {
	key := imageKey{
		radius: radius,
	}
	if img, ok := whiteRoundedRects[key]; ok {
		return img
	}

	s := radius * 3
	img := ebiten.NewImage(s, s)

	var path vector.Path
	appendRoundedRectVectorPath(&path, 0, 0, float32(s), float32(s), float32(radius))
	path.Close()

	drawPathOp := &vector.DrawPathOptions{}
	drawPathOp.AntiAlias = true
	vector.FillPath(img, &path, nil, drawPathOp)

	whiteRoundedRects[key] = img

	return img
}

func ensureWhiteRectBorder(partSize int, borderWidth float32, borderType RoundedRectBorderType, colorMode ebiten.ColorMode) *ebiten.Image {
	key := imageKey{
		radius:      partSize,
		borderWidth: borderWidth,
		borderType:  borderType,
		colorMode:   colorMode,
	}
	if img, ok := whiteRectBorders[key]; ok {
		return img
	}

	img := whiteRoundedRectBorder(partSize, borderWidth, borderType, colorMode, appendRectVectorPath)
	whiteRectBorders[key] = img
	return img
}

func ensureWhiteRoundedRectBorder(radius int, borderWidth float32, borderType RoundedRectBorderType, colorMode ebiten.ColorMode) *ebiten.Image {
	key := imageKey{
		radius:      radius,
		borderWidth: borderWidth,
		borderType:  borderType,
		colorMode:   colorMode,
	}
	if img, ok := whiteRoundedRectBorders[key]; ok {
		return img
	}

	img := whiteRoundedRectBorder(radius, borderWidth, borderType, colorMode, appendRoundedRectVectorPath)
	whiteRoundedRectBorders[key] = img
	return img
}

func whiteRoundedRectBorder(radius int, borderWidth float32, borderType RoundedRectBorderType, colorMode ebiten.ColorMode, appendPathFunc func(path *vector.Path, rx0, ry0, rx1, ry1 float32, radius float32)) *ebiten.Image {
	s := radius * 3
	inset := borderWidth

	var path vector.Path
	appendPathFunc(&path, 0, 0, float32(s), float32(s), float32(radius))
	switch borderType {
	case RoundedRectBorderTypeRegular:
		appendPathFunc(&path, inset, inset, float32(s)-inset, float32(s)-inset, float32(radius)-inset)
	case RoundedRectBorderTypeInset:
		// Use a thicker border for the dark mode, as colors tend to be contracting colors.
		if colorMode == ebiten.ColorModeDark {
			appendPathFunc(&path, inset, inset*2, float32(s)-inset, float32(s)-inset/2, float32(radius)-inset/2)
		} else {
			appendPathFunc(&path, inset, inset*3/2, float32(s)-inset, float32(s)-inset/2, float32(radius)-inset/2)
		}
	case RoundedRectBorderTypeOutset:
		// Use a thicker border for the dark mode, as colors tend to be contracting colors.
		if colorMode == ebiten.ColorModeDark {
			appendPathFunc(&path, inset, inset/2, float32(s)-inset, float32(s)-inset*2, float32(radius)-inset/2)
		} else {
			appendPathFunc(&path, inset, inset/2, float32(s)-inset, float32(s)-inset*3/2, float32(radius)-inset/2)
		}
	}
	path.Close()

	img := ebiten.NewImage(s, s)
	fillOp := &vector.FillOptions{}
	fillOp.FillRule = vector.FillRuleEvenOdd
	drawOp := &vector.DrawPathOptions{}
	drawOp.AntiAlias = true
	vector.FillPath(img, &path, fillOp, drawOp)

	return img
}

type Corners struct {
	TopStart    bool
	TopEnd      bool
	BottomStart bool
	BottomEnd   bool
}

func (s *Corners) bools() [3][3]bool {
	return [3][3]bool{
		{!s.TopStart, true, !s.TopEnd},
		{true, true, true},
		{!s.BottomStart, true, !s.BottomEnd},
	}
}

func (s *Corners) invertedBools() [3][3]bool {
	bs := s.bools()
	for j := range 3 {
		for i := range 3 {
			bs[j][i] = !bs[j][i]
		}
	}
	return bs
}

func DrawRoundedRect(context *guigui.Context, dst *ebiten.Image, bounds image.Rectangle, clr color.Color, radius int) {
	DrawRoundedRectWithSharpCorners(context, dst, bounds, clr, radius, Corners{})
}

func adjustRadius(radius int, bounds image.Rectangle) int {
	return min(radius, bounds.Dx()/2, bounds.Dy()/2)
}

func DrawRoundedRectWithSharpCorners(context *guigui.Context, dst *ebiten.Image, bounds image.Rectangle, clr color.Color, radius int, sharpCorners Corners) {
	if !dst.Bounds().Overlaps(bounds) {
		return
	}
	radius = adjustRadius(radius, bounds)
	if radius == 0 {
		return
	}

	if sharpCorners == (Corners{}) {
		draw.DrawNinePatch(dst, bounds, ensureWhiteRoundedRect(radius), clr, clr)
		return
	}

	draw.DrawNinePatchParts(dst, bounds, ensureWhiteRoundedRect(radius), clr, clr, sharpCorners.bools())
	if !dst.Bounds().Intersect(bounds).Empty() {
		theNinePatchVertices, theNinePatchIndices = draw.AppendNinePatchVertices(theNinePatchVertices[:0], theNinePatchIndices[:0], bounds, whiteImage.Bounds(), radius, radius, clr, clr, sharpCorners.invertedBools())
		op := &ebiten.DrawTrianglesOptions{}
		op.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha
		dst.DrawTriangles32(theNinePatchVertices, theNinePatchIndices, whiteImage, op)
	}
}

func DrawRoundedRectBorder(context *guigui.Context, dst *ebiten.Image, bounds image.Rectangle, clr1, clr2 color.Color, radius int, borderWidth float32, borderType RoundedRectBorderType) {
	DrawRoundedRectBorderWithSharpCorners(context, dst, bounds, clr1, clr2, radius, borderWidth, borderType, Corners{})
}

func DrawRoundedRectBorderWithSharpCorners(context *guigui.Context, dst *ebiten.Image, bounds image.Rectangle, clr1, clr2 color.Color, radius int, borderWidth float32, borderType RoundedRectBorderType, sharpCorners Corners) {
	if !dst.Bounds().Overlaps(bounds) {
		return
	}
	radius = adjustRadius(radius, bounds)
	if radius == 0 {
		return
	}

	if sharpCorners == (Corners{}) {
		draw.DrawNinePatch(dst, bounds, ensureWhiteRoundedRectBorder(radius, borderWidth, borderType, context.ResolvedColorMode()), clr1, clr2)
		return
	}

	draw.DrawNinePatchParts(dst, bounds, ensureWhiteRoundedRectBorder(radius, borderWidth, borderType, context.ResolvedColorMode()), clr1, clr2, sharpCorners.bools())
	draw.DrawNinePatchParts(dst, bounds, ensureWhiteRectBorder(radius, borderWidth, borderType, context.ResolvedColorMode()), clr1, clr2, sharpCorners.invertedBools())
}
