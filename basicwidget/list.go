// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"fmt"
	"image"
	"image/color"
	"maps"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

// EnvKeyListItemColorType is the environment key for obtaining a [ListItemColorType] from a list item.
// This is provided by the list's internal content widget, so descendant widgets of list items
// can query their color type without needing to know their index.
//
// This value is available after the build phase (e.g., during the layout phase).
//
// [guigui.Context.Env] with this key might return (nil, false) when the item is out of the viewport.
var EnvKeyListItemColorType guigui.EnvKey = guigui.GenerateEnvKey()

type ListStyle int

const (
	ListStyleNormal ListStyle = iota
	ListStyleSidebar
	ListStyleMenu
)

// ListItemColorType represents the color state of a list item.
type ListItemColorType int

const (
	ListItemColorTypeDefault ListItemColorType = iota
	ListItemColorTypeHighlighted
	ListItemColorTypeSelectedInUnfocusedList
	ListItemColorTypeItemDisabled
	ListItemColorTypeListDisabled
)

// TextColor returns the text color for the given color type.
func (t ListItemColorType) TextColor(context *guigui.Context) color.Color {
	switch t {
	case ListItemColorTypeHighlighted:
		return draw.Color2(context.ColorMode(), draw.SemanticColorBase, 1, 1)
	case ListItemColorTypeItemDisabled, ListItemColorTypeListDisabled:
		return basicwidgetdraw.TextColor(context.ColorMode(), false)
	default:
		return basicwidgetdraw.TextColor(context.ColorMode(), true)
	}
}

// BackgroundColor returns the background color for the given color type.
// BackgroundColor returns nil when no special background should be applied.
func (t ListItemColorType) BackgroundColor(context *guigui.Context) color.Color {
	switch t {
	case ListItemColorTypeHighlighted:
		return draw.Color2(context.ColorMode(), draw.SemanticColorAccent, 0.6, 0.475)
	case ListItemColorTypeListDisabled:
		return draw.Color2(context.ColorMode(), draw.SemanticColorBase, 0.8, 0.35)
	case ListItemColorTypeSelectedInUnfocusedList:
		return draw.Color2(context.ColorMode(), draw.SemanticColorBase, 0.85, 0.475)
	default:
		return nil
	}
}

var (
	listEventItemSelected        guigui.EventKey = guigui.GenerateEventKey()
	listEventItemsSelected       guigui.EventKey = guigui.GenerateEventKey()
	listEventItemsMoved          guigui.EventKey = guigui.GenerateEventKey()
	listEventItemsCanMove        guigui.EventKey = guigui.GenerateEventKey()
	listEventItemExpanderToggled guigui.EventKey = guigui.GenerateEventKey()
)

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
	Padding      guigui.Padding
	Collapsed    bool
}

func (l *ListItem[T]) selectable() bool {
	return !l.Header && !l.Unselectable && !l.Border && !l.Disabled
}

// writeStateKey writes the item's state into w.
func (l *ListItem[T]) writeStateKey(w *guigui.StateKeyWriter) {
	w.WriteString(l.Text)
	writeColor(w, l.TextColor)
	w.WriteBool(l.Header)
	w.WriteWidget(l.Content)
	w.WriteString(l.KeyText)
	w.WriteBool(l.Unselectable)
	w.WriteBool(l.Border)
	w.WriteBool(l.Disabled)
	w.WriteBool(l.Movable)
	w.WriteInt(l.IndentLevel)
	writePadding(w, l.Padding)
	w.WriteBool(l.Collapsed)

	// Value is not written because it's opaque and only used for
	// identifying the item, so it does not affect the widget state.
}

type List[T comparable] struct {
	guigui.DefaultWidget

	abstractListItems []abstractListItem[T]
	listItemWidgets   guigui.WidgetSlice[*listItemWidget[T]]
	background1       listBackground1[T]
	content           listContent[T]
	panel             listPanel[T]
	frame             listFrame

	listItemHeightPlus1 int
	headerHeight        int
	footerHeight        int
}

func (l *List[T]) SetBackground(widget guigui.Widget) {
	l.content.SetBackground(widget)
}

func (l *List[T]) SetStripeVisible(visible bool) {
	l.content.SetStripeVisible(visible)
}

// SetUnfocusedSelectionVisible sets whether to show the selection background when the list is unfocused.
// The default is true.
func (l *List[T]) SetUnfocusedSelectionVisible(visible bool) {
	l.content.SetUnfocusedSelectionVisible(visible)
}

func (l *List[T]) SetMultiSelection(multi bool) {
	l.content.abstractList.SetMultiSelection(multi)
}

func (l *List[T]) WriteStateKey(w *guigui.StateKeyWriter) {
	w.WriteInt64(int64(l.listItemHeightPlus1))
	w.WriteInt64(int64(l.headerHeight))
	w.WriteInt64(int64(l.footerHeight))
}

func (l *List[T]) SetItemHeight(height int) {
	l.listItemHeightPlus1 = height + 1
}

func (l *List[T]) OnItemSelected(f func(context *guigui.Context, index int)) {
	l.content.OnItemSelected(f)
}

func (l *List[T]) OnItemsSelected(f func(context *guigui.Context, indices []int)) {
	l.content.OnItemsSelected(f)
}

func (l *List[T]) OnItemsMoved(f func(context *guigui.Context, from, count, to int)) {
	l.content.OnItemsMoved(f)
}

func (l *List[T]) OnItemsCanMove(f func(context *guigui.Context, from, count, to int) bool) {
	l.content.OnItemsCanMove(f)
}

func (l *List[T]) OnItemExpanderToggled(f func(context *guigui.Context, index int, expanded bool)) {
	l.content.OnItemExpanderToggled(f)
}

func (l *List[T]) SetCheckmarkIndex(index int) {
	l.content.SetCheckmarkIndex(index)
}

func (l *List[T]) SetHeaderHeight(height int) {
	if l.headerHeight == height {
		return
	}
	l.headerHeight = height
	l.frame.SetHeaderHeight(height)
}

func (l *List[T]) SetFooterHeight(height int) {
	if l.footerHeight == height {
		return
	}
	l.footerHeight = height
	l.frame.SetFooterHeight(height)
}

func (l *List[T]) ItemBounds(index int) image.Rectangle {
	return l.content.ItemBounds(index)
}

func (l *List[T]) itemYFromIndexForMenu(context *guigui.Context, index int) (int, bool) {
	return l.content.itemYFromIndexForMenu(context, index)
}

func (l *List[T]) resetHoveredItemIndex() {
	l.content.resetHoveredItemIndex()
}

func (l *List[T]) setKeyboardHighlightIndex(index int) {
	l.content.setKeyboardHighlightIndex(index)
}

// IsItemAvailable reports whether the item at the given index is available in the list
// (i.e., not hidden by a collapsed ancestor).
func (l *List[T]) IsItemAvailable(index int) bool {
	return l.content.isItemAvailable(index)
}

// IsItemInViewport reports whether the item at the given index is currently visible in the viewport.
//
// IsItemInViewport is available after the layout phase.
func (l *List[T]) IsItemInViewport(index int) bool {
	return l.content.isItemInViewport(index)
}

func (l *List[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&l.background1)
	adder.AddWidget(&l.panel)
	adder.AddWidget(&l.frame)

	l.background1.setListContent(&l.content)
	l.content.listPanel = &l.panel
	l.panel.setContent(&l.content)

	for i := range l.listItemWidgets.Len() {
		item := l.listItemWidgets.At(i)
		item.text.SetBold(item.item.Header || l.content.Style() == ListStyleSidebar && l.SelectedItemIndex() == i)
	}

	return nil
}

