// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"image"
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

// infoDialog is the single-OK-button counterpart used for About-style messages.
type infoDialog struct {
	guigui.DefaultWidget

	popup   basicwidget.Popup
	content infoDialogContent
}

func (i *infoDialog) Open() {
	i.popup.SetOpen(true)
}

func (i *infoDialog) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&i.popup)
	i.content.popup = &i.popup
	i.popup.SetContent(&i.content)
	i.popup.SetModal(true)
	i.popup.SetBackgroundDark(true)
	i.popup.SetCloseByClickingOutside(true)
	i.popup.SetAnimated(true)
	return nil
}

func (i *infoDialog) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	size := i.content.Measure(context, guigui.Constraints{})
	app := context.AppBounds()
	pos := image.Pt(
		app.Min.X+(app.Dx()-size.X)/2,
		app.Min.Y+(app.Dy()-size.Y)/2,
	)
	layouter.LayoutWidget(&i.popup, image.Rectangle{Min: pos, Max: pos.Add(size)})
}

type infoDialogContent struct {
	guigui.DefaultWidget

	popup *basicwidget.Popup

	message  basicwidget.Text
	okButton basicwidget.Button

	rowItems    []guigui.LinearLayoutItem
	rowLayout   guigui.LinearLayout
	layoutItems []guigui.LinearLayoutItem
}

func (c *infoDialogContent) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&c.message)
	adder.AddWidget(&c.okButton)

	c.message.SetValue("Text Editor — Guigui example")
	c.message.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	c.message.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)

	c.okButton.SetText("OK")
	c.okButton.OnDown(func(context *guigui.Context) {
		c.popup.SetOpen(false)
	})
	return nil
}

func (c *infoDialogContent) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	u := basicwidget.UnitSize(context)
	return image.Pt(9*u, 6*u)
}

func (c *infoDialogContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)

	c.rowItems = slices.Delete(c.rowItems, 0, len(c.rowItems))
	c.rowItems = append(c.rowItems,
		guigui.LinearLayoutItem{Size: guigui.FlexibleSize(1)},
		guigui.LinearLayoutItem{
			Widget: &c.okButton,
			Size:   guigui.FixedSize(c.okButton.Measure(context, guigui.Constraints{}).X),
		},
	)
	c.rowLayout = guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     c.rowItems,
	}

	c.layoutItems = slices.Delete(c.layoutItems, 0, len(c.layoutItems))
	c.layoutItems = append(c.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &c.message,
			Size:   guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Size:   guigui.FixedSize(c.okButton.Measure(context, guigui.Constraints{}).Y),
			Layout: &c.rowLayout,
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     c.layoutItems,
		Gap:       u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}
