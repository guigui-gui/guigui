// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

type Table[T comparable] struct {
	guigui.DefaultWidget

	list            List[T]
	listItems       []ListItem[T]
	tableRows       []TableRow[T]
	tableRowWidgets guigui.WidgetSlice[*tableRowWidget[T]]
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
	Checked      bool
	Value        T
}

func (t *TableRow[T]) selectable() bool {
	return !t.Unselectable
}

type TableCell struct {
	Text      string
	TextStyle TextStyle
	Content   guigui.Widget
}

func (t *Table[T]) SetColumns(columns []TableColumn) {
	t.columns = slices.Delete(t.columns, 0, len(t.columns))
	t.columns = append(t.columns, columns...)
}

func (t *Table[T]) SetMultiSelection(multi bool) {
	t.list.SetMultiSelection(multi)
}

func (t *Table[T]) SetUnfocusedSelectionVisible(visible bool) {
	t.list.SetUnfocusedSelectionVisible(visible)
}

func (t *Table[T]) SetItemHeight(height int) {
	t.list.SetItemHeight(height)
}

func (t *Table[T]) OnItemSelected(f func(context *guigui.Context, index int)) {
	t.list.OnItemSelected(f)
}

func (t *Table[T]) OnItemsSelected(f func(context *guigui.Context, indices []int)) {
	t.list.OnItemsSelected(f)
}

func (t *Table[T]) OnItemsMoved(f func(context *guigui.Context, from, count, to int)) {
	t.list.OnItemsMoved(f)
}

func (t *Table[T]) OnItemsCanMove(f func(context *guigui.Context, from, count, to int) bool) {
	t.list.OnItemsCanMove(f)
}

// SetReservesCheckmarkSpace sets whether the table reserves space for the
// checkmark column even when no row is currently checked. See
// [List.SetReservesCheckmarkSpace] for details.
func (t *Table[T]) SetReservesCheckmarkSpace(reserves bool) {
	t.list.SetReservesCheckmarkSpace(reserves)
}

func (t *Table[T]) SetFooterHeight(height int) {
	t.list.SetFooterHeight(height)
}

func (t *Table[T]) ItemBounds(index int) image.Rectangle {
	return t.list.ItemBounds(index)
}

// CellBounds returns the bounds for the cell at the given row and column indices.
// CellBounds is valid only after the table's layout finishes.
func (t *Table[T]) CellBounds(rowIndex, colIndex int) image.Rectangle {
	itemBounds := t.list.ItemBounds(rowIndex)
	if itemBounds.Empty() {
		return image.Rectangle{}
	}
	if colIndex < 0 || colIndex >= len(t.columnWidthsInPixels) {
		return image.Rectangle{}
	}

	x := itemBounds.Min.X
	for i := range colIndex {
		x += t.columnWidthsInPixels[i]
	}
	return image.Rectangle{
		Min: image.Pt(x, itemBounds.Min.Y),
		Max: image.Pt(x+t.columnWidthsInPixels[colIndex], itemBounds.Max.Y),
	}
}

func (t *Table[T]) IsItemInViewport(index int) bool {
	return t.list.IsItemInViewport(index)
}

// IsItemAvailable reports whether the item at the given index is available in the list
// (i.e., not hidden by a collapsed ancestor).
func (t *Table[T]) IsItemAvailable(index int) bool {
	return t.list.IsItemAvailable(index)
}

func (t *Table[T]) updateTableRows() {
	t.tableRowWidgets.SetLen(len(t.tableRows))
	t.listItems = adjustSliceSize(t.listItems, len(t.tableRows))

	for i, row := range t.tableRows {
		t.tableRowWidgets.At(i).setTableRow(row)
		t.listItems[i] = t.tableRowWidgets.At(i).listItem()
	}
	t.list.SetItems(t.listItems)
}

