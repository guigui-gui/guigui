// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"image"
	"slices"
	"unicode/utf8"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type TextInputs struct {
	guigui.DefaultWidget

	textInputForm               basicwidget.Form
	singleLineText              basicwidget.Text
	singleLineTextInput         guigui.WidgetWithSize[*textInputContainer]
	errorText                   basicwidget.Text
	errorTextInput              guigui.WidgetWithSize[*textInputContainer]
	singleLineWithIconText      basicwidget.Text
	singleLineWithIconTextInput guigui.WidgetWithSize[*textInputContainer]
	multilineText               basicwidget.Text
	multilineTextInput          guigui.WidgetWithSize[*textInputContainer]
	inlineText                  basicwidget.Text
	inlineTextInput             guigui.WidgetWithSize[*textInputContainer]

	configForm                      basicwidget.Form
	horizontalAlignText             basicwidget.Text
	horizontalAlignSegmentedControl basicwidget.SegmentedControl[basicwidget.HorizontalAlign]
	verticalAlignText               basicwidget.Text
	verticalAlignSegmentedControl   basicwidget.SegmentedControl[basicwidget.VerticalAlign]
	autoWrapText                    basicwidget.Text
	autoWrapToggle                  basicwidget.Toggle
	cursorBlinkingText              basicwidget.Text
	cursorBlinkingToggle            basicwidget.Toggle
	editableText                    basicwidget.Text
	editableToggle                  basicwidget.Toggle
	enabledText                     basicwidget.Text
	enabledToggle                   basicwidget.Toggle

	layoutItems []guigui.LinearLayoutItem
}

