// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"image"
	"image/color"
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	"github.com/hajimehoshi/ebiten/v2"
)

type Selects struct {
	guigui.DefaultWidget

	listForm    basicwidget.Form
	select1Text basicwidget.Text
	select1     basicwidget.Select[int]
	select2Text basicwidget.Text
	select2     basicwidget.Select[int]

	configForm    basicwidget.Form
	enabledText   basicwidget.Text
	enabledToggle basicwidget.Toggle

	select1Items       []basicwidget.SelectItem[int]
	select2Items       []basicwidget.SelectItem[int]
	select2ItemWidgets []selectItem
}

func (s *Selects) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&s.listForm)
	adder.AddChild(&s.configForm)

	model := context.Model(s, modelKeyModel).(*Model)

	// Select (Text)
	s.select1Text.SetValue("Selects")
	s.select1Items = slices.Delete(s.select1Items, 0, len(s.select1Items))
	s.select1Items = model.Selects().AppendSelectItems(s.select1Items)
	s.select1.SetItems(s.select1Items)
	context.SetEnabled(&s.select1, model.Selects().Enabled())
	if s.select1.SelectedItemIndex() < 0 {
		s.select1.SelectItemByIndex(0)
	}

	// Select (Custom item)
	s.select2Text.SetValue("Select with custom widgets")
	// TODO: This logic is quite common. The implementation is basicwidget.adjustSliceSize. Refactor it.
	if len(s.select2ItemWidgets) < 3 {
		s.select2ItemWidgets = slices.Grow(s.select2ItemWidgets, 3-len(s.select2ItemWidgets))[:3]
	} else if len(s.select2ItemWidgets) > 3 {
		s.select2ItemWidgets = slices.Delete(s.select2ItemWidgets, 3, len(s.select2ItemWidgets))
	}
	img, err := theImageCache.Get("gopher_left")
	if err != nil {
		return err
	}
	s.select2ItemWidgets[0].SetImage(img)
	s.select2ItemWidgets[0].SetText("Left")
	img, err = theImageCache.Get("gopher_center")
	if err != nil {
		return err
	}
	s.select2ItemWidgets[1].SetImage(img)
	s.select2ItemWidgets[1].SetText("Center")
	img, err = theImageCache.Get("gopher_right")
	if err != nil {
		return err
	}
	s.select2ItemWidgets[2].SetImage(img)
	s.select2ItemWidgets[2].SetText("Right")

	s.select2Items = slices.Delete(s.select2Items, 0, len(s.select2Items))
	for i := range s.select2ItemWidgets {
		w := &s.select2ItemWidgets[i]
		s.select2Items = append(s.select2Items, basicwidget.SelectItem[int]{
			Value:   i,
			Content: w,
		})
	}
	s.select2.SetItems(s.select2Items)
	context.SetEnabled(&s.select2, model.Selects().Enabled())
	if s.select2.SelectedItemIndex() < 0 {
		s.select2.SelectItemByIndex(0)
	}

	s.listForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &s.select1Text,
			SecondaryWidget: &s.select1,
		},
		{
			PrimaryWidget:   &s.select2Text,
			SecondaryWidget: &s.select2,
		},
	})

	// Config form
	s.enabledText.SetValue("Enabled")
	s.enabledToggle.SetValue(model.Selects().Enabled())
	s.enabledToggle.SetOnValueChanged(func(context *guigui.Context, toggled bool) {
		model.Selects().SetEnabled(toggled)
	})

	s.configForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &s.enabledText,
			SecondaryWidget: &s.enabledToggle,
		},
	})

	return nil
}

func (s *Selects) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &s.listForm,
			},
			{
				Size: guigui.FlexibleSize(1),
			},
			{
				Widget: &s.configForm,
			},
		},
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

type selectItem struct {
	guigui.DefaultWidget

	image basicwidget.Image
	text  basicwidget.Text
}

func (s *selectItem) SetImage(img *ebiten.Image) {
	s.image.SetImage(img)
}

func (d *selectItem) SetText(s string) {
	d.text.SetValue(s)
}

func (s *selectItem) SetTextColor(clr color.Color) {
	s.text.SetColor(clr)
}

func (s *selectItem) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&s.image)
	adder.AddChild(&s.text)
	s.text.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	return nil
}

func (s *selectItem) layout(context *guigui.Context) guigui.LinearLayout {
	u := basicwidget.UnitSize(context)
	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &s.image,
				Size:   guigui.FixedSize(2 * u),
			},
			{
				Widget: &s.text,
			},
		},
		Gap: u / 4,
	}
}

func (s *selectItem) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	s.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (s *selectItem) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return s.layout(context).Measure(context, constraints)
}
