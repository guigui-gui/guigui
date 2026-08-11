// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

//go:build !darwin && !js

package keybindingmode

func systemMode() Mode {
	// TODO: On Linux, return ControlEmacs when the GTK Emacs key theme is in use.
	return ControlDefault
}
