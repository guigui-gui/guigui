// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

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

	background basicwidget.Background
	menubar    basicwidget.Menubar[string]
	resultText basicwidget.Text

	lastSelection string

	layoutItems []guigui.LinearLayoutItem
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.menubar)
	adder.AddWidget(&r.resultText)

	r.menubar.SetItems([]basicwidget.MenubarItem{
		{Text: "File"},
		{Text: "Edit"},
		{Text: "View"},
		{Text: "Disabled", Disabled: true},
	})
	r.menubar.PopupMenuAt(0).SetItems([]basicwidget.PopupMenuItem[string]{
		{Text: "New", Value: "file.new"},
		{Text: "Open...", Value: "file.open"},
		{Text: "Save", Value: "file.save"},
		{Border: true},
		{Text: "Quit", Value: "file.quit"},
	})
	r.menubar.PopupMenuAt(1).SetItems([]basicwidget.PopupMenuItem[string]{
		{Text: "Undo", KeyText: "Ctrl+Z", Value: "edit.undo"},
		{Text: "Redo", KeyText: "Ctrl+Y", Value: "edit.redo"},
		{Border: true},
		{Text: "Cut", KeyText: "Ctrl+X", Value: "edit.cut"},
		{Text: "Copy", KeyText: "Ctrl+C", Value: "edit.copy"},
		{Text: "Paste", KeyText: "Ctrl+V", Value: "edit.paste"},
	})
	r.menubar.PopupMenuAt(2).SetItems([]basicwidget.PopupMenuItem[string]{
		{Text: "Zoom In", Value: "view.zoomin"},
		{Text: "Zoom Out", Value: "view.zoomout"},
		{Text: "Reset Zoom", Value: "view.zoomreset"},
	})
	r.menubar.PopupMenuAt(3).SetItems([]basicwidget.PopupMenuItem[string]{
		{Text: "Should not appear"},
	})
	r.menubar.OnItemSelected(func(context *guigui.Context, menuIndex, itemIndex int) {
		r.lastSelection = fmt.Sprintf("Selected: menu %d, item %d", menuIndex, itemIndex)
	})

	if r.lastSelection == "" {
		r.resultText.SetValue("Click a title to open its menu.")
	} else {
		r.resultText.SetValue(r.lastSelection)
	}

	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	bounds := widgetBounds.Bounds()
	layouter.LayoutWidget(&r.background, bounds)

	u := basicwidget.UnitSize(context)

	// Pin the menubar to the top edge with no padding.
	mh := r.menubar.Measure(context, guigui.Constraints{}).Y
	menubarBounds := image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Min.Y+mh)
	layouter.LayoutWidget(&r.menubar, menubarBounds)

	bodyBounds := image.Rect(bounds.Min.X, menubarBounds.Max.Y, bounds.Max.X, bounds.Max.Y)
	r.layoutItems = slices.Delete(r.layoutItems, 0, len(r.layoutItems))
	r.layoutItems = append(r.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &r.resultText,
		},
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
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
	}).LayoutWidgets(context, bodyBounds, layouter)
}

func main() {
	op := &guigui.RunOptions{
		Title:         "Menubar",
		WindowMinSize: image.Pt(600, 400),
		RunGameOptions: &ebiten.RunGameOptions{
			ApplePressAndHoldEnabled: true,
		},
	}
	if err := guigui.Run(&Root{}, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
