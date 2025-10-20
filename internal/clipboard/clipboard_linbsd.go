// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

//go:build linux || freebsd

package clipboard

import (
	"log/slog"

	"github.com/atotto/clipboard"
)

var (
	// it seems atotto/clipboard does not export the error, so we have to define it here.
	errMissingCommands = "No clipboard utilities available. Please install xsel, xclip, wl-clipboard or Termux:API add-on for termux-clipboard-get/set."
	fallbackClipboard  string
)

func ReadAll() (string, error) {
	readClip, err := clipboard.ReadAll()
	if err != nil {
		if err.Error() == errMissingCommands {
			slog.Error(err.Error())
			return fallbackClipboard, nil
		} else {
			return "", err
		}
	}
	return readClip, nil
}

func WriteAll(text string) error {
	err := clipboard.WriteAll(text)
	if err != nil {
		if err.Error() == errMissingCommands {
			slog.Error(err.Error())
			fallbackClipboard = text
		} else {
			return err
		}
	}
	return nil
}
