// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"

	"github.com/guigui-gui/guigui"
)

const (
	dropdownListEventItemSelected = "itemSelected"
)

type DropdownListItem[T comparable] struct {
	Text         string
	TextColor    color.Color
	Header       bool
	Content      guigui.Widget
	Unselectable bool
	Border       bool
	Disabled     bool
	Value        T
}

type DropdownList[T comparable] struct {
	guigui.DefaultWidget

	button        Button
	buttonContent dropdownListButtonContent
	popupMenu     PopupMenu[T]

	items []DropdownListItem[T]
}

func (d *DropdownList[T]) SetOnItemSelected(f func(index int)) {
	guigui.RegisterEventHandler(d, dropdownListEventItemSelected, f)
}

func (d *DropdownList[T]) updatePopupMenuitems() {
	var popupMenuItems []PopupMenuItem[T]
	for _, item := range d.items {
		pmItem := PopupMenuItem[T](item)
		if !d.popupMenu.IsOpen() {
			pmItem.Content = nil
		}
		popupMenuItems = append(popupMenuItems, pmItem)
	}
	d.popupMenu.SetItems(popupMenuItems)
}

func (d *DropdownList[T]) updateChildren(context *guigui.Context) {
	d.updatePopupMenuitems()
	if index := d.popupMenu.SelectedItemIndex(); index >= 0 {
		if content := d.items[index].Content; content != nil {
			if d.popupMenu.IsOpen() {
				d.buttonContent.SetContentWidth(content.Measure(context, guigui.Constraints{}).X)
			} else {
				d.buttonContent.SetContent(content)
			}
			d.buttonContent.SetText("")
		} else {
			d.buttonContent.SetContent(nil)
			d.buttonContent.SetText(d.items[index].Text)
		}
	} else {
		d.buttonContent.SetContent(nil)
		d.buttonContent.SetText("")
	}
	d.button.SetContent(&d.buttonContent)
}

func (d *DropdownList[T]) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&d.button)
	adder.AddChild(&d.popupMenu)
}

func (d *DropdownList[T]) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	d.updateChildren(context)

	d.button.SetOnDown(func() {
		d.popupMenu.SetOpen(context, true)
	})
	d.button.setKeepPressed(d.popupMenu.IsOpen())
	d.button.SetIconAlign(IconAlignEnd)

	d.popupMenu.SetOnItemSelected(func(index int) {
		guigui.DispatchEventHandler(d, dropdownListEventItemSelected, index)
	})
	if !d.popupMenu.IsOpen() {
		d.popupMenu.SetCheckmarkIndex(d.SelectedItemIndex())
	}

	return nil
}

func (d *DropdownList[T]) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	p := widgetBounds.Bounds().Min
	layouter.LayoutWidget(&d.button, image.Rectangle{
		Min: p,
		Max: p.Add(d.button.Measure(context, guigui.Constraints{})),
	})

	p = widgetBounds.Bounds().Min
	p.X -= listItemCheckmarkSize(context) + listItemTextAndImagePadding(context)
	p.X = max(p.X, 0)
	// TODO: The item content in a button and a dropdown list might have different heights. Handle this case properly.
	if y, ok := d.popupMenu.itemYFromIndexForMenu(context, max(0, d.popupMenu.SelectedItemIndex())); ok {
		p.Y -= y
	}
	p.Y = max(p.Y, 0)
	layouter.LayoutWidget(&d.popupMenu, image.Rectangle{
		Min: p,
		Max: p.Add(d.popupMenu.Measure(context, guigui.Constraints{})),
	})
}

func (d *DropdownList[T]) SetItems(items []DropdownListItem[T]) {
	d.items = adjustSliceSize(d.items, len(items))
	copy(d.items, items)
	d.updatePopupMenuitems()
}

func (d *DropdownList[T]) SetItemsByStrings(items []string) {
	d.items = adjustSliceSize(d.items, len(items))
	for i, str := range items {
		d.items[i] = DropdownListItem[T]{
			Text: str,
		}
	}
	d.updatePopupMenuitems()
}

func (d *DropdownList[T]) SelectedItem() (DropdownListItem[T], bool) {
	item, ok := d.popupMenu.SelectedItem()
	if !ok {
		return DropdownListItem[T]{}, false
	}
	return DropdownListItem[T](item), true
}

func (d *DropdownList[T]) ItemByIndex(index int) (DropdownListItem[T], bool) {
	item, ok := d.popupMenu.ItemByIndex(index)
	if !ok {
		return DropdownListItem[T]{}, false
	}
	return DropdownListItem[T](item), true
}

func (d *DropdownList[T]) SelectedItemIndex() int {
	return d.popupMenu.SelectedItemIndex()
}

func (d *DropdownList[T]) SelectItemByIndex(index int) {
	d.popupMenu.SelectItemByIndex(index)
}

func (d *DropdownList[T]) SelectItemByValue(value T) {
	d.popupMenu.SelectItemByValue(value)
}

func (d *DropdownList[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	// Update the button content to reflect the current selected item.
	d.updateChildren(context)
	return d.button.Measure(context, constraints)
}

func (d *DropdownList[T]) ItemTextColor(context *guigui.Context, index int) color.Color {
	return d.popupMenu.ItemTextColor(context, index)
}

func (d *DropdownList[T]) IsPopupOpen() bool {
	return d.popupMenu.IsOpen()
}

type dropdownListButtonContent struct {
	guigui.DefaultWidget

	content           guigui.Widget
	contentWidthPlus1 int
	text              Text
	icon              Image
}

func (d *dropdownListButtonContent) SetContent(content guigui.Widget) {
	d.content = content
	d.contentWidthPlus1 = 0
}

func (d *dropdownListButtonContent) SetContentWidth(width int) {
	d.content = nil
	d.contentWidthPlus1 = width + 1
}

func (d *dropdownListButtonContent) SetText(text string) {
	d.text.SetValue(text)
}

func (d *dropdownListButtonContent) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	if d.content != nil {
		adder.AddChild(d.content)
	}
	adder.AddChild(&d.text)
	adder.AddChild(&d.icon)
}

func (d *dropdownListButtonContent) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	d.text.SetVerticalAlign(VerticalAlignMiddle)

	img, err := theResourceImages.Get("unfold_more", context.ColorMode())
	if err != nil {
		return err
	}
	d.icon.SetImage(img)
	return nil
}

func (d *dropdownListButtonContent) layout(context *guigui.Context) guigui.Layout {
	padding := guigui.Padding{
		End: buttonEdgeAndImagePadding(context),
	}

	var contentWidget guigui.Widget
	var gap int
	if d.content != nil {
		contentWidget = d.content
	} else {
		contentWidget = &d.text
		padding.Start = buttonEdgeAndTextPadding(context)
		gap = buttonTextAndImagePadding(context)
	}
	iconSize := defaultIconSize(context)
	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Gap:       gap,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: contentWidget,
			},
			{
				Widget: &d.icon,
				Size:   guigui.FixedSize(iconSize),
			},
		},
		Padding: padding,
	}
}

func (d *dropdownListButtonContent) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	d.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (d *dropdownListButtonContent) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return d.layout(context).Measure(context, constraints)
}

func DropdownListButtonTextPadding(context *guigui.Context) guigui.Padding {
	return guigui.Padding{
		Start:  buttonEdgeAndTextPadding(context),
		Top:    0,
		End:    buttonTextAndImagePadding(context),
		Bottom: 0,
	}
}