func (l *List[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	bounds := widgetBounds.Bounds()
	bounds.Min.Y += l.headerHeight
	bounds.Max.Y -= l.footerHeight

	layouter.LayoutWidget(&l.background1, widgetBounds.Bounds())
	layouter.LayoutWidget(&l.panel, bounds)
	layouter.LayoutWidget(&l.frame, widgetBounds.Bounds())
}

func (l *List[T]) SelectedItemCount() int {
	return l.content.SelectedItemCount()
}

func (l *List[T]) SelectedItemIndex() int {
	return l.content.SelectedItemIndex()
}

func (l *List[T]) AppendSelectedItemIndices(indices []int) []int {
	return l.content.AppendSelectedItemIndices(indices)
}

func (l *List[T]) SelectedItem() (ListItem[T], bool) {
	return l.ItemByIndex(l.content.SelectedItemIndex())
}

func (l *List[T]) ItemByIndex(index int) (ListItem[T], bool) {
	if index < 0 || index >= l.listItemWidgets.Len() {
		return ListItem[T]{}, false
	}
	return l.listItemWidgets.At(index).item, true
}

func (l *List[T]) IndexByValue(value T) int {
	for i := range l.listItemWidgets.Len() {
		if l.listItemWidgets.At(i).item.Value == value {
			return i
		}
	}
	return -1
}

func (l *List[T]) SetItemsByStrings(strs []string) {
	items := make([]ListItem[T], len(strs))
	for i, str := range strs {
		items[i].Text = str
	}
	l.SetItems(items)
}

func (l *List[T]) SetItems(items []ListItem[T]) {
	l.abstractListItems = adjustSliceSize(l.abstractListItems, len(items))
	l.listItemWidgets.SetLen(len(items))

	for i, item := range items {
		l.listItemWidgets.At(i).setListItem(item)
		l.listItemWidgets.At(i).setHeight(l.listItemHeightPlus1 - 1)
		l.listItemWidgets.At(i).setStyle(l.content.Style())
		l.abstractListItems[i].Content = l.listItemWidgets.At(i)
		l.abstractListItems[i].Unselectable = !item.selectable()
		l.abstractListItems[i].Movable = item.Movable
		l.abstractListItems[i].Value = item.Value
		l.abstractListItems[i].IndentLevel = item.IndentLevel
		l.abstractListItems[i].Padding = item.Padding
		l.abstractListItems[i].Collapsed = item.Collapsed
		l.abstractListItems[i].index = i
		l.abstractListItems[i].listContent = &l.content
	}
	l.content.SetItems(l.abstractListItems)
}

func (l *List[T]) ItemCount() int {
	return len(l.abstractListItems)
}

func (l *List[T]) ID(index int) any {
	return l.abstractListItems[index].Value
}

func (l *List[T]) SelectItemByIndex(index int) {
	l.content.SelectItemByIndex(index)
}

func (l *List[T]) SelectItemsByIndices(indices []int) {
	l.content.SelectItemsByIndices(indices)
}

func (l *List[T]) SelectAllItems() {
	l.content.SelectAllItems()
}

func (l *List[T]) SelectItemByValue(value T) {
	l.content.SelectItemByValue(value)
}

func (l *List[T]) SelectItemsByValues(values []T) {
	l.content.SelectItemsByValues(values)
}

func (l *List[T]) JumpToItemByIndex(index int) {
	l.content.JumpToItemByIndex(index)
}

func (l *List[T]) EnsureItemVisibleByIndex(index int) {
	l.content.EnsureItemVisibleByIndex(index)
}

func (l *List[T]) SetStyle(style ListStyle) {
	l.content.SetStyle(style)
	l.frame.SetStyle(style)
}

func (l *List[T]) SetItemString(str string, index int) {
	l.listItemWidgets.At(index).setText(str)
}

func (l *List[T]) setContentWidth(width int) {
	l.content.SetContentWidth(width)
}

func (l *List[T]) scrollOffset() (float64, float64) {
	return l.panel.scrollOffset()
}

func (l *List[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	s := l.content.Measure(context, constraints)
	s.Y += l.headerHeight + l.footerHeight
	return s
}

func (l *List[T]) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	return nil
}

type listItemWidget[T comparable] struct {
	guigui.DefaultWidget

	text    Text
	keyText Text

	item        ListItem[T]
	heightPlus1 int
	style       ListStyle

	layout             guigui.LinearLayout
	layoutItems        []guigui.LinearLayoutItem
	wrapperLayoutItems []guigui.LinearLayoutItem
}

func (l *listItemWidget[T]) WriteStateKey(w *guigui.StateKeyWriter) {
	l.item.writeStateKey(w)
	w.WriteInt(l.heightPlus1)
	w.WriteUint64(uint64(l.style))
}

func (l *listItemWidget[T]) setListItem(listItem ListItem[T]) {
	if l.item == listItem {
		return
	}
	l.item = listItem
	l.resetLayout()
}

func (l *listItemWidget[T]) setHeight(height int) {
	if l.heightPlus1 == height+1 {
		return
	}
	l.heightPlus1 = height + 1
	l.resetLayout()
}

func (l *listItemWidget[T]) setStyle(style ListStyle) {
	if l.style == style {
		return
	}
	l.style = style
	l.resetLayout()
}

func (l *listItemWidget[T]) setText(text string) {
	l.item.Text = text
}

func (l *listItemWidget[T]) textColor() color.Color {
	return l.item.TextColor
}

func (l *listItemWidget[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if l.item.Content != nil {
		adder.AddWidget(l.item.Content)
	} else {
		adder.AddWidget(&l.text)
	}
	adder.AddWidget(&l.keyText)

	l.text.SetValue(l.item.Text)
	l.text.SetVerticalAlign(VerticalAlignMiddle)
	l.keyText.SetOpacity(0.5)
	l.keyText.SetValue(l.item.KeyText)
	l.keyText.SetVerticalAlign(VerticalAlignMiddle)
	l.keyText.SetHorizontalAlign(HorizontalAlignEnd)

	context.SetEnabled(l, !l.item.Disabled)

	return nil
}

func (l *listItemWidget[T]) resetLayout() {
	l.layout = guigui.LinearLayout{}
	l.layoutItems = slices.Delete(l.layoutItems, 0, len(l.layoutItems))
}

func (l *listItemWidget[T]) ensureLayout(context *guigui.Context) guigui.LinearLayout {
	if len(l.layout.Items) > 0 {
		return l.layout
	}

	layout := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Gap:       LineHeight(context),
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
	if l.item.KeyText != "" {
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
		l.wrapperLayoutItems = slices.Delete(l.wrapperLayoutItems, 0, len(l.wrapperLayoutItems))
		l.wrapperLayoutItems = append(l.wrapperLayoutItems,
			guigui.LinearLayoutItem{
				Layout: layout,
				Size:   guigui.FixedSize(h),
			})
		l.layout = guigui.LinearLayout{
			Direction: guigui.LayoutDirectionVertical,
			Items:     l.wrapperLayoutItems,
		}
	} else {
		l.layout = layout
	}
	return l.layout
}

func (l *listItemWidget[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	// Skip if the widget is not visible and has no content widget.
	// If the widget has a content widget, this cannot be skipped because the content widget might have visible child widgets like a popup.
	if widgetBounds.VisibleBounds().Empty() && l.item.Content == nil {
		return
	}

	l.ensureLayout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)

	// Set text colors based on the item's color type.
	var clr color.Color
	if v, ok := context.Env(l, EnvKeyListItemColorType); ok {
		if ct, ok := v.(ListItemColorType); ok {
			if ct == ListItemColorTypeDefault {
				if c := l.textColor(); c != nil {
					clr = c
				} else {
					clr = ct.TextColor(context)
				}
			} else {
				clr = ct.TextColor(context)
			}
		}
	}
	l.text.SetColor(clr)
	l.keyText.SetColor(clr)
}

func (l *listItemWidget[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return l.ensureLayout(context).Measure(context, constraints)
}

func (l *listItemWidget[T]) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if l.item.Border {
		u := UnitSize(context)
		b := widgetBounds.Bounds()
		x0 := float32(b.Min.X + u/4)
		x1 := float32(b.Max.X - u/4)
		y := float32(b.Min.Y) + float32(b.Dy())/2
		width := float32(1 * context.Scale())
		vector.StrokeLine(dst, x0, y, x1, y, width, draw.Color(context.ColorMode(), draw.SemanticColorBase, 0.8), false)
		return
	}
	/*if l.item.Header {
		bounds := widgetBounds.Bounds()
		draw.DrawRoundedRect(context, dst, bounds, draw.Color(context.ColorMode(), draw.SemanticColorBase, 0.8), RoundedCornerRadius(context))
	}*/
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

type abstractListItem[T comparable] struct {
	Content      guigui.Widget
	Unselectable bool
	Movable      bool
	Value        T
	IndentLevel  int
	Padding      guigui.Padding
	Collapsed    bool

	index       int
	available   bool
	listContent *listContent[T]
}

func (a abstractListItem[T]) value() T {
	return a.Value
}

func (a abstractListItem[T]) selectable() bool {
	return !a.Unselectable
}

func (a abstractListItem[T]) visible() bool {
	return a.listContent.isItemAvailable(a.index)
}

type collapsedEntry struct {
	collapsed  bool
	generation uint64
}

type listContent[T comparable] struct {
	guigui.DefaultWidget

	customBackground guigui.Widget
	background2      listBackground2[T]
	checkmark        Image
	expanderImages   guigui.WidgetSlice[*Image]

	abstractList              abstractList[T, abstractListItem[T]]
	stripeVisible             bool
	unfocusedSelectionHidden  bool
	style                     ListStyle
	checkmarkIndexPlus1       int
	hoveredItemIndexPlus1     int
	lastHoveredItemIndexPlus1 int

	keyboardHighlightIndexPlus1 int
	lastCursorPosition          image.Point

	indexToJumpPlus1          int
	indexToEnsureVisiblePlus1 int
	jumpTick                  int64
	dragSrcIndexPlus1         int
	dragDstIndexPlus1         int
	pressStartPlus1           image.Point
	startPressingIndexPlus1   int
	contentWidthPlus1         int
	widthForCachedHeight      int
	cachedHeight              int

	itemBoundsForLayoutFromIndex []image.Rectangle
	visibleBounds                image.Rectangle

	// tmpAvailableIndices is a scratch buffer reused by appendAvailableIndices
	// callers to avoid allocation on each Build/Layout/Tick call.
	tmpAvailableIndices []int

	// tmpSelectedIndices is a scratch buffer reused by AppendSelectedItemIndices
	// callers to avoid allocation each frame.
	tmpSelectedIndices []int

	// measuredContentHeights caches measured content heights per item index
	// for the duration of a single Layout call. Cleared at the start of layoutItems.
	measuredContentHeights map[int]int

	treeItemCollapsedImage *ebiten.Image
	treeItemExpandedImage  *ebiten.Image

	widgetToIndex map[guigui.Widget]int

	// expandAnimatingIndexPlus1 is the 1-based index of the item currently animating (0 = none).
	expandAnimatingIndexPlus1 int
	// expandAnimatingChildrenEnd is the exclusive end index of the animating item's children range.
	// Children are items at indices [expandAnimatingIndexPlus1, expandAnimatingChildrenEnd).
	expandAnimatingChildrenEnd int
	// expandAnimatingCount is the remaining animation ticks.
	expandAnimatingCount int
	// prevCollapsed stores the previous Collapsed state keyed by item Value, to detect changes in SetItems.
	prevCollapsed           map[T]collapsedEntry
	prevCollapsedGeneration uint64
	// onceRendered prevents animation on the very first render.
	onceRendered bool

	onItemSelected  func(index int)
	onItemsSelected func(indices []int)

	// listPanel is a back-reference to the list panel for virtual scrolling.
	listPanel *listPanel[T]
}

func (l *listContent[T]) availableItemCount() int {
	var count int
	for i := range l.abstractList.ItemCount() {
		if l.isItemAvailable(i) {
			count++
		}
	}
	return count
}

// measureAvailableItemHeight returns the height of the item at the given
// available-item index, or 0 if the index is out of range.
func (l *listContent[T]) measureAvailableItemHeight(context *guigui.Context, availableIndex int) int {
	var idx int
	for i := range l.abstractList.ItemCount() {
		if !l.isItemAvailable(i) {
			continue
		}
		if idx == availableIndex {
			return l.measureItemHeight(context, i, l.contentWidth())
		}
		idx++
	}
	return 0
}

func (l *listContent[T]) contentWidth() int {
	if l.contentWidthPlus1 > 0 {
		return l.contentWidthPlus1 - 1
	}
	return 0
}

func (l *listContent[T]) SetBackground(widget guigui.Widget) {
	l.customBackground = widget
}

func (l *listContent[T]) OnItemSelected(f func(context *guigui.Context, index int)) {
	guigui.SetEventHandler(l, listEventItemSelected, f)
}

func (l *listContent[T]) OnItemsSelected(f func(context *guigui.Context, indices []int)) {
	guigui.SetEventHandler(l, listEventItemsSelected, f)
}

func (l *listContent[T]) OnItemsMoved(f func(context *guigui.Context, from, count, to int)) {
	guigui.SetEventHandler(l, listEventItemsMoved, f)
}

func (l *listContent[T]) OnItemsCanMove(f func(context *guigui.Context, from, count, to int) bool) {
	guigui.SetEventHandler(l, listEventItemsCanMove, f)
}

func (l *listContent[T]) OnItemExpanderToggled(f func(context *guigui.Context, index int, expanded bool)) {
	guigui.SetEventHandler(l, listEventItemExpanderToggled, f)
}

func (l *listContent[T]) WriteStateKey(w *guigui.StateKeyWriter) {
	l.abstractList.writeStateKey(w)
	w.WriteUint64(uint64(l.style))
	w.WriteInt(l.checkmarkIndexPlus1)
	w.WriteInt(l.contentWidthPlus1)
	w.WriteInt(l.hoveredItemIndexPlus1)
	w.WriteInt(l.lastHoveredItemIndexPlus1)
	w.WriteInt(l.keyboardHighlightIndexPlus1)
	w.WriteInt(l.expandAnimatingIndexPlus1)
	w.WriteInt(l.expandAnimatingChildrenEnd)
	w.WriteInt(l.expandAnimatingCount)
}

func (l *listContent[T]) SetCheckmarkIndex(index int) {
	if index < 0 {
		index = -1
	}
	l.checkmarkIndexPlus1 = index + 1
}

func (l *listContent[T]) SetContentWidth(width int) {
	l.contentWidthPlus1 = width + 1
}

func (l *listContent[T]) ItemBounds(index int) image.Rectangle {
	if index < 0 || index >= len(l.itemBoundsForLayoutFromIndex) {
		return image.Rectangle{}
	}
	return l.itemBoundsForLayoutFromIndex[index]
}

func (l *listContent[T]) isItemAvailable(index int) bool {
	item, ok := l.abstractList.ItemByIndex(index)
	if !ok {
		return false
	}
	return item.available
}

func (l *listContent[T]) prevAvailableItem(index int) (int, bool) {
	for i := index - 1; i >= 0; i-- {
		if l.isItemAvailable(i) {
			return i, true
		}
	}
	return 0, false
}

func (l *listContent[T]) nextAvailableItem(index int) (int, bool) {
	for i := index + 1; i < l.abstractList.ItemCount(); i++ {
		if l.isItemAvailable(i) {
			return i, true
		}
	}
	return 0, false
}

func (l *listContent[T]) isItemInViewport(index int) bool {
	if index < 0 || index >= len(l.itemBoundsForLayoutFromIndex) {
		return false
	}
	r := l.itemBoundsForLayoutFromIndex[index]
	if r.Empty() {
		return false
	}
	return r.Max.Y > l.visibleBounds.Min.Y && r.Min.Y < l.visibleBounds.Max.Y
}

func (l *listContent[T]) Env(context *guigui.Context, key guigui.EnvKey, source *guigui.EnvSource) (any, bool) {
	switch key {
	case EnvKeyListItemColorType:
		child := source.Child
		if child == nil {
			return nil, false
		}
		if i, ok := l.widgetToIndex[child]; ok {
			return l.itemColorType(context, i), true
		}
	}
	return nil, false
}

func (l *listContent[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if l.customBackground != nil {
		adder.AddWidget(l.customBackground)
	}
	adder.AddWidget(&l.background2)
	l.expanderImages.SetLen(l.abstractList.ItemCount())

	// Only add items around the top item, extending downward and upward until
	// the accumulated height in each direction reaches the app bounds height.
	// The actual viewport size is unknown during the Build phase, so the app
	// bounds height is used as a safe upper bound. Iterating upward is also
	// necessary for the transient case where the top item is at or near the
	// end of the list (e.g., while scrolling to the bottom), in which case the
	// items above the top item must still be added to the widget tree.
	// Item heights are measured with default constraints so each item widget
	// determines its size freely.
	l.tmpAvailableIndices = l.appendAvailableIndices(l.tmpAvailableIndices[:0])
	availableIndices := l.tmpAvailableIndices
	topIdx, _ := l.listPanel.topItem()
	if topIdx < 0 {
		topIdx = 0
	}
	if topIdx > len(availableIndices) {
		topIdx = len(availableIndices)
	}
	appBoundsHeight := context.AppBounds().Dy()

	// Find the end of the downward range [topIdx, hi).
	hi := topIdx
	var downH int
	for ai := topIdx; ai < len(availableIndices); ai++ {
		i := availableIndices[ai]
		item, _ := l.abstractList.ItemByIndex(i)
		h := item.Content.Measure(context, guigui.Constraints{}).Y
		downH += h + item.Padding.Top + item.Padding.Bottom
		hi = ai + 1
		if downH >= appBoundsHeight {
			break
		}
	}

	// Find the start of the upward range [lo, topIdx).
	lo := topIdx
	var upH int
	for ai := topIdx - 1; ai >= 0; ai-- {
		i := availableIndices[ai]
		item, _ := l.abstractList.ItemByIndex(i)
		h := item.Content.Measure(context, guigui.Constraints{}).Y
		upH += h + item.Padding.Top + item.Padding.Bottom
		lo = ai
		if upH >= appBoundsHeight {
			break
		}
	}

	// Add the items in forward order so the child order matches the visual order.
	for ai := lo; ai < hi; ai++ {
		i := availableIndices[ai]
		item, _ := l.abstractList.ItemByIndex(i)
		if l.checkmarkIndexPlus1 == i+1 {
			adder.AddWidget(&l.checkmark)
		}
		var hasChild bool
		if nextItem, ok := l.abstractList.ItemByIndex(i + 1); ok {
			hasChild = nextItem.IndentLevel > item.IndentLevel
		}

		if hasChild {
			img := l.expanderImages.At(i)
			if !item.Collapsed {
				img.SetImage(l.treeItemExpandedImage)
			} else {
				img.SetImage(l.treeItemCollapsedImage)
			}
			adder.AddWidget(img)
		}
		adder.AddWidget(item.Content)
	}

	if l.onItemSelected == nil {
		l.onItemSelected = func(index int) {
			guigui.DispatchEvent(l, listEventItemSelected, index)
		}
	}
	l.abstractList.OnItemSelected(l.onItemSelected)

	if l.onItemsSelected == nil {
		l.onItemsSelected = func(indices []int) {
			guigui.DispatchEvent(l, listEventItemsSelected, indices)
		}
	}
	l.abstractList.OnItemsSelected(l.onItemsSelected)

	l.background2.setListContent(l)

	var err error
	l.treeItemCollapsedImage, err = theResourceImages.Get("keyboard_arrow_right", context.ColorMode())
	if err != nil {
		return err
	}
	l.treeItemExpandedImage, err = theResourceImages.Get("keyboard_arrow_down", context.ColorMode())
	if err != nil {
		return err
	}

	// Build a widget-to-index map for O(1) lookup in Env.
	if l.widgetToIndex == nil {
		l.widgetToIndex = map[guigui.Widget]int{}
	}
	clear(l.widgetToIndex)
	for ai := lo; ai < hi; ai++ {
		i := availableIndices[ai]
		if item, ok := l.abstractList.ItemByIndex(i); ok {
			l.widgetToIndex[item.Content] = i
		}
	}

	return nil
}

func (l *listContent[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	cw := widgetBounds.Bounds().Dx()
	if l.contentWidthPlus1 > 0 {
		cw = l.contentWidthPlus1 - 1
	}

	l.itemBoundsForLayoutFromIndex = adjustSliceSize(l.itemBoundsForLayoutFromIndex, l.abstractList.ItemCount())
	clear(l.itemBoundsForLayoutFromIndex)

	l.visibleBounds = widgetBounds.VisibleBounds()

	if l.customBackground != nil {
		layouter.LayoutWidget(l.customBackground, widgetBounds.Bounds())
	}
	layouter.LayoutWidget(&l.background2, widgetBounds.Bounds())

	l.layoutItems(context, widgetBounds, layouter, cw)
}

// layoutItems lays out only items near the viewport using the listPanel's
// topItemIndex and topItemOffset.
func (l *listContent[T]) layoutItems(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter, cw int) {
	bounds := widgetBounds.Bounds()

	// Clear the per-Layout content height cache.
	clear(l.measuredContentHeights)

	// Build a mapping from available-item order to real index.
	l.tmpAvailableIndices = l.appendAvailableIndices(l.tmpAvailableIndices[:0])
	availableIndices := l.tmpAvailableIndices
	if len(availableIndices) == 0 {
		return
	}

	topIdx, topOff := l.listPanel.topItem()

	// Clamp topIdx to valid range.
	if topIdx >= len(availableIndices) {
		topIdx = max(0, len(availableIndices)-1)
		topOff = 0
	}
	if topIdx < 0 {
		topIdx = 0
		topOff = 0
	}

	baseX := bounds.Min.X + RoundedCornerRadius(context)
	viewportTop := bounds.Min.Y
	viewportBottom := bounds.Max.Y
	viewportHeight := viewportBottom - viewportTop

	// Normalize topIdx/topOff before layout so items are positioned correctly.
	l.normalizeTopItem(context, availableIndices, cw, bounds)
	topIdx, topOff = l.listPanel.topItem()

	animRate := l.expandAnimationRate()

	// TODO: Currently, each animating child's Y advance is scaled individually by the animation rate.
	// Ideally, children of the animating node would be treated as a single chunk (like Expander does),
	// where the chunk is laid out at full size and clipped at the animated height boundary.
	// This could be done with a two-pass approach: first lay out all children at full positions,
	// then clip and remove items outside the animated region. However, the current per-child scaling
	// is simpler and avoids the clipping complexity.

	// Pre-pass: measure heights from topIdx downward to detect bottom gap.
	// This is cheap since Measure results are typically cached.
	{
		y := RoundedCornerRadius(context) + topOff
		var reachedEnd bool
		for ai := topIdx; ai < len(availableIndices); ai++ {
			if y >= viewportHeight {
				break
			}
			h := l.measureItemHeight(context, availableIndices[ai], cw)
			if l.isExpandAnimating() && l.isChildOfExpandAnimatingItem(availableIndices[ai]) {
				y += int(float64(h) * animRate)
			} else {
				y += h
			}
			if ai == len(availableIndices)-1 {
				reachedEnd = true
			}
		}
		if reachedEnd {
			bottomY := y + RoundedCornerRadius(context)
			if gap := viewportHeight - bottomY; gap > 0 {
				topOff += gap
				// Pull topIdx backward if needed.
				for topOff > 0 && topIdx > 0 {
					topIdx--
					topOff -= l.measureItemHeight(context, availableIndices[topIdx], cw)
				}
				if topIdx == 0 && topOff > 0 {
					topOff = 0
				}
				l.listPanel.forceSetTopItem(topIdx, topOff, false)
			}
		}
	}

	// Start Y at the top of the viewport plus the topItemOffset.
	y := viewportTop + RoundedCornerRadius(context) + topOff

	// Track total measured height for average calculation.
	var totalMeasuredHeight int
	var measuredCount int

	// Lay out items downward from topIdx.
	for ai := topIdx; ai < len(availableIndices); ai++ {
		if y >= viewportBottom {
			break
		}
		i := availableIndices[ai]
		itemH := l.layoutItem(context, widgetBounds, layouter, i, baseX, y, cw)
		totalMeasuredHeight += itemH
		measuredCount++
		if l.isExpandAnimating() && l.isChildOfExpandAnimatingItem(i) {
			y += int(float64(itemH) * animRate)
		} else {
			y += itemH
		}
	}

	// Lay out items upward from topIdx-1 to fill any gap above.
	y = viewportTop + RoundedCornerRadius(context) + topOff
	for ai := topIdx - 1; ai >= 0; ai-- {
		i := availableIndices[ai]
		itemH := l.measureItemHeight(context, i, cw)
		totalMeasuredHeight += itemH
		measuredCount++

		if l.isExpandAnimating() && l.isChildOfExpandAnimatingItem(i) {
			y -= int(float64(itemH) * animRate)
		} else {
			y -= itemH
		}
		l.layoutItem(context, widgetBounds, layouter, i, baseX, y, cw)

		if y <= viewportTop {
			break
		}
	}

	// Report average measured height to listPanel.
	if measuredCount > 0 {
		l.listPanel.setEstimatedItemHeight(totalMeasuredHeight / measuredCount)
	} else {
		l.listPanel.setEstimatedItemHeight(0)
	}
}

// appendAvailableIndices appends the indices of available items to the slice.
func (l *listContent[T]) appendAvailableIndices(indices []int) []int {
	for i := range l.abstractList.ItemCount() {
		if !l.isItemAvailable(i) {
			continue
		}
		indices = append(indices, i)
	}
	return indices
}

// normalizeTopItem adjusts topItemIndex and topItemOffset so that
// topItemOffset stays within [-itemHeight, 0] by advancing or retreating
// topItemIndex when items cross boundaries.
func (l *listContent[T]) normalizeTopItem(context *guigui.Context, availableIndices []int, cw int, bounds image.Rectangle) {
	if len(availableIndices) == 0 {
		return
	}

	topIdx, topOff := l.listPanel.topItem()

	// Move topItemIndex forward when topItemOffset scrolled past an item.
	for topOff < 0 && topIdx < len(availableIndices)-1 {
		i := availableIndices[topIdx]
		itemH := l.measureItemHeight(context, i, cw)
		if -topOff >= itemH {
			topOff += itemH
			topIdx++
		} else {
			break
		}
	}

	// Move topItemIndex backward when topItemOffset is positive.
	for topOff > 0 && topIdx > 0 {
		topIdx--
		i := availableIndices[topIdx]
		itemH := l.measureItemHeight(context, i, cw)
		topOff -= itemH
	}

	// Clamp: don't allow gap above first item.
	if topIdx == 0 && topOff > 0 {
		topOff = 0
	}

	// Clamp: don't scroll past the last item.
	if topIdx >= len(availableIndices) {
		topIdx = len(availableIndices) - 1
		topOff = 0
	}

	// Clamp: don't allow negative index.
	if topIdx < 0 {
		topIdx = 0
		topOff = 0
	}

	l.listPanel.forceSetTopItem(topIdx, topOff, false)
}

// measureItemContentHeight returns the content height of an item, caching the
// result for the duration of the current Layout call.
func (l *listContent[T]) measureItemContentHeight(context *guigui.Context, index int, cw int) int {
	if h, ok := l.measuredContentHeights[index]; ok {
		return h
	}
	item, _ := l.abstractList.ItemByIndex(index)
	itemW := cw - 2*RoundedCornerRadius(context)
	itemW -= ListItemIndentSize(context, item.IndentLevel)
	itemW -= item.Padding.Start + item.Padding.End
	contentH := item.Content.Measure(context, guigui.FixedWidthConstraints(itemW)).Y
	if l.measuredContentHeights == nil {
		l.measuredContentHeights = map[int]int{}
	}
	l.measuredContentHeights[index] = contentH
	return contentH
}

// measureItemHeight returns the total height of an item (content + padding).
func (l *listContent[T]) measureItemHeight(context *guigui.Context, index int, cw int) int {
	contentH := l.measureItemContentHeight(context, index, cw)
	item, _ := l.abstractList.ItemByIndex(index)
	return contentH + item.Padding.Top + item.Padding.Bottom
}

// layoutItem lays out a single item at the given position and returns its total height.
func (l *listContent[T]) layoutItem(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter, index int, baseX int, y int, cw int) int {
	item, _ := l.abstractList.ItemByIndex(index)
	itemW := cw - 2*RoundedCornerRadius(context)
	itemW -= ListItemIndentSize(context, item.IndentLevel)
	itemW -= item.Padding.Start + item.Padding.End
	contentH := l.measureItemContentHeight(context, index, cw)
	itemH := contentH + item.Padding.Top + item.Padding.Bottom

	p := image.Pt(baseX, y)

	// Record item bounds.
	{
		itemP := p
		if l.checkmarkIndexPlus1 > 0 {
			itemP.X += listItemCheckmarkSize(context) + listItemTextAndImagePadding(context)
		}
		itemP.X += ListItemIndentSize(context, item.IndentLevel)
		itemP.X += item.Padding.Start
		itemP.Y = l.adjustItemY(context, itemP.Y)
		itemP.Y += item.Padding.Top
		l.itemBoundsForLayoutFromIndex[index] = image.Rectangle{
			Min: itemP,
			Max: itemP.Add(image.Pt(itemW, contentH)),
		}
	}

	// Skip widget layout for items outside the visible bounds.
	itemTop := p.Y
	itemBottom := p.Y + itemH
	if itemTop >= l.visibleBounds.Max.Y || itemBottom <= l.visibleBounds.Min.Y {
		return itemH
	}

	if l.checkmarkIndexPlus1 == index+1 {
		imgSize := listItemCheckmarkSize(context)
		imgP := p
		imgP.X += ListItemIndentSize(context, item.IndentLevel)
		imgP.X += UnitSize(context) / 4
		imgP.Y += (contentH - imgSize) / 2
		imgP.Y += UnitSize(context) / 16
		imgP.Y += item.Padding.Top
		imgP.Y = l.adjustItemY(context, imgP.Y)
		layouter.LayoutWidget(&l.checkmark, image.Rectangle{
			Min: imgP,
			Max: imgP.Add(image.Pt(imgSize, imgSize)),
		})
	}

	if item.IndentLevel > 0 {
		var img *ebiten.Image
		var hasChild bool
		if nextItem, ok := l.abstractList.ItemByIndex(index + 1); ok {
			hasChild = nextItem.IndentLevel > item.IndentLevel
		}
		if hasChild {
			if item.Collapsed {
				img = l.treeItemCollapsedImage
			} else {
				img = l.treeItemExpandedImage
			}
		}
		l.expanderImages.At(index).SetImage(img)
		expanderP := p
		expanderP.X += ListItemIndentSize(context, item.IndentLevel) - LineHeight(context)
		expanderP.Y += UnitSize(context) / 16
		expanderP.Y += item.Padding.Top
		s := image.Pt(
			LineHeight(context),
			contentH,
		)
		layouter.LayoutWidget(l.expanderImages.At(index), image.Rectangle{
			Min: expanderP,
			Max: expanderP.Add(s),
		})
	}

	itemP := p
	if l.checkmarkIndexPlus1 > 0 {
		itemP.X += listItemCheckmarkSize(context) + listItemTextAndImagePadding(context)
	}
	itemP.X += ListItemIndentSize(context, item.IndentLevel)
	itemP.X += item.Padding.Start
	itemP.Y = l.adjustItemY(context, itemP.Y)
	itemP.Y += item.Padding.Top
	r := image.Rectangle{
		Min: itemP,
		Max: itemP.Add(image.Pt(itemW, contentH)),
	}
	layouter.LayoutWidget(item.Content, r)
	l.itemBoundsForLayoutFromIndex[index] = r

	return itemH
}

func (l *listContent[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	var width int
	if l.contentWidthPlus1 > 0 {
		width = l.contentWidthPlus1 - 1
	} else if fixedWidth, ok := constraints.FixedWidth(); ok {
		width = fixedWidth
	}

	// Use the cached height if possible.
	// This can return an inaccurate height if the content widgets change, but this is very unlikely.
	// If a widget size is changed, widgets' Layout should be called soon anyway.
	if width > 0 && width == l.widthForCachedHeight {
		return image.Pt(width, l.cachedHeight)
	}

	hasCheckmark := l.checkmarkIndexPlus1 > 0
	offsetForCheckmark := listItemCheckmarkSize(context) + listItemTextAndImagePadding(context)

	var w, h int
	var animatingChildrenH int
	for i := range l.abstractList.ItemCount() {
		if !l.isItemAvailable(i) {
			continue
		}
		item, _ := l.abstractList.ItemByIndex(i)
		var constraint guigui.Constraints
		// If width is 0, there is no constraint.
		// This is used mainly for a menu list.
		if width > 0 {
			itemW := width - 2*RoundedCornerRadius(context)
			if hasCheckmark {
				itemW -= offsetForCheckmark
			}
			itemW -= ListItemIndentSize(context, item.IndentLevel)
			itemW -= item.Padding.Start + item.Padding.End
			constraint = guigui.FixedWidthConstraints(itemW)
		}
		s := item.Content.Measure(context, constraint)
		w = max(w, s.X+ListItemIndentSize(context, item.IndentLevel)+item.Padding.Start+item.Padding.End)
		itemH := s.Y + item.Padding.Top + item.Padding.Bottom
		if l.isExpandAnimating() && l.isChildOfExpandAnimatingItem(i) {
			animatingChildrenH += itemH
		} else {
			h += itemH
		}
	}
	if l.isExpandAnimating() && animatingChildrenH > 0 {
		h += int(float64(animatingChildrenH) * l.expandAnimationRate())
	}
	w += 2 * RoundedCornerRadius(context)
	h += 2 * RoundedCornerRadius(context)
	if hasCheckmark {
		w += offsetForCheckmark
	}
	if width > 0 {
		w = width
		// Don't cache height during animation since it changes every tick.
		if !l.isExpandAnimating() {
			l.widthForCachedHeight = width
			l.cachedHeight = h
		}
	}
	return image.Pt(w, h)
}

func (l *listContent[T]) hasMovableItems() bool {
	for i := range l.abstractList.ItemCount() {
		if !l.isItemAvailable(i) {
			continue
		}
		item, ok := l.abstractList.ItemByIndex(i)
		if !ok {
			continue
		}
		if item.Movable {
			return true
		}
	}
	return false
}

func (l *listContent[T]) ItemByIndex(index int) (abstractListItem[T], bool) {
	return l.abstractList.ItemByIndex(index)
}

func (l *listContent[T]) IsSelectedItemIndex(index int) bool {
	return l.abstractList.IsSelectedItemIndex(index)
}

func (l *listContent[T]) SelectedItemCount() int {
	return l.abstractList.SelectedItemCount()
}

func (l *listContent[T]) SelectedItemIndex() int {
	return l.abstractList.SelectedItemIndex()
}

func (l *listContent[T]) AppendSelectedItemIndices(indices []int) []int {
	return l.abstractList.AppendSelectedItemIndices(indices)
}

func (l *listContent[T]) SetItems(items []abstractListItem[T]) {
	l.prevCollapsedGeneration++
	gen := l.prevCollapsedGeneration

	if l.prevCollapsed == nil {
		l.prevCollapsed = make(map[T]collapsedEntry, len(items))
	}

	// Detect collapse state changes and start animation.
	// Also update each entry's generation to mark it as current.
	// Also precompute item availability based on collapsed ancestors.
	var lastCollapsedIndentLevel int
	for i, item := range items {
		prev, ok := l.prevCollapsed[item.Value]
		if l.onceRendered && ok && prev.collapsed != item.Collapsed {
			l.expandAnimatingIndexPlus1 = i + 1
			l.expandAnimatingCount = expandCollapseMaxCount() - l.expandAnimatingCount
			// Compute children range: all items after i with indent > item's indent,
			// stopping at the first item with indent <= item's indent.
			l.expandAnimatingChildrenEnd = len(items)
			for j := i + 1; j < len(items); j++ {
				if items[j].IndentLevel <= item.IndentLevel {
					l.expandAnimatingChildrenEnd = j
					break
				}
			}
		}
		l.prevCollapsed[item.Value] = collapsedEntry{
			collapsed:  item.Collapsed,
			generation: gen,
		}

		if lastCollapsedIndentLevel > 0 && item.IndentLevel > lastCollapsedIndentLevel {
			// During a collapse animation, treat children of the animating item as available
			// so they remain visible and can animate out.
			if l.isExpandAnimating() && l.isChildOfExpandAnimatingItem(i) {
				items[i].available = true
			} else {
				items[i].available = false
			}
		} else {
			items[i].available = true
			if item.Collapsed {
				lastCollapsedIndentLevel = item.IndentLevel
			} else {
				lastCollapsedIndentLevel = 0
			}
		}
	}

	// Remove stale entries from prevCollapsed.
	maps.DeleteFunc(l.prevCollapsed, func(_ T, e collapsedEntry) bool {
		return e.generation != gen
	})

	l.abstractList.SetItems(items)
	// Invalidate the cached height so that Measure recalculates with the new items.
	l.widthForCachedHeight = 0
}

func (l *listContent[T]) isExpandAnimating() bool {
	return l.expandAnimatingCount > 0
}

func (l *listContent[T]) expandAnimationRate() float64 {
	if !l.isExpandAnimating() {
		return 1
	}
	idx := l.expandAnimatingIndexPlus1 - 1
	item, ok := l.abstractList.ItemByIndex(idx)
	if !ok {
		return 1
	}
	rate := 1 - float64(l.expandAnimatingCount)/float64(expandCollapseMaxCount())
	if item.Collapsed {
		// Collapsing: rate goes from 1 to 0.
		rate = 1 - rate
	}
	return rate
}

// isChildOfExpandAnimatingItem reports whether the item at index is a child
// of the currently animating item, using the precomputed children range.
func (l *listContent[T]) isChildOfExpandAnimatingItem(index int) bool {
	return index > l.expandAnimatingIndexPlus1-1 && index < l.expandAnimatingChildrenEnd
}

func (l *listContent[T]) SelectItemByIndex(index int) {
	l.selectItemByIndex(index, false)
}

func (l *listContent[T]) SelectItemsByIndices(indices []int) {
	l.abstractList.SelectItemsByIndices(indices, false)
}

func (l *listContent[T]) SelectAllItems() {
	l.abstractList.SelectAllItems(false)
}

func (l *listContent[T]) selectItemByIndex(index int, forceFireEvents bool) {
	l.abstractList.SelectItemByIndex(index, forceFireEvents)
}

func (l *listContent[T]) extendItemSelectionByIndex(index int, forceFireEvents bool) {
	l.abstractList.ExtendItemSelectionByIndex(index, forceFireEvents)
}

func (l *listContent[T]) toggleItemSelectionByIndex(index int, forceFireEvents bool) {
	l.abstractList.ToggleItemSelectionByIndex(index, forceFireEvents)
}

func (l *listContent[T]) SelectItemByValue(value T) {
	l.abstractList.SelectItemByValue(value, false)
}

func (l *listContent[T]) SelectItemsByValues(values []T) {
	l.abstractList.SelectItemsByValues(values, false)
}

func (l *listContent[T]) JumpToItemByIndex(index int) {
	if index < 0 {
		return
	}
	l.indexToJumpPlus1 = index + 1
	l.indexToEnsureVisiblePlus1 = 0
	l.jumpTick = ebiten.Tick() + 1
}

func (l *listContent[T]) EnsureItemVisibleByIndex(index int) {
	if index < 0 {
		return
	}
	l.indexToEnsureVisiblePlus1 = index + 1
	l.indexToJumpPlus1 = 0
	l.jumpTick = ebiten.Tick() + 1
}

func (l *listContent[T]) SetStripeVisible(visible bool) {
	if l.stripeVisible == visible {
		return
	}
	l.stripeVisible = visible
	guigui.RequestRedraw(l)
}

func (l *listContent[T]) SetUnfocusedSelectionVisible(visible bool) {
	hidden := !visible
	if l.unfocusedSelectionHidden == hidden {
		return
	}
	l.unfocusedSelectionHidden = hidden
	guigui.RequestRedraw(l)
}

func (l *listContent[T]) isHoveringVisible() bool {
	return l.style == ListStyleMenu
}

func (l *listContent[T]) Style() ListStyle {
	return l.style
}

func (l *listContent[T]) SetStyle(style ListStyle) {
	l.style = style
}

func (l *listContent[T]) calcDropDstIndex(context *guigui.Context) int {
	_, y := ebiten.CursorPosition()
	var nonEmptyBoundsFound bool
	for i := range l.abstractList.ItemCount() {
		if !l.isItemAvailable(i) {
			continue
		}
		b := l.itemBounds(context, i)
		if b.Empty() {
			if !nonEmptyBoundsFound {
				continue
			}
			return i
		}
		nonEmptyBoundsFound = true
		if y < (b.Min.Y+b.Max.Y)/2 {
			return i
		}
	}
	return l.abstractList.ItemCount()
}

func (l *listContent[T]) resetHoveredItemIndex() {
	l.hoveredItemIndexPlus1 = 0
	l.lastHoveredItemIndexPlus1 = 0
	l.keyboardHighlightIndexPlus1 = 0
}

func (l *listContent[T]) keyboardHighlightIndex() int {
	return l.keyboardHighlightIndexPlus1 - 1
}

func (l *listContent[T]) setKeyboardHighlightIndex(index int) {
	if index < 0 {
		index = -1
	}
	l.keyboardHighlightIndexPlus1 = index + 1
	l.hoveredItemIndexPlus1 = index + 1
	l.lastHoveredItemIndexPlus1 = index + 1
}

func (l *listContent[T]) navigateKeyboardHighlight(down bool) {
	current := l.keyboardHighlightIndexPlus1 - 1
	if current < 0 {
		current = l.hoveredItemIndexPlus1 - 1
	}

	var next int
	if current < 0 {
		if down {
			next = l.nextSelectableVisibleIndex(-1, true)
		} else {
			next = l.lastSelectableVisibleIndex()
		}
	} else {
		next = l.nextSelectableVisibleIndex(current, down)
		if next < 0 {
			next = current
		}
	}

	if next >= 0 {
		l.keyboardHighlightIndexPlus1 = next + 1
		l.hoveredItemIndexPlus1 = next + 1
		l.lastHoveredItemIndexPlus1 = next + 1
		l.EnsureItemVisibleByIndex(next)
	}
}

func (l *listContent[T]) selectKeyboardHighlightedItem() bool {
	idx := l.keyboardHighlightIndexPlus1 - 1
	if idx < 0 {
		idx = l.hoveredItemIndexPlus1 - 1
	}
	if idx < 0 {
		return false
	}
	l.selectItemByIndex(idx, true)
	return true
}

func (l *listContent[T]) hoveredItemIndex(context *guigui.Context, widgetBounds *guigui.WidgetBounds) int {
	if !widgetBounds.IsHitAtCursor() {
		return -1
	}
	cp := image.Pt(ebiten.CursorPosition())
	listBounds := widgetBounds.Bounds()
	for i := range l.abstractList.ItemCount() {
		if !l.isItemAvailable(i) {
			continue
		}
		bounds := l.itemBounds(context, i)
		bounds.Min.X = listBounds.Min.X
		bounds.Max.X = listBounds.Max.X
		if cp.In(bounds) {
			return i
		}
	}
	return -1
}

func (l *listContent[T]) nextSelectableVisibleIndex(from int, forward bool) int {
	if forward {
		idx, ok := l.nextAvailableItem(from)
		if !ok {
			return -1
		}
		return idx
	}
	idx, ok := l.prevAvailableItem(from)
	if !ok {
		return -1
	}
	return idx
}

func (l *listContent[T]) HandleButtonInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	down := isKeyRepeating(ebiten.KeyDown)
	up := isKeyRepeating(ebiten.KeyUp)
	if !down && !up {
		if l.isHoveringVisible() && inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			if l.selectKeyboardHighlightedItem() {
				return guigui.HandleInputByWidget(l)
			}
		}
		return guigui.HandleInputResult{}
	}

	if l.isHoveringVisible() {
		l.navigateKeyboardHighlight(down)
		l.updateCheckmarkColor(context)
		return guigui.HandleInputByWidget(l)
	}

	// Normal/Sidebar style: navigate the selection.
	current := l.abstractList.SelectedItemIndex()

	var next int
	if current < 0 {
		if down {
			next = l.nextSelectableVisibleIndex(-1, true)
		} else {
			next = l.lastSelectableVisibleIndex()
		}
	} else {
		next = l.nextSelectableVisibleIndex(current, down)
		if next < 0 {
			next = current
		}
	}

	if next >= 0 && next != current {
		l.selectItemByIndex(next, false)
		l.EnsureItemVisibleByIndex(next)
		if item, ok := l.abstractList.ItemByIndex(next); ok {
			context.SetFocused(item.Content, true)
		}
	}
	return guigui.HandleInputByWidget(l)
}

func (l *listContent[T]) lastSelectableVisibleIndex() int {
	result := -1
	for i := range l.abstractList.ItemCount() {
		if !l.isItemAvailable(i) {
			continue
		}
		item, ok := l.abstractList.ItemByIndex(i)
		if !ok || item.Unselectable {
			continue
		}
		result = i
	}
	return result
}

func (l *listContent[T]) updateCheckmarkColor(context *guigui.Context) {
	colorMode := context.ColorMode()
	if l.hoveredItemIndexPlus1 == l.checkmarkIndexPlus1 {
		colorMode = ebiten.ColorModeDark
	}
	checkImg, err := theResourceImages.Get("check", colorMode)
	if err != nil {
		panic(fmt.Sprintf("basicwidget: failed to get check image: %v", err))
	}
	l.checkmark.SetImage(checkImg)
}

func (l *listContent[T]) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	// Reset dragging and pressing state when the list loses focus.
	// This prevents accidental drags caused by mouse events leaking through
	// a popup's closing animation (passthrough mode).
	if !context.IsFocusedOrHasFocusedChild(l) {
		l.dragSrcIndexPlus1 = 0
		l.dragDstIndexPlus1 = 0
		l.pressStartPlus1 = image.Point{}
		l.startPressingIndexPlus1 = 0
	}

	// Reset keyboard highlight when cursor moves.
	cursorPos := image.Pt(ebiten.CursorPosition())
	if l.keyboardHighlightIndexPlus1 > 0 && cursorPos != l.lastCursorPosition {
		l.keyboardHighlightIndexPlus1 = 0
	}
	l.lastCursorPosition = cursorPos

	// Skip updating the hovered item from cursor while keyboard highlight is active.
	if l.keyboardHighlightIndexPlus1 == 0 {
		l.hoveredItemIndexPlus1 = l.hoveredItemIndex(context, widgetBounds) + 1
	}

	l.updateCheckmarkColor(context)

	if l.isHoveringVisible() || l.hasMovableItems() {
		l.lastHoveredItemIndexPlus1 = l.hoveredItemIndexPlus1
	}

	// Process dragging.
	if l.dragSrcIndexPlus1 > 0 {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			_, y := ebiten.CursorPosition()
			p := widgetBounds.VisibleBounds().Min
			h := widgetBounds.VisibleBounds().Dy()
			var dy float64
			if upperY := p.Y + UnitSize(context); y < upperY {
				dy = float64(upperY-y) / 4
			}
			if lowerY := p.Y + h - UnitSize(context); y >= lowerY {
				dy = float64(lowerY-y) / 4
			}
			if dy != 0 {
				l.listPanel.forceSetScrollOffsetByDelta(0, dy)
			}
			if i := l.calcDropDstIndex(context); l.dragDstIndexPlus1-1 != i {
				droppable := true
				l.tmpSelectedIndices = l.abstractList.AppendSelectedItemIndices(l.tmpSelectedIndices[:0])
				if len(l.tmpSelectedIndices) > 0 {
					if result, handled := guigui.DispatchEvent(l, listEventItemsCanMove, l.tmpSelectedIndices[0], len(l.tmpSelectedIndices), i); handled {
						droppable = result[0].(bool)
					}
				}
				if droppable {
					l.dragDstIndexPlus1 = i + 1
				} else {
					l.dragDstIndexPlus1 = 0
				}
				guigui.RequestRedraw(l)
				return guigui.HandleInputByWidget(l)
			}
			return guigui.AbortHandlingInputByWidget(l)
		}
		if l.dragDstIndexPlus1 > 0 {
			l.tmpSelectedIndices = l.abstractList.AppendSelectedItemIndices(l.tmpSelectedIndices[:0])
			if len(l.tmpSelectedIndices) > 0 {
				from, count, to := l.tmpSelectedIndices[0], len(l.tmpSelectedIndices), l.dragDstIndexPlus1-1
				canMove := true
				if result, handled := guigui.DispatchEvent(l, listEventItemsCanMove, from, count, to); handled {
					canMove = result[0].(bool)
				}
				if canMove {
					guigui.DispatchEvent(l, listEventItemsMoved, from, count, to)
				}
			}
			l.dragDstIndexPlus1 = 0
		}
		l.dragSrcIndexPlus1 = 0
		guigui.RequestRedraw(l)
		return guigui.HandleInputByWidget(l)
	}

	if index := l.hoveredItemIndexPlus1 - 1; index >= 0 && index < l.abstractList.ItemCount() {
		c := image.Pt(ebiten.CursorPosition())

		left := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
		right := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
		switch {
		case (left || right):
			item, _ := l.abstractList.ItemByIndex(index)
			if c.X < l.itemBoundsForLayoutFromIndex[index].Min.X {
				if left {
					expanded := !item.Collapsed
					guigui.DispatchEvent(l, listEventItemExpanderToggled, index, !expanded)
				}
				l.pressStartPlus1 = image.Point{}
				l.startPressingIndexPlus1 = 0
				return guigui.AbortHandlingInputByWidget(l)
			}
			if item.Unselectable {
				l.pressStartPlus1 = image.Point{}
				l.startPressingIndexPlus1 = 0
				return guigui.AbortHandlingInputByWidget(l)
			}

			// A popup menu should not take a focus.
			// For example, a context menu for a text field should not take a focus from the text field.
			// TODO: It might be better to distinguish a menu and a popup menu in the future.
			if l.style != ListStyleMenu {
				if item, ok := l.abstractList.ItemByIndex(index); ok {
					context.SetFocused(item.Content, true)
				} else {
					context.SetFocused(l, true)
				}
			}

			if l.style == ListStyleNormal && l.abstractList.MultiSelection() {
				if ebiten.IsKeyPressed(ebiten.KeyShift) {
					l.extendItemSelectionByIndex(index, false)
				} else if !isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) ||
					isDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) {
					l.toggleItemSelectionByIndex(index, false)
				} else if !l.abstractList.IsSelectedItemIndex(index) {
					l.selectItemByIndex(index, false)
				}
				// If the index is already selected, don't change the selection by clicking,
				// or the user couldn't drag multiple items.
				// This is updated when the user releases the mouse button.
			} else {
				// If the list is for a menu, the selection should be fired even if the list is focused,
				// in order to let the user know the item is selected.
				l.selectItemByIndex(index, l.style == ListStyleMenu)
			}

			if left {
				l.pressStartPlus1 = c.Add(image.Pt(1, 1))
				l.startPressingIndexPlus1 = index + 1
				return guigui.HandleInputByWidget(l)
			}
			// For the right click, give a chance to a parent widget to handle the right click e.g. to open a context menu.
			// TODO: This behavior seems a little ad-hoc. Consider a better way.
			return guigui.HandleInputResult{}

		case ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft):
			if ebiten.IsKeyPressed(ebiten.KeyShift) {
				return guigui.AbortHandlingInputByWidget(l)
			}
			if !isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) ||
				isDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) {
				return guigui.AbortHandlingInputByWidget(l)
			}
			if l.startPressingIndexPlus1 == 0 {
				return guigui.AbortHandlingInputByWidget(l)
			}
			index := l.startPressingIndexPlus1 - 1
			if !l.abstractList.IsSelectedItemIndex(index) {
				return guigui.AbortHandlingInputByWidget(l)
			}
			l.abstractList.SelectGroupAt(index, false)
			if !l.abstractList.IsSelectedItemIndex(index) {
				return guigui.AbortHandlingInputByWidget(l)
			}
			l.tmpSelectedIndices = l.abstractList.AppendSelectedItemIndices(l.tmpSelectedIndices[:0])
			if len(l.tmpSelectedIndices) == 0 {
				return guigui.AbortHandlingInputByWidget(l)
			}
			for _, index := range l.tmpSelectedIndices {
				item, _ := l.abstractList.ItemByIndex(index)
				if !item.Movable {
					return guigui.AbortHandlingInputByWidget(l)
				}
			}
			if start := l.pressStartPlus1.Sub(image.Pt(1, 1)); start.Y != c.Y {
				itemBoundsMin := l.itemBounds(context, l.tmpSelectedIndices[0])
				itemBoundsMax := l.itemBounds(context, l.tmpSelectedIndices[len(l.tmpSelectedIndices)-1])
				minY := min((itemBoundsMin.Min.Y+start.Y)/2, (itemBoundsMin.Min.Y+itemBoundsMin.Max.Y)/2)
				maxY := max((itemBoundsMax.Max.Y+start.Y)/2, (itemBoundsMax.Min.Y+itemBoundsMax.Max.Y)/2)
				if c.Y < minY || c.Y >= maxY {
					l.dragSrcIndexPlus1 = l.tmpSelectedIndices[0] + 1
					return guigui.HandleInputByWidget(l)
				}
			}
			return guigui.AbortHandlingInputByWidget(l)

		case inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft):
			// For the multi selection, the index is updated when the user releases the mouse button.
			if l.style == ListStyleNormal && l.abstractList.MultiSelection() && l.startPressingIndexPlus1 > 0 && l.dragSrcIndexPlus1 == 0 {
				if !ebiten.IsKeyPressed(ebiten.KeyShift) &&
					!(!isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl)) &&
					!(isDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta)) {
					l.selectItemByIndex(l.startPressingIndexPlus1-1, false)
					l.pressStartPlus1 = image.Point{}
					l.startPressingIndexPlus1 = 0
					return guigui.HandleInputByWidget(l)
				}
			}
			l.pressStartPlus1 = image.Point{}
			l.startPressingIndexPlus1 = 0
			return guigui.AbortHandlingInputByWidget(l)
		}
	}

	l.dragSrcIndexPlus1 = 0
	l.pressStartPlus1 = image.Point{}
	return guigui.HandleInputResult{}
}

