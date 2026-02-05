// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

//go:build !js

package clipboard

import (
	"golang.design/x/clipboard"
)

func readAll() ([]byte, error) {
	return clipboard.Read(clipboard.FmtText), nil
}

func writeAll(text []byte) error {
	clipboard.Write(clipboard.FmtText, text)
	return nil
}
