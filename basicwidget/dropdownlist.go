// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/guigui"
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

func (d *DropdownList[T]) updateButtonContent() {
	if item, ok := d.popupMenu.SelectedItem(); ok {
		// item.Content is not available as this is used in the menu.
		// TODO: Fix this issue (#182).
		d.buttonContent.text.SetValue(item.Text)
	} else {
		d.buttonContent.text.SetValue("")
	}
	d.button.SetContent(&d.buttonContent)
}

func (d *DropdownList[T]) AppendChildWidgets(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
	appender.AppendChildWidget(&d.button)
	appender.AppendChildWidget(&d.popupMenu)
}

func (d *DropdownList[T]) Build(context *guigui.Context) error {
	d.updateButtonContent()

	d.button.SetOnDown(func() {
		d.popupMenu.Open(context)
	})
	d.button.setKeepPressed(d.popupMenu.IsOpen())
	d.button.SetIconAlign(IconAlignEnd)

	context.SetPosition(&d.button, context.Position(d))

	d.popupMenu.SetOnItemSelected(func(index int) {
		guigui.DispatchEventHandler(d, dropdownListEventItemSelected, index)
	})
	if !d.popupMenu.IsOpen() {
		d.popupMenu.SetCheckmarkIndex(d.SelectedItemIndex())
	}

	pt := context.Position(d)
	pt.X -= listItemCheckmarkSize(context) + listItemTextAndImagePadding(context)
	pt.X = max(pt.X, 0)
	pt.Y -= RoundedCornerRadius(context)
	pt.Y += int((float64(context.ActualSize(d).Y) - LineHeight(context)) / 2)
	pt.Y -= int(float64(d.popupMenu.SelectedItemIndex()) * LineHeight(context))
	pt.Y = max(pt.Y, 0)
	context.SetPosition(&d.popupMenu, pt)

	return nil
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

func (d *DropdownList[T]) DefaultSize(context *guigui.Context) image.Point {
	// Update the button content to reflect the current selected item.
	d.updateButtonContent()
	return d.button.DefaultSize(context)
}

func (d *DropdownList[T]) ItemTextColor(context *guigui.Context, index int) color.Color {
	return d.popupMenu.ItemTextColor(context, index)
}

func (d *DropdownList[T]) IsPopupOpen() bool {
	return d.popupMenu.IsOpen()
}

type dropdownListButtonContent struct {
	guigui.DefaultWidget

	content guigui.Widget
	text    Text
	image   Image
}

func (d *dropdownListButtonContent) AppendChildWidgets(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
	if d.content != nil {
		appender.AppendChildWidget(d.content)
	}
	appender.AppendChildWidget(&d.text)
	appender.AppendChildWidget(&d.image)
}

func (d *dropdownListButtonContent) Build(context *guigui.Context) error {
	paddingStartX := buttonEdgeAndTextPadding(context)

	bounds := context.Bounds(d)

	if d.content != nil {
		contentSize := d.content.DefaultSize(context)
		contentP := image.Point{
			X: bounds.Min.X + paddingStartX,
			Y: bounds.Min.Y + (bounds.Dy()-contentSize.Y)/2,
		}
		context.SetPosition(d.content, contentP)
	}

	textSize := d.text.DefaultSize(context)
	textP := image.Point{
		X: bounds.Min.X + paddingStartX,
		Y: bounds.Min.Y + (bounds.Dy()-textSize.Y)/2,
	}
	context.SetPosition(&d.text, textP)

	img, err := theResourceImages.Get("unfold_more", context.ColorMode())
	if err != nil {
		return err
	}
	d.image.SetImage(img)

	iconSize := defaultIconSize(context)
	imgP := image.Point{
		X: bounds.Max.X - buttonEdgeAndImagePadding(context) - iconSize,
		Y: bounds.Min.Y + (bounds.Dy()-iconSize)/2,
	}
	imgBounds := image.Rectangle{
		Min: imgP,
		Max: imgP.Add(image.Pt(iconSize, iconSize)),
	}
	context.SetBounds(&d.image, imgBounds, d)

	return nil
}

func (d *dropdownListButtonContent) DefaultSize(context *guigui.Context) image.Point {
	paddingStartX := buttonEdgeAndTextPadding(context)
	paddingEndX := buttonEdgeAndImagePadding(context)

	var contentSize image.Point
	if d.content != nil {
		contentSize = context.ActualSize(d.content)
	}
	textSize := d.text.DefaultSize(context)
	iconSize := defaultIconSize(context)
	return image.Point{
		X: paddingStartX + max(contentSize.X, textSize.X) + buttonTextAndImagePadding(context) + iconSize + paddingEndX,
		Y: max(contentSize.Y, textSize.Y, iconSize),
	}
}
