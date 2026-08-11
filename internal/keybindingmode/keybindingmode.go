// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package keybindingmode

// Mode represents the convention that application keyboard shortcuts follow.
type Mode int

const (
	// Unknown means that the mode is not specified.
	Unknown Mode = iota

	// Command is the convention where Command carries application shortcuts.
	Command

	// ControlDefault is the convention where Control carries application shortcuts.
	ControlDefault

	// ControlEmacs is ControlDefault with the Emacs text editing keymap.
	ControlEmacs
)

// SystemMode returns the mode of the current system.
func SystemMode() Mode {
	return systemMode()
}
