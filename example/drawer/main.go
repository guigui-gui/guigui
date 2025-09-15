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

func (r *Root) Layout(context *guigui.Context, widget guigui.Widget) image.Rectangle {
	switch widget {
	case &r.background:
		return context.Bounds(r)
	}

	return (guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &r.toolbar,
				Size:   guigui.FixedSize(r.toolbar.Measure(context, guigui.Constraints{}).Y),
			},
			{
				Size: guigui.FlexibleSize(1),
				Layout: guigui.LinearLayout{
					Direction: guigui.LayoutDirectionHorizontal,
					Items: []guigui.LinearLayoutItem{
						{
							Widget: &r.leftPanel,
							Size:   guigui.FixedSize(r.model.LeftPanelWidth(context)),
						},
						{
							Widget: &r.contentPanel,
							Size:   guigui.FlexibleSize(1),
						},
						{
							Widget: &r.rightPanel,
							Size:   guigui.FixedSize(r.model.RightPanelWidth(context)),
						},
					},
				},
			},
		},
	}).WidgetBounds(context, context.Bounds(r), widget)
}

func (r *Root) Tick(context *guigui.Context) error {
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
