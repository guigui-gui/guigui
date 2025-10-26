// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"

	"github.com/guigui-gui/guigui"
)

const (
	popupMenuEventItemSelected = "itemSelected"
)

type PopupMenuItem[T comparable] struct {
	Text         string
	TextColor    color.Color
	Header       bool
	Content      guigui.Widget
	Unselectable bool
	Border       bool
	Disabled     bool
	Value        T
}

type PopupMenu[T comparable] struct {
	guigui.DefaultWidget

	popup Popup
	list  guigui.WidgetWithSize[*List[T]]
}

func (p *PopupMenu[T]) SetOnItemSelected(f func(index int)) {
	guigui.RegisterEventHandler(p, popupMenuEventItemSelected, f)
}

func (p *PopupMenu[T]) SetCheckmarkIndex(index int) {
	p.list.Widget().SetCheckmarkIndex(index)
}

func (p *PopupMenu[T]) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&p.popup)
}

func (p *PopupMenu[T]) Update(context *guigui.Context) error {
	list := p.list.Widget()
	list.SetStyle(ListStyleMenu)
	list.list.SetOnItemSelected(func(index int) {
		p.popup.SetOpen(false)
		guigui.DispatchEventHandler(p, popupMenuEventItemSelected, index)
	})
	p.list.SetFixedSize(p.contentBounds(context).Size())

	p.popup.SetContent(&p.list)
	p.popup.SetCloseByClickingOutside(true)

	return nil
}

func (p *PopupMenu[T]) Layout(context *guigui.Context, widget guigui.Widget) image.Rectangle {
	switch widget {
	case &p.popup:
		return p.contentBounds(context)
	}
	return image.Rectangle{}
}

func (p *PopupMenu[T]) contentBounds(context *guigui.Context) image.Rectangle {
	pos := context.Bounds(p).Min
	// List size can dynamically change based on the items. Use the default size.
	s := p.list.Widget().Measure(context, guigui.Constraints{})
	s.Y = min(s.Y, 24*UnitSize(context))
	r := image.Rectangle{
		Min: pos,
		Max: pos.Add(s),
	}
	if p.IsOpen() {
		as := context.AppBounds().Size()
		if r.Max.X > as.X {
			r.Min.X = as.X - s.X
			r.Max.X = as.X
		}
		if r.Min.X < 0 {
			r.Min.X = 0
			r.Max.X = s.X
		}
		if r.Max.Y > as.Y {
			r.Min.Y = as.Y - s.Y
			r.Max.Y = as.Y
		}
		if r.Min.Y < 0 {
			r.Min.Y = 0
			r.Max.Y = s.Y
		}
	}
	return r
}

func (p *PopupMenu[T]) SetOpen(open bool) {
	p.popup.SetOpen(open)
}

func (p *PopupMenu[T]) IsOpen() bool {
	return p.popup.IsOpen()
}

func (p *PopupMenu[T]) SetItems(items []PopupMenuItem[T]) {
	var listItems []ListItem[T]
	for _, item := range items {
		listItems = append(listItems, ListItem[T]{
			Text:         item.Text,
			TextColor:    item.TextColor,
			Header:       item.Header,
			Content:      item.Content,
			Unselectable: item.Unselectable,
			Border:       item.Border,
			Disabled:     item.Disabled,
			Value:        item.Value,
		})
	}
	p.list.Widget().SetItems(listItems)
}

func (p *PopupMenu[T]) SetItemsByStrings(items []string) {
	p.list.Widget().SetItemsByStrings(items)
}

func (p *PopupMenu[T]) SelectedItem() (PopupMenuItem[T], bool) {
	listItem, ok := p.list.Widget().SelectedItem()
	if !ok {
		return PopupMenuItem[T]{}, false
	}
	return PopupMenuItem[T]{
		Text:         listItem.Text,
		TextColor:    listItem.TextColor,
		Header:       listItem.Header,
		Content:      listItem.Content,
		Unselectable: listItem.Unselectable,
		Border:       listItem.Border,
		Disabled:     listItem.Disabled,
		Value:        listItem.Value,
	}, true
}

func (p *PopupMenu[T]) ItemByIndex(index int) (PopupMenuItem[T], bool) {
	listItem, ok := p.list.Widget().ItemByIndex(index)
	if !ok {
		return PopupMenuItem[T]{}, false
	}
	return PopupMenuItem[T]{
		Text:         listItem.Text,
		TextColor:    listItem.TextColor,
		Header:       listItem.Header,
		Content:      listItem.Content,
		Unselectable: listItem.Unselectable,
		Border:       listItem.Border,
		Disabled:     listItem.Disabled,
		Value:        listItem.Value,
	}, true
}

func (p *PopupMenu[T]) SelectedItemIndex() int {
	return p.list.Widget().SelectedItemIndex()
}

func (p *PopupMenu[T]) SelectItemByIndex(index int) {
	p.list.Widget().SelectItemByIndex(index)
}

func (p *PopupMenu[T]) SelectItemByValue(value T) {
	p.list.Widget().SelectItemByValue(value)
}

func (p *PopupMenu[T]) ItemTextColor(context *guigui.Context, index int) color.Color {
	return p.list.Widget().ItemTextColor(context, index)
}
