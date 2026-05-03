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

// lineColWriter is an [io.Writer] that streams the prefix of a document
// up to the cursor and tracks the cursor's 0-indexed line plus the bytes
// of the line in progress.
//
// Line boundaries follow Unicode TR#13:
//
//	LF U+000A, VT U+000B, FF U+000C, CR U+000D, CRLF, NEL U+0085,
//	LS U+2028, PS U+2029.
type lineColWriter struct {
	line    int
	lineBuf bytes.Buffer
	pending []byte
}

func (w *lineColWriter) Write(p []byte) (int, error) {
	n := len(p)
	if len(w.pending) > 0 {
		merged := make([]byte, 0, len(w.pending)+len(p))
		merged = append(merged, w.pending...)
		merged = append(merged, p...)
		w.pending = w.pending[:0]
		p = merged
	}

	for len(p) > 0 {
		b := p[0]
		switch {
		case b == '\n', b == '\v', b == '\f':
			w.lineBreak()
			p = p[1:]
		case b == '\r':
			if len(p) < 2 {
				w.pending = append(w.pending, b)
				return n, nil
			}
			w.lineBreak()
			if p[1] == '\n' {
				p = p[2:]
			} else {
				p = p[1:]
			}
		case b == 0xC2: // NEL = 0xC2 0x85
			if len(p) < 2 {
				w.pending = append(w.pending, b)
				return n, nil
			}
			if p[1] == 0x85 {
				w.lineBreak()
				p = p[2:]
			} else {
				if err := w.lineBuf.WriteByte(b); err != nil {
					return 0, err
				}
				p = p[1:]
			}
		case b == 0xE2: // LS = 0xE2 0x80 0xA8, PS = 0xE2 0x80 0xA9
			if len(p) < 3 {
				w.pending = append(w.pending, p...)
				return n, nil
			}
			if p[1] == 0x80 && (p[2] == 0xA8 || p[2] == 0xA9) {
				w.lineBreak()
				p = p[3:]
			} else {
				if err := w.lineBuf.WriteByte(b); err != nil {
					return 0, err
				}
				p = p[1:]
			}
		default:
			i := 1
			for i < len(p) {
				c := p[i]
				if c == '\n' || c == '\r' || c == '\v' || c == '\f' || c == 0xC2 || c == 0xE2 {
					break
				}
				i++
			}
			if _, err := w.lineBuf.Write(p[:i]); err != nil {
				return 0, err
			}
			p = p[i:]
		}
	}
	return n, nil
}

func (w *lineColWriter) lineBreak() {
	w.line++
	w.lineBuf.Reset()
}

// Flush commits any pending bytes.
func (w *lineColWriter) Flush() error {
	if len(w.pending) == 0 {
		return nil
	}
	if w.pending[0] == '\r' {
		w.lineBreak()
	} else {
		if _, err := w.lineBuf.Write(w.pending); err != nil {
			return err
		}
	}
	w.pending = w.pending[:0]
	return nil
}

func formatPosition(line int, lineBytes []byte) string {
	col := 1
	var seg segmenter.Segmenter
	if err := seg.InitWithBytes(lineBytes); err != nil {
		// Invalid UTF-8: fall back to byte-based column.
		col = len(lineBytes) + 1
	} else {
		it := seg.GraphemeIterator()
		for it.Next() {
			col++
		}
	}
	return fmt.Sprintf("Line %d, Column %d", line+1, col)
}
