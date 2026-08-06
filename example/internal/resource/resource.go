// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package resource

import (
	"image/png"
	"io/fs"
	"path"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui/basicwidget"
)

type imageCacheKey struct {
	name       string
	colorMode  ebiten.ColorMode
	monochrome bool
}

// ImageLoader loads PNG images from a file system.
//
// An ImageLoader is safe for concurrent use.
type ImageLoader struct {
	fsys fs.FS

	m  map[imageCacheKey]*ebiten.Image
	mu sync.Mutex
}

// NewImageLoader returns a new ImageLoader reading images from fsys.
func NewImageLoader(fsys fs.FS) *ImageLoader {
	return &ImageLoader{
		fsys: fsys,
	}
}

// Image returns the image at "resource/<name>.png".
func (l *ImageLoader) Image(name string) (*ebiten.Image, error) {
	return l.image(imageCacheKey{
		name: name,
	})
}

// MonochromeImage returns the image at "resource/<name>.png", recolored for the given color mode.
func (l *ImageLoader) MonochromeImage(name string, colorMode ebiten.ColorMode) (*ebiten.Image, error) {
	return l.image(imageCacheKey{
		name:       name,
		colorMode:  colorMode,
		monochrome: true,
	})
}

func (l *ImageLoader) image(key imageCacheKey) (*ebiten.Image, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if img, ok := l.m[key]; ok {
		return img, nil
	}

	f, err := l.fsys.Open(path.Join("resource", key.name+".png"))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	src, err := png.Decode(f)
	if err != nil {
		return nil, err
	}

	if key.monochrome {
		src = basicwidget.CreateMonochromeImage(key.colorMode, src)
	}

	img := ebiten.NewImageFromImage(src)
	if l.m == nil {
		l.m = map[imageCacheKey]*ebiten.Image{}
	}
	l.m[key] = img
	return img, nil
}
