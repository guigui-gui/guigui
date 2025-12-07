// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

const (
	popupMenuEventItemSelected = "itemSelected"
)

type PopupMenuItem[T comparable] struct {
	Text         string
	TextColor    color.Color
	Header       bool
	Content      guigui.Widget
	KeyText      string
	Unselectable bool
	Border       bool
	Disabled     bool
	Value        T
}

type PopupMenu[T comparable] struct {
	guigui.DefaultWidget

	popup     Popup
	list      guigui.WidgetWithSize[*List[T]]
	items     []PopupMenuItem[T]
	listItems []ListItem[T]

	onItemSelected func(context *guigui.Context, widgetBounds *guigui.WidgetBounds, index int)
}

func (p *PopupMenu[T]) SetOnItemSelected(f func(context *guigui.Context, widgetBounds *guigui.WidgetBounds, index int)) {
	guigui.RegisterEventHandler(p, popupMenuEventItemSelected, f)
}

func (p *PopupMenu[T]) SetCheckmarkIndex(index int) {
	p.list.Widget().SetCheckmarkIndex(index)
}

func (p *PopupMenu[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&p.popup)

	list := p.list.Widget()
	list.SetStyle(ListStyleMenu)
	if p.onItemSelected == nil {
		p.onItemSelected = func(context *guigui.Context, widgetBounds *guigui.WidgetBounds, index int) {
			p.popup.SetOpen(false)
			guigui.DispatchEventHandler(p, popupMenuEventItemSelected, index)
		}
	}
	list.list.SetOnItemSelected(p.onItemSelected)

	p.popup.setStyle(popupStyleMenu)
	p.popup.SetContent(&p.list)
	p.popup.SetCloseByClickingOutside(true)

	return nil
}

func (p *PopupMenu[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	b := p.contentBounds(context, widgetBounds)
	p.list.SetFixedSize(b.Size())
	layouter.LayoutWidget(&p.popup, b)
}

func (p *PopupMenu[T]) contentBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	pos := widgetBounds.Bounds().Min
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

func (p *PopupMenu[T]) updateListItems() {
	p.listItems = adjustSliceSize(p.listItems, len(p.items))
	for i, item := range p.items {
		p.listItems[i] = ListItem[T]{
			Text:         item.Text,
			TextColor:    item.TextColor,
			Header:       item.Header,
			Content:      item.Content,
			KeyText:      item.KeyText,
			Unselectable: item.Unselectable,
			Border:       item.Border,
			Disabled:     item.Disabled,
			Value:        item.Value,
		}
	}
	p.list.Widget().SetItems(p.listItems)
}

func (p *PopupMenu[T]) SetItems(items []PopupMenuItem[T]) {
	if !p.popup.popup.canUpdateContent() {
		return
	}
	p.items = adjustSliceSize(p.items, len(items))
	copy(p.items, items)
	p.updateListItems()
}

func (p *PopupMenu[T]) SetItemsByStrings(items []string) {
	p.items = adjustSliceSize(p.items, len(items))
	for i, str := range items {
		p.items[i] = PopupMenuItem[T]{
			Text: str,
		}
	}
	p.updateListItems()
}

func (p *PopupMenu[T]) SelectedItem() (PopupMenuItem[T], bool) {
	index := p.list.Widget().SelectedItemIndex()
	return p.ItemByIndex(index)
}

func (p *PopupMenu[T]) ItemByIndex(index int) (PopupMenuItem[T], bool) {
	if index < 0 || index >= len(p.items) {
		return PopupMenuItem[T]{}, false
	}
	return p.items[index], true
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
	// If the popup menu is closed, the item content might be used in other places like a select list button.
	// Use the default text color in that case.
	if !p.IsOpen() {
		return draw.TextColor(context.ColorMode(), true)
	}
	return p.list.Widget().ItemTextColor(context, index)
}

func (p *PopupMenu[T]) itemYFromIndexForMenu(context *guigui.Context, index int) (int, bool) {
	return p.list.Widget().list.ItemYFromIndexForMenu(context, index)
}
