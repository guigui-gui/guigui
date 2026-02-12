// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"embed"
	"image"
	"image/png"
	"path"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

//go:embed resource/*.png
var imageResource embed.FS

type imageCacheKey struct {
	name      string
	colorMode ebiten.ColorMode
}

type resourceImages struct {
	m map[imageCacheKey]*ebiten.Image
}

var theResourceImages = &resourceImages{}

func (i *resourceImages) Get(name string, colorMode ebiten.ColorMode) (*ebiten.Image, error) {
	key := imageCacheKey{
		name:      name,
		colorMode: colorMode,
	}
	if img, ok := i.m[key]; ok {
		return img, nil
	}

	f, err := imageResource.Open(path.Join("resource", name+".png"))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	pImg, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	pImg = CreateMonochromeImage(colorMode, pImg)
	img := ebiten.NewImageFromImage(pImg)
	if i.m == nil {
		i.m = map[imageCacheKey]*ebiten.Image{}
	}
	i.m[key] = img
	return img, nil
}

func CreateMonochromeImage(colorMode ebiten.ColorMode, img image.Image) image.Image {
	base := draw.Color(colorMode, draw.ColorTypeBase, 0)
	r, g, b, _ := base.RGBA()

	bounds := img.Bounds()
	pix := make([]byte, 4*bounds.Dx()*bounds.Dy())
	for j := range bounds.Dy() {
		for i := range bounds.Dx() {
			_, _, _, a := img.At(i, j).RGBA()
			if a == 0 {
				continue
			}

			pix[4*(j*bounds.Dx()+i)] = byte((r * a / 0xFFFF) >> 8)
			pix[4*(j*bounds.Dx()+i)+1] = byte((g * a / 0xFFFF) >> 8)
			pix[4*(j*bounds.Dx()+i)+2] = byte((b * a / 0xFFFF) >> 8)
			pix[4*(j*bounds.Dx()+i)+3] = uint8(a >> 8)
		}
	}

	return &image.RGBA{
		Pix:    pix,
		Stride: 4 * bounds.Dx(),
		Rect:   bounds,
	}
}
