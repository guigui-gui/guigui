// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"fmt"
	"image"
	"os"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
)

type Toast struct {
	guigui.DefaultWidget

	popup   basicwidget.Popup
	content toastContent

	openedAt  time.Time
	duration  time.Duration
	message   string
	colorType basicwidgetdraw.ColorType
}

func (t *Toast) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.popup)

	t.content.OnClose(func(context *guigui.Context) {
		t.popup.SetOpen(false)
	})
	t.content.text.SetValue(t.message)
	t.content.text.SetColor(basicwidgetdraw.TextColorWithType(context.ColorMode(), t.colorType))
	t.popup.SetContent(&t.content)
	t.popup.SetModal(false)
	t.popup.SetCloseByClickingOutside(false)
	t.popup.SetBackgroundColor(basicwidgetdraw.PopupBackgroundColorWithType(context.ColorMode(), t.colorType))

	return nil
}

func (t *Toast) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&t.popup, widgetBounds.Bounds())
}

func (t *Toast) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if t.popup.IsOpen() && t.duration > 0 {
		if time.Since(t.openedAt) >= t.duration {
			t.popup.SetOpen(false)
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

func (t *Toast) SetDuration(duration time.Duration) {
	t.duration = duration
}

func (t *Toast) BringToFrontLayer(context *guigui.Context) {
	t.popup.BringToFrontLayer(context)
}

func (t *Toast) SetColorType(colorType basicwidgetdraw.ColorType) {
	t.colorType = colorType
}

func (t *Toast) SetOpen(open bool) {
	if open {
		t.openedAt = time.Now()
	}
	t.popup.SetOpen(open)
}

var toastContentEventClose guigui.EventKey = guigui.GenerateEventKey()

type toastContent struct {
	guigui.DefaultWidget

	text        basicwidget.Text
	closeButton basicwidget.Button

	hasCloseButton bool
}

func (t *toastContent) OnClose(f func(context *guigui.Context)) {
	guigui.SetEventHandler(t, toastContentEventClose, f)
}

func (t *toastContent) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.text)
	if t.hasCloseButton {
		adder.AddWidget(&t.closeButton)
	}

	t.text.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	t.closeButton.SetText("Close")
	t.closeButton.OnUp(func(context *guigui.Context) {
		guigui.DispatchEvent(t, toastContentEventClose)
	})

	return nil
}

func (t *toastContent) layout(context *guigui.Context) guigui.LinearLayout {
	u := basicwidget.UnitSize(context)

	var items []guigui.LinearLayoutItem
	items = append(items, guigui.LinearLayoutItem{
		Widget: &t.text,
		Size:   guigui.FlexibleSize(1),
	})
	if t.hasCloseButton {
		items = append(items, guigui.LinearLayoutItem{
			Widget: &t.closeButton,
		})
	}

	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     items,
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
	t.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (t *toastContent) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.layout(context).Measure(context, constraints)
}

type Root struct {
	guigui.DefaultWidget

	background       basicwidget.Background
	colorTypeControl basicwidget.SegmentedControl[basicwidgetdraw.ColorType]
	showToastButton  basicwidget.Button

	toasts        guigui.WidgetSlice[*Toast]
	bottomOffsets []int

	toastCounter     int
	nextBottomOffset int
	colorType        basicwidgetdraw.ColorType
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.colorTypeControl)
	adder.AddWidget(&r.showToastButton)
	for i := range r.toasts.Len() {
		adder.AddWidget(r.toasts.At(i))
	}

	r.colorTypeControl.SetItems([]basicwidget.SegmentedControlItem[basicwidgetdraw.ColorType]{
		{Text: "Base", Value: basicwidgetdraw.ColorTypeBase},
		{Text: "Accent", Value: basicwidgetdraw.ColorTypeAccent},
		{Text: "Info", Value: basicwidgetdraw.ColorTypeInfo},
		{Text: "Success", Value: basicwidgetdraw.ColorTypeSuccess},
		{Text: "Warning", Value: basicwidgetdraw.ColorTypeWarning},
		{Text: "Danger", Value: basicwidgetdraw.ColorTypeDanger},
	})
	r.colorTypeControl.SelectItemByValue(r.colorType)
	r.colorTypeControl.OnItemSelected(func(context *guigui.Context, index int) {
		if item, ok := r.colorTypeControl.ItemByIndex(index); ok {
			r.colorType = item.Value
		}
	})

	r.showToastButton.SetText("Show Toast")
	r.showToastButton.OnUp(func(context *guigui.Context) {
		r.showToast(context)
	})

	// Call BringToFrontLayer in reverse order (topmost to bottommost).
	// This ensures the bottommost toast has the highest layer,
	// so an upper toast's downward shadow is behind the lower toast's content.
	for i := r.toasts.Len() - 1; i >= 0; i-- {
		t := r.toasts.At(i)
		if !t.IsOpen() {
			continue
		}
		t.BringToFrontLayer(context)
	}

	return nil
}