func (t *Table[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	// WidgetSlice.SetLen should be called before AddChild.
	t.updateTableRows()

	adder.AddWidget(&t.list)
	adder.AddWidget(&t.tableHeader)

	context.SetClipChildren(&t.tableHeader, true)

	t.list.SetHeaderHeight(tableHeaderHeight(context))
	t.list.SetStyle(ListStyleNormal)
	t.list.SetStripeVisible(true)

	for i := range t.tableRowWidgets.Len() {
		row := t.tableRowWidgets.At(i)
		row.table = t
	}
	t.tableHeader.table = t

	return nil
}

func (t *Table[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	bounds := widgetBounds.Bounds()

	t.columnWidthsInPixels = adjustSliceSize(t.columnWidthsInPixels, len(t.columns))
	t.columnLayoutItems = adjustSliceSize(t.columnLayoutItems, len(t.columns))
	for i, column := range t.columns {
		t.columnLayoutItems[i] = guigui.LinearLayoutItem{
			Size: column.Width,
		}
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
	t.tmpItemBounds = layout.AppendItemBounds(t.tmpItemBounds[:0], context, bounds)
	for i := range t.columnWidthsInPixels {
		t.columnWidthsInPixels[i] = t.tmpItemBounds[i].Dx()
		t.columnWidthsInPixels[i] = max(t.columnWidthsInPixels[i], t.columns[i].MinWidth)
	}
	var contentWidth int
	for _, width := range t.columnWidthsInPixels {
		contentWidth += width
	}
	contentWidth += 2 * RoundedCornerRadius(context)
	t.list.setContentWidth(contentWidth)

	layouter.LayoutWidget(&t.list, bounds)

	// The header content should not be rendered on the borders.
	bounds.Min.X += int(listBorderWidth(context))
	bounds.Max.X -= int(listBorderWidth(context))
	layouter.LayoutWidget(&t.tableHeader, bounds)
}

func tableHeaderHeight(context *guigui.Context) int {
	u := UnitSize(context)
	return u
}

func (t *Table[T]) SelectedItemCount() int {
	return t.list.SelectedItemCount()
}

func (t *Table[T]) SelectedItemIndex() int {
	return t.list.SelectedItemIndex()
}

func (t *Table[T]) AppendSelectedItemIndices(indices []int) []int {
	return t.list.AppendSelectedItemIndices(indices)
}

func (t *Table[T]) SelectedItem() (TableRow[T], bool) {
	if t.list.SelectedItemIndex() < 0 || t.list.SelectedItemIndex() >= t.tableRowWidgets.Len() {
		return TableRow[T]{}, false
	}
	return t.tableRowWidgets.At(t.list.SelectedItemIndex()).row, true
}

func (t *Table[T]) ItemByIndex(index int) (TableRow[T], bool) {
	if index < 0 || index >= t.tableRowWidgets.Len() {
		return TableRow[T]{}, false
	}
	return t.tableRowWidgets.At(index).row, true
}

func (t *Table[T]) IndexByValue(value T) int {
	for i := range t.tableRowWidgets.Len() {
		if t.tableRowWidgets.At(i).row.Value == value {
			return i
		}
	}
	return -1
}

func (t *Table[T]) SetItems(items []TableRow[T]) {
	t.tableRows = adjustSliceSize(t.tableRows, len(items))
	copy(t.tableRows, items)
	t.updateTableRows()
}

func (t *Table[T]) ItemCount() int {
	return t.tableRowWidgets.Len()
}

func (t *Table[T]) ID(index int) any {
	return t.tableRowWidgets.At(index).row.Value
}

func (t *Table[T]) SelectItemByIndex(index int) {
	t.list.SelectItemByIndex(index)
}

func (t *Table[T]) SelectItemsByIndices(indices []int) {
	t.list.SelectItemsByIndices(indices)
}

func (t *Table[T]) SelectAllItems() {
	t.list.SelectAllItems()
}

func (t *Table[T]) SelectItemByValue(value T) {
	t.list.SelectItemByValue(value)
}

func (t *Table[T]) SelectItemsByValues(values []T) {
	t.list.SelectItemsByValues(values)
}

func (t *Table[T]) JumpToItemByIndex(index int) {
	t.list.JumpToItemByIndex(index)
}

func (t *Table[T]) EnsureItemVisibleByIndex(index int) {
	t.list.EnsureItemVisibleByIndex(index)
}

func (t *Table[T]) ForceEnsureItemVisibleByIndex(index int) {
	t.list.ForceEnsureItemVisibleByIndex(index)
}

// ScrollPosition returns a snapshot of the current scroll position, restorable
// with [Table.SetScrollPosition] or [Table.ForceSetScrollPosition].
func (t *Table[T]) ScrollPosition() ListScrollPosition {
	return t.list.ScrollPosition()
}

// SetScrollPosition scrolls to the given position with animation.
func (t *Table[T]) SetScrollPosition(position ListScrollPosition) {
	t.list.SetScrollPosition(position)
}

// ForceSetScrollPosition scrolls to the given position without animation.
func (t *Table[T]) ForceSetScrollPosition(position ListScrollPosition) {
	t.list.ForceSetScrollPosition(position)
}

// OnScroll sets the handler invoked when the scroll position changes.
func (t *Table[T]) OnScroll(callback func(context *guigui.Context, position ListScrollPosition)) {
	t.list.OnScroll(callback)
}

func (t *Table[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return image.Pt(12*UnitSize(context), 6*UnitSize(context))
}

type tableRowWidget[T comparable] struct {
	guigui.DefaultWidget

	row   TableRow[T]
	table *Table[T]
	texts guigui.WidgetSlice[*Text]

	linearLayoutItems     []guigui.LinearLayoutItem
	textColumnLayouts     []guigui.LinearLayout
	textColumnLayoutItems []guigui.LinearLayoutItem
}

func (t *tableRowWidget[T]) setTableRow(row TableRow[T]) {
	t.row = row
}

func (t *tableRowWidget[T]) ensureTexts() {
	t.texts.SetLen(len(t.row.Cells))
	for i, cell := range t.row.Cells {
		if cell.Content != nil {
			continue
		}
		txt := t.texts.At(i)
		txt.SetValue(cell.Text)
		// Color is adjusted at Layout.
		txt.SetHorizontalAlign(cell.TextStyle.HorizontalAlign)
		txt.SetVerticalAlign(cell.TextStyle.VerticalAlign)
		txt.SetBold(cell.TextStyle.Bold)
		txt.SetTabular(cell.TextStyle.Tabular)
		txt.SetWrapMode(cell.TextStyle.WrapMode)
	}
}

func (t *tableRowWidget[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	t.ensureTexts()
	for i, cell := range t.row.Cells {
		if cell.Content != nil {
			adder.AddWidget(cell.Content)
		} else {
			adder.AddWidget(t.texts.At(i))
		}
	}
	return nil
}

func (t *tableRowWidget[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	t.linearLayoutItems = slices.Delete(t.linearLayoutItems, 0, len(t.linearLayoutItems))
	t.textColumnLayouts = t.textColumnLayouts[:0]
	t.textColumnLayoutItems = slices.Delete(t.textColumnLayoutItems, 0, len(t.textColumnLayoutItems))
	for i := range t.table.columnWidthsInPixels {
		if i < len(t.row.Cells) && t.row.Cells[i].Content != nil {
			t.linearLayoutItems = append(t.linearLayoutItems, guigui.LinearLayoutItem{
				Widget: t.row.Cells[i].Content,
				Size:   guigui.FixedSize(t.table.columnWidthsInPixels[i]),
			})
		} else {
			if i >= t.texts.Len() {
				break
			}
			t.textColumnLayoutItems = append(t.textColumnLayoutItems, guigui.LinearLayoutItem{
				Widget: t.texts.At(i),
				Size:   guigui.FlexibleSize(1),
			})
			t.textColumnLayouts = append(t.textColumnLayouts, guigui.LinearLayout{
				Direction: guigui.LayoutDirectionHorizontal,
				Items:     t.textColumnLayoutItems[len(t.textColumnLayoutItems)-1 : len(t.textColumnLayoutItems)],
				Padding:   ListItemTextPadding(context),
			})
			t.linearLayoutItems = append(t.linearLayoutItems,
				guigui.LinearLayoutItem{
					Layout: &t.textColumnLayouts[len(t.textColumnLayouts)-1],
					Size:   guigui.FixedSize(t.table.columnWidthsInPixels[i]),
				})
		}
	}
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     t.linearLayoutItems,
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)

	// Set text colors based on the list item color type provided by the parent list widget.
	if v, ok := context.Env(t, EnvKeyListItemColorType); ok {
		ct := v.(ListItemColorType)
		clr := ct.TextColor(context)
		for i, cell := range t.row.Cells {
			if i >= t.texts.Len() {
				break
			}
			if cell.TextStyle.Color != nil {
				t.texts.At(i).SetColor(cell.TextStyle.Color)
			} else {
				t.texts.At(i).SetColor(clr)
			}
		}
	}
}

func (t *tableRowWidget[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	t.ensureTexts()

	if len(t.table.columnWidthsInPixels) == 0 {
		return image.Pt(0, LineHeight(context))
	}

	var w, h int
	for i, cell := range t.row.Cells {
		if i >= len(t.table.columnWidthsInPixels) {
			break
		}
		var s image.Point
		if cell.Content != nil {
			// TODO: t.columnWidthsInPixels should not be accessed here.
			s = cell.Content.Measure(context, guigui.FixedWidthConstraints(t.table.columnWidthsInPixels[i]))
		} else {
			// Assume that every item can use a bold font.
			p := ListItemTextPadding(context)
			w := t.table.columnWidthsInPixels[i] - p.Start - p.End
			s = t.texts.At(i).Measure(context, guigui.FixedWidthConstraints(w))
			s = s.Add(image.Pt(p.Start+p.End, p.Top+p.Bottom))
		}
		w += t.table.columnWidthsInPixels[i]
		h = max(h, s.Y)
	}
	h = max(h, LineHeight(context))
	return image.Pt(w, h)
}

func (t *tableRowWidget[T]) selectable() bool {
	return t.row.selectable()
}

func (t *tableRowWidget[T]) listItem() ListItem[T] {
	return ListItem[T]{
		Content:      t,
		Unselectable: !t.selectable(),
		Movable:      t.row.Movable,
		Checked:      t.row.Checked,
		Value:        t.row.Value,
	}
}

type tableHeader[T comparable] struct {
	guigui.DefaultWidget

	columnTexts guigui.WidgetSlice[*Text]

	table *Table[T]
}

func (t *tableHeader[T]) SetTable(table *Table[T]) {
	t.table = table
}

func (t *tableHeader[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	t.columnTexts.SetLen(len(t.table.columns))
	for i := range t.columnTexts.Len() {
		adder.AddWidget(t.columnTexts.At(i))
	}

	for i, column := range t.table.columns {
		t.columnTexts.At(i).SetValue(column.HeaderText)
		t.columnTexts.At(i).SetHorizontalAlign(column.HeaderTextHorizontalAlign)
		t.columnTexts.At(i).SetVerticalAlign(VerticalAlignMiddle)
	}

	return nil
}

func (t *tableHeader[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	bounds := widgetBounds.Bounds()
	offsetX, _ := t.table.list.scrollOffset()
	pt := bounds.Min
	pt.X += int(offsetX)
	pt.X += RoundedCornerRadius(context)
	for i := range t.columnTexts.Len() {
		if i >= len(t.table.columnWidthsInPixels) {
			break
		}
		textMin := pt.Add(image.Pt(UnitSize(context)/4, 0))
		width := t.table.columnWidthsInPixels[i] - UnitSize(context)/2
		textBounds := image.Rectangle{
			Min: textMin,
			Max: textMin.Add(image.Pt(width, tableHeaderHeight(context))),
		}
		layouter.LayoutWidget(t.columnTexts.At(i), textBounds)
		pt.X += t.table.columnWidthsInPixels[i]
	}
}

func (t *tableHeader[T]) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if len(t.table.columnWidthsInPixels) <= 1 {
		return
	}
	u := UnitSize(context)
	b := widgetBounds.Bounds()
	x := b.Min.X + RoundedCornerRadius(context)
	offsetX, _ := t.table.list.scrollOffset()
	x += int(offsetX)
	for _, width := range t.table.columnWidthsInPixels[:len(t.table.columnWidthsInPixels)-1] {
		x += width
		x0 := float32(x)
		x1 := x0
		y0 := float32(b.Min.Y + u/4)
		y1 := float32(b.Min.Y + tableHeaderHeight(context) - u/4)
		clr := draw.Color2(context.ColorMode(), draw.SemanticColorBase, 0.9, 0.4)
		if !context.IsEnabled(t) {
			clr = draw.Color2(context.ColorMode(), draw.SemanticColorBase, 0.8, 0.3)
		}
		vector.StrokeLine(dst, x0, y0, x1, y1, float32(context.Scale()), clr, false)
	}
}
