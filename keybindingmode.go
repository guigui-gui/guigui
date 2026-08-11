// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package guigui

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui/internal/keybindingmode"
)

// KeyBindingMode represents the convention that application keyboard shortcuts follow.
type KeyBindingMode int

const (
	// KeyBindingModeUnknown means that the mode is not specified.
	KeyBindingModeUnknown KeyBindingMode = KeyBindingMode(keybindingmode.Unknown)

	// KeyBindingModeCommand is the convention where Command carries application shortcuts,
	// used on macOS and iOS, including web browsers on them.
	KeyBindingModeCommand KeyBindingMode = KeyBindingMode(keybindingmode.Command)

	// KeyBindingModeControlDefault is the convention where Control carries application shortcuts,
	// used on Windows, Linux, and the other platforms.
	KeyBindingModeControlDefault KeyBindingMode = KeyBindingMode(keybindingmode.ControlDefault)

	// KeyBindingModeControlEmacs is [KeyBindingModeControlDefault] with the Emacs text editing keymap,
	// as enabled by the GTK Emacs key theme.
	KeyBindingModeControlEmacs KeyBindingMode = KeyBindingMode(keybindingmode.ControlEmacs)
)

// ShortcutModifierKey returns the modifier key that carries application shortcuts,
// either [ebiten.KeyMeta] or [ebiten.KeyControl].
//
// The returned value is not meaningful for [KeyBindingModeUnknown].
func (k KeyBindingMode) ShortcutModifierKey() ebiten.Key {
	if k == KeyBindingModeCommand {
		return ebiten.KeyMeta
	}
	return ebiten.KeyControl
}

// UsesEmacsKeymap reports whether the Emacs text editing keymap is in effect.
//
// The returned value is not meaningful for [KeyBindingModeUnknown].
func (k KeyBindingMode) UsesEmacsKeymap() bool {
	return k == KeyBindingModeCommand || k == KeyBindingModeControlEmacs
}

func systemKeyBindingMode() KeyBindingMode {
	if mode := KeyBindingMode(keybindingmode.SystemMode()); mode != KeyBindingModeUnknown {
		return mode
	}
	return KeyBindingModeControlDefault
}

var envKeyBindingMode KeyBindingMode

func init() {
	switch mode := os.Getenv("GUIGUI_KEY_BINDING_MODE"); mode {
	case "command":
		envKeyBindingMode = KeyBindingModeCommand
	case "control-default":
		envKeyBindingMode = KeyBindingModeControlDefault
	case "control-emacs":
		envKeyBindingMode = KeyBindingModeControlEmacs
	case "":
	default:
		slog.Warn(fmt.Sprintf("invalid GUIGUI_KEY_BINDING_MODE: %s", mode))
	}
}
