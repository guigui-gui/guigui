// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

// Package clipboard provides access to the system clipboard.
//
// The clipboard holds one logical value that can have several format
// representations at the same time, such as plain text and HTML. [Read]
// returns all of them, and [Write] replaces all of them at once.
package clipboard

import (
	"bytes"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/guigui-gui/guigui/internal/debugmode"
)

// Contents is the clipboard contents: one logical value represented as up to
// one data payload per format. A nil payload means its format is absent.
type Contents struct {
	// Text is the UTF-8 plain text representation.
	Text []byte

	// HTML is the UTF-8 HTML markup representation.
	HTML []byte

	// PNG is the encoded PNG stream representation. An image on the system
	// clipboard in another encoding is not reported.
	PNG []byte
}

func (c Contents) clone() Contents {
	return Contents{
		Text: bytes.Clone(c.Text),
		HTML: bytes.Clone(c.HTML),
		PNG:  bytes.Clone(c.PNG),
	}
}

var (
	clipboardWriteCh        = make(chan Contents, 1)
	cachedClipboardContents atomic.Value
)

// readContents and writeContents are implemented per platform. They are
// called only from the goroutine started here, so their implementations can
// keep unsynchronized state.
func init() {
	if debugmode.EmulatesClipboard() {
		initEmulatedClipboard()
		return
	}

	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				readToCache()
			case contents := <-clipboardWriteCh:
				if err := writeContents(contents); err != nil {
					slog.Error("failed to write clipboard", "error", err)
					continue
				}
			}
		}
	}()
}

// initEmulatedClipboard replaces the system clipboard with an in-process one.
// cachedClipboardContents is then the clipboard itself: Write already stores
// the contents there, and there is nothing to poll.
func initEmulatedClipboard() {
	if text := debugmode.EmulatedClipboardText(); text != "" {
		cachedClipboardContents.Store(Contents{
			Text: []byte(text),
		})
	}

	// Drain the writes so that Write never times out on the channel.
	go func() {
		for range clipboardWriteCh {
		}
	}()
}

func readToCache() {
	contents, err := readContents()
	if err != nil {
		slog.Error("failed to read clipboard", "error", err)
		return
	}
	cachedClipboardContents.Store(contents)
}

// Read returns the current clipboard contents.
//
// Read does not block: it returns the contents observed last, which can lag
// the system clipboard slightly.
func Read() (Contents, error) {
	contents, _ := cachedClipboardContents.Load().(Contents)
	return contents.clone(), nil
}

// Write atomically replaces the entire clipboard contents with contents:
// every format is replaced at once. A nil payload leaves its format absent.
//
// Write does not wait for the system clipboard to be updated. A failure of
// the underlying system operation is logged instead of being returned.
func Write(contents Contents) error {
	contentsCopy := contents.clone()
	select {
	case clipboardWriteCh <- contentsCopy:
	case <-time.After(100 * time.Millisecond):
		return errors.New("clipboard: timeout")
	}
	cachedClipboardContents.Store(contentsCopy)
	return nil
}
