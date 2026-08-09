// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// IsMouseButtonRepeating reports whether button is pressed and its press
// duration is at a key-repeat firing point.
func IsMouseButtonRepeating(button ebiten.MouseButton) bool {
	if !ebiten.IsMouseButtonPressed(button) {
		return false
	}
	return repeat(inpututil.MouseButtonPressDuration(button))
}

// IsKeyRepeating reports whether key is pressed and its press duration is at a
// key-repeat firing point.
func IsKeyRepeating(key ebiten.Key) bool {
	if !ebiten.IsKeyPressed(key) {
		return false
	}
	return repeat(inpututil.KeyPressDuration(key))
}

// IsModifierPressed reports whether a modifier key was held at any point during
// the current tick. A non-modifier key always reports false.
func IsModifierPressed(key ebiten.Key) bool {
	switch key {
	case ebiten.KeyAlt, ebiten.KeyAltLeft, ebiten.KeyAltRight,
		ebiten.KeyControl, ebiten.KeyControlLeft, ebiten.KeyControlRight,
		ebiten.KeyMeta, ebiten.KeyMetaLeft, ebiten.KeyMetaRight,
		ebiten.KeyShift, ebiten.KeyShiftLeft, ebiten.KeyShiftRight:
	default:
		return false
	}

	if ebiten.IsKeyPressed(key) {
		return true
	}

	// A modifier can be released in the same tick as the key it qualifies, when a
	// stalled event queue delivers both at once. An input event is stamped with
	// the tick it is processed in, so a press edge that arrives together with the
	// modifier's release edge sees the modifier as already up
	// (hajimehoshi/ebiten#3497).
	//
	// ebiten.IsKeyPressed maps a virtual modifier key onto its left and right
	// variants, but inpututil.IsKeyJustReleased tracks the variants only and
	// never reports the virtual key (hajimehoshi/ebiten#3498). Check the variants
	// explicitly.
	switch key {
	case ebiten.KeyAlt:
		return inpututil.IsKeyJustReleased(ebiten.KeyAltLeft) || inpututil.IsKeyJustReleased(ebiten.KeyAltRight)
	case ebiten.KeyControl:
		return inpututil.IsKeyJustReleased(ebiten.KeyControlLeft) || inpututil.IsKeyJustReleased(ebiten.KeyControlRight)
	case ebiten.KeyMeta:
		return inpututil.IsKeyJustReleased(ebiten.KeyMetaLeft) || inpututil.IsKeyJustReleased(ebiten.KeyMetaRight)
	case ebiten.KeyShift:
		return inpututil.IsKeyJustReleased(ebiten.KeyShiftLeft) || inpututil.IsKeyJustReleased(ebiten.KeyShiftRight)
	}
	return inpututil.IsKeyJustReleased(key)
}

func repeat(duration int) bool {
	// duration can be 0 e.g. when pressing Ctrl+A on macOS.
	// A release event might be sent too quickly after the press event.
	if duration <= 1 {
		return true
	}
	delay := ebiten.TPS() * 2 / 5
	if duration < delay {
		return false
	}
	return (duration-delay)%4 == 0
}

// doubleClickLimitInTicks returns the maximum number of ticks between two
// clicks that count as a double click.
func doubleClickLimitInTicks() int {
	return ebiten.TPS() / 2
}
