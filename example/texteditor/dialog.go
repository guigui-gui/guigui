// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"errors"

	"github.com/hajimehoshi/dialog"
)

// dialog.File() blocks the calling goroutine. To keep the UI responsive,
// callers spawn a goroutine and poll the returned channel from the UI tick.
// Yes/No confirms and info messages are handled in-app via Guigui Popups
// (see confirmdialog.go) rather than dialog.Message — the native message
// box has a noticeable display delay on macOS when invoked from inside the
// Ebiten loop.

type fileResult struct {
	path      string
	cancelled bool
	err       error
}

func openFileAsync() <-chan fileResult {
	ch := make(chan fileResult, 1)
	go func() {
		path, err := dialog.File().Title("Open").Load()
		ch <- toFileResult(path, err)
	}()
	return ch
}

func saveFileAsync(suggested string) <-chan fileResult {
	ch := make(chan fileResult, 1)
	go func() {
		b := dialog.File().Title("Save As")
		if suggested != "" {
			b = b.SetStartFile(suggested)
		}
		path, err := b.Save()
		ch <- toFileResult(path, err)
	}()
	return ch
}

func toFileResult(path string, err error) fileResult {
	if errors.Is(err, dialog.ErrCancelled) {
		return fileResult{cancelled: true}
	}
	return fileResult{path: path, err: err}
}
