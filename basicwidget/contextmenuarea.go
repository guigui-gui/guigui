// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
)

// ContextMenuArea is a standalone widget that shows a popup menu when the user
// right-clicks inside the area specified by its bounds.
//
// ContextMenuArea shows a modeless popup that closes when the user clicks outside.
// The previously focused widget retains focus while the context menu is open,
// so that widgets like [TextInput] can continue to show their cursor and selection.
//
// Keyboard navigation (Up/Down/Enter/Escape) is supported via [guigui.Context.SetButtonInputReceptive].
//
// Use [ContextMenuArea.PopupMenu] to configure the menu items.
type ContextMenuArea[T comparable] struct {
	guigui.DefaultWidget

	popupMenu PopupMenu[T]

	menuPosition image.Point
}

// PopupMenu returns the popup menu so that the caller can configure its items
// and event handlers.
func (c *ContextMenuArea[T]) PopupMenu() *PopupMenu[T] {
	return &c.popupMenu
}

// Build implements [guigui.Widget.Build].
func (c *ContextMenuArea[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if c.popupMenu.IsOpen() {
		adder.AddWidget(&c.popupMenu)
	}
	c.popupMenu.setModal(false)
	context.SetButtonInputReceptive(c, c.popupMenu.IsOpen())
	return nil
}

// Layout implements [guigui.Widget.Layout].
func (c *ContextMenuArea[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	menuSize := c.popupMenu.Measure(context, guigui.Constraints{})
	layouter.LayoutWidget(&c.popupMenu, image.Rectangle{
		Min: c.menuPosition,
		Max: c.menuPosition.Add(menuSize),
	})
}

// HandlePointingInput implements [guigui.Widget.HandlePointingInput].
func (c *ContextMenuArea[T]) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if widgetBounds.IsHitAtCursor() {
			c.menuPosition = image.Pt(ebiten.CursorPosition())
			c.popupMenu.SetOpen(true)
			return guigui.HandleInputByWidget(c)
		}
	}
	return guigui.HandleInputResult{}
}
