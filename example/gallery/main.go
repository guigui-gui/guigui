// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package main

import (
	"fmt"
	"image"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	_ "github.com/guigui-gui/guigui/basicwidget/cjkfont"
)

type modelKey int

const (
	modelKeyModel modelKey = iota
)

type Root struct {
	guigui.DefaultWidget

	background   basicwidget.Background
	sidebar      Sidebar
	settings     Settings
	basic        Basic
	buttons      Buttons
	texts        Texts
	textInputs   TextInputs
	numberInputs NumberInputs
	lists        Lists
	selects      Selects
	tables       Tables
	popups       Popups

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

func (r *Root) contentWidgeet() guigui.Widget {
	switch r.model.Mode() {
	case "settings":
		return &r.settings
	case "basic":
		return &r.basic
	case "buttons":
		return &r.buttons
	case "texts":
		return &r.texts
	case "textinputs":
		return &r.textInputs
	case "numberinputs":
		return &r.numberInputs
	case "lists":
		return &r.lists
	case "selects":
		return &r.selects
	case "tables":
		return &r.tables
	case "popups":
		return &r.popups
	}
	return nil
}

func (r *Root) Build(context *guigui.Context, adder *guigui.WidgetAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.sidebar)
	if content := r.contentWidgeet(); content != nil {
		adder.AddWidget(content)
	}
	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &r.sidebar,
				Size:   guigui.FixedSize(8 * basicwidget.UnitSize(context)),
			},
			{
				Widget: r.contentWidgeet(),
				Size:   guigui.FlexibleSize(1),
			},
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func main() {
	op := &guigui.RunOptions{
		Title:      "Component Gallery",
		WindowSize: image.Pt(800, 800),
		RunGameOptions: &ebiten.RunGameOptions{
			ApplePressAndHoldEnabled: true,
		},
	}
	if err := guigui.Run(&Root{}, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
