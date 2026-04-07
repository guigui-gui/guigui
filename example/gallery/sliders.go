// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Sliders struct {
	guigui.DefaultWidget

	sliderForm                       basicwidget.Form
	sliderText                       basicwidget.Text
	slider                           guigui.WidgetWithSize[*basicwidget.Slider]
	sliderWithoutRangeText           basicwidget.Text
	sliderWithoutRange               guigui.WidgetWithSize[*basicwidget.Slider]
	sliderWithSnapsText              basicwidget.Text
	sliderWithSnaps                  guigui.WidgetWithSize[*basicwidget.Slider]
	sliderWithSnapsNoRestrictionText basicwidget.Text
	sliderWithSnapsNoRestriction     guigui.WidgetWithSize[*basicwidget.Slider]

	configForm    basicwidget.Form
	enabledText   basicwidget.Text
	enabledToggle basicwidget.Toggle

	layoutItems []guigui.LinearLayoutItem
}

func (s *Sliders) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&s.sliderForm)
	adder.AddWidget(&s.configForm)

	v, ok := context.Env(s, modelKeyModel)
	if !ok {
		return nil
	}
	model := v.(*Model)

	u := basicwidget.UnitSize(context)
	width := 12 * u

	s.sliderText.SetValue("Slider (Range: [-100, 100])")
	s.slider.Widget().OnValueChanged(func(context *guigui.Context, value int) {
		model.Sliders().SetSliderValue(value)
	})
	s.slider.Widget().SetMinimumValue(-100)
	s.slider.Widget().SetMaximumValue(100)
	s.slider.Widget().SetValue(model.Sliders().SliderValue())
	context.SetEnabled(&s.slider, model.Sliders().Enabled())
	s.slider.SetFixedWidth(width)

	s.sliderWithoutRangeText.SetValue("Slider w/o range")
	context.SetEnabled(&s.sliderWithoutRange, model.Sliders().Enabled())
	s.sliderWithoutRange.SetFixedWidth(width)

	s.sliderWithSnapsText.SetValue("Slider with snaps (Range: [-100, 100], Step: 10)")
	s.sliderWithSnaps.Widget().OnValueChanged(func(context *guigui.Context, value int) {
		model.Sliders().SetSliderValue(value)
	})
	s.sliderWithSnaps.Widget().SetMinimumValue(-100)
	s.sliderWithSnaps.Widget().SetMaximumValue(100)
	s.sliderWithSnaps.Widget().SetStep(10)
	s.sliderWithSnaps.Widget().SetSnapOnly(true)
	s.sliderWithSnaps.Widget().SetValue(model.Sliders().SliderValue())
	context.SetEnabled(&s.sliderWithSnaps, model.Sliders().Enabled())
	s.sliderWithSnaps.SetFixedWidth(width)

	s.sliderWithSnapsNoRestrictionText.SetValue("Slider with snaps, no restriction (Range: [-100, 100], Step: 10)")
	s.sliderWithSnapsNoRestriction.Widget().OnValueChanged(func(context *guigui.Context, value int) {
		model.Sliders().SetSliderValue(value)
	})
	s.sliderWithSnapsNoRestriction.Widget().SetMinimumValue(-100)
	s.sliderWithSnapsNoRestriction.Widget().SetMaximumValue(100)
	s.sliderWithSnapsNoRestriction.Widget().SetStep(10)
	s.sliderWithSnapsNoRestriction.Widget().SetValue(model.Sliders().SliderValue())
	context.SetEnabled(&s.sliderWithSnapsNoRestriction, model.Sliders().Enabled())
	s.sliderWithSnapsNoRestriction.SetFixedWidth(width)

	s.sliderForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &s.sliderText,
			SecondaryWidget: &s.slider,
		},
		{
			PrimaryWidget:   &s.sliderWithoutRangeText,
			SecondaryWidget: &s.sliderWithoutRange,
		},
		{
			PrimaryWidget:   &s.sliderWithSnapsText,
			SecondaryWidget: &s.sliderWithSnaps,
		},
		{
			PrimaryWidget:   &s.sliderWithSnapsNoRestrictionText,
			SecondaryWidget: &s.sliderWithSnapsNoRestriction,
		},
	})

	// Configurations
	s.enabledText.SetValue("Enabled")
	s.enabledToggle.OnValueChanged(func(context *guigui.Context, value bool) {
		model.Sliders().SetEnabled(value)
	})
	s.enabledToggle.SetValue(model.Sliders().Enabled())

	s.configForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &s.enabledText,
			SecondaryWidget: &s.enabledToggle,
		},
	})

	return nil
}

func (s *Sliders) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	s.layoutItems = slices.Delete(s.layoutItems, 0, len(s.layoutItems))
	s.layoutItems = append(s.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &s.sliderForm,
		},
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: &s.configForm,
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     s.layoutItems,
		Gap:       u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}
