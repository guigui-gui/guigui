// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"
	"slices"

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
	KeyText      string
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

func (l *List[T]) SetOnItemSelected(f func(context *guigui.Context, widgetBounds *guigui.WidgetBounds, index int)) {
	l.list.SetOnItemSelected(f)
}

func (l *List[T]) SetOnItemsMoved(f func(context *guigui.Context, widgetBounds *guigui.WidgetBounds, from, count, to int)) {
	l.list.SetOnItemsMoved(f)
}

func (l *List[T]) SetOnItemExpanderToggled(f func(context *guigui.Context, widgetBounds *guigui.WidgetBounds, index int, expanded bool)) {
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

func (l *List[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&l.list)

	l.updateListItems()
	for i := range l.listItemWidgets {
		item := &l.listItemWidgets[i]
		item.text.SetBold(item.item.Header || l.list.Style() == ListStyleSidebar && l.SelectedItemIndex() == i)
		item.text.SetColor(l.ItemTextColor(context, i))
		item.keyText.SetColor(l.ItemTextColor(context, i))
		context.SetEnabled(item, !item.item.Disabled)
	}
	return nil
}

func (l *List[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
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

	item    ListItem[T]
	text    Text
	keyText Text

	heightPlus1 int
	style       ListStyle

	layoutItems []guigui.LinearLayoutItem
}

func (l *listItemWidget[T]) setListItem(listItem ListItem[T]) {
	l.item = listItem
	l.text.SetValue(listItem.Text)
	l.keyText.SetValue(listItem.KeyText)
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

func (l *listItemWidget[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if l.item.Content != nil {
		adder.AddChild(l.item.Content)
	} else {
		adder.AddChild(&l.text)
	}
	adder.AddChild(&l.keyText)

	l.text.SetValue(l.item.Text)
	l.text.SetVerticalAlign(VerticalAlignMiddle)
	l.keyText.SetOpacity(0.5)
	l.keyText.SetValue(l.item.KeyText)
	l.keyText.SetVerticalAlign(VerticalAlignMiddle)
	l.keyText.SetHorizontalAlign(HorizontalAlignEnd)
	return nil
}

func (l *listItemWidget[T]) layout(context *guigui.Context) guigui.LinearLayout {
	layout := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Gap:       int(LineHeight(context)),
	}
	l.layoutItems = slices.Delete(l.layoutItems, 0, len(l.layoutItems))
	if l.item.Content != nil {
		l.layoutItems = append(l.layoutItems, guigui.LinearLayoutItem{
			Widget: l.item.Content,
			Size:   guigui.FlexibleSize(1),
		})
	} else {
		// TODO: Use bold font to measure the size, maybe?
		l.layoutItems = append(l.layoutItems, guigui.LinearLayoutItem{
			Widget: &l.text,
			Size:   guigui.FlexibleSize(1),
		})
		layout.Padding = ListItemTextPadding(context)
	}
	if l.keyText.Value() != "" {
		l.layoutItems = append(l.layoutItems, guigui.LinearLayoutItem{
			Widget: &l.keyText,
		})
		layout.Padding.End = ListItemTextPadding(context).End
	}
	layout.Items = l.layoutItems
	var h int
	if l.heightPlus1 > 0 {
		h = l.heightPlus1 - 1
	} else if l.item.Border && l.item.Content == nil {
		h = UnitSize(context) / 2
	} else if l.item.Header && l.item.Content == nil {
		h = UnitSize(context) * 3 / 2
	}
	if h > 0 {
		return guigui.LinearLayout{
			Direction: guigui.LayoutDirectionVertical,
			Items: []guigui.LinearLayoutItem{
				{
					Layout: layout,
					Size:   guigui.FixedSize(h),
				},
			},
		}
	}
	return layout
}

func (l *listItemWidget[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	l.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (l *listItemWidget[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return l.layout(context).Measure(context, constraints)
}

func (l *listItemWidget[T]) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if l.item.Border {
		u := UnitSize(context)
		b := widgetBounds.Bounds()
		x0 := float32(b.Min.X + u/4)
		x1 := float32(b.Max.X - u/4)
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
