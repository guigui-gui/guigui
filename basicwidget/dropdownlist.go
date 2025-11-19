// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"
	"slices"

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

	items                 []DropdownListItem[T]
	popupMenuItems        []PopupMenuItem[T]
	popupMenuItemContents []dropdownListItemContent

	indexAtOpen int

	onDown                  func()
	onPopupMenuItemSelected func(index int)
}

func (d *DropdownList[T]) SetOnItemSelected(f func(index int)) {
	guigui.RegisterEventHandler(d, dropdownListEventItemSelected, f)
}

func (d *DropdownList[T]) updatePopupMenuitems() {
	d.popupMenuItems = adjustSliceSize(d.popupMenuItems, len(d.items))
	d.popupMenuItemContents = adjustSliceSize(d.popupMenuItemContents, len(d.items))
	for i, item := range d.items {
		pmItem := PopupMenuItem[T](item)
		if d.popupMenu.IsOpen() && pmItem.Content != nil {
			d.popupMenuItemContents[i].SetContent(pmItem.Content)
			pmItem.Content = &d.popupMenuItemContents[i]
		} else {
			d.popupMenuItemContents[i].SetContent(nil)
			pmItem.Content = nil
		}
		d.popupMenuItems[i] = pmItem
	}
	d.popupMenu.SetItems(d.popupMenuItems)
}

func (d *DropdownList[T]) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&d.button)
	adder.AddChild(&d.popupMenu)
}

func (d *DropdownList[T]) Update(context *guigui.Context) error {
	d.updatePopupMenuitems()
	if index := d.popupMenu.SelectedItemIndex(); index >= 0 {
		if content := d.items[index].Content; content != nil {
			if d.popupMenu.IsOpen() {
				d.buttonContent.SetContentSize(content.Measure(context, guigui.Constraints{}))
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

	if d.onDown == nil {
		d.onDown = func() {
			d.popupMenu.SetOpen(true)
			d.indexAtOpen = d.popupMenu.SelectedItemIndex()
		}
	}
	d.button.SetOnDown(d.onDown)
	d.button.setKeepPressed(d.popupMenu.IsOpen())
	d.button.SetIconAlign(IconAlignEnd)

	if d.onPopupMenuItemSelected == nil {
		d.onPopupMenuItemSelected = func(index int) {
			guigui.DispatchEventHandler(d, dropdownListEventItemSelected, index)
		}
	}
	d.popupMenu.SetOnItemSelected(d.onPopupMenuItemSelected)
	d.popupMenu.SetCheckmarkIndex(d.indexAtOpen)

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

	content      guigui.Widget
	dummyContent guigui.WidgetWithSize[*guigui.DefaultWidget]

	contentSizePlus1 image.Point
	text             Text
	icon             Image

	layoutItems []guigui.LinearLayoutItem
}

func (d *dropdownListButtonContent) SetContent(content guigui.Widget) {
	d.content = content
	d.contentSizePlus1 = image.Point{}
}

func (d *dropdownListButtonContent) SetContentSize(size image.Point) {
	d.content = nil
	d.contentSizePlus1 = size.Add(image.Point{1, 1})
}

func (d *dropdownListButtonContent) SetText(text string) {
	d.text.SetValue(text)
}

func (d *dropdownListButtonContent) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	if d.content != nil {
		adder.AddChild(d.content)
	}
	adder.AddChild(&d.dummyContent)
	adder.AddChild(&d.text)
	adder.AddChild(&d.icon)
}

func (d *dropdownListButtonContent) Update(context *guigui.Context) error {
	d.text.SetVerticalAlign(VerticalAlignMiddle)

	img, err := theResourceImages.Get("unfold_more", context.ColorMode())
	if err != nil {
		return err
	}
	d.icon.SetImage(img)
	return nil
}

func (d *dropdownListButtonContent) layout(context *guigui.Context) guigui.LinearLayout {
	d.layoutItems = slices.Delete(d.layoutItems, 0, len(d.layoutItems))

	var paddingTop int
	var paddingBottom int
	u := UnitSize(context)
	if d.contentSizePlus1.X != 0 || d.contentSizePlus1.Y != 0 {
		d.dummyContent.SetFixedSize(d.contentSizePlus1.Sub(image.Pt(1, 1)))
		d.layoutItems = append(d.layoutItems,
			guigui.LinearLayoutItem{
				Widget: &d.dummyContent,
			})
		paddingTop = u / 4
		paddingBottom = u / 4
	} else if d.content != nil {
		d.layoutItems = append(d.layoutItems,
			guigui.LinearLayoutItem{
				Widget: d.content,
			})
		paddingTop = u / 4
		paddingBottom = u / 4
	} else {
		d.layoutItems = append(d.layoutItems,
			guigui.LinearLayoutItem{
				Widget: &d.text,
			})
	}

	iconSize := defaultIconSize(context)
	d.layoutItems = append(d.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &d.icon,
			Size:   guigui.FixedSize(iconSize),
		})

	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Gap:       buttonTextAndImagePadding(context),
		Items:     d.layoutItems,
		Padding: guigui.Padding{
			Start:  buttonEdgeAndTextPadding(context),
			Top:    paddingTop,
			End:    buttonTextAndImagePadding(context),
			Bottom: paddingBottom,
		},
	}
}

func (d *dropdownListButtonContent) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	d.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (d *dropdownListButtonContent) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return d.layout(context).Measure(context, constraints)
}

type dropdownListItemContent struct {
	guigui.DefaultWidget

	content guigui.Widget
}

func (d *dropdownListItemContent) SetContent(content guigui.Widget) {
	d.content = content
}

func (d *dropdownListItemContent) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(d.content)
}

func (d *dropdownListItemContent) layout(context *guigui.Context) guigui.LinearLayout {
	u := UnitSize(context)
	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: d.content,
			},
		},
		Padding: guigui.Padding{
			Start:  u / 4,
			Top:    int(context.Scale()),
			End:    u / 4,
			Bottom: int(context.Scale()),
		},
	}
}

func (d *dropdownListItemContent) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	d.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (d *dropdownListItemContent) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return d.layout(context).Measure(context, constraints)
}
