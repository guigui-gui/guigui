// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Root struct {
	guigui.DefaultWidget

	fill bool
	gap  bool

	configForm basicwidget.Form
	fillText   basicwidget.Text
	fillToggle basicwidget.Toggle
	gapText    basicwidget.Text
	gapToggle  basicwidget.Toggle

	background basicwidget.Background
	buttons    [16]basicwidget.Button
}

func (r *Root) Build(context *guigui.Context, adder *guigui.WidgetAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.configForm)
	for i := range r.buttons {
		adder.AddWidget(&r.buttons[i])
	}

	r.fillText.SetValue("Fill Widgets into Grid Cells")
	r.fillToggle.SetValue(r.fill)
	r.fillToggle.SetOnValueChanged(func(context *guigui.Context, value bool) {
		r.fill = value
	})
	r.gapText.SetValue("Use Gap")
	r.gapToggle.SetValue(r.gap)
	r.gapToggle.SetOnValueChanged(func(context *guigui.Context, value bool) {
		r.gap = value
	})
	r.configForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &r.fillText,
			SecondaryWidget: &r.fillToggle,
		},
		{
			PrimaryWidget:   &r.gapText,
			SecondaryWidget: &r.gapToggle,
		},
	})

	for i := range r.buttons {
		r.buttons[i].SetText(fmt.Sprintf("Button %d", i))
	}

	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())

	u := basicwidget.UnitSize(context)
	var gridGap int
	if r.gap {
		gridGap = int(u / 2)
	}

	var firstColumnWidth int
	for i := range 4 {
		firstColumnWidth = max(firstColumnWidth, r.buttons[4*i].Measure(context, guigui.Constraints{}).X)
	}
	var firstRowHeight int
	for i := range 4 {
		firstRowHeight = max(firstRowHeight, r.buttons[i].Measure(context, guigui.Constraints{}).Y)
	}

	center := func(widget guigui.Widget) guigui.LinearLayout {
		return guigui.LinearLayout{
			Direction: guigui.LayoutDirectionHorizontal,
			Items: []guigui.LinearLayoutItem{
				{
					Size: guigui.FlexibleSize(1),
				},
				{
					Layout: guigui.LinearLayout{
						Direction: guigui.LayoutDirectionVertical,
						Items: []guigui.LinearLayoutItem{
							{
								Size: guigui.FlexibleSize(1),
							},
							{
								Widget: widget,
							},
							{
								Size: guigui.FlexibleSize(1),
							},
						},
					},
				},
				{
					Size: guigui.FlexibleSize(1),
				},
			},
		}
	}

	gridRowLayout := func(row int) guigui.LinearLayout {
		if r.fill {
			return guigui.LinearLayout{
				Direction: guigui.LayoutDirectionHorizontal,
				Items: []guigui.LinearLayoutItem{
					{
						Widget: &r.buttons[4*row],
						Size:   guigui.FixedSize(firstColumnWidth),
					},
					{
						Widget: &r.buttons[4*row+1],
						Size:   guigui.FixedSize(200),
					},
					{
						Widget: &r.buttons[4*row+2],
						Size:   guigui.FlexibleSize(1),
					},
					{
						Widget: &r.buttons[4*row+3],
						Size:   guigui.FlexibleSize(2),
					},
				},
				Gap: gridGap,
			}
		}
		return guigui.LinearLayout{
			Direction: guigui.LayoutDirectionHorizontal,
			Items: []guigui.LinearLayoutItem{
				{
					Layout: center(&r.buttons[4*row]),
					Size:   guigui.FixedSize(firstColumnWidth),
				},
				{
					Layout: center(&r.buttons[4*row+1]),
					Size:   guigui.FixedSize(200),
				},
				{
					Layout: center(&r.buttons[4*row+2]),
					Size:   guigui.FlexibleSize(1),
				},
				{
					Layout: center(&r.buttons[4*row+3]),
					Size:   guigui.FlexibleSize(2),
				},
			},
			Gap: gridGap,
		}
	}

	layout := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &r.configForm,
			},
			{
				Size: guigui.FlexibleSize(1),
				Layout: guigui.LinearLayout{
					Direction: guigui.LayoutDirectionVertical,
					Items: []guigui.LinearLayoutItem{
						{
							Size:   guigui.FixedSize(firstRowHeight),
							Layout: gridRowLayout(0),
						},
						{
							Size:   guigui.FixedSize(100),
							Layout: gridRowLayout(1),
						},
						{
							Size:   guigui.FlexibleSize(1),
							Layout: gridRowLayout(2),
						},
						{
							Size:   guigui.FlexibleSize(2),
							Layout: gridRowLayout(3),
						},
					},
					Gap: gridGap,
				},
			},
		},
		Gap: u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}
	layout.LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func main() {
	op := &guigui.RunOptions{
		Title: "Grid Layout",
		RunGameOptions: &ebiten.RunGameOptions{
			ApplePressAndHoldEnabled: true,
		},
	}
	if err := guigui.Run(&Root{}, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
