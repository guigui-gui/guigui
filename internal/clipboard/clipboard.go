// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package clipboard

import (
	"log/slog"
	"sync/atomic"
	"time"
)

var (
	clipboardWriteCh    = make(chan []byte, 1)
	cachedClipboardData atomic.Value
)

func init() {
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				readToCache()
			case text := <-clipboardWriteCh:
				if err := writeAll(text); err != nil {
					slog.Error("failed to write clipboard", "error", err)
					continue
				}
			}
		}
	}()
}

func readToCache() {
	data, err := readAll()
	if err != nil {
		slog.Error("failed to read clipboard", "error", err)
		return
	}
	cachedClipboardData.Store(data)
}

func ReadAll() ([]byte, error) {
	v, ok := cachedClipboardData.Load().([]byte)
	if !ok {
		return nil, nil
	}
	return v, nil
}

func WriteAll(bs []byte) error {
	v := make([]byte, len(bs))
	copy(v, bs)
	clipboardWriteCh <- v
	cachedClipboardData.Store(v)
	return nil
}
