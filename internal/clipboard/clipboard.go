// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package clipboard

import (
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	cachedClipboardData           atomic.Value
	cachedClipboardDataExpireTime atomic.Int64
)

func ReadAll() (string, error) {
	if ebiten.Tick() <= cachedClipboardDataExpireTime.Load() {
		v, ok := cachedClipboardData.Load().(string)
		if !ok {
			return "", nil
		}
		return v, nil
	}
	// TODO: Read clipboard data asynchronously.
	data, err := readAll()
	if err != nil {
		return "", err
	}
	cachedClipboardData.Store(data)
	cachedClipboardDataExpireTime.Store(ebiten.Tick() + int64(ebiten.TPS()))
	return data, nil
}

func WriteAll(text string) error {
	// TODO: Write clipboard data asynchronously.
	if err := writeAll(text); err != nil {
		return err
	}
	cachedClipboardData.Store(text)
	cachedClipboardDataExpireTime.Store(ebiten.Tick() + int64(ebiten.TPS()))
	return nil
}
