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

	background    basicwidget.Background
	configForm    basicwidget.Form
	edgeText      basicwidget.Text
	edgeSelect    basicwidget.Select[basicwidget.DrawerEdge]
	showButton    basicwidget.Button
	drawer        basicwidget.Drawer
	drawerContent drawerContent

	edge basicwidget.DrawerEdge

	layoutItems []guigui.LinearLayoutItem
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.configForm)
	adder.AddWidget(&r.drawer)

	r.edgeText.SetValue("Edge")

	r.edgeSelect.SetItems([]basicwidget.SelectItem[basicwidget.DrawerEdge]{
		{
			Value: basicwidget.DrawerEdgeStart,
			Text:  "Start",
		},
		{
			Value: basicwidget.DrawerEdgeTop,
			Text:  "Top",
		},
		{
			Value: basicwidget.DrawerEdgeEnd,
			Text:  "End",
		},
		{
			Value: basicwidget.DrawerEdgeBottom,
			Text:  "Bottom",
		},
	})
	r.edgeSelect.SelectItemByValue(r.edge)
	r.edgeSelect.OnItemSelected(func(context *guigui.Context, index int) {
		item, ok := r.edgeSelect.ItemByIndex(index)
		if !ok {
			return
		}
		r.edge = item.Value
	})

	r.showButton.SetText("Show")
	r.showButton.OnDown(func(context *guigui.Context) {
		r.drawer.SetOpen(true)
	})

	r.configForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &r.edgeText,
			SecondaryWidget: &r.edgeSelect,
		},
		{
			SecondaryWidget: &r.showButton,
		},
	})

	r.drawer.SetDrawerEdge(r.edge)
	r.drawer.SetAnimated(true)
	r.drawer.SetCloseByClickingOutside(true)
	r.drawer.SetBackgroundDark(true)
	r.drawer.SetContent(&r.drawerContent)
	r.drawerContent.OnClose(func(context *guigui.Context) {
		r.drawer.SetOpen(false)
	})

	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())

	u := basicwidget.UnitSize(context)
	r.layoutItems = slices.Delete(r.layoutItems, 0, len(r.layoutItems))
	r.layoutItems = append(r.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &r.configForm,
		},
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     r.layoutItems,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)

	drawerBounds := context.AppBounds()
	switch r.edge {
	case basicwidget.DrawerEdgeStart:
		drawerBounds.Max.X = drawerBounds.Min.X + 6*u
	case basicwidget.DrawerEdgeTop:
		drawerBounds.Max.Y = drawerBounds.Min.Y + 4*u
	case basicwidget.DrawerEdgeEnd:
		drawerBounds.Min.X = drawerBounds.Max.X - 6*u
	case basicwidget.DrawerEdgeBottom:
		drawerBounds.Min.Y = drawerBounds.Max.Y - 4*u
	}
	layouter.LayoutWidget(&r.drawer, drawerBounds)
}

var (
	drawerContentEventClose guigui.EventKey = guigui.GenerateEventKey()
)

type drawerContent struct {
	guigui.DefaultWidget

	closeButton basicwidget.Button

	innerLayout guigui.LinearLayout
	innerItems  []guigui.LinearLayoutItem
	outerItems  []guigui.LinearLayoutItem
}

func (d *drawerContent) OnClose(f func(context *guigui.Context)) {
	guigui.SetEventHandler(d, drawerContentEventClose, f)
}

func (d *drawerContent) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&d.closeButton)

	d.closeButton.SetText("Close")
	d.closeButton.OnDown(func(context *guigui.Context) {
		guigui.DispatchEvent(d, drawerContentEventClose)
	})

	return nil
}

func (d *drawerContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	d.innerItems = slices.Delete(d.innerItems, 0, len(d.innerItems))
	d.innerItems = append(d.innerItems,
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: &d.closeButton,
		},
	)
	d.innerLayout = guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     d.innerItems,
	}
	d.outerItems = slices.Delete(d.outerItems, 0, len(d.outerItems))
	d.outerItems = append(d.outerItems,
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Layout: &d.innerLayout,
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     d.outerItems,
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
		Title: "Drawer",
		RunGameOptions: &ebiten.RunGameOptions{
			ApplePressAndHoldEnabled: true,
		},
	}
	if err := guigui.Run(&Root{}, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
