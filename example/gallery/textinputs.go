// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"image"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type TextInputs struct {
	guigui.DefaultWidget

	textInputForm               basicwidget.Form
	singleLineText              basicwidget.Text
	singleLineTextInput         guigui.WidgetWithSize[*basicwidget.TextInput]
	singleLineWithIconText      basicwidget.Text
	singleLineWithIconTextInput guigui.WidgetWithSize[*basicwidget.TextInput]
	multilineText               basicwidget.Text
	multilineTextInput          guigui.WidgetWithSize[*basicwidget.TextInput]
	inlineText                  basicwidget.Text
	inlineTextInput             guigui.WidgetWithSize[*inlineTextInputContainer]

	configForm                      basicwidget.Form
	horizontalAlignText             basicwidget.Text
	horizontalAlignSegmentedControl basicwidget.SegmentedControl[basicwidget.HorizontalAlign]
	verticalAlignText               basicwidget.Text
	verticalAlignSegmentedControl   basicwidget.SegmentedControl[basicwidget.VerticalAlign]
	autoWrapText                    basicwidget.Text
	autoWrapToggle                  basicwidget.Toggle
	editableText                    basicwidget.Text
	editableToggle                  basicwidget.Toggle
	enabledText                     basicwidget.Text
	enabledToggle                   basicwidget.Toggle
}

func (t *TextInputs) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.textInputForm)
	adder.AddWidget(&t.configForm)

	model := context.Data(t, modelKeyModel).(*Model)

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
	imgSearch, err := theImageCache.GetMonochrome("search", context.ColorMode())
	if err != nil {
		return err
	}

	u := basicwidget.UnitSize(context)

	// Text Inputs
	width := 12 * u

	t.singleLineText.SetValue("Single line")
	t.singleLineTextInput.Widget().OnValueChanged(func(context *guigui.Context, text string, committed bool) {
		if committed {
			model.TextInputs().SetSingleLineText(text)
		}
	})
	t.singleLineTextInput.Widget().SetValue(model.TextInputs().SingleLineText())
	t.singleLineTextInput.Widget().SetHorizontalAlign(model.TextInputs().HorizontalAlign())
	t.singleLineTextInput.Widget().SetVerticalAlign(model.TextInputs().VerticalAlign())
	t.singleLineTextInput.Widget().SetEditable(model.TextInputs().Editable())
	context.SetEnabled(&t.singleLineTextInput, model.TextInputs().Enabled())
	t.singleLineTextInput.SetFixedWidth(width)

	t.singleLineWithIconText.SetValue("Single line with icon")
	t.singleLineWithIconTextInput.Widget().SetHorizontalAlign(model.TextInputs().HorizontalAlign())
	t.singleLineWithIconTextInput.Widget().SetVerticalAlign(model.TextInputs().VerticalAlign())
	t.singleLineWithIconTextInput.Widget().SetEditable(model.TextInputs().Editable())
	t.singleLineWithIconTextInput.Widget().SetIcon(imgSearch)
	context.SetEnabled(&t.singleLineWithIconTextInput, model.TextInputs().Enabled())
	t.singleLineWithIconTextInput.SetFixedWidth(width)

	t.multilineText.SetValue("Multiline")
	t.multilineTextInput.Widget().OnValueChanged(func(context *guigui.Context, text string, committed bool) {
		if committed {
			model.TextInputs().SetMultilineText(text)
		}
	})
	t.multilineTextInput.Widget().SetValue(model.TextInputs().MultilineText())
	t.multilineTextInput.Widget().SetMultiline(true)
	t.multilineTextInput.Widget().SetHorizontalAlign(model.TextInputs().HorizontalAlign())
	t.multilineTextInput.Widget().SetVerticalAlign(model.TextInputs().VerticalAlign())
	t.multilineTextInput.Widget().SetAutoWrap(model.TextInputs().AutoWrap())
	t.multilineTextInput.Widget().SetEditable(model.TextInputs().Editable())
	context.SetEnabled(&t.multilineTextInput, model.TextInputs().Enabled())
	t.multilineTextInput.SetFixedSize(image.Pt(width, 4*u))

	t.inlineText.SetValue("Inline")
	t.inlineTextInput.Widget().SetHorizontalAlign(model.TextInputs().HorizontalAlign())
	t.inlineTextInput.Widget().textInput.SetVerticalAlign(model.TextInputs().VerticalAlign())
	t.inlineTextInput.Widget().textInput.SetEditable(model.TextInputs().Editable())
	context.SetEnabled(&t.inlineTextInput, model.TextInputs().Enabled())
	t.inlineTextInput.SetFixedWidth(width)

	t.textInputForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &t.singleLineText,
			SecondaryWidget: &t.singleLineTextInput,
		},
		{
			PrimaryWidget:   &t.singleLineWithIconText,
			SecondaryWidget: &t.singleLineWithIconTextInput,
		},
		{
			PrimaryWidget:   &t.multilineText,
			SecondaryWidget: &t.multilineTextInput,
		},
		{
			PrimaryWidget:   &t.inlineText,
			SecondaryWidget: &t.inlineTextInput,
		},
	})

	// Configurations
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
	t.horizontalAlignSegmentedControl.OnItemSelected(func(context *guigui.Context, index int) {
		item, ok := t.horizontalAlignSegmentedControl.ItemByIndex(index)
		if !ok {
			model.TextInputs().SetHorizontalAlign(basicwidget.HorizontalAlignStart)
			return
		}
		model.TextInputs().SetHorizontalAlign(item.Value)
	})
	t.horizontalAlignSegmentedControl.SelectItemByValue(model.TextInputs().HorizontalAlign())

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
	t.verticalAlignSegmentedControl.OnItemSelected(func(context *guigui.Context, index int) {
		item, ok := t.verticalAlignSegmentedControl.ItemByIndex(index)
		if !ok {
			model.TextInputs().SetVerticalAlign(basicwidget.VerticalAlignTop)
			return
		}
		model.TextInputs().SetVerticalAlign(item.Value)
	})
	t.verticalAlignSegmentedControl.SelectItemByValue(model.TextInputs().VerticalAlign())

	t.autoWrapText.SetValue("Auto wrap")
	t.autoWrapToggle.OnValueChanged(func(context *guigui.Context, value bool) {
		model.TextInputs().SetAutoWrap(value)
	})
	t.autoWrapToggle.SetValue(model.TextInputs().AutoWrap())

	t.editableText.SetValue("Editable")
	t.editableToggle.OnValueChanged(func(context *guigui.Context, value bool) {
		model.TextInputs().SetEditable(value)
	})
	t.editableToggle.SetValue(model.TextInputs().Editable())

	t.enabledText.SetValue("Enabled")
	t.enabledToggle.OnValueChanged(func(context *guigui.Context, value bool) {
		model.TextInputs().SetEnabled(value)
	})
	t.enabledToggle.SetValue(model.TextInputs().Enabled())

	t.configForm.SetItems([]basicwidget.FormItem{
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
			PrimaryWidget:   &t.editableText,
			SecondaryWidget: &t.editableToggle,
		},
		{
			PrimaryWidget:   &t.enabledText,
			SecondaryWidget: &t.enabledToggle,
		},
	})

	return nil
}

