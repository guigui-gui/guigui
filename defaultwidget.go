// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package guigui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type DefaultWidget struct {
	s widgetState

	_    noCopy
	addr *DefaultWidget
}

var _ Widget = (*DefaultWidget)(nil)

func (d *DefaultWidget) copyCheck() {
	if d.addr == nil {
		d.addr = d
	} else if d.addr != d {
		panic("guigui: illegal use of DefaultWidget copied by value")
	}
}

func (*DefaultWidget) Model(key ModelKey) any {
	return nil
}

func (*DefaultWidget) Build(context *Context, adder *ChildAdder) error {
	return nil
}

func (*DefaultWidget) Layout(context *Context, widgetBounds *WidgetBounds, layouter *ChildLayouter) {
}

func (*DefaultWidget) HandlePointingInput(context *Context, widgetBounds *WidgetBounds) HandleInputResult {
	context.setDefaultMethodCalledFlag()
	return HandleInputResult{}
}

func (*DefaultWidget) HandleButtonInput(context *Context, widgetBounds *WidgetBounds) HandleInputResult {
	return HandleInputResult{}
}

func (*DefaultWidget) Tick(context *Context, widgetBounds *WidgetBounds) error {
	return nil
}

func (*DefaultWidget) CursorShape(context *Context, widgetBounds *WidgetBounds) (ebiten.CursorShapeType, bool) {
	context.setDefaultMethodCalledFlag()
	return 0, false
}

func (*DefaultWidget) Draw(context *Context, widgetBounds *WidgetBounds, dst *ebiten.Image) {
	context.setDefaultMethodCalledFlag()
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

func (d *DefaultWidget) widgetState() *widgetState {
	d.copyCheck()
	return &d.s
}
