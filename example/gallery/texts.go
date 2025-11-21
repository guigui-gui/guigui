// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Texts struct {
	guigui.DefaultWidget

	form                            basicwidget.Form
	horizontalAlignText             basicwidget.Text
	horizontalAlignSegmentedControl basicwidget.SegmentedControl[basicwidget.HorizontalAlign]
	verticalAlignText               basicwidget.Text
	verticalAlignSegmentedControl   basicwidget.SegmentedControl[basicwidget.VerticalAlign]
	autoWrapText                    basicwidget.Text
	autoWrapToggle                  basicwidget.Toggle
	boldText                        basicwidget.Text
	boldToggle                      basicwidget.Toggle
	selectableText                  basicwidget.Text
	selectableToggle                basicwidget.Toggle
	editableText                    basicwidget.Text
	editableToggle                  basicwidget.Toggle
	sampleText                      basicwidget.Text
}

func (t *Texts) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&t.sampleText)
	adder.AddChild(&t.form)

	model := context.Model(t, modelKeyModel).(*Model)

	imgAlignStart, err := theImageCache.GetMonochrome("format_align_left", context.ColorMode())
	if err != nil {
		return err
	}
	imgAlignCenter, err := theImageCache.GetMonochrome("format_align_center", context.ColorMode())
	if err != nil {
		return err
	}
	imgAlignEnd, err := theImageCache.GetMonochrome("format_align_right", context.ColorMode())
	if err != nil {
		return err
	}
	imgAlignTop, err := theImageCache.GetMonochrome("vertical_align_top", context.ColorMode())
	if err != nil {
		return err
	}
	imgAlignMiddle, err := theImageCache.GetMonochrome("vertical_align_center", context.ColorMode())
	if err != nil {
		return err
	}
	imgAlignBottom, err := theImageCache.GetMonochrome("vertical_align_bottom", context.ColorMode())
	if err != nil {
		return err
	}

	t.horizontalAlignText.SetValue("Horizontal align")
	t.horizontalAlignSegmentedControl.SetItems([]basicwidget.SegmentedControlItem[basicwidget.HorizontalAlign]{
		{
			Icon:  imgAlignStart,
			Value: basicwidget.HorizontalAlignStart,
		},
		{
			Icon:  imgAlignCenter,
			Value: basicwidget.HorizontalAlignCenter,
		},
		{
			Icon:  imgAlignEnd,
			Value: basicwidget.HorizontalAlignEnd,
		},
	})
	t.horizontalAlignSegmentedControl.SetOnItemSelected(func(index int) {
		item, ok := t.horizontalAlignSegmentedControl.ItemByIndex(index)
		if !ok {
			model.Texts().SetHorizontalAlign(basicwidget.HorizontalAlignStart)
			return
		}
		model.Texts().SetHorizontalAlign(item.Value)
	})
	t.horizontalAlignSegmentedControl.SelectItemByValue(model.Texts().HorizontalAlign())

	t.verticalAlignText.SetValue("Vertical align")
	t.verticalAlignSegmentedControl.SetItems([]basicwidget.SegmentedControlItem[basicwidget.VerticalAlign]{
		{
			Icon:  imgAlignTop,
			Value: basicwidget.VerticalAlignTop,
		},
		{
			Icon:  imgAlignMiddle,
			Value: basicwidget.VerticalAlignMiddle,
		},
		{
			Icon:  imgAlignBottom,
			Value: basicwidget.VerticalAlignBottom,
		},
	})
	t.verticalAlignSegmentedControl.SetOnItemSelected(func(index int) {
		item, ok := t.verticalAlignSegmentedControl.ItemByIndex(index)
		if !ok {
			model.Texts().SetVerticalAlign(basicwidget.VerticalAlignTop)
			return
		}
		model.Texts().SetVerticalAlign(item.Value)
	})
	t.verticalAlignSegmentedControl.SelectItemByValue(model.Texts().VerticalAlign())

	t.autoWrapText.SetValue("Auto wrap")
	t.autoWrapToggle.SetOnValueChanged(func(value bool) {
		model.Texts().SetAutoWrap(value)
	})
	t.autoWrapToggle.SetValue(model.Texts().AutoWrap())

	t.boldText.SetValue("Bold")
	t.boldToggle.SetOnValueChanged(func(value bool) {
		model.Texts().SetBold(value)
	})
	t.boldToggle.SetValue(model.Texts().Bold())

	t.selectableText.SetValue("Selectable")
	t.selectableToggle.SetOnValueChanged(func(checked bool) {
		model.Texts().SetSelectable(checked)
	})
	t.selectableToggle.SetValue(model.Texts().Selectable())

	t.editableText.SetValue("Editable")
	t.editableToggle.SetOnValueChanged(func(value bool) {
		model.Texts().SetEditable(value)
	})
	t.editableToggle.SetValue(model.Texts().Editable())

	t.form.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &t.horizontalAlignText,
			SecondaryWidget: &t.horizontalAlignSegmentedControl,
		},
		{
			PrimaryWidget:   &t.verticalAlignText,
			SecondaryWidget: &t.verticalAlignSegmentedControl,
		},
		{
			PrimaryWidget:   &t.autoWrapText,
			SecondaryWidget: &t.autoWrapToggle,
		},
		{
			PrimaryWidget:   &t.boldText,
			SecondaryWidget: &t.boldToggle,
		},
		{
			PrimaryWidget:   &t.selectableText,
			SecondaryWidget: &t.selectableToggle,
		},
		{
			PrimaryWidget:   &t.editableText,
			SecondaryWidget: &t.editableToggle,
		},
	})

	t.sampleText.SetMultiline(true)
	t.sampleText.SetHorizontalAlign(model.Texts().HorizontalAlign())
	t.sampleText.SetVerticalAlign(model.Texts().VerticalAlign())
	t.sampleText.SetAutoWrap(model.Texts().AutoWrap())
	t.sampleText.SetBold(model.Texts().Bold())
	t.sampleText.SetSelectable(model.Texts().Selectable())
	t.sampleText.SetEditable(model.Texts().Editable())
	t.sampleText.SetOnValueChanged(func(text string, committed bool) {
		if committed {
			model.Texts().SetText(text)
		}
	})
	t.sampleText.SetOnKeyJustPressed(func(key ebiten.Key) bool {
		if !t.sampleText.IsEditable() {
			return false
		}
		if key == ebiten.KeyTab {
			t.sampleText.ReplaceValueAtSelection("\t")
			return true
		}
		return false
	})
	t.sampleText.SetValue(model.Texts().Text())

	return nil
}

func (t *Texts) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &t.sampleText,
				Size:   guigui.FlexibleSize(1),
			},
			{
				Widget: &t.form,
			},
		},
		Gap: u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}