func (t *TextInputs) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &t.textInputForm,
			},
			{
				Size: guigui.FlexibleSize(1),
			},
			{
				Widget: &t.configForm,
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

type inlineTextInputContainer struct {
	guigui.DefaultWidget

	textInput       basicwidget.TextInput
	horizontalAlign basicwidget.HorizontalAlign
}

func (c *inlineTextInputContainer) SetHorizontalAlign(align basicwidget.HorizontalAlign) {
	c.horizontalAlign = align
	c.textInput.SetHorizontalAlign(align)
}

func (c *inlineTextInputContainer) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&c.textInput)
	c.textInput.SetStyle(basicwidget.TextInputStyleInline)
	return nil
}

func (c *inlineTextInputContainer) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	size := c.textInput.Measure(context, guigui.Constraints{})
	if size.X > widgetBounds.Bounds().Dx() {
		size = c.textInput.Measure(context, guigui.FixedHeightConstraints(widgetBounds.Bounds().Dx()))
	}
	pos := widgetBounds.Bounds().Min
	switch c.horizontalAlign {
	case basicwidget.HorizontalAlignStart:
	case basicwidget.HorizontalAlignCenter:
		pos.X += (widgetBounds.Bounds().Dx() - size.X) / 2
	case basicwidget.HorizontalAlignEnd:
		pos.X += widgetBounds.Bounds().Dx() - size.X
	}
	layouter.LayoutWidget(&c.textInput, image.Rectangle{
		Min: pos,
		Max: pos.Add(size),
	})
}

func (c *inlineTextInputContainer) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	c.textInput.SetStyle(basicwidget.TextInputStyleInline)
	return c.textInput.Measure(context, constraints)
}
