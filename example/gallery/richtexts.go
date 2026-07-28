// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"fmt"
	"image"
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

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
		// ForceSetValue synchronously dispatches the value-changed handler,
		// which writes the widget's cleared styles and hotspots back to the
		// model, so the model is reset afterwards and then reinstalled.
		r.sampleText.ForceSetValue(richTextsSampleText)
		model.RichTexts().ResetSampleText()
		r.sampleText.SetStyles(model.RichTexts().SampleTextStyles())
		r.sampleText.SetHotspotRanges(model.RichTexts().SampleTextHotspots())
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

	// The model owns the sample value, its ranged styles, and its hotspot
	// ranges. Each build restores them into the widget — SetValue comes
	// first, as a changed value clears the widget's followed styles and
	// hotspots before the model's are installed — and each change writes
	// the widget's followed state back to the model.
	t.SetValue(model.RichTexts().SampleText())
	t.SetStyles(model.RichTexts().SampleTextStyles())
	t.SetHotspotRanges(model.RichTexts().SampleTextHotspots())
	t.OnValueChanged(func(context *guigui.Context, text string, committed bool) {
		model.RichTexts().SetSampleText(text)
		model.RichTexts().SetSampleTextStyles(r.sampleText.Styles())
		model.RichTexts().SetSampleTextHotspots(r.sampleText.AppendHotspotRanges(nil))
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
