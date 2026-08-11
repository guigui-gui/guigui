// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

//go:build unix && !android && !darwin

package keybindingmode_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/guigui-gui/guigui/internal/keybindingmode"
)

func TestModeForKeyTheme(t *testing.T) {
	testCases := []struct {
		theme string
		want  keybindingmode.Mode
	}{
		{
			theme: "Emacs",
			want:  keybindingmode.ControlEmacs,
		},
		{
			theme: "Default",
			want:  keybindingmode.ControlDefault,
		},
		{
			theme: "",
			want:  keybindingmode.ControlDefault,
		},
		{
			// A key theme name is the basename of a directory, so the match is case sensitive.
			theme: "emacs",
			want:  keybindingmode.ControlDefault,
		},
		{
			theme: "Emacs2",
			want:  keybindingmode.ControlDefault,
		},
		{
			theme: "Mac",
			want:  keybindingmode.ControlDefault,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.theme, func(t *testing.T) {
			if got, want := keybindingmode.ModeForKeyTheme(tc.theme), tc.want; got != want {
				t.Errorf("ModeForKeyTheme(%q): got %d, want %d", tc.theme, got, want)
			}
		})
	}
}

func TestKeyThemeSourceForDesktop(t *testing.T) {
	testCases := []struct {
		currentDesktop string
		want           keybindingmode.KeyThemeSource
	}{
		{
			currentDesktop: "GNOME",
			want:           keybindingmode.KeyThemeSourceGNOME,
		},
		{
			currentDesktop: "ubuntu:GNOME",
			want:           keybindingmode.KeyThemeSourceGNOME,
		},
		{
			currentDesktop: "GNOME-Flashback:GNOME",
			want:           keybindingmode.KeyThemeSourceGNOME,
		},
		{
			currentDesktop: "Budgie:GNOME",
			want:           keybindingmode.KeyThemeSourceGNOME,
		},
		{
			currentDesktop: "Unity:Unity7:ubuntu",
			want:           keybindingmode.KeyThemeSourceGNOME,
		},
		{
			currentDesktop: "Pantheon",
			want:           keybindingmode.KeyThemeSourceGNOME,
		},
		{
			currentDesktop: "X-Cinnamon",
			want:           keybindingmode.KeyThemeSourceCinnamon,
		},
		{
			currentDesktop: "MATE",
			want:           keybindingmode.KeyThemeSourceMATE,
		},
		{
			currentDesktop: "XFCE",
			want:           keybindingmode.KeyThemeSourceXfce,
		},
		{
			currentDesktop: "xfce",
			want:           keybindingmode.KeyThemeSourceXfce,
		},
		{
			// The first desktop with a source wins, so an Xfce session is not answered by a
			// GNOME schema that merely happens to be installed.
			currentDesktop: "XFCE:GNOME",
			want:           keybindingmode.KeyThemeSourceXfce,
		},
		{
			// A desktop without a source is skipped rather than ending the search.
			currentDesktop: "Enlightenment:GNOME",
			want:           keybindingmode.KeyThemeSourceGNOME,
		},
		{
			currentDesktop: "KDE",
			want:           keybindingmode.KeyThemeSourceNone,
		},
		{
			currentDesktop: "sway",
			want:           keybindingmode.KeyThemeSourceNone,
		},
		{
			currentDesktop: "",
			want:           keybindingmode.KeyThemeSourceNone,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.currentDesktop, func(t *testing.T) {
			if got, want := keybindingmode.KeyThemeSourceForDesktop(tc.currentDesktop), tc.want; got != want {
				t.Errorf("KeyThemeSourceForDesktop(%q): got %d, want %d", tc.currentDesktop, got, want)
			}
		})
	}
}

func TestUnquoteGSettingsString(t *testing.T) {
	testCases := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "quoted",
			value: "'Emacs'\n",
			want:  "Emacs",
		},
		{
			name:  "default",
			value: "'Default'\n",
			want:  "Default",
		},
		{
			name:  "empty",
			value: "''\n",
			want:  "",
		},
		{
			name:  "no newline",
			value: "'Emacs'",
			want:  "Emacs",
		},
		{
			name:  "unquoted",
			value: "Emacs\n",
			want:  "Emacs",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got, want := keybindingmode.UnquoteGSettingsString(tc.value), tc.want; got != want {
				t.Errorf("UnquoteGSettingsString(%q): got %q, want %q", tc.value, got, want)
			}
		})
	}
}

func TestKeyThemeInGTKSettings(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "typical",
			content: "[Settings]\ngtk-key-theme-name=Emacs\n",
			want:    "Emacs",
		},
		{
			name:    "spaces around the separator",
			content: "[Settings]\ngtk-key-theme-name = Emacs\n",
			want:    "Emacs",
		},
		{
			name:    "among other keys",
			content: "[Settings]\ngtk-theme-name=Adwaita\ngtk-key-theme-name=Emacs\ngtk-font-name=Sans 10\n",
			want:    "Emacs",
		},
		{
			name:    "commented out",
			content: "[Settings]\n#gtk-key-theme-name=Emacs\n",
			want:    "",
		},
		{
			name:    "key not set",
			content: "[Settings]\ngtk-theme-name=Adwaita\n",
			want:    "",
		},
		{
			name:    "empty value",
			content: "[Settings]\ngtk-key-theme-name=\n",
			want:    "",
		},
		{
			name:    "last assignment wins",
			content: "[Settings]\ngtk-key-theme-name=Emacs\ngtk-key-theme-name=Default\n",
			want:    "Default",
		},
		{
			name:    "another key ending with the same name",
			content: "[Settings]\nxgtk-key-theme-name=Emacs\n",
			want:    "",
		},
		{
			name:    "no trailing newline",
			content: "[Settings]\ngtk-key-theme-name=Emacs",
			want:    "Emacs",
		},
		{
			name:    "CRLF",
			content: "[Settings]\r\ngtk-key-theme-name=Emacs\r\n",
			want:    "Emacs",
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got, want := keybindingmode.KeyThemeInGTKSettings(tc.content), tc.want; got != want {
				t.Errorf("KeyThemeInGTKSettings(%q): got %q, want %q", tc.content, got, want)
			}
		})
	}
}

func TestGTKSettingsFileKeyTheme(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "settings.ini")
	if err := os.WriteFile(path, []byte("[Settings]\ngtk-key-theme-name=Emacs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, want := keybindingmode.GTKSettingsFileKeyTheme(path), "Emacs"; got != want {
		t.Errorf("GTKSettingsFileKeyTheme: got %q, want %q", got, want)
	}

	if got, want := keybindingmode.GTKSettingsFileKeyTheme(filepath.Join(dir, "nonexistent.ini")), ""; got != want {
		t.Errorf("GTKSettingsFileKeyTheme for a missing file: got %q, want %q", got, want)
	}
}
