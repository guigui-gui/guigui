// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

//go:build unix && !android && !darwin

package keybindingmode

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// emacsKeyTheme is the name of the GTK key theme that selects the Emacs text editing keymap.
const emacsKeyTheme = "Emacs"

// cachedSystemMode is the system mode, computed at the first use. The value never expires: a key
// theme change is reflected only at the next run.
var cachedSystemMode = sync.OnceValue(func() Mode {
	return modeForKeyTheme(gtkKeyTheme())
})

func systemMode() Mode {
	return cachedSystemMode()
}

// modeForKeyTheme returns the mode for a GTK key theme name, matched case sensitively.
//
// Every name other than the Emacs key theme, including an empty one, maps to ControlDefault.
func modeForKeyTheme(theme string) Mode {
	if theme == emacsKeyTheme {
		return ControlEmacs
	}
	return ControlDefault
}

// keyThemeSource identifies where the GTK settings of a desktop are read from.
type keyThemeSource int

const (
	keyThemeSourceNone keyThemeSource = iota
	keyThemeSourceGNOME
	keyThemeSourceCinnamon
	keyThemeSourceMATE
	keyThemeSourceXfce
)

// keyThemeSources maps a desktop name as it appears in XDG_CURRENT_DESKTOP, lowercased, to the
// source of its GTK settings.
var keyThemeSources = map[string]keyThemeSource{
	"gnome":      keyThemeSourceGNOME,
	"unity":      keyThemeSourceGNOME,
	"pantheon":   keyThemeSourceGNOME,
	"cinnamon":   keyThemeSourceCinnamon,
	"x-cinnamon": keyThemeSourceCinnamon,
	"mate":       keyThemeSourceMATE,
	"xfce":       keyThemeSourceXfce,
}

// keyThemeSourceForDesktop returns the source for the first desktop named in an XDG_CURRENT_DESKTOP
// value that has one, or keyThemeSourceNone.
func keyThemeSourceForDesktop(currentDesktop string) keyThemeSource {
	for desktop := range strings.SplitSeq(currentDesktop, ":") {
		if source, ok := keyThemeSources[strings.ToLower(strings.TrimSpace(desktop))]; ok {
			return source
		}
	}
	return keyThemeSourceNone
}

// gtkKeyTheme returns the GTK key theme name configured for the current environment, or an empty
// string if none is configured or it cannot be read.
func gtkKeyTheme() string {
	// A desktop that answers wins over the settings file: its setting reaches GTK through XSETTINGS
	// or the desktop portal, which take precedence over the file.
	if theme, ok := desktopKeyTheme(); ok {
		return theme
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	// Only GTK3 is consulted: GTK4 dropped key themes.
	return gtkSettingsFileKeyTheme(filepath.Join(homeDir, ".config", "gtk-3.0", "settings.ini"))
}

// desktopKeyTheme returns the GTK key theme name the current desktop is configured with, and
// reports whether the desktop answered.
func desktopKeyTheme() (string, bool) {
	switch keyThemeSourceForDesktop(os.Getenv("XDG_CURRENT_DESKTOP")) {
	case keyThemeSourceGNOME:
		return gsettingsKeyTheme("org.gnome.desktop.interface")
	case keyThemeSourceCinnamon:
		return gsettingsKeyTheme("org.cinnamon.desktop.interface")
	case keyThemeSourceMATE:
		return gsettingsKeyTheme("org.mate.interface")
	case keyThemeSourceXfce:
		return xfconfKeyTheme()
	}
	return "", false
}

// gsettingsKeyTheme returns the gtk-key-theme value in a GSettings schema, and reports whether the
// value could be read.
func gsettingsKeyTheme(schema string) (string, bool) {
	out, err := exec.Command("gsettings", "get", schema, "gtk-key-theme").Output()
	if err != nil {
		return "", false
	}
	return unquoteGSettingsString(string(out)), true
}

// unquoteGSettingsString returns the content of a GVariant string literal as printed by gsettings.
func unquoteGSettingsString(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "'")
	value = strings.TrimSuffix(value, "'")
	return value
}

// xfconfKeyTheme returns the Xfce XSETTINGS key theme name, and reports whether the value could be
// read.
func xfconfKeyTheme() (string, bool) {
	out, err := exec.Command("xfconf-query", "-c", "xsettings", "-p", "/Gtk/KeyThemeName").Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

// gtkSettingsFileKeyTheme returns the gtk-key-theme-name value in the GTK settings file at path, or
// an empty string if the file cannot be read or sets no such key.
func gtkSettingsFileKeyTheme(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return keyThemeInGTKSettings(string(data))
}

// keyThemeInGTKSettings returns the gtk-key-theme-name value in the content of a GTK settings file,
// or an empty string if the content sets no such key. The last assignment wins.
func keyThemeInGTKSettings(content string) string {
	var theme string
	for line := range strings.Lines(content) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(key) != "gtk-key-theme-name" {
			continue
		}
		theme = strings.TrimSpace(value)
	}
	return theme
}
