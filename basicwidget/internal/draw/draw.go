// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package draw

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type RoundedRectBorderType int

const (
	RoundedRectBorderTypeRegular RoundedRectBorderType = iota
	RoundedRectBorderTypeInset
	RoundedRectBorderTypeOutset
)

var (
	theNinePatchVertices []ebiten.Vertex
	theNinePatchIndices  []uint32
)

func DrawNinePatch(dst *ebiten.Image, bounds image.Rectangle, src *ebiten.Image, clr1, clr2 color.Color) {
	DrawNinePatchParts(dst, bounds, src, clr1, clr2, [3][3]bool{
		{true, true, true},
		{true, true, true},
		{true, true, true},
	})
}

func DrawNinePatchParts(dst *ebiten.Image, bounds image.Rectangle, src *ebiten.Image, clr1, clr2 color.Color, renderingParts [3][3]bool) {
	if dst.Bounds().Intersect(bounds).Empty() {
		return
	}
	cornerW, cornerH := src.Bounds().Dx()/3, src.Bounds().Dy()/3
	theNinePatchVertices, theNinePatchIndices = AppendNinePatchVertices(theNinePatchVertices[:0], theNinePatchIndices[:0], bounds, src.Bounds(), cornerW, cornerH, clr1, clr2, renderingParts)
	op := &ebiten.DrawTrianglesOptions{}
	op.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha
	dst.DrawTriangles32(theNinePatchVertices, theNinePatchIndices, src, op)
}

func AppendNinePatchVertices(vertices []ebiten.Vertex, indices []uint32, dstBounds, srcBounds image.Rectangle, cornerW, cornerH int, clr1, clr2 color.Color, renderingParts [3][3]bool) ([]ebiten.Vertex, []uint32) {
	var c1 [4]uint32
	var c2 [4]uint32
	c1[0], c1[1], c1[2], c1[3] = clr1.RGBA()
	c2[0], c2[1], c2[2], c2[3] = clr2.RGBA()

	mix := func(a, b uint32, rate float32) float32 {
		return (1-rate)*float32(a)/0xffff + rate*float32(b)/0xffff
	}
	rates := [...]float32{
		0,
		float32(cornerH) / float32(dstBounds.Dy()),
		float32(dstBounds.Dy()-cornerH) / float32(dstBounds.Dy()),
		1,
	}
	var clrs [4][4]float32
	for j, rate := range rates {
		for i := range clrs[j] {
			clrs[j][i] = mix(c1[i], c2[i], rate)
		}
	}

	for j := range 3 {
		for i := range 3 {
			if !renderingParts[j][i] {
				continue
			}

			var scaleX float32 = 1.0
			var scaleY float32 = 1.0
			var tx, ty int

			switch i {
			case 0:
				tx = dstBounds.Min.X
			case 1:
				scaleX = float32(dstBounds.Dx()-2*cornerW) / float32(cornerW)
				tx = dstBounds.Min.X + cornerW
			case 2:
				tx = dstBounds.Max.X - cornerW
			}
			switch j {
			case 0:
				ty = dstBounds.Min.Y
			case 1:
				scaleY = float32(dstBounds.Dy()-2*cornerH) / float32(cornerH)
				ty = dstBounds.Min.Y + cornerH
			case 2:
				ty = dstBounds.Max.Y - cornerH
			}

			tx0 := float32(tx)
			tx1 := float32(tx) + scaleX*float32(cornerW)
			ty0 := float32(ty)
			ty1 := float32(ty) + scaleY*float32(cornerH)
			sx0 := float32(i * srcBounds.Dx() / 3)
			sy0 := float32(j * srcBounds.Dy() / 3)
			sx1 := float32(i+1) * float32(srcBounds.Dx()/3)
			sy1 := float32(j+1) * float32(srcBounds.Dy()/3)

			base := uint32(len(vertices))
			vertices = append(vertices,
				ebiten.Vertex{
					DstX:   tx0,
					DstY:   ty0,
					SrcX:   sx0,
					SrcY:   sy0,
					ColorR: clrs[j][0],
					ColorG: clrs[j][1],
					ColorB: clrs[j][2],
					ColorA: clrs[j][3],
				},
				ebiten.Vertex{
					DstX:   tx1,
					DstY:   ty0,
					SrcX:   sx1,
					SrcY:   sy0,
					ColorR: clrs[j][0],
					ColorG: clrs[j][1],
					ColorB: clrs[j][2],
					ColorA: clrs[j][3],
				},
				ebiten.Vertex{
					DstX:   tx0,
					DstY:   ty1,
					SrcX:   sx0,
					SrcY:   sy1,
					ColorR: clrs[j+1][0],
					ColorG: clrs[j+1][1],
					ColorB: clrs[j+1][2],
					ColorA: clrs[j+1][3],
				},
				ebiten.Vertex{
					DstX:   tx1,
					DstY:   ty1,
					SrcX:   sx1,
					SrcY:   sy1,
					ColorR: clrs[j+1][0],
					ColorG: clrs[j+1][1],
					ColorB: clrs[j+1][2],
					ColorA: clrs[j+1][3],
				},
			)
			indices = append(indices, base+0, base+1, base+2, base+1, base+2, base+3)
		}
	}

	return vertices, indices
}

