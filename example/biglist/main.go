// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"fmt"
	"image"
	"image/color"
	"math/rand/v2"
	"os"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

const itemCount = 10000

type itemWidget struct {
	guigui.DefaultWidget

	text   basicwidget.Text
	height int

	layoutItems []guigui.LinearLayoutItem
}

func (w *itemWidget) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&w.text)
	return nil
}

func (w *itemWidget) layout(context *guigui.Context) guigui.LinearLayout {
	w.layoutItems = slices.Delete(w.layoutItems, 0, len(w.layoutItems))
	w.layoutItems = append(w.layoutItems, guigui.LinearLayoutItem{
		Widget: &w.text,
		Size:   guigui.FlexibleSize(1),
	})
	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     w.layoutItems,
		Padding:   basicwidget.ListItemTextPadding(context),
	}
}

func (w *itemWidget) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	var clr color.Color
	if v, ok := context.Env(w, basicwidget.EnvKeyListItemColorType); ok {
		if ct, ok := v.(basicwidget.ListItemColorType); ok {
			clr = ct.TextColor(context)
		}
	}
	w.text.SetColor(clr)
	w.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (w *itemWidget) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	s := w.layout(context).Measure(context, constraints)
	if w.height > 0 {
		s.Y = w.height
	}
	return s
}

func (w *itemWidget) SetValue(text string) {
	w.text.SetValue(text)
}

func (w *itemWidget) SetHeight(height int) {
	if w.height == height {
		return
	}
	w.height = height
	guigui.RequestRebuild(w)
}

type Root struct {
	guigui.DefaultWidget

	background                   basicwidget.Background
	list                         guigui.WidgetWithSize[*basicwidget.List[int]]
	randomizeButton              basicwidget.Button
	randomizeAboveViewportButton basicwidget.Button

	itemWidgets guigui.WidgetSlice[*itemWidget]
	items       []basicwidget.ListItem[int]

	itemHeightScales [itemCount]int

	layoutItems []guigui.LinearLayoutItem
}

func (r *Root) randomizeHeights() {
	for i := range itemCount {
		r.itemHeightScales[i] = 1 + rand.IntN(5)
	}
}

func (r *Root) randomizeHeightsAboveViewport() {
	for i := range itemCount {
		if r.list.Widget().IsItemInViewport(i) {
			break
		}
		r.itemHeightScales[i] = 1 + rand.IntN(5)
	}
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.list)
	adder.AddWidget(&r.randomizeButton)
	adder.AddWidget(&r.randomizeAboveViewportButton)

	u := basicwidget.UnitSize(context)

	r.randomizeButton.SetText("Randomize Heights")
	r.randomizeButton.OnUp(func(context *guigui.Context) {
		r.randomizeHeights()
		guigui.RequestRebuild(r)
	})

	r.randomizeAboveViewportButton.SetText("Randomize Heights Above Viewport")
	r.randomizeAboveViewportButton.OnUp(func(context *guigui.Context) {
		r.randomizeHeightsAboveViewport()
		guigui.RequestRebuild(r)
	})

	r.itemWidgets.SetLen(itemCount)
	r.items = slices.Delete(r.items, 0, len(r.items))
	for i := range itemCount {
		scale := r.itemHeightScales[i]
		if scale == 0 {
			scale = 1
		}
		w := r.itemWidgets.At(i)
		w.SetValue(fmt.Sprintf("Item %d", i+1))
		w.SetHeight(u * scale)
		r.items = append(r.items, basicwidget.ListItem[int]{
			Content: w,
			Value:   i,
		})
	}
	r.list.Widget().SetItems(r.items)
	r.list.Widget().SetStripeVisible(true)

	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())

	u := basicwidget.UnitSize(context)
	r.layoutItems = slices.Delete(r.layoutItems, 0, len(r.layoutItems))
	r.layoutItems = append(r.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &r.list,
			Size:   guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: &r.randomizeButton,
			Size:   guigui.FixedSize(u),
		},
		guigui.LinearLayoutItem{
			Widget: &r.randomizeAboveViewportButton,
			Size:   guigui.FixedSize(u),
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     r.layoutItems,
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
	r := &Root{}
	r.randomizeHeights()

	op := &guigui.RunOptions{
		Title:         "Big List",
		WindowMinSize: image.Pt(400, 300),
		WindowSize:    image.Pt(600, 600),
		RunGameOptions: &ebiten.RunGameOptions{
			ApplePressAndHoldEnabled: true,
		},
	}
	if err := guigui.Run(r, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
