// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"embed"
	"image/png"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui/basicwidget"
)

//go:embed resource/*.png
var pngImages embed.FS

type imageCacheKey struct {
	name       string
	colorMode  ebiten.ColorMode
	monochrome bool
}

type imageCache struct {
	m map[imageCacheKey]*ebiten.Image
}

var theImageCache = &imageCache{}

func (i *imageCache) GetMonochrome(name string, colorMode ebiten.ColorMode) (*ebiten.Image, error) {
	return i.get(imageCacheKey{
		name:       name,
		colorMode:  colorMode,
		monochrome: true,
	})
}

func (i *imageCache) Get(name string) (*ebiten.Image, error) {
	return i.get(imageCacheKey{
		name: name,
	})
}

func (i *imageCache) get(key imageCacheKey) (*ebiten.Image, error) {
	if img, ok := i.m[key]; ok {
		return img, nil
	}

	f, err := pngImages.Open("resource/" + key.name + ".png")
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

	if key.monochrome {
		pImg = basicwidget.CreateMonochromeImage(key.colorMode, pImg)
	}

	img := ebiten.NewImageFromImage(pImg)
	if i.m == nil {
		i.m = map[imageCacheKey]*ebiten.Image{}
	}
	i.m[key] = img
	return img, nil
}
