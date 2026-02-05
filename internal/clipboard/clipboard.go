// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package clipboard

import (
	"log/slog"
	"sync/atomic"
	"time"
)

var (
	clipboardWriteCh    = make(chan string, 1)
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

// TODO: Use []byte?
func ReadAll() (string, error) {
	v, ok := cachedClipboardData.Load().(string)
	if !ok {
		return "", nil
	}
	return v, nil
}

// TODO: Use []byte?
func WriteAll(text string) error {
	clipboardWriteCh <- text
	cachedClipboardData.Store(text)
	return nil
}
