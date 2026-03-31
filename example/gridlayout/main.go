// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"fmt"
	"os"
	"slices"

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

	// Layout state for center function (non-fill case): 4 rows * 4 cols = 16.
	centerInnerItems   [16][]guigui.LinearLayoutItem
	centerInnerLayouts []guigui.LinearLayout
	centerOuterItems   [16][]guigui.LinearLayoutItem
	centerOuterLayouts []guigui.LinearLayout

	// Layout state for grid rows.
	gridRowItems   [4][]guigui.LinearLayoutItem
	gridRowLayouts []guigui.LinearLayout

	// Layout state for the inner grid layout.
	gridItems  []guigui.LinearLayoutItem
	gridLayout guigui.LinearLayout

	// Layout state for the outer layout.
	outerItems []guigui.LinearLayoutItem
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.configForm)
	for i := range r.buttons {
		adder.AddWidget(&r.buttons[i])
	}

	r.fillText.SetValue("Fill Widgets into Grid Cells")
	r.fillToggle.SetValue(r.fill)
	r.fillToggle.OnValueChanged(func(context *guigui.Context, value bool) {
		r.fill = value
	})
	r.gapText.SetValue("Use Gap")
	r.gapToggle.SetValue(r.gap)
	r.gapToggle.OnValueChanged(func(context *guigui.Context, value bool) {
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

func (r *Root) setupCenter(idx int, widget guigui.Widget) {
	r.centerInnerItems[idx] = slices.Delete(r.centerInnerItems[idx], 0, len(r.centerInnerItems[idx]))
	r.centerInnerItems[idx] = append(r.centerInnerItems[idx],
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: widget,
		},
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
	)
	r.centerInnerLayouts = append(r.centerInnerLayouts, guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     r.centerInnerItems[idx],
	})
	r.centerOuterItems[idx] = slices.Delete(r.centerOuterItems[idx], 0, len(r.centerOuterItems[idx]))
	r.centerOuterItems[idx] = append(r.centerOuterItems[idx],
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Layout: &r.centerInnerLayouts[idx],
		},
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
	)
	r.centerOuterLayouts = append(r.centerOuterLayouts, guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     r.centerOuterItems[idx],
	})
}

func (r *Root) setupGridRow(row, firstColumnWidth, gridGap int) {
	r.gridRowItems[row] = slices.Delete(r.gridRowItems[row], 0, len(r.gridRowItems[row]))
	if r.fill {
		r.gridRowItems[row] = append(r.gridRowItems[row],
			guigui.LinearLayoutItem{
				Widget: &r.buttons[4*row],
				Size:   guigui.FixedSize(firstColumnWidth),
			},
			guigui.LinearLayoutItem{
				Widget: &r.buttons[4*row+1],
				Size:   guigui.FixedSize(200),
			},
			guigui.LinearLayoutItem{
				Widget: &r.buttons[4*row+2],
				Size:   guigui.FlexibleSize(1),
			},
			guigui.LinearLayoutItem{
				Widget: &r.buttons[4*row+3],
				Size:   guigui.FlexibleSize(2),
			},
		)
	} else {
		centerBase := row * 4
		r.setupCenter(centerBase, &r.buttons[4*row])
		r.setupCenter(centerBase+1, &r.buttons[4*row+1])
		r.setupCenter(centerBase+2, &r.buttons[4*row+2])
		r.setupCenter(centerBase+3, &r.buttons[4*row+3])
		r.gridRowItems[row] = append(r.gridRowItems[row],
			guigui.LinearLayoutItem{
				Layout: &r.centerOuterLayouts[centerBase],
				Size:   guigui.FixedSize(firstColumnWidth),
			},
			guigui.LinearLayoutItem{
				Layout: &r.centerOuterLayouts[centerBase+1],
				Size:   guigui.FixedSize(200),
			},
			guigui.LinearLayoutItem{
				Layout: &r.centerOuterLayouts[centerBase+2],
				Size:   guigui.FlexibleSize(1),
			},
			guigui.LinearLayoutItem{
				Layout: &r.centerOuterLayouts[centerBase+3],
				Size:   guigui.FlexibleSize(2),
			},
		)
	}
	r.gridRowLayouts = append(r.gridRowLayouts, guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     r.gridRowItems[row],
		Gap:       gridGap,
	})
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

	r.centerInnerLayouts = slices.Delete(r.centerInnerLayouts, 0, len(r.centerInnerLayouts))
	r.centerOuterLayouts = slices.Delete(r.centerOuterLayouts, 0, len(r.centerOuterLayouts))
	r.gridRowLayouts = slices.Delete(r.gridRowLayouts, 0, len(r.gridRowLayouts))
	for row := range 4 {
		r.setupGridRow(row, firstColumnWidth, gridGap)
	}

	r.gridItems = slices.Delete(r.gridItems, 0, len(r.gridItems))
	r.gridItems = append(r.gridItems,
		guigui.LinearLayoutItem{
			Size:   guigui.FixedSize(firstRowHeight),
			Layout: &r.gridRowLayouts[0],
		},
		guigui.LinearLayoutItem{
			Size:   guigui.FixedSize(100),
			Layout: &r.gridRowLayouts[1],
		},
		guigui.LinearLayoutItem{
			Size:   guigui.FlexibleSize(1),
			Layout: &r.gridRowLayouts[2],
		},
		guigui.LinearLayoutItem{
			Size:   guigui.FlexibleSize(2),
			Layout: &r.gridRowLayouts[3],
		},
	)
	r.gridLayout = guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     r.gridItems,
		Gap:       gridGap,
	}

	r.outerItems = slices.Delete(r.outerItems, 0, len(r.outerItems))
	r.outerItems = append(r.outerItems,
		guigui.LinearLayoutItem{
			Widget: &r.configForm,
		},
		guigui.LinearLayoutItem{
			Size:   guigui.FlexibleSize(1),
			Layout: &r.gridLayout,
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     r.outerItems,
		Gap:       u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
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
