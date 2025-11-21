// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"image"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Buttons struct {
	guigui.DefaultWidget

	buttonsForm           basicwidget.Form
	buttonText            basicwidget.Text
	button                basicwidget.Button
	textIconButton1Text   basicwidget.Text
	textIconButton1       guigui.WidgetWithSize[*basicwidget.Button]
	textIconButton2Text   basicwidget.Text
	textIconButton2       guigui.WidgetWithSize[*basicwidget.Button]
	imageButtonText       basicwidget.Text
	imageButton           guigui.WidgetWithSize[*basicwidget.Button]
	segmentedControlHText basicwidget.Text
	segmentedControlH     basicwidget.SegmentedControl[int]
	segmentedControlVText basicwidget.Text
	segmentedControlV     basicwidget.SegmentedControl[int]
	toggleText            basicwidget.Text
	toggle                basicwidget.Toggle

	configForm    basicwidget.Form
	enabledText   basicwidget.Text
	enabledToggle basicwidget.Toggle
}

func (b *Buttons) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&b.buttonsForm)
	adder.AddChild(&b.configForm)

	model := context.Model(b, modelKeyModel).(*Model)

	u := basicwidget.UnitSize(context)

	b.buttonText.SetValue("Button")
	b.button.SetText("Button")
	context.SetEnabled(&b.button, model.Buttons().Enabled())

	b.textIconButton1Text.SetValue("Button w/ text and icon (1)")
	b.textIconButton1.Widget().SetText("Button")
	img, err := theImageCache.GetMonochrome("check", context.ColorMode())
	if err != nil {
		return err
	}
	b.textIconButton1.Widget().SetIcon(img)
	context.SetEnabled(&b.textIconButton1, model.Buttons().Enabled())
	b.textIconButton1.SetFixedWidth(6 * u)

	b.textIconButton2Text.SetValue("Button w/ text and icon (2)")
	b.textIconButton2.Widget().SetText("Button")
	b.textIconButton2.Widget().SetIcon(img)
	b.textIconButton2.Widget().SetIconAlign(basicwidget.IconAlignEnd)
	context.SetEnabled(&b.textIconButton2, model.Buttons().Enabled())
	b.textIconButton2.SetFixedWidth(6 * u)

	b.imageButtonText.SetValue("Image button")
	img, err = theImageCache.Get("gopher_center")
	if err != nil {
		return err
	}
	b.imageButton.Widget().SetIcon(img)
	context.SetEnabled(&b.imageButton, model.Buttons().Enabled())
	b.imageButton.SetFixedSize(image.Pt(2*u, 2*u))

	b.segmentedControlHText.SetValue("Segmented control (Horizontal)")
	b.segmentedControlH.SetItems([]basicwidget.SegmentedControlItem[int]{
		{
			Text: "One",
		},
		{
			Text: "Two",
		},
		{
			Text: "Three",
		},
	})
	b.segmentedControlH.SetDirection(basicwidget.SegmentedControlDirectionHorizontal)
	context.SetEnabled(&b.segmentedControlH, model.Buttons().Enabled())

	b.segmentedControlVText.SetValue("Segmented control (Vertical)")
	b.segmentedControlV.SetItems([]basicwidget.SegmentedControlItem[int]{
		{
			Text: "One",
		},
		{
			Text: "Two",
		},
		{
			Text: "Three",
		},
	})
	b.segmentedControlV.SetDirection(basicwidget.SegmentedControlDirectionVertical)
	context.SetEnabled(&b.segmentedControlV, model.Buttons().Enabled())

	b.toggleText.SetValue("Toggle")
	context.SetEnabled(&b.toggle, model.Buttons().Enabled())

	b.buttonsForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &b.buttonText,
			SecondaryWidget: &b.button,
		},
		{
			PrimaryWidget:   &b.textIconButton1Text,
			SecondaryWidget: &b.textIconButton1,
		},
		{
			PrimaryWidget:   &b.textIconButton2Text,
			SecondaryWidget: &b.textIconButton2,
		},
		{
			PrimaryWidget:   &b.imageButtonText,
			SecondaryWidget: &b.imageButton,
		},
		{
			PrimaryWidget:   &b.segmentedControlHText,
			SecondaryWidget: &b.segmentedControlH,
		},
		{
			PrimaryWidget:   &b.segmentedControlVText,
			SecondaryWidget: &b.segmentedControlV,
		},
		{
			PrimaryWidget:   &b.toggleText,
			SecondaryWidget: &b.toggle,
		},
	})

	b.enabledText.SetValue("Enabled")
	b.enabledToggle.SetOnValueChanged(func(enabled bool) {
		model.Buttons().SetEnabled(enabled)
	})
	b.enabledToggle.SetValue(model.Buttons().Enabled())

	b.configForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &b.enabledText,
			SecondaryWidget: &b.enabledToggle,
		},
	})

	return nil
}

func (b *Buttons) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &b.buttonsForm,
			},
			{
				Size: guigui.FlexibleSize(1),
			},
			{
				Widget: &b.configForm,
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
