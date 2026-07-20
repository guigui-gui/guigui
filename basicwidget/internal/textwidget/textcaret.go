// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

func textCaretWidth(context *guigui.Context) int {
	return int(math.Ceil(2 * context.Scale()))
}

type textCaret struct {
	guigui.DefaultWidget

	text *Text

	counter   int
	prevAlpha float64
	prevPos   textutil.TextPosition
	prevOK    bool
}

func (t *textCaret) resetCounter() {
	t.counter = 0
}

func (t *textCaret) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	pos, ok := t.text.caretPosition(context, t.text.widgetBoundsRect)
	if t.prevPos != pos {
		t.resetCounter()
	}
	t.prevPos = pos
	t.prevOK = ok

	t.counter++
	if a := t.alpha(context); t.prevAlpha != a {
		t.prevAlpha = a
		guigui.RequestRedraw(t)
	}
	return nil
}

func (t *textCaret) alpha(context *guigui.Context) float64 {
	// prevOK reflects the current tick: Tick refreshes it before alpha
	// is called, and Draw runs after Tick in the same tick.
	if !t.prevOK {
		return 0
	}
	s, e, ok := t.text.selectionToDraw(context)
	if !ok {
		return 0
	}
	if s != e {
		return 0
	}
	if t.text.caretStatic {
		return 1
	}
	offset := ebiten.TPS() / 2
	if t.counter <= offset {
		return 1
	}
	interval := ebiten.TPS()
	c := (t.counter - offset) % interval
	if c < interval/5 {
		return 1 - float64(c)/float64(interval/5)
	}
	if c < interval*2/5 {
		return 0
	}
	if c < interval*3/5 {
		return float64(c-interval*2/5) / float64(interval/5)
	}
	return 1
}

func (t *textCaret) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	alpha := t.alpha(context)
	if alpha == 0 {
		return
	}
	b := widgetBounds.Bounds()
	if b.Empty() {
		return
	}
	w := textCaretWidth(context)
	region := t.text.widgetBoundsRect
	region.Min.X -= w
	region.Max.X += w
	if !b.In(region) {
		return
	}
	clr := draw.ScaleAlpha(t.text.baseStyle.caretColor, alpha)
	draw.DrawRoundedRect(context, dst, b, clr, b.Dx()/2)
}
