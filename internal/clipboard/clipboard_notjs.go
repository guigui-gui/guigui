// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

//go:build !js

package clipboard

import (
	"golang.design/x/clipboard"
)

func readAll() (string, error) {
	return string(clipboard.Read(clipboard.FmtText)), nil
}

func writeAll(text string) error {
	clipboard.Write(clipboard.FmtText, []byte(text))
	return nil
}