func (t *TextInputs) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.textInputForm)
	adder.AddWidget(&t.configForm)

	v, ok := context.Env(t, modelKeyModel)
	if !ok {
		return nil
	}
	model := v.(*Model)

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
	t.singleLineTextInput.Widget().TextInput().OnValueChanged(func(context *guigui.Context, text string, committed bool) {
		if committed {
			model.TextInputs().SetSingleLineText(text)
		}
	})
	t.singleLineTextInput.Widget().TextInput().SetValue(model.TextInputs().SingleLineText())
	t.singleLineTextInput.Widget().TextInput().SetHorizontalAlign(model.TextInputs().HorizontalAlign())
	t.singleLineTextInput.Widget().TextInput().SetVerticalAlign(model.TextInputs().VerticalAlign())
	t.singleLineTextInput.Widget().TextInput().SetEditable(model.TextInputs().Editable())
	t.singleLineTextInput.Widget().TextInput().SetCursorBlinking(model.TextInputs().CursorBlinking())
	t.singleLineTextInput.Widget().SetContextMenu(true)
	context.SetEnabled(&t.singleLineTextInput, model.TextInputs().Enabled())
	t.singleLineTextInput.SetFixedWidth(width)

	t.errorText.SetValue("With validation")
	t.errorTextInput.Widget().TextInput().OnValueChanged(func(context *guigui.Context, text string, committed bool) {
		n := utf8.RuneCountInString(text)
		if n < 3 || n > 5 {
			t.errorTextInput.Widget().TextInput().SetError(true)
			t.errorTextInput.Widget().TextInput().SetSupportText("Must be between 3 and 5 characters")
		} else {
			t.errorTextInput.Widget().TextInput().SetError(false)
			t.errorTextInput.Widget().TextInput().SetSupportText("")
		}
	})
	t.errorTextInput.Widget().TextInput().SetHorizontalAlign(model.TextInputs().HorizontalAlign())
	t.errorTextInput.Widget().TextInput().SetVerticalAlign(model.TextInputs().VerticalAlign())
	t.errorTextInput.Widget().TextInput().SetEditable(model.TextInputs().Editable())
	t.errorTextInput.Widget().TextInput().SetCursorBlinking(model.TextInputs().CursorBlinking())
	t.errorTextInput.Widget().SetContextMenu(true)
	context.SetEnabled(&t.errorTextInput, model.TextInputs().Enabled())
	t.errorTextInput.SetFixedWidth(width)

	t.singleLineWithIconText.SetValue("Single line with icon")
	t.singleLineWithIconTextInput.Widget().TextInput().SetHorizontalAlign(model.TextInputs().HorizontalAlign())
	t.singleLineWithIconTextInput.Widget().TextInput().SetVerticalAlign(model.TextInputs().VerticalAlign())
	t.singleLineWithIconTextInput.Widget().TextInput().SetEditable(model.TextInputs().Editable())
	t.singleLineWithIconTextInput.Widget().TextInput().SetCursorBlinking(model.TextInputs().CursorBlinking())
	t.singleLineWithIconTextInput.Widget().TextInput().SetIcon(imgSearch)
	t.singleLineWithIconTextInput.Widget().SetContextMenu(true)
	context.SetEnabled(&t.singleLineWithIconTextInput, model.TextInputs().Enabled())
	t.singleLineWithIconTextInput.SetFixedWidth(width)

	t.multilineText.SetValue("Multiline")
	t.multilineTextInput.Widget().TextInput().OnValueChanged(func(context *guigui.Context, text string, committed bool) {
		if committed {
			model.TextInputs().SetMultilineText(text)
		}
	})
	t.multilineTextInput.Widget().TextInput().SetValue(model.TextInputs().MultilineText())
	t.multilineTextInput.Widget().TextInput().SetMultiline(true)
	t.multilineTextInput.Widget().TextInput().SetHorizontalAlign(model.TextInputs().HorizontalAlign())
	t.multilineTextInput.Widget().TextInput().SetVerticalAlign(model.TextInputs().VerticalAlign())
	t.multilineTextInput.Widget().TextInput().SetAutoWrap(model.TextInputs().AutoWrap())
	t.multilineTextInput.Widget().TextInput().SetEditable(model.TextInputs().Editable())
	t.multilineTextInput.Widget().TextInput().SetCursorBlinking(model.TextInputs().CursorBlinking())
	t.multilineTextInput.Widget().SetContextMenu(true)
	context.SetEnabled(&t.multilineTextInput, model.TextInputs().Enabled())
	t.multilineTextInput.SetFixedSize(image.Pt(width, 4*u))

	t.inlineText.SetValue("Inline")
	t.inlineTextInput.Widget().SetStyle(basicwidget.TextInputStyleInline)
	t.inlineTextInput.Widget().SetHorizontalAlign(model.TextInputs().HorizontalAlign())
	t.inlineTextInput.Widget().TextInput().SetVerticalAlign(model.TextInputs().VerticalAlign())
	t.inlineTextInput.Widget().TextInput().SetEditable(model.TextInputs().Editable())
	t.inlineTextInput.Widget().TextInput().SetCursorBlinking(model.TextInputs().CursorBlinking())
	t.inlineTextInput.Widget().SetContextMenu(true)
	context.SetEnabled(&t.inlineTextInput, model.TextInputs().Enabled())
	t.inlineTextInput.SetFixedWidth(width)

	t.textInputForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &t.singleLineText,
			SecondaryWidget: &t.singleLineTextInput,
		},
		{
			PrimaryWidget:   &t.errorText,
			SecondaryWidget: &t.errorTextInput,
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

	t.cursorBlinkingText.SetValue("Cursor blinking")
	t.cursorBlinkingToggle.OnValueChanged(func(context *guigui.Context, value bool) {
		model.TextInputs().SetCursorBlinking(value)
	})
	t.cursorBlinkingToggle.SetValue(model.TextInputs().CursorBlinking())

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
			PrimaryWidget:   &t.cursorBlinkingText,
			SecondaryWidget: &t.cursorBlinkingToggle,
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
	t.layoutItems = slices.Delete(t.layoutItems, 0, len(t.layoutItems))
	t.layoutItems = append(t.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &t.textInputForm,
		},
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: &t.configForm,
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     t.layoutItems,
		Gap:       u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

type textInputContainer struct {
	guigui.DefaultWidget

	textInput       basicwidget.TextInput
	contextMenuArea basicwidget.ContextMenuArea[int]

	style           basicwidget.TextInputStyle
	horizontalAlign basicwidget.HorizontalAlign
	showContextMenu bool
}

func (t *textInputContainer) TextInput() *basicwidget.TextInput {
	return &t.textInput
}

func (t *textInputContainer) SetStyle(style basicwidget.TextInputStyle) {
	t.style = style
}

func (t *textInputContainer) SetHorizontalAlign(align basicwidget.HorizontalAlign) {
	t.horizontalAlign = align
	t.textInput.SetHorizontalAlign(align)
}

func (t *textInputContainer) SetContextMenu(show bool) {
	t.showContextMenu = show
}

func (t *textInputContainer) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.textInput)
	t.textInput.SetStyle(t.style)
	if t.showContextMenu {
		adder.AddWidget(&t.contextMenuArea)
		t.contextMenuArea.PopupMenu().SetItems([]basicwidget.PopupMenuItem[int]{
			{
				Text:     "Cut",
				Value:    0,
				Disabled: !t.textInput.CanCut(),
			},
			{
				Text:     "Copy",
				Value:    1,
				Disabled: !t.textInput.CanCopy(),
			},
			{
				Text:     "Paste",
				Value:    2,
				Disabled: !t.textInput.CanPaste(),
			},
		})
		t.contextMenuArea.PopupMenu().OnItemSelected(func(context *guigui.Context, index int) {
			item, ok := t.contextMenuArea.PopupMenu().ItemByIndex(index)
			if !ok {
				return
			}
			switch item.Value {
			case 0:
				t.textInput.Cut()
			case 1:
				t.textInput.Copy()
			case 2:
				t.textInput.Paste()
			}
		})
	}
	return nil
}

func (t *textInputContainer) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	var b image.Rectangle
	if t.style == basicwidget.TextInputStyleInline {
		size := t.textInput.Measure(context, guigui.Constraints{})
		if size.X > widgetBounds.Bounds().Dx() {
			size = t.textInput.Measure(context, guigui.FixedHeightConstraints(widgetBounds.Bounds().Dx()))
		}
		pos := widgetBounds.Bounds().Min
		switch t.horizontalAlign {
		case basicwidget.HorizontalAlignStart:
		case basicwidget.HorizontalAlignCenter:
			pos.X += (widgetBounds.Bounds().Dx() - size.X) / 2
		case basicwidget.HorizontalAlignEnd:
			pos.X += widgetBounds.Bounds().Dx() - size.X
		}
		b = image.Rectangle{
			Min: pos,
			Max: pos.Add(size),
		}
	} else {
		b = widgetBounds.Bounds()
	}
	layouter.LayoutWidget(&t.textInput, b)
	if t.showContextMenu {
		layouter.LayoutWidget(&t.contextMenuArea, b)
	}
}

func (t *textInputContainer) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	t.textInput.SetStyle(t.style)
	return t.textInput.Measure(context, constraints)
}
