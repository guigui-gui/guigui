// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package ebitenginewidget

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
)

// forwardInput sends the window's input to the guest. Positions are translated into the widget's area
// and divided by the full scale (device scale times application scale), matching the guest's outside
// coordinates: the guest screen is bounds/AppScale physical pixels, of which the guest sees the
// device-independent size.
func (e *Ebitengine) forwardInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
	s := e.gp.session
	bounds := widgetBounds.Bounds()
	scale := context.Scale()
	if scale <= 0 {
		scale = 1
	}
	localX := func(x int) float64 {
		return float64(x-bounds.Min.X) / scale
	}
	localY := func(y int) float64 {
		return float64(y-bounds.Min.Y) / scale
	}

	// Key presses and typed characters are forwarded only while the game area is focused, so typing into
	// surrounding Guigui widgets does not reach the guest.
	if context.IsFocused(e) {
		e.keyBuf = inpututil.AppendJustPressedKeys(e.keyBuf[:0])
		for _, k := range e.keyBuf {
			s.PressKey(k)
			if e.pressedKeys == nil {
				e.pressedKeys = map[ebiten.Key]struct{}{}
			}
			e.pressedKeys[k] = struct{}{}
		}
		e.runeBuf = ebiten.AppendInputChars(e.runeBuf[:0])
		for _, r := range e.runeBuf {
			s.TypeRune(r)
		}
	}

	// Key releases are forwarded regardless of focus, like the mouse button releases below: dropping a
	// release would leave the guest with a stuck key.
	e.keyBuf = inpututil.AppendJustReleasedKeys(e.keyBuf[:0])
	for _, k := range e.keyBuf {
		if _, ok := e.pressedKeys[k]; !ok {
			continue
		}
		s.ReleaseKey(k)
		delete(e.pressedKeys, k)
	}

	// The cursor is forwarded while it is over the game, or while a button pressed over the game is still
	// held (a drag that left the area).
	hit := widgetBounds.IsHitAtCursor()
	dragging := len(e.pressedMouseButtons) > 0
	if hit || dragging {
		cx, cy := ebiten.CursorPosition()
		s.MoveCursor(localX(cx), localY(cy))
	}
	for _, b := range []ebiten.MouseButton{ebiten.MouseButtonLeft, ebiten.MouseButtonRight, ebiten.MouseButtonMiddle} {
		if !hit || !inpututil.IsMouseButtonJustPressed(b) {
			continue
		}
		s.PressMouseButton(b)
		if e.pressedMouseButtons == nil {
			e.pressedMouseButtons = map[ebiten.MouseButton]struct{}{}
		}
		e.pressedMouseButtons[b] = struct{}{}
	}

	// Mouse button releases are forwarded regardless of the cursor's location, so a drag that ends
	// outside the game does not leave the guest with a stuck button.
	for _, b := range []ebiten.MouseButton{ebiten.MouseButtonLeft, ebiten.MouseButtonRight, ebiten.MouseButtonMiddle} {
		if !inpututil.IsMouseButtonJustReleased(b) {
			continue
		}
		if _, ok := e.pressedMouseButtons[b]; !ok {
			continue
		}
		s.ReleaseMouseButton(b)
		delete(e.pressedMouseButtons, b)
	}
	if hit {
		if wx, wy := ebiten.Wheel(); wx != 0 || wy != 0 {
			s.ScrollWheel(wx, wy)
		}
	}

	// A touch that begins over the game is forwarded, then tracked so its moves and release follow even
	// if it leaves the area.
	if e.forwardedTouches == nil {
		e.forwardedTouches = map[ebiten.TouchID]struct{}{}
	}
	e.touchIDsBuf = inpututil.AppendJustPressedTouchIDs(e.touchIDsBuf[:0])
	for _, id := range e.touchIDsBuf {
		x, y := ebiten.TouchPosition(id)
		if !widgetBounds.IsHitAt(image.Pt(x, y)) {
			continue
		}
		s.PressTouch(id, localX(x), localY(y))
		e.forwardedTouches[id] = struct{}{}
	}
	e.touchIDsBuf = ebiten.AppendTouchIDs(e.touchIDsBuf[:0])
	for _, id := range e.touchIDsBuf {
		if _, ok := e.forwardedTouches[id]; !ok {
			continue
		}
		// A just-pressed touch was already positioned by PressTouch above; only a continuing touch moves.
		if inpututil.TouchPressDuration(id) == 1 {
			continue
		}
		x, y := ebiten.TouchPosition(id)
		s.MoveTouch(id, localX(x), localY(y))
	}
	e.touchIDsBuf = inpututil.AppendJustReleasedTouchIDs(e.touchIDsBuf[:0])
	for _, id := range e.touchIDsBuf {
		if _, ok := e.forwardedTouches[id]; !ok {
			continue
		}
		s.ReleaseTouch(id)
		delete(e.forwardedTouches, id)
	}

	// Gamepads are mirrored unconditionally, since they are not tied to the cursor or focus.
	// UpdateGamepads copies the snapshot out, so the states and their inner slices and maps are reused
	// across ticks.
	e.gamepadIDsBuf = ebiten.AppendGamepadIDs(e.gamepadIDsBuf[:0])
	// Reslicing within the capacity keeps the elements, so their slices and maps are reused; growing
	// appends zero elements while keeping the existing ones.
	if n := len(e.gamepadIDsBuf); n <= cap(e.gamepadStatesBuf) {
		e.gamepadStatesBuf = e.gamepadStatesBuf[:n]
	} else {
		e.gamepadStatesBuf = append(e.gamepadStatesBuf[:cap(e.gamepadStatesBuf)], make([]vmhost.GamepadState, n-cap(e.gamepadStatesBuf))...)
	}
	for i, id := range e.gamepadIDsBuf {
		updateGamepadState(&e.gamepadStatesBuf[i], id)
	}
	s.UpdateGamepads(e.gamepadStatesBuf)
}

