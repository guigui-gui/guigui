// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"fmt"

	"github.com/go-text/typesetting/segmenter"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type statusBar struct {
	guigui.DefaultWidget

	text basicwidget.Text
	seg  segmenter.Segmenter
}

func (s *statusBar) SetStatus(line int, lineBytes []byte) {
	col := 1
	if err := s.seg.InitWithBytes(lineBytes); err != nil {
		// Invalid UTF-8: fall back to byte-based column.
		col = len(lineBytes) + 1
	} else {
		it := s.seg.GraphemeIterator()
		for it.Next() {
			col++
		}
	}
	text := fmt.Sprintf("Line %d, Column %d", line+1, col)
	s.text.SetValue(text)
}

func (s *statusBar) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&s.text)
	s.text.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	return nil
}

func (s *statusBar) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	b := widgetBounds.Bounds()
	b.Min.X += u / 2
	b.Max.X -= u / 2
	layouter.LayoutWidget(&s.text, b)
}
