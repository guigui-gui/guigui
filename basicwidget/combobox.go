// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image"
	"slices"
	"strings"

	"github.com/guigui-gui/guigui"
)

var (
	comboboxEventValueChanged guigui.EventKey = guigui.GenerateEventKey()
)

// Combobox is a composite widget that combines a [TextInput] with a [PopupMenu].
// The popup menu shows filtered items based on the current input text.
// When the user focuses the text input, the popup opens below (or above) the text input.
// The popup is modeless, so the text input retains focus while the popup is shown.
// The popup closes when the text input loses focus.
type Combobox struct {
	guigui.DefaultWidget

	textInput TextInput
	popupMenu PopupMenu[string]

	items          []string
	filteredItems  []PopupMenuItem[string]
	allowFreeInput bool
	lastValidValue string

	prevFocused bool

	onTextInputValueChanged func(context *guigui.Context, text string, committed bool)
	onPopupMenuItemSelected func(context *guigui.Context, index int)
}

// SetItems sets the list of items for the combobox.
func (c *Combobox) SetItems(items []string) {
	c.items = adjustSliceSize(c.items, len(items))
	copy(c.items, items)
}

// SetAllowFreeInput sets whether the combobox allows values that are not in the items list.
// When false, the value is reverted to the last valid value on commit if it does not match any item.
func (c *Combobox) SetAllowFreeInput(allow bool) {
	if c.allowFreeInput == allow {
		return
	}
	c.allowFreeInput = allow
	if !allow {
		// If the current value is not in the items, clear it.
		v := c.textInput.Value()
		if v == "" {
			return
		}
		if slices.Contains(c.items, v) {
			return
		}
		c.textInput.ForceSetValue("")
		c.lastValidValue = ""
	}
}

// TextInput returns the internal [TextInput] widget for customization.
func (c *Combobox) TextInput() *TextInput {
	return &c.textInput
}

// IsError reports whether the combobox is in the error state.
func (c *Combobox) IsError() bool {
	return c.textInput.IsError()
}

// SetError sets whether the combobox is in the error state.
// When the error state is true, the combobox border is drawn in a danger color.
func (c *Combobox) SetError(hasError bool) {
	c.textInput.SetError(hasError)
}

// SupportText returns the support text displayed below the combobox.
func (c *Combobox) SupportText() string {
	return c.textInput.SupportText()
}

// SetSupportText sets the support text displayed below the combobox.
// The support text is shown in a subdued color, or in a danger color when the error state is true.
func (c *Combobox) SetSupportText(text string) {
	c.textInput.SetSupportText(text)
}

// Value returns the current text value.
func (c *Combobox) Value() string {
	return c.textInput.Value()
}

// SetValue sets the text value.
func (c *Combobox) SetValue(value string) {
	c.textInput.SetValue(value)
	c.lastValidValue = value
}

// OnValueChanged sets the event handler that is called when the combobox value changes.
// The handler receives the current text and whether the change is committed.
func (c *Combobox) OnValueChanged(f func(context *guigui.Context, value string, committed bool)) {
	guigui.SetEventHandler(c, comboboxEventValueChanged, f)
}

func (c *Combobox) updateFilteredItems() {
	input := strings.ToLower(c.textInput.Value())
	c.filteredItems = c.filteredItems[:0]
	for _, item := range c.items {
		if input == "" || strings.Contains(strings.ToLower(item), input) {
			c.filteredItems = append(c.filteredItems, PopupMenuItem[string]{
				Text:  item,
				Value: item,
			})
		}
	}
	// Don't update the popup with an empty list. This keeps the previous items
	// visible during the fade-out animation instead of showing a thin empty popup.
	if len(c.filteredItems) > 0 {
		c.popupMenu.SetItems(c.filteredItems)
	}
}

func (c *Combobox) highlightClosestCandidate(input string) {
	if input == "" || len(c.filteredItems) == 0 {
		c.popupMenu.setKeyboardHighlightIndex(-1)
		return
	}
	// Prefer the first item with a prefix match.
	var bestIndex int
	for i, item := range c.filteredItems {
		if strings.HasPrefix(strings.ToLower(item.Text), input) {
			bestIndex = i
			break
		}
	}
	c.popupMenu.setKeyboardHighlightIndex(bestIndex)
}

