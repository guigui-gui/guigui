// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package main

import (
	"fmt"
	"image"
	"os"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	"github.com/guigui-gui/guigui/basicwidget/cjkfont"
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
	tables       Tables
	popups       Popups

	model Model

	locales           []language.Tag
	faceSourceEntries []basicwidget.FaceSourceEntry
}

func (r *Root) updateFontFaceSources(context *guigui.Context) {
	r.locales = slices.Delete(r.locales, 0, len(r.locales))
	r.locales = context.AppendLocales(r.locales)

	r.faceSourceEntries = slices.Delete(r.faceSourceEntries, 0, len(r.faceSourceEntries))
	r.faceSourceEntries = cjkfont.AppendRecommendedFaceSourceEntries(r.faceSourceEntries, r.locales)
	basicwidget.SetFaceSources(r.faceSourceEntries)
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
	adder.AddChild(&r.sidebar)
	switch r.model.Mode() {
	case "settings":
		adder.AddChild(&r.settings)
	case "basic":
		adder.AddChild(&r.basic)
	case "buttons":
		adder.AddChild(&r.buttons)
	case "texts":
		adder.AddChild(&r.texts)
	case "textinputs":
		adder.AddChild(&r.textInputs)
	case "numberinputs":
		adder.AddChild(&r.numberInputs)
	case "lists":
		adder.AddChild(&r.lists)
	case "tables":
		adder.AddChild(&r.tables)
	case "popups":
		adder.AddChild(&r.popups)
	}
}

func (r *Root) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	r.updateFontFaceSources(context)
	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, widget guigui.Widget) image.Rectangle {
	switch widget {
	case &r.background:
		return widgetBounds.Bounds()
	}

	layout := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &r.sidebar,
				Size:   guigui.FixedSize(8 * basicwidget.UnitSize(context)),
			},
			{
				Size: guigui.FlexibleSize(1),
			},
		},
	}
	if widget == &r.sidebar {
		return layout.WidgetBounds(context, widgetBounds.Bounds(), widget)
	}
	return layout.ItemBounds(context, widgetBounds.Bounds(), 1)
}

func main() {
	op := &guigui.RunOptions{
		Title:      "Component Gallery",
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
