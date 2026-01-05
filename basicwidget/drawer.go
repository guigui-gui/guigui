// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"

	"github.com/guigui-gui/guigui"
)

type DrawerEdge int

const (
	DrawerEdgeStart DrawerEdge = iota
	DrawerEdgeTop
	DrawerEdgeEnd
	DrawerEdgeBottom
)

type Drawer struct {
	guigui.DefaultWidget

	popup Popup
}

func (d *Drawer) SetOpen(open bool) {
	d.popup.SetOpen(open)
}

func (d *Drawer) IsOpen() bool {
	return d.popup.IsOpen()
}

func (d *Drawer) SetOnClose(onClose func(context *guigui.Context, reason PopupCloseReason)) {
	d.popup.SetOnClose(onClose)
}

func (d *Drawer) SetContent(widget guigui.Widget) {
	d.popup.SetContent(widget)
}

func (d *Drawer) SetBackgroundDark(dark bool) {
	d.popup.SetBackgroundDark(dark)
}

func (d *Drawer) SetBackgroundBlurred(blurred bool) {
	d.popup.SetBackgroundBlurred(blurred)
}

func (d *Drawer) SetCloseByClickingOutside(closeByClickingOutside bool) {
	d.popup.SetCloseByClickingOutside(closeByClickingOutside)
}

func (d *Drawer) SetAnimated(animateOnFading bool) {
	d.popup.SetAnimated(animateOnFading)
}

func (d *Drawer) SetBackgroundBounds(bounds image.Rectangle) {
	d.popup.SetBackgroundBounds(bounds)
}

func (d *Drawer) SetDrawerEdge(edge DrawerEdge) {
	d.popup.setDrawerEdge(edge)
}

func (d *Drawer) Build(context *guigui.Context, adder *guigui.WidgetAdder) error {
	adder.AddChild(&d.popup)

	d.popup.setStyle(popupStyleDrawer)

	return nil
}

func (d *Drawer) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&d.popup, widgetBounds.Bounds())
}

func (d *Drawer) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return d.popup.Measure(context, constraints)
}
