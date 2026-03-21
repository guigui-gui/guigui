// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package guigui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type inputState struct {
	touchIDs         []ebiten.TouchID
	anyMousePressed  bool
	anyTouch         bool
	wheelX, wheelY   float64
	cursorX, cursorY int
	pressedKeys      []ebiten.Key
	justReleasedKeys []ebiten.Key

	prevAnyMousePressed      bool
	prevAnyTouch             bool
	prevCursorX, prevCursorY int
}

func (s *inputState) update() {
	s.prevAnyMousePressed = s.anyMousePressed
	s.prevAnyTouch = s.anyTouch
	s.prevCursorX = s.cursorX
	s.prevCursorY = s.cursorY

	s.anyMousePressed = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) ||
		ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) ||
		ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle)
	s.touchIDs = ebiten.AppendTouchIDs(s.touchIDs[:0])
	s.anyTouch = len(s.touchIDs) > 0
	s.wheelX, s.wheelY = ebiten.Wheel()
	s.cursorX, s.cursorY = ebiten.CursorPosition()
	s.pressedKeys = inpututil.AppendPressedKeys(s.pressedKeys[:0])
	s.justReleasedKeys = inpututil.AppendJustReleasedKeys(s.justReleasedKeys[:0])
}

func (s *inputState) isButtonActive() bool {
	return len(s.pressedKeys) > 0 || len(s.justReleasedKeys) > 0
}

func (s *inputState) isPointingActive(layoutChanged bool) bool {
	cursorMoved := s.cursorX != s.prevCursorX || s.cursorY != s.prevCursorY
	return layoutChanged || cursorMoved ||
		s.anyMousePressed ||
		(!s.anyMousePressed && s.prevAnyMousePressed) ||
		s.anyTouch ||
		(!s.anyTouch && s.prevAnyTouch) ||
		s.wheelX != 0 || s.wheelY != 0
}
