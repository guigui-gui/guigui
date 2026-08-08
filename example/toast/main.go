// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Toast struct {
	guigui.DefaultWidget

	popup   basicwidget.Popup
	content toastContent

	openedTicks   int
	durationTicks int
	message       string
	tint          color.Color
}

func (t *Toast) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.popup)

	t.content.OnClose(func(context *guigui.Context) {
		t.popup.SetOpen(false)
	})
	t.content.SetText(t.message)
	t.content.SetTintColor(t.tint)
	t.popup.SetContent(&t.content)
	t.popup.SetModal(false)
	t.popup.SetCloseByClickingOutside(false)
	t.popup.SetTintColor(t.tint)

	return nil
}

func (t *Toast) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&t.popup, widgetBounds.Bounds())
}

func (t *Toast) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if t.popup.IsOpen() && t.durationTicks > 0 {
		// Check if the cursor is on the toast by a simple geometric check.
		// IsHitAtCursor is not suitable here because the popup content is in a higher layer,
		// which blocks the Toast widget from being considered "hit".
		// TODO: There might be a need for an API to check another widget's hit test (e.g., WidgetBounds.IsWidgetHitAtCursor),
		// but this has not been decided yet.
		if image.Pt(ebiten.CursorPosition()).In(widgetBounds.VisibleBounds()) {
			// Reset the timer while the cursor is on the toast.
			t.openedTicks = 0
		} else {
			t.openedTicks++
			if t.openedTicks >= t.durationTicks {
				t.popup.SetOpen(false)
			}
		}
	}
	return nil
}

func (t *Toast) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.content.Measure(context, constraints)
}

func (t *Toast) IsOpen() bool {
	return t.popup.IsOpen()
}

func (t *Toast) SetMessage(message string) {
	t.message = message
	t.content.text.SetValue(message)
}

func (t *Toast) SetHasCloseButton(hasCloseButton bool) {
	t.content.hasCloseButton = hasCloseButton
}

func (t *Toast) SetDurationInTicks(ticks int) {
	t.durationTicks = ticks
}

func (t *Toast) SetTintColor(tint color.Color) {
	t.tint = tint
}

func (t *Toast) SetOpen(open bool) {
	if open {
		t.openedTicks = 0
	}
	t.popup.SetOpen(open)
}

var toastContentEventClose guigui.EventKey = guigui.GenerateEventKey()

type toastContent struct {
	guigui.DefaultWidget

	text        basicwidget.Text
	closeButton basicwidget.Button

	tint           color.Color
	hasCloseButton bool

	linearLayout      guigui.LinearLayout
	linearLayoutItems []guigui.LinearLayoutItem
}

func (t *toastContent) SetText(text string) {
	t.text.SetValue(text)
}

func (t *toastContent) SetTintColor(tint color.Color) {
	t.tint = tint
	t.closeButton.SetTintColor(tint)
}

func (t *toastContent) OnClose(f func(context *guigui.Context)) {
	guigui.SetEventHandler(t, toastContentEventClose, f)
}

func (t *toastContent) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.text)
	if t.hasCloseButton {
		adder.AddWidget(&t.closeButton)
	}

	var style basicwidget.TextStyle
	if t.tint != nil {
		style.SetColor(basicwidget.TextColorFromTint(context, t.tint))
	}
	t.text.SetBaseStyle(&style)
	t.text.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	t.closeButton.SetText("Close")
	t.closeButton.OnUp(func(context *guigui.Context) {
		guigui.DispatchEvent(t, toastContentEventClose)
	})

	return nil
}

func (t *toastContent) buildLayout(context *guigui.Context) {
	u := basicwidget.UnitSize(context)

	t.linearLayoutItems = slices.Delete(t.linearLayoutItems, 0, len(t.linearLayoutItems))
	t.linearLayoutItems = append(t.linearLayoutItems, guigui.LinearLayoutItem{
		Widget: &t.text,
		Size:   guigui.FlexibleSize(1),
	})
	if t.hasCloseButton {
		t.linearLayoutItems = append(t.linearLayoutItems, guigui.LinearLayoutItem{
			Widget: &t.closeButton,
		})
	}

	t.linearLayout = guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     t.linearLayoutItems,
		Gap:       u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 4,
			End:    u / 2,
			Bottom: u / 4,
		},
	}
}

