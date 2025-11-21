// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

type List[T comparable] struct {
	guigui.DefaultWidget

	list            baseList[T]
	baseListItems   []baseListItem[T]
	listItems       []ListItem[T]
	listItemWidgets []listItemWidget[T]

	listItemHeightPlus1 int
}

type ListItem[T comparable] struct {
	Text         string
	TextColor    color.Color
	Header       bool
	Content      guigui.Widget
	Unselectable bool
	Border       bool
	Disabled     bool
	Movable      bool
	Value        T
	IndentLevel  int
	Collapsed    bool
}

func (l *ListItem[T]) selectable() bool {
	return !l.Header && !l.Unselectable && !l.Border && !l.Disabled
}

func (l *List[T]) SetStripeVisible(visible bool) {
	l.list.SetStripeVisible(visible)
}

func (l *List[T]) SetItemHeight(height int) {
	if l.listItemHeightPlus1 == height+1 {
		return
	}
	l.listItemHeightPlus1 = height + 1
	guigui.RequestRedraw(l)
}

func (l *List[T]) SetOnItemSelected(f func(index int)) {
	l.list.SetOnItemSelected(f)
}

func (l *List[T]) SetOnItemsMoved(f func(from, count, to int)) {
	l.list.SetOnItemsMoved(f)
}

func (l *List[T]) SetOnItemExpanderToggled(f func(index int, expanded bool)) {
	l.list.SetOnItemExpanderToggled(f)
}

func (l *List[T]) SetCheckmarkIndex(index int) {
	l.list.SetCheckmarkIndex(index)
}

func (l *List[T]) SetHeaderHeight(height int) {
	l.list.SetHeaderHeight(height)
}

func (l *List[T]) SetFooterHeight(height int) {
	l.list.SetFooterHeight(height)
}

func (l *List[T]) updateListItems() {
	l.listItemWidgets = adjustSliceSize(l.listItemWidgets, len(l.listItems))
	l.baseListItems = adjustSliceSize(l.baseListItems, len(l.listItems))

	for i, item := range l.listItems {
		l.listItemWidgets[i].setListItem(item)
		l.listItemWidgets[i].setHeight(l.listItemHeightPlus1 - 1)
		l.listItemWidgets[i].setStyle(l.list.Style())
		l.baseListItems[i] = l.listItemWidgets[i].listItem()
	}
	l.list.SetItems(l.baseListItems)
}

func (l *List[T]) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&l.list)
}

func (l *List[T]) Update(context *guigui.Context) error {
	l.updateListItems()
	for i := range l.listItemWidgets {
		item := &l.listItemWidgets[i]
		item.text.SetBold(item.item.Header || l.list.Style() == ListStyleSidebar && l.SelectedItemIndex() == i)
		item.text.SetColor(l.ItemTextColor(context, i))
		context.SetEnabled(item, !item.item.Disabled)
	}
	return nil
}

func (l *List[T]) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&l.list, widgetBounds.Bounds())
}

func (l *List[T]) HighlightedItemIndex(context *guigui.Context) int {
	index := -1
	switch l.list.Style() {
	case ListStyleNormal, ListStyleSidebar:
		index = l.list.SelectedItemIndex()
	case ListStyleMenu:
		if !l.list.IsHoveringVisible() {
			return -1
		}
		index = l.list.HoveredItemIndex()
	}
	if index < 0 || index >= len(l.listItemWidgets) {
		return -1
	}
	item := &l.listItemWidgets[index]
	if !item.selectable() {
		return -1
	}
	if !context.IsEnabled(item) {
		return -1
	}
	return index
}

func (l *List[T]) ItemTextColor(context *guigui.Context, index int) color.Color {
	if l.HighlightedItemIndex(context) == index {
		return DefaultActiveListItemTextColor(context)
	}
	item := &l.listItemWidgets[index]
	if item.item.TextColor != nil {
		return item.item.TextColor
	}
	return draw.TextColor(context.ColorMode(), context.IsEnabled(item))
}

func (l *List[T]) SelectedItemIndex() int {
	return l.list.SelectedItemIndex()
}

func (l *List[T]) SelectedItem() (ListItem[T], bool) {
	if l.list.SelectedItemIndex() < 0 || l.list.SelectedItemIndex() >= len(l.listItemWidgets) {
		return ListItem[T]{}, false
	}
	return l.listItemWidgets[l.list.SelectedItemIndex()].item, true
}

func (l *List[T]) ItemByIndex(index int) (ListItem[T], bool) {
	if index < 0 || index >= len(l.listItemWidgets) {
		return ListItem[T]{}, false
	}
	return l.listItemWidgets[index].item, true
}

func (l *List[T]) SetItemsByStrings(strs []string) {
	items := make([]ListItem[T], len(strs))
	for i, str := range strs {
		items[i].Text = str
	}
	l.SetItems(items)
}

func (l *List[T]) SetItems(items []ListItem[T]) {
	l.listItems = adjustSliceSize(l.listItems, len(items))
	copy(l.listItems, items)

	// Updating list items at Update might be too late, when the text list is not visible like a select.
	// Update it here.
	l.updateListItems()
}

func (l *List[T]) ItemCount() int {
	return len(l.listItemWidgets)
}

