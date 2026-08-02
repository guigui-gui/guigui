// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package texteditor

import (
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
)

// CmdPressed reports whether the platform's primary command modifier is
// pressed: Command on macOS, Control elsewhere.
func CmdPressed() bool {
	if runtime.GOOS == "darwin" {
		return ebiten.IsKeyPressed(ebiten.KeyMeta)
	}
	return ebiten.IsKeyPressed(ebiten.KeyControl)
}

// Hotkey returns the platform-conventional display label of a shortcut with
// the primary command modifier.
func Hotkey(key string) string {
	if runtime.GOOS == "darwin" {
		return "⌘" + key
	}
	return "Ctrl+" + key
}

// HotkeyShift returns the platform-conventional display label of a shortcut
// with the Shift and primary command modifiers.
func HotkeyShift(key string) string {
	if runtime.GOOS == "darwin" {
		return "⇧⌘" + key
	}
	return "Ctrl+Shift+" + key
}
