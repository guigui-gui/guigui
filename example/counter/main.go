// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package main

import (
	"fmt"
	"image"
	"os"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Root struct {
	guigui.DefaultWidget

	background  basicwidget.Background
	resetButton basicwidget.Button
	decButton   basicwidget.Button
	incButton   basicwidget.Button
	counterText basicwidget.Text

	counter int

	buttonRowLayout guigui.LinearLayout
	buttonRowItems  []guigui.LinearLayoutItem
	layoutItems     []guigui.LinearLayoutItem
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.counterText)
	adder.AddWidget(&r.resetButton)
	adder.AddWidget(&r.decButton)
	adder.AddWidget(&r.incButton)

	r.counterText.SetSelectable(true)
	r.counterText.SetBold(true)
	r.counterText.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)
	r.counterText.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	r.counterText.SetScale(4)
	r.counterText.SetValue(fmt.Sprintf("%d", r.counter))

	r.resetButton.SetText("Reset")
	r.resetButton.OnUp(func(context *guigui.Context) {
		r.counter = 0
	})
	context.SetEnabled(&r.resetButton, r.counter != 0)

	r.decButton.SetText("Decrement")
	r.decButton.OnUp(func(context *guigui.Context) {
		r.counter--
	})

	r.incButton.SetText("Increment")
	r.incButton.OnUp(func(context *guigui.Context) {
		r.counter++
	})

	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())

	u := basicwidget.UnitSize(context)
	r.buttonRowItems = slices.Delete(r.buttonRowItems, 0, len(r.buttonRowItems))
	r.buttonRowItems = append(r.buttonRowItems,
		guigui.LinearLayoutItem{
			Widget: &r.resetButton,
			Size:   guigui.FixedSize(6 * u),
		},
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: &r.decButton,
			Size:   guigui.FixedSize(6 * u),
		},
		guigui.LinearLayoutItem{
			Widget: &r.incButton,
			Size:   guigui.FixedSize(6 * u),
		},
	)
	r.buttonRowLayout = guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     r.buttonRowItems,
		Gap:       u / 2,
	}
	r.layoutItems = slices.Delete(r.layoutItems, 0, len(r.layoutItems))
	r.layoutItems = append(r.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &r.counterText,
			Size:   guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Size:   guigui.FixedSize(u),
			Layout: &r.buttonRowLayout,
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     r.layoutItems,
		Gap:       u,
		Padding: guigui.Padding{
			Start:  u,
			Top:    u,
			End:    u,
			Bottom: u,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func main() {
	op := &guigui.RunOptions{
		Title:         "Counter",
		WindowMinSize: image.Pt(600, 300),
		RunGameOptions: &ebiten.RunGameOptions{
			ApplePressAndHoldEnabled: true,
		},
	}
	if err := guigui.Run(&Root{}, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
