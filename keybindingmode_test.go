// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package guigui_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
)

func TestKeyBindingModeShortcutModifierKey(t *testing.T) {
	tests := []struct {
		mode guigui.KeyBindingMode
		want ebiten.Key
	}{
		{
			mode: guigui.KeyBindingModeCommand,
			want: ebiten.KeyMeta,
		},
		{
			mode: guigui.KeyBindingModeControlDefault,
			want: ebiten.KeyControl,
		},
		{
			mode: guigui.KeyBindingModeControlEmacs,
			want: ebiten.KeyControl,
		},
	}
	for _, tc := range tests {
		if got := tc.mode.ShortcutModifierKey(); got != tc.want {
			t.Errorf("mode %d: ShortcutModifierKey() = %v, want %v", tc.mode, got, tc.want)
		}
	}
}

func TestKeyBindingModeUsesEmacsKeymap(t *testing.T) {
	tests := []struct {
		mode guigui.KeyBindingMode
		want bool
	}{
		{
			mode: guigui.KeyBindingModeUnknown,
			want: false,
		},
		{
			mode: guigui.KeyBindingModeCommand,
			want: true,
		},
		{
			mode: guigui.KeyBindingModeControlDefault,
			want: false,
		},
		{
			mode: guigui.KeyBindingModeControlEmacs,
			want: true,
		},
	}
	for _, tc := range tests {
		if got := tc.mode.UsesEmacsKeymap(); got != tc.want {
			t.Errorf("mode %d: UsesEmacsKeymap() = %t, want %t", tc.mode, got, tc.want)
		}
	}
}
