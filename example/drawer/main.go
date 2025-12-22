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

	background    basicwidget.Background
	configForm    basicwidget.Form
	edgeText      basicwidget.Text
	edgeSelect    basicwidget.Select[basicwidget.DrawerEdge]
	showButton    basicwidget.Button
	drawer        basicwidget.Drawer
	drawerContent drawerContent

	edge basicwidget.DrawerEdge
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&r.background)
	adder.AddChild(&r.configForm)
	adder.AddChild(&r.drawer)

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
	r.edgeSelect.SetOnItemSelected(func(context *guigui.Context, index int) {
		item, ok := r.edgeSelect.ItemByIndex(index)
		if !ok {
			return
		}
		r.edge = item.Value
	})

	r.showButton.SetText("Show")
	r.showButton.SetOnDown(func(context *guigui.Context) {
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
	r.drawerContent.SetOnClose(func(context *guigui.Context) {
		r.drawer.SetOpen(false)
	})

	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())

	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &r.configForm,
			},
			{
				Size: guigui.FlexibleSize(1),
			},
		},
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

type drawerContent struct {
	guigui.DefaultWidget

	closeButton basicwidget.Button
}

func (d *drawerContent) SetOnClose(f func(context *guigui.Context)) {
	guigui.SetEventHandler(d, "close", f)
}

func (d *drawerContent) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&d.closeButton)

	d.closeButton.SetText("Close")
	d.closeButton.SetOnDown(func(context *guigui.Context) {
		guigui.DispatchEvent(d, "close")
	})

	return nil
}

func (d *drawerContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
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
							Widget: &d.closeButton,
						},
					},
				},
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
