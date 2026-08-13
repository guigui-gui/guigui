// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type ProgressBars struct {
	guigui.DefaultWidget

	progressBarFormPanel basicwidget.Panel
	progressBarForm      basicwidget.Form

	progressBarText             basicwidget.Text
	progressBar                 guigui.WidgetWithSize[*basicwidget.ProgressBar]
	progressBarWithoutRangeText basicwidget.Text
	progressBarWithoutRange     guigui.WidgetWithSize[*basicwidget.ProgressBar]
	valueText                   basicwidget.Text
	valueNumberInput            guigui.WidgetWithSize[*basicwidget.NumberInput]

	configForm    basicwidget.Form
	enabledText   basicwidget.Text
	enabledToggle basicwidget.Toggle

	layoutItems []guigui.LinearLayoutItem
}

func (p *ProgressBars) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&p.progressBarFormPanel)
	adder.AddWidget(&p.configForm)

	p.progressBarFormPanel.SetContent(&p.progressBarForm)
	p.progressBarFormPanel.SetAutoBorder(true)
	p.progressBarFormPanel.SetContentConstraints(basicwidget.PanelContentConstraintsFixedWidth)

	v, ok := context.Env(p, modelKeyModel)
	if !ok {
		return nil
	}
	model := v.(*Model)

	u := basicwidget.UnitSize(context)
	width := 12 * u

	p.progressBarText.SetValue("Progress bar")
	p.progressBar.Widget().SetMinimumValue(0)
	p.progressBar.Widget().SetMaximumValue(100)
	p.progressBar.Widget().SetValue(model.ProgressBars().Value())
	context.SetEnabled(&p.progressBar, model.ProgressBars().Enabled())
	p.progressBar.SetFixedWidth(width)

	p.progressBarWithoutRangeText.SetValue("Progress bar w/o range")
	p.progressBarWithoutRange.Widget().SetValue(model.ProgressBars().Value())
	context.SetEnabled(&p.progressBarWithoutRange, model.ProgressBars().Enabled())
	p.progressBarWithoutRange.SetFixedWidth(width)

	p.valueText.SetValue("Value (%)")
	p.valueNumberInput.Widget().OnValueChanged(func(context *guigui.Context, value int, committed bool) {
		if !committed {
			return
		}
		model.ProgressBars().SetValue(value)
	})
	p.valueNumberInput.Widget().SetMinimumValue(0)
	p.valueNumberInput.Widget().SetMaximumValue(100)
	p.valueNumberInput.Widget().SetStep(5)
	p.valueNumberInput.Widget().SetValue(model.ProgressBars().Value())
	p.valueNumberInput.SetFixedWidth(width)

	p.progressBarForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &p.progressBarText,
			SecondaryWidget: &p.progressBar,
		},
		{
			PrimaryWidget:   &p.progressBarWithoutRangeText,
			SecondaryWidget: &p.progressBarWithoutRange,
		},
		{
			PrimaryWidget:   &p.valueText,
			SecondaryWidget: &p.valueNumberInput,
		},
	})

	// Configurations
	p.enabledText.SetValue("Enabled")
	p.enabledToggle.OnValueChanged(func(context *guigui.Context, value bool) {
		model.ProgressBars().SetEnabled(value)
	})
	p.enabledToggle.SetValue(model.ProgressBars().Enabled())

	p.configForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &p.enabledText,
			SecondaryWidget: &p.enabledToggle,
		},
	})

	return nil
}

func (p *ProgressBars) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	p.layoutItems = slices.Delete(p.layoutItems, 0, len(p.layoutItems))
	p.layoutItems = append(p.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &p.progressBarFormPanel,
			Size:   guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: &p.configForm,
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     p.layoutItems,
		Gap:       u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}
