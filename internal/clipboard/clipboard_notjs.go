// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

//go:build !js

package clipboard

import (
	"github.com/atotto/clipboard"
)

func readAll() (string, error) {
	return clipboard.ReadAll()
}

func writeAll(text string) error {
	return clipboard.WriteAll(text)
}
