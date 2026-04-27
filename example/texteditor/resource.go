// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"embed"
	"image/png"
	"path"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui/basicwidget"
)

//go:embed resource/*.png
var imageResource embed.FS

type imageCacheKey struct {
	name      string
	colorMode ebiten.ColorMode
}

var (
	imageCache   = map[imageCacheKey]*ebiten.Image{}
	imageCacheMu sync.Mutex
)

// loadImage returns the named PNG (without extension) under resource/, recolored
// for the given color mode. Decoded images are cached after first use.
func loadImage(name string, colorMode ebiten.ColorMode) *ebiten.Image {
	imageCacheMu.Lock()
	defer imageCacheMu.Unlock()

	key := imageCacheKey{name: name, colorMode: colorMode}
	if img, ok := imageCache[key]; ok {
		return img
	}
	f, err := imageResource.Open(path.Join("resource", name+".png"))
	if err != nil {
		return nil
	}
	defer func() {
		_ = f.Close()
	}()

	src, err := png.Decode(f)
	if err != nil {
		return nil
	}
	img := ebiten.NewImageFromImage(basicwidget.CreateMonochromeImage(colorMode, src))
	imageCache[key] = img
	return img
}