func (l *listContent[T]) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	// Jump to the item if requested.
	// This is done in Tick to wait for the list items are updated, or an item cannot be measured correctly.
	if l.jumpTick > 0 && ebiten.Tick() >= l.jumpTick {
		if idx := l.indexToJumpPlus1 - 1; idx >= 0 && idx < l.abstractList.ItemCount() {
			// Convert item index to available-item index.
			if ai := l.availableIndexForItemIndex(idx); ai >= 0 {
				l.listPanel.setTopItem(ai, 0)
			}
			l.indexToJumpPlus1 = 0
		}
		if idx := l.indexToEnsureVisiblePlus1 - 1; idx >= 0 && idx < l.abstractList.ItemCount() {
			l.scrollToEnsureItemVisible(context, widgetBounds, idx)
			l.indexToEnsureVisiblePlus1 = 0
		}
		l.jumpTick = 0
	}

	// Advance expand/collapse animation.
	if l.expandAnimatingCount > 0 {
		l.expandAnimatingCount--
		if l.expandAnimatingCount == 0 {
			l.expandAnimatingIndexPlus1 = 0
			l.expandAnimatingChildrenEnd = 0
		}
	}

	return nil
}

func (l *listContent[T]) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	l.onceRendered = true
}

// availableIndexForItemIndex converts a raw item index to the position in the available items list.
func (l *listContent[T]) availableIndexForItemIndex(itemIndex int) int {
	var ai int
	for i := range l.abstractList.ItemCount() {
		if !l.isItemAvailable(i) {
			continue
		}
		if i == itemIndex {
			return ai
		}
		ai++
	}
	return -1
}

