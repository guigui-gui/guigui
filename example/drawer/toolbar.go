// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"image"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Toolbar struct {
	guigui.DefaultWidget

	panel   basicwidget.Panel
	content guigui.WidgetWithSize[*toolbarContent]
}

func (t *Toolbar) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&t.panel)
	t.panel.SetStyle(basicwidget.PanelStyleSide)
	t.panel.SetBorders(basicwidget.PanelBorder{
		Bottom: true,
	})
	t.panel.SetContent(&t.content)
	return nil
}

func (t *Toolbar) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	t.content.SetFixedSize(widgetBounds.Bounds().Size())
	layouter.LayoutWidget(&t.panel, widgetBounds.Bounds())
}

func (t *Toolbar) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	u := basicwidget.UnitSize(context)
	return image.Pt(t.DefaultWidget.Measure(context, constraints).X, 2*u)
}

type toolbarContent struct {
	guigui.DefaultWidget

	leftPanelButton  basicwidget.Button
	rightPanelButton basicwidget.Button
}

func (t *toolbarContent) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&t.leftPanelButton)
	adder.AddChild(&t.rightPanelButton)

	model := context.Model(t, modelKeyModel).(*Model)

	if model.IsLeftPanelOpen() {
		img, err := theImageCache.GetMonochrome("left_panel_close", context.ColorMode())
		if err != nil {
			return err
		}
		t.leftPanelButton.SetIcon(img)
	} else {
		img, err := theImageCache.GetMonochrome("left_panel_open", context.ColorMode())
		if err != nil {
			return err
		}
		t.leftPanelButton.SetIcon(img)
	}
	if model.IsRightPanelOpen() {
		img, err := theImageCache.GetMonochrome("right_panel_close", context.ColorMode())
		if err != nil {
			return err
		}
		t.rightPanelButton.SetIcon(img)
	} else {
		img, err := theImageCache.GetMonochrome("right_panel_open", context.ColorMode())
		if err != nil {
			return err
		}
		t.rightPanelButton.SetIcon(img)
	}
	t.leftPanelButton.SetOnDown(func() {
		model.SetLeftPanelOpen(!model.IsLeftPanelOpen())
	})
	t.rightPanelButton.SetOnDown(func() {
		model.SetRightPanelOpen(!model.IsRightPanelOpen())
	})

	return nil
}

func (t *toolbarContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &t.leftPanelButton,
				Size:   guigui.FixedSize(u * 3 / 2),
			},
			{
				Size: guigui.FlexibleSize(1),
			},
			{
				Widget: &t.rightPanelButton,
				Size:   guigui.FixedSize(u * 3 / 2),
			},
		},
		Padding: guigui.Padding{
			Start:  u / 4,
			Top:    u / 4,
			End:    u / 4,
			Bottom: u / 4,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}
