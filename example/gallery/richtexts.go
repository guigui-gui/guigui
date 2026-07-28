// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"fmt"
	"image"
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
	editableText        basicwidget.Text
	editableToggle      basicwidget.Toggle
	resetText           basicwidget.Text
	resetButton         basicwidget.Button
	clickCountText      basicwidget.Text
	clickCountValueText basicwidget.Text
	sampleText          basicwidget.Text
	hotspotTooltipArea  basicwidget.TooltipArea

	hotspotRanges    []basicwidget.TextRange
	layoutItems      []guigui.LinearLayoutItem
	hotspotBoundsArr []image.Rectangle
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
	adder.AddWidget(&r.hotspotTooltipArea)
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

	r.editableText.SetValue("Editable")
	r.editableToggle.OnValueChanged(func(context *guigui.Context, value bool) {
		model.RichTexts().SetEditable(value)
	})
	r.editableToggle.SetValue(model.RichTexts().Editable())

	r.resetText.SetValue("Reset the sample text")
	r.resetButton.SetText("Reset")
	r.resetButton.OnDown(func(context *guigui.Context) {
		model.RichTexts().SetSampleText(richTextsSampleText)
		r.sampleText.ForceSetValue(richTextsSampleText)
		applyRichTextsSampleStyles(&r.sampleText)
	})

	r.clickCountText.SetValue("Clickable range clicks")
	r.clickCountValueText.SetValue(fmt.Sprintf("%d", model.RichTexts().ClickCount()))

	r.sampleText.OnHotspotUp(func(context *guigui.Context, textRange basicwidget.TextRange) {
		model.RichTexts().IncrementClickCount()
	})

	r.hotspotTooltipArea.SetText(fmt.Sprintf("Clicks: %d", model.RichTexts().ClickCount()))

	r.form.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &r.selectableText,
			SecondaryWidget: &r.selectableToggle,
		},
		{
			PrimaryWidget:   &r.editableText,
			SecondaryWidget: &r.editableToggle,
		},
		{
			PrimaryWidget:   &r.resetText,
			SecondaryWidget: &r.resetButton,
		},
		{
			PrimaryWidget:   &r.clickCountText,
			SecondaryWidget: &r.clickCountValueText,
		},
	})

	t := &r.sampleText
	t.SetMultiline(true)
	t.SetWrapMode(basicwidget.WrapModeNormal)
	t.SetSelectable(model.RichTexts().Selectable())
	t.SetEditable(model.RichTexts().Editable())

	// The ranged styles are applied once, at the first build and at a
	// reset; afterwards they follow the text through edits on their own,
	// so the sample value round-trips through the model without any style
	// re-application.
	value, inited := model.RichTexts().SampleText()
	if !inited {
		value = richTextsSampleText
		model.RichTexts().SetSampleText(value)
		t.SetValue(value)
		applyRichTextsSampleStyles(t)
	} else {
		t.SetValue(value)
	}
	t.OnValueChanged(func(context *guigui.Context, text string, committed bool) {
		model.RichTexts().SetSampleText(text)
	})

	return nil
}

// applyRichTextsSampleStyles applies the ranged styles for the pristine
// sample text. The current value must be richTextsSampleText, as the byte
// ranges refer to it.
func applyRichTextsSampleStyles(t *basicwidget.Text) {
	red := color.RGBA{R: 0xff, G: 0x4b, B: 0x00, A: 0xff}
	blue := color.RGBA{R: 0x00, G: 0x5a, B: 0xff, A: 0xff}
	yellow := color.NRGBA{R: 0xff, G: 0xf1, B: 0x00, A: 0x60}
	green := color.NRGBA{R: 0x03, G: 0xaf, B: 0x7a, A: 0x50}

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
		t.SetHotspotRanges([]basicwidget.TextRange{
			{
				StartInBytes: start,
				EndInBytes:   end,
			},
		})
	})
	styleRichTextsSampleRange("下線", func(start, end int) {
		t.SetUnderlineInRange(start, end, true)
	})
	styleRichTextsSampleRange("背景", func(start, end int) {
		t.SetBackgroundColorInRange(start, end, yellow)
	})
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
	layout := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     r.layoutItems,
		Gap:       u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}
	layout.LayoutWidgets(context, widgetBounds.Bounds(), layouter)

	// Cover the sample text with the tooltip area, and restrict its hit areas to the
	// hotspot ranges' rectangles so that only the clickable range shows the tooltip.
	// The hotspots are inert while the sample is editable, so no tooltip either.
	sampleTextBounds := layout.ItemBoundsAt(0, context, widgetBounds.Bounds())
	r.hotspotBoundsArr = r.hotspotBoundsArr[:0]
	var editable bool
	if v, ok := context.Env(r, modelKeyModel); ok {
		editable = v.(*Model).RichTexts().Editable()
	}
	if !editable {
		r.hotspotRanges = r.sampleText.AppendHotspotRanges(r.hotspotRanges[:0])
		for _, hr := range r.hotspotRanges {
			r.hotspotBoundsArr = r.sampleText.AppendBoundsOfTextRange(r.hotspotBoundsArr, context, sampleTextBounds, hr.StartInBytes, hr.EndInBytes)
		}
	}
	r.hotspotTooltipArea.SetHitAreas(r.hotspotBoundsArr)
	layouter.LayoutWidget(&r.hotspotTooltipArea, sampleTextBounds)
}
