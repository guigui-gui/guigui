// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"image"
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

var confirmDialogEventClose = guigui.GenerateEventKey()

// confirmResult identifies how the user dismissed the dialog. The zero value
// is [confirmResultCancel] so that an outside click or any close path that
// doesn't go through a button reports as cancellation.
type confirmResult int

const (
	confirmResultCancel confirmResult = iota
	confirmResultSave
	confirmResultDontSave
)

// confirmDialog is the in-app modal "you have unsaved changes" prompt with
// the standard Save / Don't save / Cancel triad. Using Guigui widgets keeps
// everything on the main goroutine, sidestepping the macOS dispatch-queue
// timing issue that affects native dialogs called from a non-main goroutine.
type confirmDialog struct {
	guigui.DefaultWidget

	popup   basicwidget.Popup
	content confirmDialogContent

	// pendingResult is the result selected by the user, latched in the button
	// OnDown before SetOpen(false). The popup's OnClose then dispatches it.
	pendingResult confirmResult
}

// SetMessage sets the prompt text shown above the buttons.
func (c *confirmDialog) SetMessage(message string) {
	c.content.message.SetValue(message)
}

// SetOpen shows or hides the dialog. Opening also resets the pending result
// to Cancel so a previous selection that didn't get dispatched can't leak.
func (c *confirmDialog) SetOpen(open bool) {
	if open {
		c.pendingResult = confirmResultCancel
	}
	c.popup.SetOpen(open)
}

// OnClose registers the handler that fires whenever the dialog closes,
// reporting which button the user picked.
func (c *confirmDialog) OnClose(fn func(context *guigui.Context, result confirmResult)) {
	guigui.SetEventHandler(c, confirmDialogEventClose, fn)
}

func (c *confirmDialog) IsOpen() bool {
	return c.popup.IsOpen()
}

func (c *confirmDialog) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&c.popup)
	c.content.dialog = c
	c.popup.SetContent(&c.content)
	c.popup.SetModal(true)
	c.popup.SetBackgroundDark(true)
	c.popup.SetCloseByClickingOutside(true)
	c.popup.SetAnimated(true)
	c.popup.OnClose(func(context *guigui.Context, reason basicwidget.PopupCloseReason) {
		result := c.pendingResult
		c.pendingResult = confirmResultCancel
		guigui.DispatchEvent(c, confirmDialogEventClose, result)
	})
	return nil
}

func (c *confirmDialog) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	size := c.content.Measure(context, guigui.Constraints{})
	app := context.AppBounds()
	pos := image.Pt(
		app.Min.X+(app.Dx()-size.X)/2,
		app.Min.Y+(app.Dy()-size.Y)/2,
	)
	layouter.LayoutWidget(&c.popup, image.Rectangle{Min: pos, Max: pos.Add(size)})
}

type confirmDialogContent struct {
	guigui.DefaultWidget

	dialog *confirmDialog

	message       basicwidget.Text
	saveButton    basicwidget.Button
	discardButton basicwidget.Button
	cancelButton  basicwidget.Button

	layoutItems []guigui.LinearLayoutItem
}

func (c *confirmDialogContent) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&c.message)
	adder.AddWidget(&c.saveButton)
	adder.AddWidget(&c.discardButton)
	adder.AddWidget(&c.cancelButton)

	c.message.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	c.message.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)

	c.saveButton.SetText("Save")
	c.saveButton.OnDown(func(context *guigui.Context) {
		c.dialog.pendingResult = confirmResultSave
		c.dialog.popup.SetOpen(false)
	})

	c.discardButton.SetText("Don't save")
	c.discardButton.OnDown(func(context *guigui.Context) {
		c.dialog.pendingResult = confirmResultDontSave
		c.dialog.popup.SetOpen(false)
	})

	c.cancelButton.SetText("Cancel")
	c.cancelButton.OnDown(func(context *guigui.Context) {
		// Leave pending at confirmResultCancel.
		c.dialog.popup.SetOpen(false)
	})

	return nil
}

func (c *confirmDialogContent) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	u := basicwidget.UnitSize(context)
	return image.Pt(9*u, 7*u)
}

func (c *confirmDialogContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	btnH := c.saveButton.Measure(context, guigui.Constraints{}).Y

	c.layoutItems = slices.Delete(c.layoutItems, 0, len(c.layoutItems))
	// Explicit spacer items between rows instead of LinearLayout.Gap, so the
	// extra room between Cancel and the proceed actions is visible at the
	// call site rather than implicit in a global gap value.
	c.layoutItems = append(c.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &c.message,
			Size:   guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Size: guigui.FixedSize(u / 2),
		},
		guigui.LinearLayoutItem{
			Widget: &c.saveButton,
			Size:   guigui.FixedSize(btnH),
		},
		guigui.LinearLayoutItem{
			Size: guigui.FixedSize(u / 4),
		},
		guigui.LinearLayoutItem{
			Widget: &c.discardButton,
			Size:   guigui.FixedSize(btnH),
		},
		guigui.LinearLayoutItem{
			Size: guigui.FixedSize(u / 2),
		},
		guigui.LinearLayoutItem{
			Widget: &c.cancelButton,
			Size:   guigui.FixedSize(btnH),
		},
	)

	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     c.layoutItems,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}