func (t *toastContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	t.buildLayout(context)
	t.linearLayout.LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (t *toastContent) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	t.buildLayout(context)
	return t.linearLayout.Measure(context, constraints)
}

type Root struct {
	guigui.DefaultWidget

	background      basicwidget.Background
	tintControl     basicwidget.SegmentedControl[color.Color]
	showToastButton basicwidget.Button

	toasts guigui.WidgetSlice[*Toast]

	toastCounter int
	tint         color.Color
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.tintControl)
	adder.AddWidget(&r.showToastButton)
	for i := range r.toasts.Len() {
		adder.AddWidget(r.toasts.At(i))
	}

	r.tintControl.SetItems([]basicwidget.SegmentedControlItem[color.Color]{
		{Text: "Default", Value: nil},
		{Text: "Accent", Value: basicwidget.AccentTintColor()},
		{Text: "Info", Value: basicwidget.InfoTintColor()},
		{Text: "Success", Value: basicwidget.SuccessTintColor()},
		{Text: "Warning", Value: basicwidget.WarningTintColor()},
		{Text: "Danger", Value: basicwidget.DangerTintColor()},
	})
	r.tintControl.SelectItemByValue(r.tint)
	r.tintControl.OnItemSelected(func(context *guigui.Context, index int) {
		if item, ok := r.tintControl.ItemByIndex(index); ok {
			r.tint = item.Value
		}
	})

	r.showToastButton.SetText("Show Toast")
	r.showToastButton.OnDown(func(context *guigui.Context) {
		r.showToast()
	})

	return nil
}

func (r *Root) showToast() {
	r.toastCounter++

	// Append a new toast so that the slot order matches the order the toasts were shown in.
	idx := r.toasts.Len()
	r.toasts.SetLen(idx + 1)

	t := r.toasts.At(idx)
	t.SetMessage(fmt.Sprintf("Toast #%d", r.toastCounter))
	t.SetHasCloseButton(r.toastCounter%2 == 0)
	t.SetDurationInTicks(3 * ebiten.TPS())
	t.SetTintColor(r.tint)
	t.SetOpen(true)
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())

	u := basicwidget.UnitSize(context)
	appBounds := context.AppBounds()

	// Position the segmented control and button at the top-left with some padding.
	topLeft := appBounds.Min.Add(image.Pt(u, u))
	controlSize := r.tintControl.Measure(context, guigui.Constraints{})
	layouter.LayoutWidget(&r.tintControl, image.Rectangle{
		Min: topLeft,
		Max: topLeft.Add(controlSize),
	})

	buttonTop := topLeft.Y + controlSize.Y + u/2
	buttonSize := r.showToastButton.Measure(context, guigui.Constraints{})
	layouter.LayoutWidget(&r.showToastButton, image.Rectangle{
		Min: image.Pt(topLeft.X, buttonTop),
		Max: image.Pt(topLeft.X+buttonSize.X, buttonTop+buttonSize.Y),
	})

	// Stack the toasts upwards from the bottom in the order they were shown.
	// A closed toast keeps reserving its space so that the others stay at their position.
	margin := u / 2
	gap := u / 4
	baseY := appBounds.Max.Y - margin

	var bottomOffset int
	allClosed := true
	for i := range r.toasts.Len() {
		t := r.toasts.At(i)
		contentSize := t.Measure(context, guigui.Constraints{})
		if t.IsOpen() {
			allClosed = false
			bottomY := baseY - bottomOffset
			toastBounds := image.Rectangle{
				Min: image.Pt(appBounds.Max.X-margin-contentSize.X, bottomY-contentSize.Y),
				Max: image.Pt(appBounds.Max.X-margin, bottomY),
			}
			layouter.LayoutWidget(t, toastBounds)
		}
		bottomOffset += contentSize.Y + gap
	}

	// Reset when all toasts are closed.
	if allClosed {
		r.toasts.SetLen(0)
	}
}

func main() {
	op := &guigui.RunOptions{
		Title:         "Toast",
		WindowMinSize: image.Pt(400, 300),
	}
	if err := guigui.Run(&Root{}, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
