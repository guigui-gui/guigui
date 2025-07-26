// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	"github.com/hajimehoshi/guigui/layout"
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

func (t *Texts) AppendChildWidgets(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
	appender.AppendChildWidget(&t.sampleText)
	appender.AppendChildWidget(&t.form)
}

func (t *Texts) Build(context *guigui.Context) error {
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
			Icon: imgAlignStart,
			ID:   basicwidget.HorizontalAlignStart,
		},
		{
			Icon: imgAlignCenter,
			ID:   basicwidget.HorizontalAlignCenter,
		},
		{
			Icon: imgAlignEnd,
			ID:   basicwidget.HorizontalAlignEnd,
		},
	})
	t.horizontalAlignSegmentedControl.SetOnItemSelected(func(index int) {
		item, ok := t.horizontalAlignSegmentedControl.ItemByIndex(index)
		if !ok {
			model.Texts().SetHorizontalAlign(basicwidget.HorizontalAlignStart)
			return
		}
		model.Texts().SetHorizontalAlign(item.ID)
	})
	t.horizontalAlignSegmentedControl.SelectItemByID(model.Texts().HorizontalAlign())

	t.verticalAlignText.SetValue("Vertical align")
	t.verticalAlignSegmentedControl.SetItems([]basicwidget.SegmentedControlItem[basicwidget.VerticalAlign]{
		{
			Icon: imgAlignTop,
			ID:   basicwidget.VerticalAlignTop,
		},
		{
			Icon: imgAlignMiddle,
			ID:   basicwidget.VerticalAlignMiddle,
		},
		{
			Icon: imgAlignBottom,
			ID:   basicwidget.VerticalAlignBottom,
		},
	})
	t.verticalAlignSegmentedControl.SetOnItemSelected(func(index int) {
		item, ok := t.verticalAlignSegmentedControl.ItemByIndex(index)
		if !ok {
			model.Texts().SetVerticalAlign(basicwidget.VerticalAlignTop)
			return
		}
		model.Texts().SetVerticalAlign(item.ID)
	})
	t.verticalAlignSegmentedControl.SelectItemByID(model.Texts().VerticalAlign())

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

	u := basicwidget.UnitSize(context)
	gl := layout.GridLayout{
		Bounds: context.Bounds(t).Inset(u / 2),
		Heights: []layout.Size{
			layout.FlexibleSize(1),
			layout.FixedSize(t.form.DefaultSizeInContainer(context, context.Bounds(t).Dx()-u).Y),
		},
		RowGap: u / 2,
	}
	context.SetBounds(&t.sampleText, gl.CellBounds(0, 0), t)
	context.SetBounds(&t.form, gl.CellBounds(0, 1), t)

	return nil
}
