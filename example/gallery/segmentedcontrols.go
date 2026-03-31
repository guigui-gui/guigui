// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type SegmentedControls struct {
	guigui.DefaultWidget

	segmentedControlsForm basicwidget.Form
	segmentedControlHText basicwidget.Text
	segmentedControlH     basicwidget.SegmentedControl[int]
	segmentedControlVText basicwidget.Text
	segmentedControlV     basicwidget.SegmentedControl[int]
	segmentedControlMText basicwidget.Text
	segmentedControlM     basicwidget.SegmentedControl[int]

	configForm    basicwidget.Form
	enabledText   basicwidget.Text
	enabledToggle basicwidget.Toggle

	layoutItems []guigui.LinearLayoutItem
}

func (s *SegmentedControls) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&s.segmentedControlsForm)
	adder.AddWidget(&s.configForm)

	v, ok := context.Env(s, modelKeyModel)
	if !ok {
		return nil
	}
	model := v.(*Model)

	s.segmentedControlHText.SetValue("Segmented control (Horizontal)")
	s.segmentedControlH.SetItems([]basicwidget.SegmentedControlItem[int]{
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
	s.segmentedControlH.SetDirection(basicwidget.SegmentedControlDirectionHorizontal)
	context.SetEnabled(&s.segmentedControlH, model.SegmentedControls().Enabled())

	s.segmentedControlVText.SetValue("Segmented control (Vertical)")
	s.segmentedControlV.SetItems([]basicwidget.SegmentedControlItem[int]{
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
	s.segmentedControlV.SetDirection(basicwidget.SegmentedControlDirectionVertical)
	context.SetEnabled(&s.segmentedControlV, model.SegmentedControls().Enabled())

	s.segmentedControlMText.SetValue("Segmented control (Multi-selection)")
	s.segmentedControlM.SetMultiSelection(true)
	s.segmentedControlM.SetItems([]basicwidget.SegmentedControlItem[int]{
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
	s.segmentedControlM.SetDirection(basicwidget.SegmentedControlDirectionHorizontal)
	context.SetEnabled(&s.segmentedControlM, model.SegmentedControls().Enabled())

	s.segmentedControlsForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &s.segmentedControlHText,
			SecondaryWidget: &s.segmentedControlH,
		},
		{
			PrimaryWidget:   &s.segmentedControlVText,
			SecondaryWidget: &s.segmentedControlV,
		},
		{
			PrimaryWidget:   &s.segmentedControlMText,
			SecondaryWidget: &s.segmentedControlM,
		},
	})

	s.enabledText.SetValue("Enabled")
	s.enabledToggle.OnValueChanged(func(context *guigui.Context, enabled bool) {
		model.SegmentedControls().SetEnabled(enabled)
	})
	s.enabledToggle.SetValue(model.SegmentedControls().Enabled())

	s.configForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &s.enabledText,
			SecondaryWidget: &s.enabledToggle,
		},
	})

	return nil
}

func (s *SegmentedControls) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	s.layoutItems = slices.Delete(s.layoutItems, 0, len(s.layoutItems))
	s.layoutItems = append(s.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &s.segmentedControlsForm,
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
