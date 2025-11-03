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

	list            baseList[T]
	baseListItems   []baseListItem[T]
	tableRows       []TableRow[T]
	tableRowWidgets []tableRowWidget[T]
	columnTexts     []Text
	tableHeader     tableHeader[T]

	columns              []TableColumn
	columnLayoutItems    []guigui.LinearLayoutItem
	columnWidthsInPixels []int

	tmpItemBounds []image.Rectangle
}

type TableColumn struct {
	HeaderText                string
	HeaderTextHorizontalAlign HorizontalAlign
	Width                     guigui.Size
	MinWidth                  int
}

type TableRow[T comparable] struct {
	Cells        []TableCell
	Unselectable bool
	Movable      bool
	Value        T
}

func (t *TableRow[T]) selectable() bool {
	return !t.Unselectable
}

type TableCell struct {
	Text                string
	TextColor           color.Color
	TextHorizontalAlign HorizontalAlign
	TextVerticalAlign   VerticalAlign
	TextBold            bool
	TextTabular         bool
	Content             guigui.Widget
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

func (t *Table[T]) updateTableRows() {
	t.tableRowWidgets = adjustSliceSize(t.tableRowWidgets, len(t.tableRows))
	t.baseListItems = adjustSliceSize(t.baseListItems, len(t.tableRows))

	for i, row := range t.tableRows {
		t.tableRowWidgets[i].setTableRow(row)
		t.baseListItems[i] = t.tableRowWidgets[i].listItem()
	}
	t.list.SetItems(t.baseListItems)
}

func (t *Table[T]) AddChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, adder *guigui.ChildAdder) {
	adder.AddChild(&t.list)
	for i := range t.columnTexts {
		adder.AddChild(&t.columnTexts[i])
	}
	adder.AddChild(&t.tableHeader)
}

func (t *Table[T]) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	t.list.SetHeaderHeight(tableHeaderHeight(context))
	t.list.SetStyle(ListStyleNormal)
	t.list.SetStripeVisible(true)

	t.updateTableRows()

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
	// TODO: Use this at Layout. The issue is that the current LinearLayout cannot treat MinWidth well.
	layout := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     t.columnLayoutItems,
		Padding: guigui.Padding{
			Start: RoundedCornerRadius(context),
			End:   RoundedCornerRadius(context),
		},
	}
	t.tmpItemBounds = layout.AppendItemBounds(t.tmpItemBounds[:0], context, widgetBounds.Bounds())
	for i := range t.columnWidthsInPixels {
		t.columnWidthsInPixels[i] = t.tmpItemBounds[i].Dx()
		t.columnWidthsInPixels[i] = max(t.columnWidthsInPixels[i], t.columns[i].MinWidth)
	}
	var contentWidth int
	for _, width := range t.columnWidthsInPixels {
		contentWidth += width
	}
	contentWidth += 2 * RoundedCornerRadius(context)
	t.list.SetContentWidth(contentWidth)

	for i := range t.tableRowWidgets {
		row := &t.tableRowWidgets[i]
		row.table = t
	}

	t.tableHeader.table = t

	return nil
}

