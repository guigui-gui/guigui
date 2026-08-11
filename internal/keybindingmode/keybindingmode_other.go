// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

//go:build !darwin && !js && !(unix && !android)

package keybindingmode

func systemMode() Mode {
	return ControlDefault
}
