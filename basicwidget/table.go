// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Table[T comparable] struct {
	guigui.DefaultWidget

	list             baseList[T]
	baseListItems    []baseListItem[T]
	tableItems       []TableItem[T]
	tableItemWidgets []tableItemWidget[T]
	columnTexts      []Text
	tableHeader      tableHeader[T]

	columns              []TableColumn
	columnLayoutItems    []guigui.LinearLayoutItem
	columnWidthsInPixels []int
}

type TableColumn struct {
	HeaderText                string
	HeaderTextHorizontalAlign HorizontalAlign
	Width                     guigui.Size
	MinWidth                  int
}

type TableItem[T comparable] struct {
	Contents     []guigui.Widget
	Unselectable bool
	Movable      bool
	Value        T
}

func (t *TableItem[T]) selectable() bool {
	return !t.Unselectable
}

func (t *Table[T]) SetColumns(columns []TableColumn) {
	t.columns = slices.Delete(t.columns, 0, len(t.columns))
	t.columns = append(t.columns, columns...)
}

func (t *Table[T]) SetOnItemSelected(f func(index int)) {
	t.list.SetOnItemSelected(f)
}

func (t *Table[T]) SetOnItemsMoved(f func(from, count, to int)) {
	t.list.SetOnItemsMoved(f)
}

func (t *Table[T]) SetCheckmarkIndex(index int) {
	t.list.SetCheckmarkIndex(index)
}

func (t *Table[T]) SetFooterHeight(height int) {
	t.list.SetFooterHeight(height)
}

func (t *Table[T]) updateTableItems() {
	t.tableItemWidgets = adjustSliceSize(t.tableItemWidgets, len(t.tableItems))
	t.baseListItems = adjustSliceSize(t.baseListItems, len(t.tableItems))

	for i, item := range t.tableItems {
		t.tableItemWidgets[i].setListItem(item)
		t.baseListItems[i] = t.tableItemWidgets[i].listItem()
	}
	t.list.SetItems(t.baseListItems)
}

func (t *Table[T]) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&t.list)
	for i := range t.columnTexts {
		adder.AddChild(&t.columnTexts[i])
	}
	adder.AddChild(&t.tableHeader)
}

func (t *Table[T]) Update(context *guigui.Context) error {
	t.list.SetHeaderHeight(tableHeaderHeight(context))
	t.list.SetStyle(ListStyleNormal)
	t.list.SetStripeVisible(true)

	t.updateTableItems()

	t.columnWidthsInPixels = adjustSliceSize(t.columnWidthsInPixels, len(t.columns))
	t.columnLayoutItems = adjustSliceSize(t.columnLayoutItems, len(t.columns))
	t.columnTexts = adjustSliceSize(t.columnTexts, len(t.columns))
	for i, column := range t.columns {
		t.columnLayoutItems[i] = guigui.LinearLayoutItem{
			Size: column.Width,
		}
		t.columnTexts[i].SetValue(column.HeaderText)
		t.columnTexts[i].SetHorizontalAlign(column.HeaderTextHorizontalAlign)
		t.columnTexts[i].SetVerticalAlign(VerticalAlignMiddle)
	}
	layout := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     t.columnLayoutItems,
		Gap:       tableColumnGap(context),
		Padding: guigui.Padding{
			Start: listItemPadding(context),
			End:   listItemPadding(context),
		},
	}
	for i := range t.columnWidthsInPixels {
		t.columnWidthsInPixels[i] = layout.ItemBounds(context, context.Bounds(t), i).Dx()
		t.columnWidthsInPixels[i] = max(t.columnWidthsInPixels[i], t.columns[i].MinWidth)
	}
	var contentWidth int
	if len(t.columnWidthsInPixels) > 0 {
		for _, width := range t.columnWidthsInPixels {
			contentWidth += width
		}
		contentWidth += (len(t.columnWidthsInPixels)-1)*tableColumnGap(context) + 2*listItemPadding(context)
	}
	t.list.SetContentWidth(contentWidth)

	for i := range t.tableItemWidgets {
		item := &t.tableItemWidgets[i]
		item.table = t
	}

	t.tableHeader.table = t

	return nil
}

func (t *Table[T]) Layout(context *guigui.Context, widget guigui.Widget) image.Rectangle {
	switch widget {
	case &t.list:
		return context.Bounds(t)
	case &t.tableHeader:
		return context.Bounds(t)
	}

	offsetX, _ := t.list.ScrollOffset()
	pt := context.Bounds(t).Min
	pt.X += int(offsetX)
	pt.X += listItemPadding(context)
	for i := range t.columnTexts {
		if widget == &t.columnTexts[i] {
			return image.Rectangle{
				Min: pt,
				Max: pt.Add(image.Pt(t.columnWidthsInPixels[i], tableHeaderHeight(context))),
			}
		}
		pt.X += t.columnWidthsInPixels[i] + tableColumnGap(context)
	}

	return image.Rectangle{}
}

func tableColumnGap(context *guigui.Context) int {
	u := UnitSize(context)
	return u / 2
}

func tableHeaderHeight(context *guigui.Context) int {
	u := UnitSize(context)
	return u
}

func (t *Table[T]) ItemTextColor(context *guigui.Context, index int) color.Color {
	item := &t.tableItemWidgets[index]
	switch {
	case t.list.SelectedItemIndex() == index && item.selectable():
		return DefaultActiveListItemTextColor(context)
	default:
		return draw.TextColor(context.ColorMode(), context.IsEnabled(item))
	}
}

