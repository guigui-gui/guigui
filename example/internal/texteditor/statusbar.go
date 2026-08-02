// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package texteditor

import (
	"fmt"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

// StatusBar displays the caret position of an editor.
type StatusBar struct {
	guigui.DefaultWidget

	text basicwidget.Text
}

// SetPosition sets the displayed caret position. line and column are
// 1-based display values; the column unit is the caller's choice.
func (s *StatusBar) SetPosition(line, column int) {
	s.text.SetValue(fmt.Sprintf("Line %d, Column %d", line, column))
}

func (s *StatusBar) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&s.text)
	s.text.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	return nil
}

func (s *StatusBar) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	b := widgetBounds.Bounds()
	b.Min.X += u / 2
	b.Max.X -= u / 2
	layouter.LayoutWidget(&s.text, b)
}
