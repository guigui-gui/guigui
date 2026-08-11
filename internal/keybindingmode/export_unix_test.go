// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

//go:build unix && !android && !darwin

package keybindingmode

type KeyThemeSource = keyThemeSource

const (
	KeyThemeSourceNone     = keyThemeSourceNone
	KeyThemeSourceGNOME    = keyThemeSourceGNOME
	KeyThemeSourceCinnamon = keyThemeSourceCinnamon
	KeyThemeSourceMATE     = keyThemeSourceMATE
	KeyThemeSourceXfce     = keyThemeSourceXfce
)

var (
	ModeForKeyTheme          = modeForKeyTheme
	KeyThemeSourceForDesktop = keyThemeSourceForDesktop
	UnquoteGSettingsString   = unquoteGSettingsString
	KeyThemeInGTKSettings    = keyThemeInGTKSettings
	GTKSettingsFileKeyTheme  = gtkSettingsFileKeyTheme
)
