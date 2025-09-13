// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package guigui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type DefaultWidget struct {
	s widgetState
}

var _ Widget = (*DefaultWidget)(nil)

func (*DefaultWidget) Model(key any) any {
	return nil
}

func (*DefaultWidget) AddChildren(context *Context, adder *ChildAdder) {
}

func (*DefaultWidget) Update(context *Context) error {
	return nil
}

func (*DefaultWidget) Layout(context *Context, widget Widget) image.Rectangle {
	// TODO: Return appropriate bounds for the child widgets.
	return image.Rectangle{}
}

func (*DefaultWidget) HandlePointingInput(context *Context) HandleInputResult {
	return HandleInputResult{}
}

func (*DefaultWidget) HandleButtonInput(context *Context) HandleInputResult {
	return HandleInputResult{}
}

func (*DefaultWidget) Tick(context *Context) error {
	return nil
}

func (*DefaultWidget) CursorShape(context *Context) (ebiten.CursorShapeType, bool) {
	return 0, false
}

func (*DefaultWidget) Draw(context *Context, dst *ebiten.Image) {
}

func (*DefaultWidget) ZDelta() int {
	return 0
}

func (d *DefaultWidget) Measure(context *Context, constraints Constraints) image.Point {
	var s image.Point
	if d.widgetState().root {
		s = context.app.bounds().Size()
	} else {
		s = image.Pt(int(144*context.Scale()), int(144*context.Scale()))
	}
	if w, ok := constraints.FixedWidth(); ok {
		s.X = w
	}
	if h, ok := constraints.FixedHeight(); ok {
		s.Y = h
	}
	return s
}

func (*DefaultWidget) PassThrough() bool {
	return false
}

func (d *DefaultWidget) widgetState() *widgetState {
	return &d.s
}
