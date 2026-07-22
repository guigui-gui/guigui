// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"fmt"
	"image/color"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

const richTextsSampleText = "Colored, light, highlighted, underlined, struck-through, bold, and scaled ranges.\n" +
	"Combined styles apply to a single range.\n" +
	"A sufficiently long styled range continues across the visual line boundary when the text wraps, keeping its background and underline on every visual line it covers.\n" +
	"The clickable range counts clicks.\n" +
	"日本語のテキストにも下線と背景を適用できます。"

type RichTexts struct {
	guigui.DefaultWidget

	form                basicwidget.Form
	selectableText      basicwidget.Text
	selectableToggle    basicwidget.Toggle
	clickCountText      basicwidget.Text
	clickCountValueText basicwidget.Text
	sampleText          basicwidget.Text

	layoutItems []guigui.LinearLayoutItem
}

func styleRichTextsSampleRange(sub string, f func(start, end int)) {
	idx := strings.Index(richTextsSampleText, sub)
	if idx < 0 {
		return
	}
	f(idx, idx+len(sub))
}

func (r *RichTexts) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.sampleText)
	adder.AddWidget(&r.form)

	v, ok := context.Env(r, modelKeyModel)
	if !ok {
		return nil
	}
	model := v.(*Model)

	r.selectableText.SetValue("Selectable")
	r.selectableToggle.OnValueChanged(func(context *guigui.Context, value bool) {
		model.RichTexts().SetSelectable(value)
	})
	r.selectableToggle.SetValue(model.RichTexts().Selectable())

	r.clickCountText.SetValue("Clickable range clicks")
	r.clickCountValueText.SetValue(fmt.Sprintf("%d", model.RichTexts().ClickCount()))

	styleRichTextsSampleRange("clickable range", func(start, end int) {
		r.sampleText.SetHotspotRanges([]basicwidget.TextRange{
			{
				StartInBytes: start,
				EndInBytes:   end,
			},
		})
	})
	r.sampleText.OnHotspotUp(func(context *guigui.Context, textRange basicwidget.TextRange) {
		model.RichTexts().IncrementClickCount()
	})

	r.form.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &r.selectableText,
			SecondaryWidget: &r.selectableToggle,
		},
		{
			PrimaryWidget:   &r.clickCountText,
			SecondaryWidget: &r.clickCountValueText,
		},
	})

	red := color.RGBA{R: 0xff, G: 0x4b, B: 0x00, A: 0xff}
	blue := color.RGBA{R: 0x00, G: 0x5a, B: 0xff, A: 0xff}
	yellow := color.NRGBA{R: 0xff, G: 0xf1, B: 0x00, A: 0x60}
	green := color.NRGBA{R: 0x03, G: 0xaf, B: 0x7a, A: 0x50}

	t := &r.sampleText
	t.SetMultiline(true)
	t.SetWrapMode(basicwidget.WrapModeNormal)
	t.SetSelectable(model.RichTexts().Selectable())
	t.SetValue(richTextsSampleText)
	styleRichTextsSampleRange("Colored", func(start, end int) {
		t.SetColorInRange(start, end, red)
	})
	styleRichTextsSampleRange("highlighted", func(start, end int) {
		t.SetBackgroundColorInRange(start, end, yellow)
	})
	styleRichTextsSampleRange("underlined", func(start, end int) {
		t.SetUnderlineInRange(start, end, true)
	})
	styleRichTextsSampleRange("struck-through", func(start, end int) {
		t.SetStrikethroughInRange(start, end, true)
	})
	styleRichTextsSampleRange("bold", func(start, end int) {
		t.SetWeightInRange(start, end, text.WeightBold)
	})
	styleRichTextsSampleRange("light", func(start, end int) {
		t.SetWeightInRange(start, end, text.WeightLight)
	})
	styleRichTextsSampleRange("scaled", func(start, end int) {
		t.SetScaleInRange(start, end, 1.5)
	})
	styleRichTextsSampleRange("Combined styles", func(start, end int) {
		t.SetColorInRange(start, end, blue)
		t.SetBackgroundColorInRange(start, end, green)
		t.SetUnderlineInRange(start, end, true)
		t.SetWeightInRange(start, end, text.WeightBold)
	})
	styleRichTextsSampleRange("continues across the visual line boundary when the text wraps, keeping its background and underline", func(start, end int) {
		t.SetColorInRange(start, end, blue)
		t.SetBackgroundColorInRange(start, end, green)
		t.SetUnderlineInRange(start, end, true)
	})
	styleRichTextsSampleRange("clickable range", func(start, end int) {
		t.SetColorInRange(start, end, blue)
		t.SetUnderlineInRange(start, end, true)
	})
	styleRichTextsSampleRange("下線", func(start, end int) {
		t.SetUnderlineInRange(start, end, true)
	})
	styleRichTextsSampleRange("背景", func(start, end int) {
		t.SetBackgroundColorInRange(start, end, yellow)
	})

	return nil
}

func (r *RichTexts) WriteStateKey(context *guigui.Context, w *guigui.StateKeyWriter) {
	if v, ok := context.Env(r, modelKeyModel); ok {
		w.WriteInt(v.(*Model).RichTexts().ClickCount())
	}
}

func (r *RichTexts) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	r.layoutItems = slices.Delete(r.layoutItems, 0, len(r.layoutItems))
	r.layoutItems = append(r.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &r.sampleText,
			Size:   guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: &r.form,
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     r.layoutItems,
		Gap:       u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}
