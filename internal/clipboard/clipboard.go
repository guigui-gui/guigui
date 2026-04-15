// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package clipboard

import (
	"errors"
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
				if err := writeWithTimeout(text, 100*time.Millisecond); err != nil {
					slog.Error("failed to write clipboard", "error", err)
					continue
				}
			}
		}
	}()
}

func writeWithTimeout(text []byte, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if err := writeAll(text); err != nil {
			if time.Now().After(deadline) {
				return errors.New("clipboard: timeout")
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}
		return nil
	}
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

	select {
	case clipboardWriteCh <- v:
	case <-time.After(100 * time.Millisecond):
		return errors.New("clipboard: timeout")
	}

	cachedClipboardData.Store(v)
	return nil
}