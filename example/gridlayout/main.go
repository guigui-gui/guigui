// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"fmt"
	"image"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	_ "github.com/hajimehoshi/guigui/basicwidget/cjkfont"
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

func (r *Root) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&r.background)
	adder.AddChild(&r.configForm)
	for i := range r.buttons {
		adder.AddChild(&r.buttons[i])
	}
}

func (r *Root) Update(context *guigui.Context) error {
	r.fillText.SetValue("Fill Widgets into Grid Cells")
	r.fillToggle.SetValue(r.fill)
	r.fillToggle.SetOnValueChanged(func(value bool) {
		r.fill = value
	})
	r.gapText.SetValue("Use Gap")
	r.gapToggle.SetValue(r.gap)
	r.gapToggle.SetOnValueChanged(func(value bool) {
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

func (r *Root) Layout(context *guigui.Context, widget guigui.Widget) image.Rectangle {
	switch widget {
	case &r.background:
		return context.Bounds(r)
	}

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

	gridRowLayout := func(row int) guigui.LinearLayout {
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

	bounds := (guigui.LinearLayout{
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
	}).WidgetBounds(context, context.Bounds(r).Inset(u/2), widget)

	if !r.fill {
		if _, ok := widget.(*basicwidget.Button); ok {
			pt := bounds.Min
			s := widget.Measure(context, guigui.Constraints{})
			pt.X += (bounds.Dx() - s.X) / 2
			pt.Y += (bounds.Dy() - s.Y) / 2
			return image.Rectangle{
				Min: pt,
				Max: pt.Add(s),
			}
		}
	}

	return bounds
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