func (t *Table[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, widget guigui.Widget) image.Rectangle {
	switch widget {
	case &t.list:
		return widgetBounds.Bounds()
	case &t.tableHeader:
		return widgetBounds.Bounds()
	}

	offsetX, _ := t.list.ScrollOffset()
	pt := widgetBounds.Bounds().Min
	pt.X += int(offsetX)
	pt.X += RoundedCornerRadius(context)
	for i := range t.columnTexts {
		if widget == &t.columnTexts[i] {
			pt = pt.Add(image.Pt(UnitSize(context)/4, 0))
			w := t.columnWidthsInPixels[i] - UnitSize(context)/2
			return image.Rectangle{
				Min: pt,
				Max: pt.Add(image.Pt(w, tableHeaderHeight(context))),
			}
		}
		pt.X += t.columnWidthsInPixels[i]
	}

	return image.Rectangle{}
}

func tableHeaderHeight(context *guigui.Context) int {
	u := UnitSize(context)
	return u
}

func (t *Table[T]) ItemTextColor(context *guigui.Context, index int) color.Color {
	item := &t.tableRowWidgets[index]
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

func (t *Table[T]) SelectedItem() (TableRow[T], bool) {
	if t.list.SelectedItemIndex() < 0 || t.list.SelectedItemIndex() >= len(t.tableRowWidgets) {
		return TableRow[T]{}, false
	}
	return t.tableRowWidgets[t.list.SelectedItemIndex()].row, true
}

func (t *Table[T]) ItemByIndex(index int) (TableRow[T], bool) {
	if index < 0 || index >= len(t.tableRowWidgets) {
		return TableRow[T]{}, false
	}
	return t.tableRowWidgets[index].row, true
}

func (t *Table[T]) SetItems(items []TableRow[T]) {
	t.tableRows = adjustSliceSize(t.tableRows, len(items))
	copy(t.tableRows, items)
	t.updateTableRows()
}

func (t *Table[T]) ItemsCount() int {
	return len(t.tableRowWidgets)
}

func (t *Table[T]) ID(index int) any {
	return t.tableRowWidgets[index].row.Value
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

type tableRowWidget[T comparable] struct {
	guigui.DefaultWidget

	row   TableRow[T]
	table *Table[T]
	texts []Text

	//contentBounds map[guigui.Widget]image.Rectangle
	layout guigui.Layout
}

func (t *tableRowWidget[T]) setTableRow(row TableRow[T]) {
	t.row = row
}

func (t *tableRowWidget[T]) ensureTexts() {
	t.texts = adjustSliceSize(t.texts, len(t.row.Cells))
	for i, cell := range t.row.Cells {
		if cell.Content != nil {
			continue
		}
		t.texts[i].SetValue(cell.Text)
		t.texts[i].SetColor(cell.TextColor)
		t.texts[i].SetHorizontalAlign(cell.TextHorizontalAlign)
		t.texts[i].SetVerticalAlign(cell.TextVerticalAlign)
		t.texts[i].SetBold(cell.TextBold)
		t.texts[i].SetTabular(cell.TextTabular)
		t.texts[i].SetAutoWrap(true)
	}
}

func (t *tableRowWidget[T]) AddChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, adder *guigui.ChildAdder) {
	t.ensureTexts()
	for i, cell := range t.row.Cells {
		if cell.Content != nil {
			adder.AddChild(cell.Content)
		} else {
			adder.AddChild(&t.texts[i])
		}
	}
}

func (t *tableRowWidget[T]) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	l := &guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
	}
	for i := range t.table.columnWidthsInPixels {
		if t.row.Cells[i].Content != nil {
			l.Items = append(l.Items, guigui.LinearLayoutItem{
				Widget: t.row.Cells[i].Content,
				Size:   guigui.FixedSize(t.table.columnWidthsInPixels[i]),
			})
		} else {
			l.Items = append(l.Items,
				guigui.LinearLayoutItem{
					Layout: guigui.LinearLayout{
						Direction: guigui.LayoutDirectionHorizontal,
						Items: []guigui.LinearLayoutItem{
							{
								Widget: &t.texts[i],
								Size:   guigui.FlexibleSize(1),
							},
						},
						Padding: ListItemTextPadding(context),
					},
					Size: guigui.FixedSize(t.table.columnWidthsInPixels[i]),
				})
		}

	}
	t.layout = l
	return nil
}

func (t *tableRowWidget[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, widget guigui.Widget) image.Rectangle {
	return t.layout.WidgetBounds(context, widgetBounds.Bounds(), widget)
}

func (t *tableRowWidget[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	t.ensureTexts()

	var w, h int
	for i, cell := range t.row.Cells {
		var s image.Point
		if cell.Content != nil {
			s = cell.Content.Measure(context, guigui.FixedWidthConstraints(t.table.columnWidthsInPixels[i]))
		} else {
			// Assume that every item can use a bold font.
			p := ListItemTextPadding(context)
			w := t.table.columnWidthsInPixels[i] - p.Start - p.End
			s = t.texts[i].Measure(context, guigui.FixedWidthConstraints(w))
			s = s.Add(image.Pt(p.Start+p.End, p.Top+p.Bottom))
		}
		w += t.table.columnWidthsInPixels[i]
		h = max(h, s.Y)
	}
	h = max(h, int(LineHeight(context)))
	return image.Pt(w, h)
}

func (t *tableRowWidget[T]) selectable() bool {
	return t.row.selectable()
}

func (t *tableRowWidget[T]) listItem() baseListItem[T] {
	return baseListItem[T]{
		Content:    t,
		Selectable: t.selectable(),
		Movable:    t.row.Movable,
		Value:      t.row.Value,
	}
}

type tableHeader[T comparable] struct {
	guigui.DefaultWidget

	table *Table[T]
}

func (t *tableHeader[T]) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if len(t.table.columnWidthsInPixels) <= 1 {
		return
	}
	u := UnitSize(context)
	b := widgetBounds.Bounds()
	x := b.Min.X + RoundedCornerRadius(context)
	offsetX, _ := t.table.list.ScrollOffset()
	x += int(offsetX)
	for _, width := range t.table.columnWidthsInPixels[:len(t.table.columnWidthsInPixels)-1] {
		x += width
		x0 := float32(x)
		x1 := x0
		y0 := float32(b.Min.Y + u/4)
		y1 := float32(b.Min.Y + tableHeaderHeight(context) - u/4)
		clr := draw.Color2(context.ColorMode(), draw.ColorTypeBase, 0.9, 0.4)
		if !context.IsEnabled(t) {
			clr = draw.Color2(context.ColorMode(), draw.ColorTypeBase, 0.8, 0.3)
		}
		vector.StrokeLine(dst, x0, y0, x1, y1, float32(context.Scale()), clr, false)
	}
}