// updateGamepadState reads the current state of one gamepad through the public ebiten API into state,
// reusing state's slices and maps.
func updateGamepadState(state *vmhost.GamepadState, id ebiten.GamepadID) {
	state.ID = id
	state.SDLID = ebiten.GamepadSDLID(id)
	state.Name = ebiten.GamepadName(id)

	state.Axes = state.Axes[:0]
	for a := 0; a < ebiten.GamepadAxisCount(id); a++ {
		state.Axes = append(state.Axes, ebiten.GamepadAxisValue(id, a))
	}
	state.Buttons = state.Buttons[:0]
	for b := 0; b < ebiten.GamepadButtonCount(id); b++ {
		state.Buttons = append(state.Buttons, ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton(b)))
	}

	// The standard-layout view is keyed on present entries, so leaving the maps empty means the layout
	// is unavailable.
	if state.StandardAxes == nil {
		state.StandardAxes = map[ebiten.StandardGamepadAxis]float64{}
	} else {
		clear(state.StandardAxes)
	}
	if state.StandardButtons == nil {
		state.StandardButtons = map[ebiten.StandardGamepadButton]vmhost.GamepadStandardButtonState{}
	} else {
		clear(state.StandardButtons)
	}
	if !ebiten.IsStandardGamepadLayoutAvailable(id) {
		return
	}
	for a := ebiten.StandardGamepadAxis(0); a <= ebiten.StandardGamepadAxisMax; a++ {
		if !ebiten.IsStandardGamepadAxisAvailable(id, a) {
			continue
		}
		state.StandardAxes[a] = ebiten.StandardGamepadAxisValue(id, a)
	}
	for b := ebiten.StandardGamepadButton(0); b <= ebiten.StandardGamepadButtonMax; b++ {
		if !ebiten.IsStandardGamepadButtonAvailable(id, b) {
			continue
		}
		state.StandardButtons[b] = vmhost.GamepadStandardButtonState{
			Pressed: ebiten.IsStandardGamepadButtonPressed(id, b),
			Value:   ebiten.StandardGamepadButtonValue(id, b),
		}
	}
}

// updateTextInput serves the guest's text-input sessions on the host's platform IME through a
// [vmhostutil.ComposerForwarder]. It runs before the guest's ticks are advanced so that a commit
// reaches the guest in the same tick as the raw input that produced it; the guest's own composer then
// reports the input as handled and its game does not process it twice.
func (e *Ebitengine) updateTextInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
	if textInput := e.gp.takeNewTextInput(); textInput != nil {
		e.textInput = textInput
	}
	if e.textInput != nil && e.textInput.IsClosed() {
		// The guest released the session (its game ended text inputting), or it was already answered
		// with a commit or an error.
		e.textInput = nil
	}
	// The host IME is driven only while the widget is focused, like key forwarding, so text inputting
	// into surrounding Guigui widgets does not reach the guest.
	if e.textInput == nil || e.inputForwardingDisabled || !context.IsFocused(e) {
		e.composerForwarder.Reset()
		return
	}
	// The guest caret's rectangle is in the guest's device-independent pixels; scaling by the full
	// scale from the widget's origin translates it into the widget's coordinate space (the inverse of
	// forwardInput's position translation).
	bounds := widgetBounds.Bounds()
	scale := context.Scale()
	if scale <= 0 {
		scale = 1
	}
	cb := e.textInput.CaretBounds()
	caretBounds := image.Rect(
		bounds.Min.X+int(math.Round(float64(cb.Min.X)*scale)),
		bounds.Min.Y+int(math.Round(float64(cb.Min.Y)*scale)),
		bounds.Min.X+int(math.Round(float64(cb.Max.X)*scale)),
		bounds.Min.Y+int(math.Round(float64(cb.Max.Y)*scale)),
	)
	e.composerForwarder.Forward(e.textInput, caretBounds)
	// The handled result is not consulted: the widget mirrors the host's raw input as-is, and the
	// forwarded states make the guest's own composer report the same result to the guest's game.
	e.composerForwarder.Update()
}
