// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

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

var (
	modelKeyModel = guigui.GenerateEnvKey()
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

	mainLayoutItems  []guigui.LinearLayoutItem
	panelLayoutItems []guigui.LinearLayoutItem
}

func (r *Root) Env(context *guigui.Context, key guigui.EnvKey, source *guigui.EnvSource) (any, bool) {
	switch key {
	case modelKeyModel:
		return &r.model, true
	default:
		return nil, false
	}
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.toolbar)
	adder.AddWidget(&r.leftPanel)
	adder.AddWidget(&r.contentPanel)
	adder.AddWidget(&r.rightPanel)
	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())

	r.mainLayoutItems = slices.Delete(r.mainLayoutItems, 0, len(r.mainLayoutItems))
	r.mainLayoutItems = append(r.mainLayoutItems,
		guigui.LinearLayoutItem{
			Widget: &r.toolbar,
			Size:   guigui.FixedSize(r.toolbar.Measure(context, guigui.Constraints{}).Y),
		},
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
	)
	mainLayout := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     r.mainLayoutItems,
	}
	mainLayout.LayoutWidgets(context, widgetBounds.Bounds(), layouter)

	boundsArr := mainLayout.AppendItemBounds(nil, context, widgetBounds.Bounds())
	bounds := boundsArr[1]
	bounds.Min.X -= r.model.DefaultPanelWidth(context)
	bounds.Min.X += r.model.LeftPanelWidth(context)
	bounds.Max.X += r.model.DefaultPanelWidth(context)
	bounds.Max.X -= r.model.RightPanelWidth(context)
	r.panelLayoutItems = slices.Delete(r.panelLayoutItems, 0, len(r.panelLayoutItems))
	r.panelLayoutItems = append(r.panelLayoutItems,
		guigui.LinearLayoutItem{
			Widget: &r.leftPanel,
			Size:   guigui.FixedSize(r.model.DefaultPanelWidth(context)),
		},
		guigui.LinearLayoutItem{
			Widget: &r.contentPanel,
			Size:   guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: &r.rightPanel,
			Size:   guigui.FixedSize(r.model.DefaultPanelWidth(context)),
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     r.panelLayoutItems,
	}).LayoutWidgets(context, bounds, layouter)
}

func (r *Root) BuildKey(h *guigui.BuildKeyHasher) {
	r.model.writeBuildKey(h)
}

func (r *Root) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	r.model.Tick()
	return nil
}

func main() {
	op := &guigui.RunOptions{
		Title:      "Panels",
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