const maskShaderSource = `//kage:unit pixels

package main

var DstOrigin vec2
var Bounds vec4
var Radius float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	c1 := Bounds.xy + vec2(Radius, Radius)
	c2 := Bounds.zy + vec2(-Radius, Radius)
	c3 := Bounds.xw + vec2(Radius, -Radius)
	c4 := Bounds.zw + vec2(-Radius, -Radius)
	dpos := dstPos.xy - imageDstOrigin() + DstOrigin
	if (dpos.x >= Bounds.x+Radius && dpos.x < Bounds.z-Radius) ||
		(dpos.y >= Bounds.y+Radius && dpos.y < Bounds.w-Radius) ||
		distance(c1, dpos) <= Radius ||
		distance(c2, dpos) <= Radius ||
		distance(c3, dpos) <= Radius ||
		distance(c4, dpos) <= Radius {
		discard()
	}
	return imageSrc0At(srcPos) * color
}
`

var maskShader *ebiten.Shader

func init() {
	s, err := ebiten.NewShader([]byte(maskShaderSource))
	if err != nil {
		panic(err)
	}
	maskShader = s
}

func adjustRadius(radius int, bounds image.Rectangle) int {
	return min(radius, bounds.Dx()/2, bounds.Dy()/2)
}

func DrawRoundedCorners(dst *ebiten.Image, src *ebiten.Image, bounds image.Rectangle, radius int, op *ebiten.DrawImageOptions) {
	radius = adjustRadius(radius, bounds)
	sOp := &ebiten.DrawRectShaderOptions{}
	if op != nil {
		sOp.GeoM = op.GeoM
		sOp.ColorScale = op.ColorScale
		sOp.CompositeMode = op.CompositeMode
		sOp.Blend = op.Blend
	}
	sOp.Images[0] = src
	sOp.Uniforms = map[string]any{
		"DstOrigin": []float32{
			float32(dst.Bounds().Min.X),
			float32(dst.Bounds().Min.Y),
		},
		"Bounds": []float32{
			float32(bounds.Min.X),
			float32(bounds.Min.Y),
			float32(bounds.Max.X),
			float32(bounds.Max.Y),
		},
		"Radius": float32(radius),
	}
	dst.DrawRectShader(src.Bounds().Dx(), src.Bounds().Dy(), maskShader, sOp)
}

func OverlapsWithRoundedCorner(bounds image.Rectangle, radius int, srcBounds image.Rectangle) bool {
	b1 := image.Rectangle{
		Min: bounds.Min,
		Max: bounds.Min.Add(image.Pt(radius, radius)),
	}
	b2 := image.Rectangle{
		Min: image.Pt(bounds.Max.X-radius, bounds.Min.Y),
		Max: image.Pt(bounds.Max.X, bounds.Min.Y+radius),
	}
	b3 := image.Rectangle{
		Min: image.Pt(bounds.Min.X, bounds.Max.Y-radius),
		Max: image.Pt(bounds.Min.X+radius, bounds.Max.Y),
	}
	b4 := image.Rectangle{
		Min: image.Pt(bounds.Max.X-radius, bounds.Max.Y-radius),
		Max: bounds.Max,
	}
	return srcBounds.Overlaps(b1) || srcBounds.Overlaps(b2) || srcBounds.Overlaps(b3) || srcBounds.Overlaps(b4)
}
