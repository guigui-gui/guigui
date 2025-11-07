// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package clipboard

import (
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	cachedClipboardData     atomic.Value
	cachedClipboardDataTime atomic.Int64
)

func ReadAll() (string, error) {
	if ebiten.Tick() <= cachedClipboardDataTime.Load() {
		v, ok := cachedClipboardData.Load().(string)
		if !ok {
			return "", nil
		}
		return v, nil
	}
	data, err := readAll()
	if err != nil {
		return "", err
	}
	cachedClipboardData.Store(data)
	cachedClipboardDataTime.Store(ebiten.Tick())
	return data, nil
}

func WriteAll(text string) error {
	if err := writeAll(text); err != nil {
		return err
	}
	cachedClipboardData.Store(text)
	cachedClipboardDataTime.Store(ebiten.Tick())
	return nil
}
