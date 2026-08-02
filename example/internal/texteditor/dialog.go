// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package texteditor

import (
	"errors"

	"github.com/hajimehoshi/dialog"
)

// dialog.File() blocks the calling goroutine. To keep the UI responsive,
// callers spawn a goroutine and poll the returned channel from the UI tick.
// The unsaved-changes confirmation is handled in-app via a Guigui Popup
// (see [ConfirmDialog]) rather than dialog.Message: the native message box
// has a noticeable display delay on macOS when invoked from inside the
// Ebiten loop.

// FileResult is the outcome of an asynchronous file dialog.
type FileResult struct {
	// Path is the selected file path when the dialog was confirmed.
	Path string

	// Cancelled reports whether the user dismissed the dialog.
	Cancelled bool

	// Err is the error the dialog failed with, if any.
	Err error
}

// FileFilter restricts the files an open dialog offers.
type FileFilter struct {
	// Description is the human-readable name of the filter.
	Description string

	// Extensions is the list of file extensions without the leading dot.
	Extensions []string
}

// OpenFileAsync shows a native open dialog on a new goroutine and delivers
// the result on the returned channel. A nil filter offers all files.
func OpenFileAsync(filter *FileFilter) <-chan FileResult {
	ch := make(chan FileResult, 1)
	go func() {
		b := dialog.File().Title("Open")
		if filter != nil {
			b = b.Filter(filter.Description, filter.Extensions...)
		}
		path, err := b.Load()
		ch <- toFileResult(path, err)
	}()
	return ch
}

// SaveFileAsync shows a native save dialog on a new goroutine and delivers
// the result on the returned channel. A non-empty suggested is the
// pre-filled file name.
func SaveFileAsync(suggested string) <-chan FileResult {
	ch := make(chan FileResult, 1)
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

func toFileResult(path string, err error) FileResult {
	if errors.Is(err, dialog.ErrCancelled) {
		return FileResult{Cancelled: true}
	}
	return FileResult{Path: path, Err: err}
}