// scrollToEnsureItemVisible adjusts the listPanel's topItem to make the given item visible.
func (l *listContent[T]) scrollToEnsureItemVisible(context *guigui.Context, widgetBounds *guigui.WidgetBounds, itemIndex int) {
	ai := l.availableIndexForItemIndex(itemIndex)
	if ai < 0 {
		return
	}

	topIdx, topOff := l.listPanel.topItem()

	if ai < topIdx {
		// Item is above viewport — scroll up to it.
		l.listPanel.setTopItem(ai, 0)
		return
	}

	if ai == topIdx && topOff < 0 {
		// Item is partially above viewport — align to top.
		l.listPanel.setTopItem(ai, 0)
		return
	}

	// Check if item is below viewport.
	// We need to compute the Y position of the item relative to the viewport top.
	bounds := widgetBounds.Bounds()
	cw := bounds.Dx()
	if l.contentWidthPlus1 > 0 {
		cw = l.contentWidthPlus1 - 1
	}

	l.tmpAvailableIndices = l.appendAvailableIndices(l.tmpAvailableIndices[:0])
	availableIndices := l.tmpAvailableIndices
	y := topOff + RoundedCornerRadius(context)
	for aIdx := topIdx; aIdx <= ai && aIdx < len(availableIndices); aIdx++ {
		h := l.measureItemHeight(context, availableIndices[aIdx], cw)
		if aIdx == ai {
			// Check if the bottom of this item is below the viewport.
			itemBottom := y + h
			viewportHeight := bounds.Dy()
			if itemBottom > viewportHeight-RoundedCornerRadius(context) {
				// Need to scroll down. Set the offset so this item's bottom aligns with viewport bottom.
				diff := itemBottom - (viewportHeight - RoundedCornerRadius(context))
				l.listPanel.setTopItem(topIdx, topOff-diff)
				// normalizeTopItem will fix the indices during the next layout.
			}
			return
		}
		y += h
	}
}

