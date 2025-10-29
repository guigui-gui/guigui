// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package draw

import (
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
)

const blurShaderSourceTmpl = `//kage:unit pixels

package main

const blurSize = {{.BlurSize}}

var Rate float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	minPos := imageSrc0Origin()
	maxPos := minPos + imageSrc0Size()
	var sum vec4
	var factorSum float
	const sigma = float(2*blurSize)/6.0
	const expDenom = 2.0 * sigma * sigma
	for j := -blurSize; j <= blurSize; j++ {
		for i := -blurSize; i <= blurSize; i++ {
			pos := vec2(float(i), float(j)) + srcPos
			if minPos.x <= pos.x && pos.x <= maxPos.x && minPos.y <= pos.y && pos.y <= maxPos.y {
				factor := exp(-float(i*i+j*j) / expDenom)
				sum += imageSrc0UnsafeAt(pos) * factor
				factorSum += factor
			}
		}
	}
	return imageSrc0UnsafeAt(srcPos) * (1-Rate) + sum / factorSum * Rate
}
`

var blurShaders = map[int]*ebiten.Shader{}

func DrawBlurredImage(context *guigui.Context, dst *ebiten.Image, src *ebiten.Image, rate float64) {
	if rate == 0 {
		return
	}

	const baseBlurSize = 5
	blurSize := int(baseBlurSize * context.Scale())
	shader, ok := blurShaders[blurSize]
	if !ok {
		src := strings.Replace(blurShaderSourceTmpl, "{{.BlurSize}}", strconv.Itoa(blurSize), -1)
		s, err := ebiten.NewShader([]byte(src))
		if err != nil {
			panic(err)
		}
		shader = s
		blurShaders[blurSize] = shader
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = src
	op.GeoM.Translate(float64(src.Bounds().Min.X), float64(src.Bounds().Min.Y))
	op.Uniforms = map[string]any{
		"Rate": rate,
	}
	op.Blend = ebiten.BlendCopy
	dst.DrawRectShader(src.Bounds().Dx(), src.Bounds().Dy(), shader, op)
}
