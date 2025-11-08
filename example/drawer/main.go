// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"fmt"
	"image"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type modelKey int

const (
	modelKeyModel modelKey = iota
)

const dummyText = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."

type Root struct {
	guigui.DefaultWidget

	background   basicwidget.Background
	toolbar      Toolbar
	leftPanel    LeftPanel
	contentPanel ContentPanel
	rightPanel   RightPanel

	model Model
}

func (r *Root) Model(key any) any {
	switch key {
	case modelKeyModel:
		return &r.model
	default:
		return nil
	}
}

func (r *Root) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&r.background)
	adder.AddChild(&r.toolbar)
	adder.AddChild(&r.leftPanel)
	adder.AddChild(&r.contentPanel)
	adder.AddChild(&r.rightPanel)
}

func (r *Root) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())

	mainLayout := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &r.toolbar,
				Size:   guigui.FixedSize(r.toolbar.Measure(context, guigui.Constraints{}).Y),
			},
			{
				Size: guigui.FlexibleSize(1),
			},
		},
	}
	boundsArr := mainLayout.AppendItemBounds(nil, context, widgetBounds.Bounds())
	layouter.LayoutWidget(&r.toolbar, boundsArr[0])

	bounds := boundsArr[1]
	bounds.Min.X -= r.model.DefaultPanelWidth(context)
	bounds.Min.X += r.model.LeftPanelWidth(context)
	bounds.Max.X += r.model.DefaultPanelWidth(context)
	bounds.Max.X -= r.model.RightPanelWidth(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &r.leftPanel,
				Size:   guigui.FixedSize(r.model.DefaultPanelWidth(context)),
			},
			{
				Widget: &r.contentPanel,
				Size:   guigui.FlexibleSize(1),
			},
			{
				Widget: &r.rightPanel,
				Size:   guigui.FixedSize(r.model.DefaultPanelWidth(context)),
			},
		},
	}).LayoutWidgets(context, bounds, layouter)
}

func (r *Root) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	r.model.Tick()
	return nil
}

func main() {
	op := &guigui.RunOptions{
		Title:      "Drawers",
		WindowSize: image.Pt(800, 600),
		RunGameOptions: &ebiten.RunGameOptions{
			ApplePressAndHoldEnabled: true,
		},
	}
	if err := guigui.Run(&Root{}, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