// itemYFromIndexForMenu returns the Y position of the item at the given index relative to the top of the List widget.
// itemYFromIndexForMenu returns the same value whatever the List position is.
//
// itemYFromIndexForMenu is available anytime even before Build is called.
func (l *listContent[T]) itemYFromIndexForMenu(context *guigui.Context, index int) (int, bool) {
	y := RoundedCornerRadius(context)
	for i := range l.abstractList.ItemCount() {
		if !l.isItemAvailable(i) {
			continue
		}
		if i == index {
			return y, true
		}
		if i > index {
			break
		}
		item, _ := l.abstractList.ItemByIndex(i)
		// Use a free constraints to measure the item height for menu.
		y += item.Content.Measure(context, guigui.Constraints{}).Y
	}

	return 0, false
}

func (l *listContent[T]) adjustItemY(context *guigui.Context, y int) int {
	// Adjust the bounds based on the list style (inset or outset).
	switch l.style {
	case ListStyleNormal:
		y += int(0.5 * context.Scale())
	case ListStyleMenu:
		y += int(-0.5 * context.Scale())
	}
	return y
}

func (l *listContent[T]) itemBounds(context *guigui.Context, index int) image.Rectangle {
	if index < 0 || index >= len(l.itemBoundsForLayoutFromIndex) {
		return image.Rectangle{}
	}
	r := l.itemBoundsForLayoutFromIndex[index]
	if l.checkmarkIndexPlus1 > 0 {
		r.Min.X -= listItemCheckmarkSize(context) + listItemTextAndImagePadding(context)
	}
	return r
}

