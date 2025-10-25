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
}

func (d *DropdownList[T]) SetOnItemSelected(f func(index int)) {
	guigui.RegisterEventHandler(d, dropdownListEventItemSelected, f)
}

func (d *DropdownList[T]) updateButtonContent(context *guigui.Context) {
	if item, ok := d.popupMenu.SelectedItem(); ok {
		if item.Content != nil {
			if d.popupMenu.IsOpen() {
				d.buttonContent.SetContentWidth(item.Content.Measure(context, guigui.Constraints{}).X)
			} else {
				d.buttonContent.SetContent(item.Content)
			}
			d.buttonContent.SetText("")
		} else {
			d.buttonContent.SetContent(nil)
			d.buttonContent.SetText(item.Text)
		}
	} else {
		d.buttonContent.SetContent(nil)
		d.buttonContent.SetText("")
	}
	d.button.SetContent(&d.buttonContent)
}

func (d *DropdownList[T]) AddChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, adder *guigui.ChildAdder) {
	adder.AddChild(&d.button)
	adder.AddChild(&d.popupMenu)
}

func (d *DropdownList[T]) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	d.updateButtonContent(context)

	d.button.SetOnDown(func() {
		d.popupMenu.SetOpen(true)
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

func (d *DropdownList[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, widget guigui.Widget) image.Rectangle {
	switch widget {
	case &d.button:
		p := widgetBounds.Bounds().Min
		return image.Rectangle{
			Min: p,
			Max: p.Add(d.button.Measure(context, guigui.Constraints{})),
		}
	case &d.popupMenu:
		p := widgetBounds.Bounds().Min
		p.X -= listItemCheckmarkSize(context) + listItemTextAndImagePadding(context)
		p.X = max(p.X, 0)
		p.Y -= RoundedCornerRadius(context)
		p.Y += int((float64(widgetBounds.Bounds().Dy()) - LineHeight(context)) / 2)
		p.Y -= max(0, d.popupMenu.SelectedItemIndex()) * int(LineHeight(context)+2*listItemTextPadding(context))
		p.Y = max(p.Y, 0)
		return image.Rectangle{
			Min: p,
			Max: p.Add(d.popupMenu.Measure(context, guigui.Constraints{})),
		}
	}
	return image.Rectangle{}
}

func (d *DropdownList[T]) SetItems(items []DropdownListItem[T]) {
	var popupMenuItems []PopupMenuItem[T]
	for _, item := range items {
		popupMenuItems = append(popupMenuItems, PopupMenuItem[T](item))
	}
	d.popupMenu.SetItems(popupMenuItems)
}

func (d *DropdownList[T]) SetItemsByStrings(items []string) {
	d.popupMenu.SetItemsByStrings(items)
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
	d.updateButtonContent(context)
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
	image             Image
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

func (d *dropdownListButtonContent) AddChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, adder *guigui.ChildAdder) {
	if d.content != nil {
		adder.AddChild(d.content)
	}
	adder.AddChild(&d.text)
	adder.AddChild(&d.image)
}

func (d *dropdownListButtonContent) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	img, err := theResourceImages.Get("unfold_more", context.ColorMode())
	if err != nil {
		return err
	}
	d.image.SetImage(img)
	return nil
}

func (d *dropdownListButtonContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, widget guigui.Widget) image.Rectangle {
	bounds := widgetBounds.Bounds()
	paddingStartX := buttonEdgeAndTextPadding(context)

	switch widget {
	case d.content:
		contentSize := d.content.Measure(context, guigui.Constraints{})
		contentP := image.Point{
			X: bounds.Min.X + paddingStartX,
			Y: bounds.Min.Y + (bounds.Dy()-contentSize.Y)/2,
		}
		return image.Rectangle{
			Min: contentP,
			Max: contentP.Add(contentSize),
		}
	case &d.text:
		textSize := d.text.Measure(context, guigui.Constraints{})
		textP := image.Point{
			X: bounds.Min.X + paddingStartX,
			Y: bounds.Min.Y + (bounds.Dy()-textSize.Y)/2,
		}
		return image.Rectangle{
			Min: textP,
			Max: textP.Add(textSize),
		}
	case &d.image:
		iconSize := defaultIconSize(context)
		imgP := image.Point{
			X: bounds.Max.X - buttonEdgeAndImagePadding(context) - iconSize,
			Y: bounds.Min.Y + (bounds.Dy()-iconSize)/2,
		}
		return image.Rectangle{
			Min: imgP,
			Max: imgP.Add(image.Pt(iconSize, iconSize)),
		}
	}
	return image.Rectangle{}
}

func (d *dropdownListButtonContent) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	paddingStartX := buttonEdgeAndTextPadding(context)
	paddingEndX := buttonEdgeAndImagePadding(context)

	var contentSize image.Point
	if d.content != nil {
		contentSize = d.content.Measure(context, guigui.Constraints{})
	}
	if d.contentWidthPlus1 > 0 {
		contentSize.X = d.contentWidthPlus1 - 1
	}
	textSize := d.text.Measure(context, constraints)
	iconSize := defaultIconSize(context)
	return image.Point{
		X: paddingStartX + max(contentSize.X, textSize.X) + buttonTextAndImagePadding(context) + iconSize + paddingEndX,
		Y: max(contentSize.Y, textSize.Y, iconSize),
	}
}