func (t *Table[T]) SelectedItemIndex() int {
	return t.list.SelectedItemIndex()
}

func (t *Table[T]) SelectedItem() (TableItem[T], bool) {
	if t.list.SelectedItemIndex() < 0 || t.list.SelectedItemIndex() >= len(t.tableItemWidgets) {
		return TableItem[T]{}, false
	}
	return t.tableItemWidgets[t.list.SelectedItemIndex()].item, true
}

func (t *Table[T]) ItemByIndex(index int) (TableItem[T], bool) {
	if index < 0 || index >= len(t.tableItemWidgets) {
		return TableItem[T]{}, false
	}
	return t.tableItemWidgets[index].item, true
}

func (t *Table[T]) SetItems(items []TableItem[T]) {
	t.tableItems = adjustSliceSize(t.tableItems, len(items))
	copy(t.tableItems, items)
	t.updateTableItems()
}

func (t *Table[T]) ItemsCount() int {
	return len(t.tableItemWidgets)
}

func (t *Table[T]) ID(index int) any {
	return t.tableItemWidgets[index].item.Value
}

func (t *Table[T]) SelectItemByIndex(index int) {
	t.list.SelectItemByIndex(index)
}

func (t *Table[T]) SelectItemByValue(value T) {
	t.list.SelectItemByValue(value)
}

func (t *Table[T]) JumpToItemIndex(index int) {
	t.list.JumpToItemIndex(index)
}

func (t *Table[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return image.Pt(12*UnitSize(context), 6*UnitSize(context))
}

type tableItemWidget[T comparable] struct {
	guigui.DefaultWidget

	item  TableItem[T]
	table *Table[T]

	contentBounds map[guigui.Widget]image.Rectangle
}

func (t *tableItemWidget[T]) setListItem(listItem TableItem[T]) {
	t.item = listItem
}

func (t *tableItemWidget[T]) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	for _, content := range t.item.Contents {
		if content != nil {
			adder.AddChild(content)
		}
	}
}

func (t *tableItemWidget[T]) Update(context *guigui.Context) error {
	b := context.Bounds(t)
	x := b.Min.X
	clear(t.contentBounds)
	if t.contentBounds == nil {
		t.contentBounds = map[guigui.Widget]image.Rectangle{}
	}
	for i, content := range t.item.Contents {
		if content != nil {
			w := t.table.columnWidthsInPixels[i]
			pt := image.Pt(x, b.Min.Y)
			pt.Y += int(listItemTextPadding(context))
			s := image.Pt(w, content.Measure(context, guigui.FixedHeightConstraints(w)).Y)
			t.contentBounds[content] = image.Rectangle{
				Min: pt,
				Max: pt.Add(s),
			}
		}
		x += t.table.columnWidthsInPixels[i] + tableColumnGap(context)
	}
	return nil
}

func (t *tableItemWidget[T]) Layout(context *guigui.Context, widget guigui.Widget) image.Rectangle {
	return t.contentBounds[widget]
}

func (t *tableItemWidget[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	var w, h int
	for i, content := range t.item.Contents {
		if content == nil {
			continue
		}
		s := content.Measure(context, guigui.FixedWidthConstraints(t.table.columnWidthsInPixels[i]))
		// s.X is not reliable because the content might return an arbitrary value.
		if i > 0 {
			w += tableColumnGap(context)
		}
		w += t.table.columnWidthsInPixels[i]
		h = max(h, s.Y)
	}
	h = max(h, int(LineHeight(context)))
	// As TableItem.Text doesn't exist unlike ListItem.Text, always add padding.
	h += int(2 * listItemTextPadding(context))
	return image.Pt(w, h)
}

func (t *tableItemWidget[T]) selectable() bool {
	return t.item.selectable()
}

func (t *tableItemWidget[T]) listItem() baseListItem[T] {
	return baseListItem[T]{
		Content:    t,
		Selectable: t.selectable(),
		Movable:    t.item.Movable,
		Value:      t.item.Value,
	}
}

type tableHeader[T comparable] struct {
	guigui.DefaultWidget

	table *Table[T]
}

func (t *tableHeader[T]) Draw(context *guigui.Context, dst *ebiten.Image) {
	if len(t.table.columnWidthsInPixels) <= 1 {
		return
	}
	u := UnitSize(context)
	b := context.Bounds(t)
	x := b.Min.X + listItemPadding(context)
	offsetX, _ := t.table.list.ScrollOffset()
	x += int(offsetX)
	for _, width := range t.table.columnWidthsInPixels[:len(t.table.columnWidthsInPixels)-1] {
		x += width
		x0 := float32(x + tableColumnGap(context)/2)
		x1 := x0
		y0 := float32(b.Min.Y + u/4)
		y1 := float32(b.Min.Y + tableHeaderHeight(context) - u/4)
		clr := draw.Color2(context.ColorMode(), draw.ColorTypeBase, 0.9, 0.4)
		if !context.IsEnabled(t) {
			clr = draw.Color2(context.ColorMode(), draw.ColorTypeBase, 0.8, 0.3)
		}
		vector.StrokeLine(dst, x0, y0, x1, y1, float32(context.Scale()), clr, false)
		x += tableColumnGap(context)
	}
}
