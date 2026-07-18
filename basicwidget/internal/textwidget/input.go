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
