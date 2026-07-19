// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"image/color"
	"slices"
	"strings"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

const richTextsSampleText = "Colored, highlighted, underlined, and struck-through ranges.\n" +
	"Combined styles apply to a single range.\n" +
	"A sufficiently long styled range continues across the visual line boundary when the text wraps, keeping its background and underline on every visual line it covers.\n" +
	"日本語のテキストにも下線と背景を適用できます。"

type RichTexts struct {
	guigui.DefaultWidget

	form             basicwidget.Form
	boldText         basicwidget.Text
	boldToggle       basicwidget.Toggle
	selectableText   basicwidget.Text
	selectableToggle basicwidget.Toggle
	sampleText       basicwidget.Text

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

	r.boldText.SetValue("Bold")
	r.boldToggle.OnValueChanged(func(context *guigui.Context, value bool) {
		model.RichTexts().SetBold(value)
	})
	r.boldToggle.SetValue(model.RichTexts().Bold())

	r.selectableText.SetValue("Selectable")
	r.selectableToggle.OnValueChanged(func(context *guigui.Context, value bool) {
		model.RichTexts().SetSelectable(value)
	})
	r.selectableToggle.SetValue(model.RichTexts().Selectable())

	r.form.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &r.boldText,
			SecondaryWidget: &r.boldToggle,
		},
		{
			PrimaryWidget:   &r.selectableText,
			SecondaryWidget: &r.selectableToggle,
		},
	})

	red := color.RGBA{R: 0xff, G: 0x4b, B: 0x00, A: 0xff}
	blue := color.RGBA{R: 0x00, G: 0x5a, B: 0xff, A: 0xff}
	yellow := color.NRGBA{R: 0xff, G: 0xf1, B: 0x00, A: 0x60}
	green := color.NRGBA{R: 0x03, G: 0xaf, B: 0x7a, A: 0x50}

	t := &r.sampleText
	t.SetMultiline(true)
	t.SetWrapMode(basicwidget.WrapModeNormal)
	t.SetBold(model.RichTexts().Bold())
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
	styleRichTextsSampleRange("Combined styles", func(start, end int) {
		t.SetColorInRange(start, end, blue)
		t.SetBackgroundColorInRange(start, end, green)
		t.SetUnderlineInRange(start, end, true)
	})
	styleRichTextsSampleRange("continues across the visual line boundary when the text wraps, keeping its background and underline", func(start, end int) {
		t.SetColorInRange(start, end, blue)
		t.SetBackgroundColorInRange(start, end, green)
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
