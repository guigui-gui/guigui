// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"fmt"
	"slices"
	"unicode/utf8"

	"github.com/go-text/typesetting/segmenter"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type statusBar struct {
	guigui.DefaultWidget

	text basicwidget.Text
	seg  segmenter.Segmenter

	cachedLinePlus1 int
	cachedCols      []int
}

func (s *statusBar) InvalidateCache() {
	s.cachedLinePlus1 = 0
	s.cachedCols = s.cachedCols[:0]
}

func (s *statusBar) SetStatus(line int, lineBytes []byte, cursorOffsetInLine int) {
	if s.cachedLinePlus1 != line+1 {
		s.rebuildCache(line, lineBytes)
	}
	col := s.cachedCols[cursorOffsetInLine]
	text := fmt.Sprintf("Line %d, Column %d", line+1, col)
	s.text.SetValue(text)
}

func (s *statusBar) rebuildCache(line int, lineBytes []byte) {
	s.cachedCols = slices.Grow(s.cachedCols[:0], len(lineBytes)+1)[:len(lineBytes)+1]

	if err := s.seg.InitWithBytes(lineBytes); err != nil {
		// Invalid UTF-8: byte-based column.
		for b := range s.cachedCols {
			s.cachedCols[b] = b + 1
		}
		s.cachedLinePlus1 = line + 1
		return
	}

	var bytePos int
	col := 1
	it := s.seg.GraphemeIterator()
	for it.Next() {
		gr := it.Grapheme()
		var n int
		for _, r := range gr.Text {
			n += utf8.RuneLen(r)
		}
		end := bytePos + n
		for b := bytePos; b < end; b++ {
			s.cachedCols[b] = col
		}
		bytePos = end
		col++
	}
	s.cachedCols[len(lineBytes)] = col
	s.cachedLinePlus1 = line + 1
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