func (l *List[T]) ID(index int) any {
	return l.listItemWidgets[index].item.Value
}

func (l *List[T]) SelectItemByIndex(index int) {
	l.list.SelectItemByIndex(index)
}

func (l *List[T]) SelectItemByValue(value T) {
	l.list.SelectItemByValue(value)
}

func (l *List[T]) JumpToItemByIndex(index int) {
	l.list.JumpToItemByIndex(index)
}

func (l *List[T]) EnsureItemVisibleByIndex(index int) {
	l.list.EnsureItemVisibleByIndex(index)
}

func (l *List[T]) SetStyle(style ListStyle) {
	l.list.SetStyle(style)
}

func (l *List[T]) SetItemString(str string, index int) {
	l.listItemWidgets[index].item.Text = str
}

func (l *List[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return l.list.Measure(context, constraints)
}

type listItemWidget[T comparable] struct {
	guigui.DefaultWidget

	item ListItem[T]
	text Text

	heightPlus1 int
	style       ListStyle
}

func (l *listItemWidget[T]) setListItem(listItem ListItem[T]) {
	l.item = listItem
	l.text.SetValue(listItem.Text)
}

func (l *listItemWidget[T]) setHeight(height int) {
	if l.heightPlus1 == height+1 {
		return
	}
	l.heightPlus1 = height + 1
}

func (l *listItemWidget[T]) setStyle(style ListStyle) {
	l.style = style
}

func (l *listItemWidget[T]) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	if l.item.Content != nil {
		adder.AddChild(l.item.Content)
		return
	}
	adder.AddChild(&l.text)
}

func (l *listItemWidget[T]) Update(context *guigui.Context) error {
	l.text.SetValue(l.item.Text)
	l.text.SetVerticalAlign(VerticalAlignMiddle)
	return nil
}

func (l *listItemWidget[T]) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	var widget guigui.Widget
	if l.item.Content != nil {
		widget = l.item.Content
	} else {
		widget = &l.text
	}

	b := widgetBounds.Bounds()
	s := widget.Measure(context, guigui.FixedWidthConstraints(b.Dx()))
	if l.style != ListStyleMenu {
		s.X = b.Dx()
	}
	if l.heightPlus1 > 0 {
		s.Y = l.heightPlus1 - 1
	}
	offY := (b.Dy() - s.Y) / 2
	pt := b.Min.Add(image.Pt(0, offY))
	if widget == &l.text {
		pt.X += ListItemTextPadding(context).Start
	}
	layouter.LayoutWidget(widget, image.Rectangle{
		Min: pt,
		Max: pt.Add(s),
	})
}

func (l *listItemWidget[T]) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if l.item.Border {
		b := widgetBounds.Bounds()
		x0 := float32(b.Min.X)
		x1 := float32(b.Max.X)
		y := float32(b.Min.Y) + float32(b.Dy())/2
		width := float32(1 * context.Scale())
		vector.StrokeLine(dst, x0, y, x1, y, width, draw.Color(context.ColorMode(), draw.ColorTypeBase, 0.8), false)
		return
	}
	/*if l.item.Header {
		bounds := widgetBounds.Bounds()
		draw.DrawRoundedRect(context, dst, bounds, draw.Color(context.ColorMode(), draw.ColorTypeBase, 0.8), RoundedCornerRadius(context))
	}*/
}

func (l *listItemWidget[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	var s image.Point
	if l.item.Content != nil {
		s = l.item.Content.Measure(context, constraints)
	} else {
		// Assume that every item can use a bold font.
		p := ListItemTextPadding(context)
		if fixedWidth, ok := constraints.FixedWidth(); ok {
			constraints = guigui.FixedWidthConstraints(fixedWidth - p.Start - p.End)
		}
		if fixedHeight, ok := constraints.FixedHeight(); ok {
			constraints = guigui.FixedHeightConstraints(fixedHeight - p.Top - p.Bottom)
		}
		s = l.text.boldTextSize(context, constraints)
		s = s.Add(image.Pt(p.Start+p.End, p.Top+p.Bottom))
	}

	if l.style != ListStyleMenu || l.item.Border {
		if w, ok := constraints.FixedWidth(); ok {
			s.X = max(s.X, w)
		}
	}

	if l.heightPlus1 > 0 {
		s.Y = l.heightPlus1 - 1
	} else if l.item.Border {
		s.Y = UnitSize(context) / 2
	} else if l.item.Header {
		s.Y = UnitSize(context) * 3 / 2
	}
	return s
}

func (l *listItemWidget[T]) selectable() bool {
	return l.item.selectable() && !l.item.Border
}

func (l *listItemWidget[T]) listItem() baseListItem[T] {
	return baseListItem[T]{
		Content:     l,
		Selectable:  l.selectable(),
		Movable:     l.item.Movable,
		Value:       l.item.Value,
		IndentLevel: l.item.IndentLevel,
		Collapsed:   l.item.Collapsed,
	}
}

func ListItemTextPadding(context *guigui.Context) guigui.Padding {
	u := UnitSize(context)
	return guigui.Padding{
		Start:  u / 4,
		Top:    int(context.Scale()),
		End:    u / 4,
		Bottom: int(context.Scale()),
	}
}
