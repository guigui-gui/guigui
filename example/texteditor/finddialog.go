// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"fmt"
	"image"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

var (
	findDialogEventFindNext     = guigui.GenerateEventKey()
	findDialogEventFindPrev     = guigui.GenerateEventKey()
	findDialogEventQueryChanged = guigui.GenerateEventKey()
	findDialogEventClose        = guigui.GenerateEventKey()
)

type findDialog struct {
	guigui.DefaultWidget

	popup   basicwidget.Popup
	content findDialogContent
}

// OnFindNext registers the handler for the down-arrow button (and Enter on
// the query input). The handler receives the current query string.
func (f *findDialog) OnFindNext(fn func(context *guigui.Context, query string)) {
	guigui.SetEventHandler(f, findDialogEventFindNext, fn)
}

// OnFindPrev registers the handler for the up-arrow button (and Shift+Enter
// on the query input).
func (f *findDialog) OnFindPrev(fn func(context *guigui.Context, query string)) {
	guigui.SetEventHandler(f, findDialogEventFindPrev, fn)
}

// OnQueryChanged registers the handler that fires whenever the query input
// changes value, even mid-typing.
func (f *findDialog) OnQueryChanged(fn func(context *guigui.Context, query string)) {
	guigui.SetEventHandler(f, findDialogEventQueryChanged, fn)
}

// OnClose registers the handler that fires after the popup closes.
func (f *findDialog) OnClose(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(f, findDialogEventClose, fn)
}

func (f *findDialog) SetOpen(open bool) {
	f.popup.SetOpen(open)
}

func (f *findDialog) IsOpen() bool {
	return f.popup.IsOpen()
}

func (f *findDialog) Query() string {
	return f.content.queryInput.Value()
}

// SetCount displays the current match index (1-based) and the total number
// of matches. Pass total = 0 to clear the count display.
func (f *findDialog) SetCount(current, total int) {
	if total == 0 {
		f.content.count.SetValue("")
		return
	}
	f.content.count.SetValue(fmt.Sprintf("%d / %d", current, total))
}

func (f *findDialog) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&f.popup)
	f.content.dialog = f
	f.popup.SetContent(&f.content)
	f.popup.SetCloseByClickingOutside(false)
	f.popup.SetModal(false)
	f.popup.SetAnimated(true)
	f.popup.OnClose(func(context *guigui.Context, reason basicwidget.PopupCloseReason) {
		guigui.DispatchEvent(f, findDialogEventClose)
	})
	return nil
}

func (f *findDialog) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	size := f.content.Measure(context, guigui.Constraints{})
	app := context.AppBounds()
	u := basicwidget.UnitSize(context)
	pos := image.Pt(app.Max.X-size.X-u/2, app.Min.Y+u/2)
	layouter.LayoutWidget(&f.popup, image.Rectangle{Min: pos, Max: pos.Add(size)})
}

type findDialogContent struct {
	guigui.DefaultWidget

	dialog *findDialog

	queryInput  basicwidget.TextInput
	count       basicwidget.Text
	prevButton  basicwidget.Button
	nextButton  basicwidget.Button
	closeButton basicwidget.Button

	buttonGroupItems  []guigui.LinearLayoutItem
	buttonGroupLayout guigui.LinearLayout
	rowItems          []guigui.LinearLayoutItem
}

func (c *findDialogContent) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&c.queryInput)
	adder.AddWidget(&c.count)
	adder.AddWidget(&c.prevButton)
	adder.AddWidget(&c.nextButton)
	adder.AddWidget(&c.closeButton)

	c.queryInput.OnValueChanged(func(context *guigui.Context, text string, committed bool) {
		guigui.DispatchEvent(c.dialog, findDialogEventQueryChanged, text)
	})
	c.queryInput.OnHandleButtonInput(func(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			if ebiten.IsKeyPressed(ebiten.KeyShift) {
				guigui.DispatchEvent(c.dialog, findDialogEventFindPrev, c.queryInput.Value())
			} else {
				guigui.DispatchEvent(c.dialog, findDialogEventFindNext, c.queryInput.Value())
			}
			return guigui.HandleInputByWidget(&c.queryInput)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			c.dialog.popup.SetOpen(false)
			return guigui.HandleInputByWidget(&c.queryInput)
		}
		// Cmd/Ctrl+F toggles: when the popup is already open, treat the same
		// shortcut as a close.
		if cmdPressed() && inpututil.IsKeyJustPressed(ebiten.KeyF) {
			c.dialog.popup.SetOpen(false)
			return guigui.HandleInputByWidget(&c.queryInput)
		}
		return guigui.HandleInputResult{}
	})

	c.count.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	c.count.SetHorizontalAlign(basicwidget.HorizontalAlignEnd)

	cm := context.ColorMode()
	c.prevButton.SetIcon(loadImage("keyboard_arrow_up", cm))
	c.prevButton.SetSharpCorners(basicwidget.Corners{TopEnd: true, BottomEnd: true})
	c.prevButton.OnDown(func(context *guigui.Context) {
		guigui.DispatchEvent(c.dialog, findDialogEventFindPrev, c.queryInput.Value())
	})

	c.nextButton.SetIcon(loadImage("keyboard_arrow_down", cm))
	c.nextButton.SetSharpCorners(basicwidget.Corners{TopStart: true, BottomStart: true})
	c.nextButton.OnDown(func(context *guigui.Context) {
		guigui.DispatchEvent(c.dialog, findDialogEventFindNext, c.queryInput.Value())
	})

	c.closeButton.SetIcon(loadImage("close_small", cm))
	c.closeButton.OnDown(func(context *guigui.Context) {
		c.dialog.popup.SetOpen(false)
	})

	return nil
}

func (c *findDialogContent) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	u := basicwidget.UnitSize(context)
	return image.Pt(16*u, 2*u)
}

func (c *findDialogContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	btnW := u

	// prev + next share an inner row with no gap so the rounded outer corners
	// make them read as a single up/down control.
	c.buttonGroupItems = slices.Delete(c.buttonGroupItems, 0, len(c.buttonGroupItems))
	c.buttonGroupItems = append(c.buttonGroupItems,
		guigui.LinearLayoutItem{
			Widget: &c.prevButton,
			Size:   guigui.FixedSize(btnW),
		},
		guigui.LinearLayoutItem{
			Widget: &c.nextButton,
			Size:   guigui.FixedSize(btnW),
		},
	)
	c.buttonGroupLayout = guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     c.buttonGroupItems,
	}

	c.rowItems = slices.Delete(c.rowItems, 0, len(c.rowItems))
	c.rowItems = append(c.rowItems,
		guigui.LinearLayoutItem{
			Widget: &c.queryInput,
			Size:   guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: &c.count,
			Size:   guigui.FixedSize(2 * u),
		},
		guigui.LinearLayoutItem{
			Layout: &c.buttonGroupLayout,
			Size:   guigui.FixedSize(2 * btnW),
		},
		guigui.LinearLayoutItem{
			Widget: &c.closeButton,
			Size:   guigui.FixedSize(btnW),
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     c.rowItems,
		Gap:       u / 4,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}