func (r *Root) showToast(context *guigui.Context) {
	r.toastCounter++
	hasCloseButton := r.toastCounter%2 == 0

	// Find a free slot.
	idx := -1
	for i := range r.toasts.Len() {
		if !r.toasts.At(i).IsOpen() {
			idx = i
			break
		}
	}

	// If no free slot, add a new one.
	if idx == -1 {
		idx = r.toasts.Len()
		r.toasts.SetLen(idx + 1)
	}

	t := r.toasts.At(idx)
	t.SetMessage(fmt.Sprintf("Toast #%d", r.toastCounter))
	t.SetHasCloseButton(hasCloseButton)
	t.SetDuration(3 * time.Second)
	t.SetColorType(r.colorType)

	// Grow the offsets slice if needed.
	if len(r.bottomOffsets) <= idx {
		r.bottomOffsets = slices.Grow(r.bottomOffsets, idx+1-len(r.bottomOffsets))[:idx+1]
	}
	r.bottomOffsets[idx] = r.nextBottomOffset

	u := basicwidget.UnitSize(context)
	gap := u / 4
	contentHeight := t.Measure(context, guigui.Constraints{}).Y
	r.nextBottomOffset += contentHeight + gap

	t.SetOpen(true)
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())

	u := basicwidget.UnitSize(context)
	appBounds := context.AppBounds()

	// Position the segmented control and button at the top-left with some padding.
	topLeft := appBounds.Min.Add(image.Pt(u, u))
	controlSize := r.colorTypeControl.Measure(context, guigui.Constraints{})
	layouter.LayoutWidget(&r.colorTypeControl, image.Rectangle{
		Min: topLeft,
		Max: topLeft.Add(controlSize),
	})

	buttonTop := topLeft.Y + controlSize.Y + u/2
	buttonSize := r.showToastButton.Measure(context, guigui.Constraints{})
	layouter.LayoutWidget(&r.showToastButton, image.Rectangle{
		Min: image.Pt(topLeft.X, buttonTop),
		Max: image.Pt(topLeft.X+buttonSize.X, buttonTop+buttonSize.Y),
	})

	// Position each toast based on its assigned bottom offset.
	// Toasts stay at their assigned position even when others close.
	margin := u / 2
	baseY := appBounds.Max.Y - margin

	allClosed := true
	for i := range r.toasts.Len() {
		t := r.toasts.At(i)
		if !t.IsOpen() {
			continue
		}
		allClosed = false

		contentSize := t.Measure(context, guigui.Constraints{})
		bottomY := baseY - r.bottomOffsets[i]
		toastBounds := image.Rectangle{
			Min: image.Pt(appBounds.Max.X-margin-contentSize.X, bottomY-contentSize.Y),
			Max: image.Pt(appBounds.Max.X-margin, bottomY),
		}
		layouter.LayoutWidget(t, toastBounds)
	}

	// Reset when all toasts are closed.
	if allClosed {
		r.toasts.SetLen(0)
		r.bottomOffsets = r.bottomOffsets[:0]
		r.nextBottomOffset = 0
	}
}

func main() {
	op := &guigui.RunOptions{
		Title:         "Toast",
		WindowMinSize: image.Pt(400, 300),
		RunGameOptions: &ebiten.RunGameOptions{
			ApplePressAndHoldEnabled: true,
		},
	}
	if err := guigui.Run(&Root{}, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
