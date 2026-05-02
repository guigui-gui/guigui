// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"fmt"
	"strings"

	"github.com/go-text/typesetting/segmenter"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type statusBar struct {
	guigui.DefaultWidget

	text basicwidget.Text
}

func (s *statusBar) SetText(text string) {
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

// lineCol returns 1-indexed line and grapheme-cluster column for the given
// byte offset into text. Newlines count as the start of a new line.
func lineCol(text string, offset int) (line, col int) {
	offset = min(max(offset, 0), len(text))
	prefix := text[:offset]
	line = 1 + strings.Count(prefix, "\n")

	var lineStart int
	if i := strings.LastIndexByte(prefix, '\n'); i >= 0 {
		lineStart = i + 1
	}

	col = 1
	var seg segmenter.Segmenter
	if err := seg.InitWithString(text[lineStart:offset]); err != nil {
		// Invalid UTF-8: fall back to byte-based column.
		col = offset - lineStart + 1
		return
	}
	it := seg.GraphemeIterator()
	for it.Next() {
		col++
	}
	return
}

func formatPosition(text string, sel int) string {
	line, col := lineCol(text, sel)
	return fmt.Sprintf("Line %d, Column %d", line, col)
}