func (c *Combobox) handleCommit(context *guigui.Context, text string) {
	if c.allowFreeInput {
		c.lastValidValue = text
		guigui.DispatchEvent(c, comboboxEventValueChanged, text, true)
		return
	}

	// Accept empty text or an exact item match.
	if text == "" {
		c.lastValidValue = text
		guigui.DispatchEvent(c, comboboxEventValueChanged, text, true)
		return
	}
	if slices.Contains(c.items, text) {
		c.lastValidValue = text
		guigui.DispatchEvent(c, comboboxEventValueChanged, text, true)
		return
	}

	// Revert to the last valid value.
	c.textInput.ForceSetValue(c.lastValidValue)
	guigui.DispatchEvent(c, comboboxEventValueChanged, c.lastValidValue, true)
}

func (c *Combobox) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&c.textInput)
	adder.AddWidget(&c.popupMenu)

	c.popupMenu.setModal(false)
	context.DelegateFocus(c, &c.textInput)
	context.SetButtonInputReceptive(c, c.popupMenu.IsOpen())

	c.updateFilteredItems()
	if len(c.filteredItems) == 0 && c.popupMenu.IsOpen() {
		c.popupMenu.SetOpen(false)
	}

	if c.onTextInputValueChanged == nil {
		c.onTextInputValueChanged = func(context *guigui.Context, text string, committed bool) {
			if committed {
				c.handleCommit(context, text)
				c.popupMenu.SetOpen(false)
				return
			}
			c.updateFilteredItems()
			c.highlightClosestCandidate(strings.ToLower(text))
			if len(c.filteredItems) == 0 {
				c.popupMenu.SetOpen(false)
			} else if !c.popupMenu.IsOpen() && context.IsFocusedOrHasFocusedChild(&c.textInput) {
				c.popupMenu.SetOpen(true)
			}
			guigui.DispatchEvent(c, comboboxEventValueChanged, text, false)
		}
	}
	c.textInput.OnValueChanged(c.onTextInputValueChanged)

	if c.onPopupMenuItemSelected == nil {
		c.onPopupMenuItemSelected = func(context *guigui.Context, index int) {
			if item, ok := c.popupMenu.ItemByIndex(index); ok {
				c.textInput.ForceSetValue(item.Text)
				c.textInput.setSelection(len(item.Text), len(item.Text))
				c.lastValidValue = item.Text
				guigui.DispatchEvent(c, comboboxEventValueChanged, item.Text, true)
			}
		}
	}
	c.popupMenu.OnItemSelected(c.onPopupMenuItemSelected)

	return nil
}

func (c *Combobox) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	bounds := widgetBounds.Bounds()
	layouter.LayoutWidget(&c.textInput, bounds)

	// Exclude the text input bounds from close-by-clicking-outside detection,
	// so clicking the text input while the popup is open doesn't close the popup.
	c.popupMenu.setCloseByClickingOutsideExcludedRect(bounds)

	// Match the popup width to the text input width.
	c.popupMenu.setMinWidth(bounds.Dx())
	popupSize := c.popupMenu.Measure(context, guigui.Constraints{})
	appBounds := context.AppBounds()

	// Position popup below the text input.
	popupPos := image.Pt(bounds.Min.X, bounds.Max.Y)

	// If there is not enough room below, position above.
	if popupPos.Y+popupSize.Y > appBounds.Max.Y {
		popupPos.Y = bounds.Min.Y - popupSize.Y
	}

	layouter.LayoutWidget(&c.popupMenu, image.Rectangle{
		Min: popupPos,
		Max: popupPos.Add(popupSize),
	})
}

func (c *Combobox) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	focused := context.IsFocusedOrHasFocusedChild(&c.textInput)

	if focused && !c.prevFocused {
		// Focus gained: open popup if there are items.
		c.updateFilteredItems()
		if len(c.filteredItems) > 0 {
			c.popupMenu.SetOpen(true)
		}
	}
	if !focused && c.prevFocused {
		// Focus lost: close popup.
		c.popupMenu.SetOpen(false)
	}

	c.prevFocused = focused
	return nil
}

func (c *Combobox) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return c.textInput.Measure(context, constraints)
}
