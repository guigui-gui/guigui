// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package texteditor

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
)

// ShortcutModifierPressed reports whether the modifier key that carries
// application shortcuts is pressed.
func ShortcutModifierPressed(context *guigui.Context) bool {
	return ebiten.IsKeyPressed(context.KeyBindingMode().ShortcutModifierKey())
}

// Hotkey returns the display label of a shortcut with the shortcut modifier.
func Hotkey(context *guigui.Context, key string) string {
	if context.KeyBindingMode() == guigui.KeyBindingModeCommand {
		return "⌘" + key
	}
	return "Ctrl+" + key
}

// HotkeyShift returns the display label of a shortcut with the Shift and the
// shortcut modifier.
func HotkeyShift(context *guigui.Context, key string) string {
	if context.KeyBindingMode() == guigui.KeyBindingModeCommand {
		return "⇧⌘" + key
	}
	return "Ctrl+Shift+" + key
}