func (l *listContent[T]) isHighlightedItemIndex(context *guigui.Context, index int) bool {
	if !l.useHighlightedBackgroundColor(context) {
		return false
	}
	if l.isHoveringVisible() {
		if l.hoveredItemIndexPlus1-1 != index {
			return false
		}
		item, ok := l.abstractList.ItemByIndex(index)
		if !ok {
			return false
		}
		return !item.Unselectable
	}
	return l.IsSelectedItemIndex(index)
}

func (l *listContent[T]) itemColorType(context *guigui.Context, index int) ListItemColorType {
	if !context.IsEnabled(l) {
		return ListItemColorTypeListDisabled
	}
	if item, ok := l.ItemByIndex(index); ok && !context.IsEnabled(item.Content) {
		return ListItemColorTypeItemDisabled
	}
	if l.isHighlightedItemIndex(context, index) {
		return ListItemColorTypeHighlighted
	}
	if l.IsSelectedItemIndex(index) && !l.unfocusedSelectionHidden {
		return ListItemColorTypeSelectedInUnfocusedList
	}
	return ListItemColorTypeDefault
}

func (l *listContent[T]) useHighlightedBackgroundColor(context *guigui.Context) bool {
	if !context.IsEnabled(l) {
		return false
	}
	return l.style == ListStyleSidebar || context.IsFocusedOrHasFocusedChild(l) || l.style == ListStyleMenu
}

