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
	guigui.RegisterEventHandler2(t, &t.horizontalAlignSegmentedControl)
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
	guigui.RegisterEventHandler2(t, &t.verticalAlignSegmentedControl)
	t.verticalAlignSegmentedControl.SelectItemByValue(model.Texts().VerticalAlign())

	t.autoWrapText.SetValue("Auto wrap")
	guigui.RegisterEventHandler2(t, &t.autoWrapToggle)
	t.autoWrapToggle.SetValue(model.Texts().AutoWrap())

	t.boldText.SetValue("Bold")
	guigui.RegisterEventHandler2(t, &t.boldToggle)
	t.boldToggle.SetValue(model.Texts().Bold())

	t.selectableText.SetValue("Selectable")
	guigui.RegisterEventHandler2(t, &t.selectableToggle)
	t.selectableToggle.SetValue(model.Texts().Selectable())

	t.editableText.SetValue("Editable")
	guigui.RegisterEventHandler2(t, &t.editableToggle)
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
	guigui.RegisterEventHandler2(t, &t.sampleText)

	t.sampleText.SetValue(model.Texts().Text())

	return nil
}

func (t *Texts) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
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

func (t *Texts) HandleEvent(context *guigui.Context, targetWidget guigui.Widget, eventArgs any) {
	model := context.Model(t, modelKeyModel).(*Model)
	switch targetWidget {
	case &t.horizontalAlignSegmentedControl:
		switch eventArgs := eventArgs.(type) {
		case *basicwidget.SegmentedControlEventArgsItemSelected:
			item, ok := t.horizontalAlignSegmentedControl.ItemByIndex(eventArgs.Index)
			if !ok {
				model.Texts().SetHorizontalAlign(basicwidget.HorizontalAlignStart)
				return
			}
			model.Texts().SetHorizontalAlign(item.Value)
		}
	case &t.verticalAlignSegmentedControl:
		switch eventArgs := eventArgs.(type) {
		case *basicwidget.SegmentedControlEventArgsItemSelected:
			item, ok := t.verticalAlignSegmentedControl.ItemByIndex(eventArgs.Index)
			if !ok {
				model.Texts().SetVerticalAlign(basicwidget.VerticalAlignTop)
				return
			}
			model.Texts().SetVerticalAlign(item.Value)
		}
	case &t.sampleText:
		switch eventArgs := eventArgs.(type) {
		case *basicwidget.TextEventArgsValueChanged:
			if eventArgs.Committed {
				model.Texts().SetText(eventArgs.Value)
			}
		case *basicwidget.TextEventArgsKeyJustPressed:
			if !t.sampleText.IsEditable() {
				return
			}
			if eventArgs.Key == ebiten.KeyTab {
				t.sampleText.ReplaceValueAtSelection("\t")
			}
		}
	case &t.autoWrapToggle:
		switch eventArgs := eventArgs.(type) {
		case *basicwidget.ToggleEventArgsValueChanged:
			model.Texts().SetAutoWrap(eventArgs.Value)
		}
	case &t.boldToggle:
		switch eventArgs := eventArgs.(type) {
		case *basicwidget.ToggleEventArgsValueChanged:
			model.Texts().SetBold(eventArgs.Value)
		}
	case &t.selectableToggle:
		switch eventArgs := eventArgs.(type) {
		case *basicwidget.ToggleEventArgsValueChanged:
			model.Texts().SetSelectable(eventArgs.Value)
		}
	case &t.editableToggle:
		switch eventArgs := eventArgs.(type) {
		case *basicwidget.ToggleEventArgsValueChanged:
			model.Texts().SetEditable(eventArgs.Value)
		}
	}
}
