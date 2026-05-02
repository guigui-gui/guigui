// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"bytes"
	"fmt"

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
// prefix (the bytes of the document up to the cursor). Newlines count as
// the start of a new line.
//
// TODO: Read only the bytes of the current line so this scales with line
// length rather than cursor offset.
func lineCol(prefix []byte) (line, col int) {
	line = 1 + bytes.Count(prefix, []byte{'\n'})

	var lineStart int
	if i := bytes.LastIndexByte(prefix, '\n'); i >= 0 {
		lineStart = i + 1
	}

	col = 1
	var seg segmenter.Segmenter
	if err := seg.InitWithBytes(prefix[lineStart:]); err != nil {
		// Invalid UTF-8: fall back to byte-based column.
		col = len(prefix) - lineStart + 1
		return
	}
	it := seg.GraphemeIterator()
	for it.Next() {
		col++
	}
	return
}

func formatPosition(prefix []byte) string {
	line, col := lineCol(prefix)
	return fmt.Sprintf("Line %d, Column %d", line, col)
}