func (l *listContent[T]) selectedItemBackgroundColor(context *guigui.Context, index int) color.Color {
	return l.itemColorType(context, index).BackgroundColor(context)
}

type listBackground1[T comparable] struct {
	guigui.DefaultWidget

	content *listContent[T]
}

func (l *listBackground1[T]) setListContent(content *listContent[T]) {
	l.content = content
}

func (l *listBackground1[T]) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	var clr color.Color
	switch l.content.style {
	case ListStyleSidebar:
	case ListStyleNormal:
		clr = basicwidgetdraw.ControlColor(context.ColorMode(), context.IsEnabled(l))
	case ListStyleMenu:
		clr = basicwidgetdraw.ControlSecondaryColor(context.ColorMode(), context.IsEnabled(l))
	}
	if clr != nil {
		bounds := widgetBounds.Bounds()
		basicwidgetdraw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
	}

	if l.content.stripeVisible && l.content.abstractList.ItemCount() > 0 {
		vb := widgetBounds.VisibleBounds()
		// Draw item stripes.
		// TODO: Get indices of items that are visible.
		var count int
		for i := range l.content.abstractList.ItemCount() {
			if !l.content.isItemAvailable(i) {
				continue
			}
			count++
			if count%2 == 1 {
				continue
			}
			bounds := l.content.itemBounds(context, i)
			// Reset the X position to ignore indentation.
			item, _ := l.content.abstractList.ItemByIndex(i)
			bounds.Min.X -= ListItemIndentSize(context, item.IndentLevel)
			if bounds.Min.Y > vb.Max.Y {
				break
			}
			if !bounds.Overlaps(vb) {
				continue
			}
			clr := basicwidgetdraw.ControlSecondaryColor(context.ColorMode(), context.IsEnabled(l))
			basicwidgetdraw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
		}
	}
}

type listBackground2[T comparable] struct {
	guigui.DefaultWidget

	tmpItemIndices []int

	content *listContent[T]
}

func (l *listBackground2[T]) setListContent(content *listContent[T]) {
	l.content = content
}

func (l *listBackground2[T]) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	vb := widgetBounds.VisibleBounds()

	// Draw the selected item background.
	if !l.content.isHoveringVisible() {
		l.tmpItemIndices = l.content.AppendSelectedItemIndices(l.tmpItemIndices[:0])
		for _, index := range l.tmpItemIndices {
			clr := l.content.selectedItemBackgroundColor(context, index)
			if clr == nil {
				continue
			}
			if !l.content.isItemAvailable(index) {
				continue
			}
			bounds := l.content.itemBounds(context, index)
			if l.content.style == ListStyleMenu {
				bounds.Max.X = bounds.Min.X + widgetBounds.Bounds().Dx() - 2*RoundedCornerRadius(context)
			}
			if bounds.Overlaps(vb) {
				item, _ := l.content.ItemByIndex(index)
				var corners basicwidgetdraw.Corners
				// If prev available item is adjacent to this item, don't draw the top corner.
				if prevIndex, ok := l.content.prevAvailableItem(index); ok && item.Padding.Top == 0 {
					if prevItem, ok := l.content.ItemByIndex(prevIndex); ok && prevItem.Padding.Bottom == 0 {
						if l.content.IsSelectedItemIndex(prevIndex) {
							corners.TopStart = prevItem.IndentLevel <= item.IndentLevel &&
								prevItem.Padding.Start == item.Padding.Start
							corners.TopEnd = prevItem.Padding.End == item.Padding.End
						}
					}
				}
				// If next available item is adjacent to this item, don't draw the bottom corner.
				if nextIndex, ok := l.content.nextAvailableItem(index); ok && item.Padding.Bottom == 0 {
					if nextItem, ok := l.content.ItemByIndex(nextIndex); ok && nextItem.Padding.Top == 0 {
						if l.content.IsSelectedItemIndex(nextIndex) {
							corners.BottomStart = nextItem.IndentLevel <= item.IndentLevel &&
								nextItem.Padding.Start == item.Padding.Start
							corners.BottomEnd = nextItem.Padding.End == item.Padding.End
						}
					}
				}
				basicwidgetdraw.DrawRoundedRectWithSharpCorners(context, dst, bounds, clr, RoundedCornerRadius(context), corners)
			}
		}
	}

	hoveredItemIndex := l.content.hoveredItemIndexPlus1 - 1
	hoveredItem, ok := l.content.abstractList.ItemByIndex(hoveredItemIndex)
	if ok && l.content.isHoveringVisible() && hoveredItemIndex >= 0 && hoveredItemIndex < l.content.abstractList.ItemCount() && !hoveredItem.Unselectable && l.content.isItemAvailable(hoveredItemIndex) {
		clr := l.content.selectedItemBackgroundColor(context, hoveredItemIndex)
		bounds := l.content.itemBounds(context, hoveredItemIndex)
		if l.content.style == ListStyleMenu {
			bounds.Max.X = bounds.Min.X + widgetBounds.Bounds().Dx() - 2*RoundedCornerRadius(context)
		}
		if clr != nil && bounds.Overlaps(vb) {
			basicwidgetdraw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
		}
	}

	// Draw a drag indicator.
	if context.IsEnabled(l) && l.content.dragSrcIndexPlus1 == 0 {
		if item, ok := l.content.abstractList.ItemByIndex(hoveredItemIndex); ok && item.Movable && item.selectable() {
			img, err := theResourceImages.Get("drag_indicator", context.ColorMode())
			if err != nil {
				panic(fmt.Sprintf("basicwidget: failed to get drag indicator image: %v", err))
			}
			op := &ebiten.DrawImageOptions{}
			s := float64(2*RoundedCornerRadius(context)) / float64(img.Bounds().Dy())
			op.GeoM.Scale(s, s)
			bounds := l.content.itemBounds(context, hoveredItemIndex)
			p := bounds.Min
			p.X = widgetBounds.Bounds().Min.X
			op.GeoM.Translate(float64(p.X), float64(p.Y)+(float64(bounds.Dy())-float64(img.Bounds().Dy())*s)/2)
			op.ColorScale.ScaleAlpha(0.5)
			op.Filter = ebiten.FilterLinear
			dst.DrawImage(img, op)
		}
	}

	// Draw a dragging guideline.
	// Compute the guideline Y directly from item bounds in screen coordinates.
	// Using itemYFromIndex would be incorrect when scrolled because it relies on
	// itemBoundsForLayoutFromIndex[0] as a baseline, which is zeroed when item 0
	// is scrolled off-screen.
	if dstIdx := l.content.dragDstIndexPlus1 - 1; dstIdx >= 0 {
		p := widgetBounds.Bounds().Min
		x0 := float32(p.X) + float32(RoundedCornerRadius(context))
		cw := widgetBounds.Bounds().Dx()
		if l.content.contentWidthPlus1 > 0 {
			cw = l.content.contentWidthPlus1 - 1
		}
		x1 := x0 + float32(cw)
		x1 -= 2 * float32(RoundedCornerRadius(context))

		adjustY := l.content.adjustItemY(context, 0)
		var y float32
		var ok bool
		if dstIdx < len(l.content.itemBoundsForLayoutFromIndex) {
			if item, itemOk := l.content.abstractList.ItemByIndex(dstIdx); itemOk {
				y = float32(l.content.itemBoundsForLayoutFromIndex[dstIdx].Min.Y - item.Padding.Top - adjustY)
				ok = true
			}
		} else {
			// This is needed especially when dragging to the end of the list, where dstIdx can be equal to the item count.
			if item, itemOk := l.content.abstractList.ItemByIndex(dstIdx - 1); itemOk {
				y = float32(l.content.itemBoundsForLayoutFromIndex[dstIdx-1].Max.Y + item.Padding.Bottom - adjustY)
				ok = true
			}
		}
		if ok {
			vector.StrokeLine(dst, x0, y, x1, y, 2*float32(context.Scale()), draw.Color(context.ColorMode(), draw.SemanticColorAccent, 0.5), false)
		}
	}
}

type listFrame struct {
	guigui.DefaultWidget

	headerHeight int
	footerHeight int
	style        ListStyle
}

func (l *listFrame) WriteStateKey(w *guigui.StateKeyWriter) {
	w.WriteInt(l.headerHeight)
	w.WriteInt(l.footerHeight)
	w.WriteUint64(uint64(l.style))
}

func (l *listFrame) SetHeaderHeight(height int) {
	l.headerHeight = height
}

func (l *listFrame) SetFooterHeight(height int) {
	l.footerHeight = height
}

func (l *listFrame) SetStyle(style ListStyle) {
	l.style = style
}

func (l *listFrame) headerBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	bounds := widgetBounds.Bounds()
	bounds.Max.Y = bounds.Min.Y + l.headerHeight
	return bounds
}

func (l *listFrame) footerBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	bounds := widgetBounds.Bounds()
	bounds.Min.Y = bounds.Max.Y - l.footerHeight
	return bounds
}

func (l *listFrame) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if l.style == ListStyleSidebar || l.style == ListStyleMenu {
		return
	}

	// Draw a header.
	if l.headerHeight > 0 {
		bounds := l.headerBounds(context, widgetBounds)
		basicwidgetdraw.DrawRoundedRectWithSharpCorners(context, dst, bounds, basicwidgetdraw.ControlColor(context.ColorMode(), context.IsEnabled(l)), RoundedCornerRadius(context), basicwidgetdraw.Corners{
			TopStart:    false,
			TopEnd:      false,
			BottomStart: true,
			BottomEnd:   true,
		})

		x0 := float32(bounds.Min.X)
		x1 := float32(bounds.Max.X)
		y0 := float32(bounds.Max.Y)
		y1 := float32(bounds.Max.Y)
		clr := draw.Color2(context.ColorMode(), draw.SemanticColorBase, 0.9, 0.4)
		if !context.IsEnabled(l) {
			clr = draw.Color2(context.ColorMode(), draw.SemanticColorBase, 0.8, 0.3)
		}
		vector.StrokeLine(dst, x0, y0, x1, y1, float32(context.Scale()), clr, false)
	}

	// Draw a footer.
	if l.footerHeight > 0 {
		bounds := l.footerBounds(context, widgetBounds)
		basicwidgetdraw.DrawRoundedRectWithSharpCorners(context, dst, bounds, basicwidgetdraw.ControlColor(context.ColorMode(), context.IsEnabled(l)), RoundedCornerRadius(context), basicwidgetdraw.Corners{
			TopStart:    true,
			TopEnd:      true,
			BottomStart: false,
			BottomEnd:   false,
		})

		x0 := float32(bounds.Min.X)
		x1 := float32(bounds.Max.X)
		y0 := float32(bounds.Min.Y)
		y1 := float32(bounds.Min.Y)
		clr := draw.Color2(context.ColorMode(), draw.SemanticColorBase, 0.9, 0.4)
		if !context.IsEnabled(l) {
			clr = draw.Color2(context.ColorMode(), draw.SemanticColorBase, 0.8, 0.3)
		}
		vector.StrokeLine(dst, x0, y0, x1, y1, float32(context.Scale()), clr, false)
	}

	bounds := widgetBounds.Bounds()
	border := basicwidgetdraw.RoundedRectBorderTypeInset
	if l.style != ListStyleNormal {
		border = basicwidgetdraw.RoundedRectBorderTypeOutset
	}
	clr1, clr2 := basicwidgetdraw.BorderColors(context.ColorMode(), basicwidgetdraw.RoundedRectBorderType(border))
	borderWidth := listBorderWidth(context)
	basicwidgetdraw.DrawRoundedRectBorder(context, dst, bounds, clr1, clr2, RoundedCornerRadius(context), borderWidth, border)
}

func listItemCheckmarkSize(context *guigui.Context) int {
	return LineHeight(context) * 3 / 4
}

func listItemTextAndImagePadding(context *guigui.Context) int {
	return UnitSize(context) / 8
}

func ListItemIndentSize(context *guigui.Context, level int) int {
	if level == 0 {
		return 0
	}
	return LineHeight(context) + LineHeight(context)/2*(level-1)
}

func listBorderWidth(context *guigui.Context) float32 {
	return float32(1 * context.Scale())
}
