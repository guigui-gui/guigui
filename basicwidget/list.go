// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"fmt"
	"image"
	"image/color"
	"iter"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

type ListStyle int

const (
	ListStyleNormal ListStyle = iota
	ListStyleSidebar
	ListStyleMenu
)

var (
	listEventItemSelected         guigui.EventKey = guigui.GenerateEventKey()
	listEventItemsSelected        guigui.EventKey = guigui.GenerateEventKey()
	listEventItemsMoved           guigui.EventKey = guigui.GenerateEventKey()
	listEventItemExpanderToggled  guigui.EventKey = guigui.GenerateEventKey()
	listEventScrollY              guigui.EventKey = guigui.GenerateEventKey()
	listEventScrollYEnsureVisible guigui.EventKey = guigui.GenerateEventKey()
	listEventScrollDeltaY         guigui.EventKey = guigui.GenerateEventKey()
)

func defaultActiveListItemTextColor(context *guigui.Context) color.Color {
	return draw.Color2(context.ColorMode(), draw.ColorTypeBase, 1, 1)
}

type List[T comparable] struct {
	guigui.DefaultWidget

	abstractListItems []abstractListItem[T]
	listItemWidgets   guigui.WidgetSlice[*listItemWidget[T]]
	background1       listBackground1[T]
	content           listContent[T]
	panel             Panel
	frame             listFrame

	listItemHeightPlus1 int
	headerHeight        int
	footerHeight        int

	onScrollY                 func(context *guigui.Context, offsetY float64)
	onScrollYEnsureVisible    func(context *guigui.Context, offsetYTop, offsetYBottom float64)
	onScrollDeltaY            func(context *guigui.Context, deltaY float64)
	scrollOffsetYTopMinus1    float64
	scrollOffsetYBottomMinus1 float64
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
	Padding      guigui.Padding
	Collapsed    bool
}

func (l *ListItem[T]) selectable() bool {
	return !l.Header && !l.Unselectable && !l.Border && !l.Disabled
}

func (l *List[T]) SetBackground(widget guigui.Widget) {
	l.content.SetBackground(widget)
}

func (l *List[T]) SetStripeVisible(visible bool) {
	l.content.SetStripeVisible(visible)
}

func (l *List[T]) SetMultiSelection(multi bool) {
	l.content.abstractList.SetMultiSelection(multi)
}

func (l *List[T]) SetItemHeight(height int) {
	if l.listItemHeightPlus1 == height+1 {
		return
	}
	l.listItemHeightPlus1 = height + 1
	guigui.RequestRebuild(l)
}

func (l *List[T]) SetOnItemSelected(f func(context *guigui.Context, index int)) {
	l.content.SetOnItemSelected(f)
}

func (l *List[T]) SetOnItemsSelected(f func(context *guigui.Context, indices []int)) {
	l.content.SetOnItemsSelected(f)
}

func (l *List[T]) SetOnItemsMoved(f func(context *guigui.Context, from, count, to int)) {
	l.content.SetOnItemsMoved(f)
}

func (l *List[T]) SetOnItemExpanderToggled(f func(context *guigui.Context, index int, expanded bool)) {
	l.content.SetOnItemExpanderToggled(f)
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
	guigui.RequestRebuild(l)
}

func (l *List[T]) SetFooterHeight(height int) {
	if l.footerHeight == height {
		return
	}
	l.footerHeight = height
	l.frame.SetFooterHeight(height)
	guigui.RequestRebuild(l)
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

func (l *List[T]) IsItemVisible(index int) bool {
	return l.content.isItemVisible(index)
}

func (l *List[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&l.background1)
	adder.AddChild(&l.panel)
	adder.AddChild(&l.frame)

	l.background1.setListContent(&l.content)
	l.panel.SetContent(&l.content)
	l.panel.SetContentConstraints(PanelContentConstraintsFixedWidth)

	if l.onScrollY == nil {
		l.onScrollY = func(context *guigui.Context, offsetY float64) {
			offsetX, _ := l.panel.scrollOffset()
			l.panel.SetScrollOffset(offsetX, offsetY)
		}
	}
	guigui.SetEventHandler(&l.content, listEventScrollY, l.onScrollY)

	if l.onScrollYEnsureVisible == nil {
		l.onScrollYEnsureVisible = func(context *guigui.Context, offsetYTop, offsetYBottom float64) {
			l.scrollOffsetYTopMinus1 = offsetYTop - 1
			l.scrollOffsetYBottomMinus1 = offsetYBottom - 1
		}
	}
	guigui.SetEventHandler(&l.content, listEventScrollYEnsureVisible, l.onScrollYEnsureVisible)

	if l.onScrollDeltaY == nil {
		l.onScrollDeltaY = func(context *guigui.Context, deltaY float64) {
			l.panel.SetScrollOffsetByDelta(0, deltaY)
		}
	}
	guigui.SetEventHandler(&l.content, listEventScrollDeltaY, l.onScrollDeltaY)

	for i := range l.listItemWidgets.Len() {
		item := l.listItemWidgets.At(i)
		item.text.SetBold(item.item.Header || l.content.Style() == ListStyleSidebar && l.SelectedItemIndex() == i)
		item.text.SetColor(l.ItemTextColor(context, i))
		item.keyText.SetColor(l.ItemTextColor(context, i))
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

func (l *List[T]) isHighlightedItemIndex(context *guigui.Context, index int) bool {
	if l.content.Style() != ListStyleMenu {
		return l.content.IsSelectedItemIndex(index)
	}

	if !l.content.isHoveringVisible() {
		return false
	}
	// TODO: The hovered item index is not updated yet.
	// This requires the list's widgetBounds.
	if l.content.hoveredItemIndexPlus1-1 != index {
		return false
	}
	if index < 0 || index >= l.listItemWidgets.Len() {
		return false
	}
	if !l.abstractListItems[index].selectable() {
		return false
	}
	if !context.IsEnabled(l.listItemWidgets.At(index)) {
		return false
	}
	return true
}

func (l *List[T]) ItemTextColor(context *guigui.Context, index int) color.Color {
	if l.isHighlightedItemIndex(context, index) {
		return defaultActiveListItemTextColor(context)
	}
	item := l.listItemWidgets.At(index)
	if clr := item.textColor(); clr != nil {
		return clr
	}
	return basicwidgetdraw.TextColor(context.ColorMode(), context.IsEnabled(item))
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
	return l.content.Measure(context, constraints)
}

func (l *List[T]) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if l.scrollOffsetYTopMinus1 != 0 || l.scrollOffsetYBottomMinus1 != 0 {
		// Adjust the bottom first.
		if l.scrollOffsetYBottomMinus1 != 0 {
			y := l.scrollOffsetYBottomMinus1 + 1
			y += float64(widgetBounds.Bounds().Dy())
			y -= float64(l.headerHeight + l.footerHeight)
			y -= float64(RoundedCornerRadius(context))
			if offsetX, offsetY := l.panel.scrollOffset(); y < offsetY {
				l.panel.SetScrollOffset(offsetX, y)
			}
		}
		// Then adjust the top.
		if l.scrollOffsetYTopMinus1 != 0 {
			y := l.scrollOffsetYTopMinus1 + 1
			y += float64(RoundedCornerRadius(context))
			// Reget the offset as it may be changed by the above bottom adjustment.
			if offsetX, offsetY := l.panel.scrollOffset(); y > offsetY {
				l.panel.SetScrollOffset(offsetX, y)
			}
		}
		l.scrollOffsetYTopMinus1 = 0
		l.scrollOffsetYBottomMinus1 = 0
	}
	return nil
}

type listItemWidget[T comparable] struct {
	guigui.DefaultWidget

	text    Text
	keyText Text

	item        ListItem[T]
	heightPlus1 int
	style       ListStyle

	layout      guigui.LinearLayout
	layoutItems []guigui.LinearLayoutItem
}

func (l *listItemWidget[T]) setListItem(listItem ListItem[T]) {
	if l.item == listItem {
		return
	}
	l.item = listItem
	l.resetLayout()
	guigui.RequestRebuild(l)
}

func (l *listItemWidget[T]) setHeight(height int) {
	if l.heightPlus1 == height+1 {
		return
	}
	l.heightPlus1 = height + 1
	l.resetLayout()
	guigui.RequestRebuild(l)
}

func (l *listItemWidget[T]) setStyle(style ListStyle) {
	if l.style == style {
		return
	}
	l.style = style
	l.resetLayout()
	guigui.RequestRebuild(l)
}

func (l *listItemWidget[T]) setText(text string) {
	if l.item.Text == text {
		return
	}
	l.item.Text = text
	guigui.RequestRebuild(l)
}

func (l *listItemWidget[T]) textColor() color.Color {
	return l.item.TextColor
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
		l.layout = guigui.LinearLayout{
			Direction: guigui.LayoutDirectionVertical,
			Items: []guigui.LinearLayoutItem{
				{
					Layout: layout,
					Size:   guigui.FixedSize(h),
				},
			},
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
		vector.StrokeLine(dst, x0, y, x1, y, width, draw.Color(context.ColorMode(), draw.ColorTypeBase, 0.8), false)
		return
	}
	/*if l.item.Header {
		bounds := widgetBounds.Bounds()
		draw.DrawRoundedRect(context, dst, bounds, draw.Color(context.ColorMode(), draw.ColorTypeBase, 0.8), RoundedCornerRadius(context))
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
	listContent *listContent[T]
}

func (a abstractListItem[T]) value() T {
	return a.Value
}

func (a abstractListItem[T]) selectable() bool {
	return !a.Unselectable
}

func (a abstractListItem[T]) visible() bool {
	return a.listContent.isItemVisible(a.index)
}

type listContent[T comparable] struct {
	guigui.DefaultWidget

	customBackground guigui.Widget
	background2      listBackground2[T]
	checkmark        Image
	expanderImages   guigui.WidgetSlice[*Image]

	abstractList              abstractList[T, abstractListItem[T]]
	stripeVisible             bool
	style                     ListStyle
	checkmarkIndexPlus1       int
	hoveredItemIndexPlus1     int
	lastHoveredItemIndexPlus1 int

	indexToJumpPlus1          int
	indexToEnsureVisiblePlus1 int
	jumpTick                  int64
	dragSrcIndexPlus1         int
	dragDstIndexPlus1         int
	pressStartPlus1           image.Point
	startPressingIndexPlus1   int
	contentWidthPlus1         int
	prevFocused               bool
	widthForCachedHeight      int
	cachedHeight              int

	widgetBoundsForLayout        map[guigui.Widget]image.Rectangle
	itemBoundsForLayoutFromIndex []image.Rectangle

	treeItemCollapsedImage *ebiten.Image
	treeItemExpandedImage  *ebiten.Image

	onItemSelected  func(index int)
	onItemsSelected func(indices []int)
}

func (l *listContent[T]) SetBackground(widget guigui.Widget) {
	l.customBackground = widget
}

func (l *listContent[T]) SetOnItemSelected(f func(context *guigui.Context, index int)) {
	guigui.SetEventHandler(l, listEventItemSelected, f)
}

func (l *listContent[T]) SetOnItemsSelected(f func(context *guigui.Context, indices []int)) {
	guigui.SetEventHandler(l, listEventItemsSelected, f)
}

func (l *listContent[T]) SetOnItemsMoved(f func(context *guigui.Context, from, count, to int)) {
	guigui.SetEventHandler(l, listEventItemsMoved, f)
}

func (l *listContent[T]) SetOnItemExpanderToggled(f func(context *guigui.Context, index int, expanded bool)) {
	guigui.SetEventHandler(l, listEventItemExpanderToggled, f)
}

func (l *listContent[T]) SetCheckmarkIndex(index int) {
	if index < 0 {
		index = -1
	}
	if l.checkmarkIndexPlus1 == index+1 {
		return
	}
	l.checkmarkIndexPlus1 = index + 1
	guigui.RequestRebuild(l)
}

func (l *listContent[T]) SetContentWidth(width int) {
	if l.contentWidthPlus1 == width+1 {
		return
	}
	l.contentWidthPlus1 = width + 1
	guigui.RequestRebuild(l)
}

func (l *listContent[T]) ItemBounds(index int) image.Rectangle {
	return l.itemBoundsForLayoutFromIndex[index]
}

func (l *listContent[T]) visibleItems() iter.Seq[int] {
	return func(yield func(int) bool) {
		var lastCollapsedIndentLevel int
		for i := range l.abstractList.ItemCount() {
			item, _ := l.abstractList.ItemByIndex(i)
			if lastCollapsedIndentLevel > 0 && item.IndentLevel > lastCollapsedIndentLevel {
				continue
			}
			if item.Collapsed {
				lastCollapsedIndentLevel = item.IndentLevel
			} else {
				lastCollapsedIndentLevel = 0
			}
			if !yield(i) {
				return
			}
		}
	}
}

func (l *listContent[T]) isItemVisible(index int) bool {
	item, ok := l.abstractList.ItemByIndex(index)
	if !ok {
		return false
	}
	indent := item.IndentLevel
	for {
		if indent == 0 {
			break
		}
		index--
		if index < 0 {
			break
		}
		item, ok := l.abstractList.ItemByIndex(index)
		if !ok {
			continue
		}
		if item.IndentLevel < indent {
			if item.Collapsed {
				return false
			}
			indent = item.IndentLevel
		}
	}
	return true
}

func (l *listContent[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if l.customBackground != nil {
		adder.AddChild(l.customBackground)
	}
	adder.AddChild(&l.background2)
	l.expanderImages.SetLen(l.abstractList.ItemCount())
	for i := range l.visibleItems() {
		item, _ := l.abstractList.ItemByIndex(i)
		if l.checkmarkIndexPlus1 == i+1 {
			adder.AddChild(&l.checkmark)
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
			adder.AddChild(img)
		}
		adder.AddChild(item.Content)
	}

	if l.onItemSelected == nil {
		l.onItemSelected = func(index int) {
			guigui.DispatchEvent(l, listEventItemSelected, index)
		}
	}
	l.abstractList.SetOnItemSelected(l.onItemSelected)

	if l.onItemsSelected == nil {
		l.onItemsSelected = func(indices []int) {
			guigui.DispatchEvent(l, listEventItemsSelected, indices)
		}
	}
	l.abstractList.SetOnItemsSelected(l.onItemsSelected)

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
	return nil
}

func (l *listContent[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	cw := widgetBounds.Bounds().Dx()
	if l.contentWidthPlus1 > 0 {
		cw = l.contentWidthPlus1 - 1
	}

	p := widgetBounds.Bounds().Min
	p.X += RoundedCornerRadius(context)
	p.Y += RoundedCornerRadius(context)

	clear(l.widgetBoundsForLayout)
	if l.widgetBoundsForLayout == nil {
		l.widgetBoundsForLayout = map[guigui.Widget]image.Rectangle{}
	}

	l.itemBoundsForLayoutFromIndex = adjustSliceSize(l.itemBoundsForLayoutFromIndex, l.abstractList.ItemCount())
	clear(l.itemBoundsForLayoutFromIndex)

	for i := range l.visibleItems() {
		item, _ := l.abstractList.ItemByIndex(i)
		itemW := cw - 2*RoundedCornerRadius(context)
		itemW -= listItemIndentSize(context, item.IndentLevel)
		itemW -= item.Padding.Start + item.Padding.End
		contentH := item.Content.Measure(context, guigui.FixedWidthConstraints(itemW)).Y

		if l.checkmarkIndexPlus1 == i+1 {
			imgSize := listItemCheckmarkSize(context)
			imgP := p
			imgP.X += listItemIndentSize(context, item.IndentLevel)
			imgP.X += UnitSize(context) / 4
			itemH := contentH
			imgP.Y += (itemH - imgSize) / 2
			// Adjust the position a bit for better appearance.
			imgP.Y += UnitSize(context) / 16
			imgP.Y += item.Padding.Top
			imgP.Y = l.adjustItemY(context, imgP.Y)
			l.widgetBoundsForLayout[&l.checkmark] = image.Rectangle{
				Min: imgP,
				Max: imgP.Add(image.Pt(imgSize, imgSize)),
			}
		}

		if item.IndentLevel > 0 {
			var img *ebiten.Image
			var hasChild bool
			if nextItem, ok := l.abstractList.ItemByIndex(i + 1); ok {
				hasChild = nextItem.IndentLevel > item.IndentLevel
			}
			if hasChild {
				if item.Collapsed {
					img = l.treeItemCollapsedImage
				} else {
					img = l.treeItemExpandedImage
				}
			}
			l.expanderImages.At(i).SetImage(img)
			expanderP := p
			expanderP.X += listItemIndentSize(context, item.IndentLevel) - LineHeight(context)
			// Adjust the position a bit for better appearance.
			expanderP.Y += UnitSize(context) / 16
			expanderP.Y += item.Padding.Top
			s := image.Pt(
				LineHeight(context),
				contentH,
			)
			l.widgetBoundsForLayout[l.expanderImages.At(i)] = image.Rectangle{
				Min: expanderP,
				Max: expanderP.Add(s),
			}
		}

		itemP := p
		if l.checkmarkIndexPlus1 > 0 {
			itemP.X += listItemCheckmarkSize(context) + listItemTextAndImagePadding(context)
		}
		itemP.X += listItemIndentSize(context, item.IndentLevel)
		itemP.X += item.Padding.Start
		itemP.Y = l.adjustItemY(context, itemP.Y)
		itemP.Y += item.Padding.Top
		r := image.Rectangle{
			Min: itemP,
			Max: itemP.Add(image.Pt(itemW, contentH)),
		}
		l.widgetBoundsForLayout[item.Content] = r
		l.itemBoundsForLayoutFromIndex[i] = r

		p.Y += contentH + item.Padding.Top + item.Padding.Bottom
	}

	l.widthForCachedHeight = cw
	l.cachedHeight = p.Y - widgetBounds.Bounds().Min.Y + RoundedCornerRadius(context)

	if l.customBackground != nil {
		layouter.LayoutWidget(l.customBackground, widgetBounds.Bounds())
	}
	layouter.LayoutWidget(&l.background2, widgetBounds.Bounds())
	for widget, bounds := range l.widgetBoundsForLayout {
		layouter.LayoutWidget(widget, bounds)
	}
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
	for i := range l.visibleItems() {
		item, _ := l.abstractList.ItemByIndex(i)
		var constraint guigui.Constraints
		// If width is 0, there is no constraint.
		// This is used mainly for a menu list.
		if width > 0 {
			itemW := width - 2*RoundedCornerRadius(context)
			if hasCheckmark {
				itemW -= offsetForCheckmark
			}
			itemW -= listItemIndentSize(context, item.IndentLevel)
			itemW -= item.Padding.Start + item.Padding.End
			constraint = guigui.FixedWidthConstraints(itemW)
		}
		s := item.Content.Measure(context, constraint)
		w = max(w, s.X+listItemIndentSize(context, item.IndentLevel)+item.Padding.Start+item.Padding.End)
		h += s.Y + item.Padding.Top + item.Padding.Bottom
	}
	w += 2 * RoundedCornerRadius(context)
	h += 2 * RoundedCornerRadius(context)
	if hasCheckmark {
		w += offsetForCheckmark
	}
	if width > 0 {
		w = width
		l.widthForCachedHeight = width
		l.cachedHeight = h
	}
	return image.Pt(w, h)
}

func (l *listContent[T]) hasMovableItems() bool {
	for i := range l.visibleItems() {
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
	l.abstractList.SetItems(items)
}

func (l *listContent[T]) SelectItemByIndex(index int) {
	l.selectItemByIndex(index, false)
}

func (l *listContent[T]) SelectItemsByIndices(indices []int) {
	if l.abstractList.SelectItemsByIndices(indices, false) {
		guigui.RequestRebuild(l)
	}
}

func (l *listContent[T]) selectItemByIndex(index int, forceFireEvents bool) {
	if l.abstractList.SelectItemByIndex(index, forceFireEvents) {
		guigui.RequestRebuild(l)
	}
}

func (l *listContent[T]) extendItemSelectionByIndex(index int, forceFireEvents bool) {
	if l.abstractList.ExtendItemSelectionByIndex(index, forceFireEvents) {
		guigui.RequestRebuild(l)
	}
}

func (l *listContent[T]) toggleItemSelectionByIndex(index int, forceFireEvents bool) {
	if l.abstractList.ToggleItemSelectionByIndex(index, forceFireEvents) {
		guigui.RequestRebuild(l)
	}
}

func (l *listContent[T]) SelectItemByValue(value T) {
	if l.abstractList.SelectItemByValue(value, false) {
		guigui.RequestRebuild(l)
	}
}

func (l *listContent[T]) SelectItemsByValues(values []T) {
	if l.abstractList.SelectItemsByValues(values, false) {
		guigui.RequestRebuild(l)
	}
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

func (l *listContent[T]) isHoveringVisible() bool {
	return l.style == ListStyleMenu
}

func (l *listContent[T]) Style() ListStyle {
	return l.style
}

func (l *listContent[T]) SetStyle(style ListStyle) {
	if l.style == style {
		return
	}
	l.style = style
	guigui.RequestRebuild(l)
}

func (l *listContent[T]) calcDropDstIndex(context *guigui.Context) int {
	_, y := ebiten.CursorPosition()
	for i := range l.visibleItems() {
		if b := l.itemBounds(context, i); y < (b.Min.Y+b.Max.Y)/2 {
			return i
		}
	}
	return l.abstractList.ItemCount()
}

func (l *listContent[T]) resetHoveredItemIndex() {
	if l.hoveredItemIndexPlus1 == 0 && l.lastHoveredItemIndexPlus1 == 0 {
		return
	}
	l.hoveredItemIndexPlus1 = 0
	l.lastHoveredItemIndexPlus1 = 0
	guigui.RequestRebuild(l)
}

func (l *listContent[T]) hoveredItemIndex(context *guigui.Context, widgetBounds *guigui.WidgetBounds) int {
	if !widgetBounds.IsHitAtCursor() {
		return -1
	}
	cp := image.Pt(ebiten.CursorPosition())
	listBounds := widgetBounds.Bounds()
	for i := range l.visibleItems() {
		bounds := l.itemBounds(context, i)
		bounds.Min.X = listBounds.Min.X
		bounds.Max.X = listBounds.Max.X
		if cp.In(bounds) {
			return i
		}
	}
	return -1
}

func (l *listContent[T]) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	l.hoveredItemIndexPlus1 = l.hoveredItemIndex(context, widgetBounds) + 1

	colorMode := context.ColorMode()
	if l.hoveredItemIndexPlus1 == l.checkmarkIndexPlus1 {
		colorMode = guigui.ColorModeDark
	}
	checkImg, err := theResourceImages.Get("check", colorMode)
	if err != nil {
		panic(fmt.Sprintf("basicwidget: failed to get check image: %v", err))
	}
	l.checkmark.SetImage(checkImg)

	if l.isHoveringVisible() || l.hasMovableItems() {
		if l.lastHoveredItemIndexPlus1 != l.hoveredItemIndexPlus1 {
			l.lastHoveredItemIndexPlus1 = l.hoveredItemIndexPlus1
			guigui.RequestRebuild(l)
		}
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
				guigui.DispatchEvent(l, listEventScrollDeltaY, dy)
			}
			if i := l.calcDropDstIndex(context); l.dragDstIndexPlus1-1 != i {
				l.dragDstIndexPlus1 = i + 1
				guigui.RequestRedraw(l)
				return guigui.HandleInputByWidget(l)
			}
			return guigui.AbortHandlingInputByWidget(l)
		}
		if l.dragDstIndexPlus1 > 0 {
			indices := l.abstractList.AppendSelectedItemIndices(nil)
			if len(indices) > 0 {
				guigui.DispatchEvent(l, listEventItemsMoved, indices[0], len(indices), l.dragDstIndexPlus1-1)
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
				} else if (!isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl)) ||
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

			l.pressStartPlus1 = c.Add(image.Pt(1, 1))
			l.startPressingIndexPlus1 = index + 1
			if left {
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
			if l.abstractList.SelectGroupAt(index, false) {
				guigui.RequestRebuild(l)
			}
			if !l.abstractList.IsSelectedItemIndex(index) {
				return guigui.AbortHandlingInputByWidget(l)
			}
			indices := l.abstractList.AppendSelectedItemIndices(nil)
			if len(indices) == 0 {
				return guigui.AbortHandlingInputByWidget(l)
			}
			for _, index := range indices {
				item, _ := l.abstractList.ItemByIndex(index)
				if !item.Movable {
					return guigui.AbortHandlingInputByWidget(l)
				}
			}
			if start := l.pressStartPlus1.Sub(image.Pt(1, 1)); start.Y != c.Y {
				itemBoundsMin := l.itemBounds(context, indices[0])
				itemBoundsMax := l.itemBounds(context, indices[len(indices)-1])
				minY := min((itemBoundsMin.Min.Y+start.Y)/2, (itemBoundsMin.Min.Y+itemBoundsMin.Max.Y)/2)
				maxY := max((itemBoundsMax.Max.Y+start.Y)/2, (itemBoundsMax.Min.Y+itemBoundsMax.Max.Y)/2)
				if c.Y < minY || c.Y >= maxY {
					l.dragSrcIndexPlus1 = indices[0] + 1
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
			if y, ok := l.itemYFromIndex(context, idx); ok {
				y -= RoundedCornerRadius(context)
				guigui.DispatchEvent(l, listEventScrollY, float64(-y))
			}
			l.indexToJumpPlus1 = 0
		}
		if idx := l.indexToEnsureVisiblePlus1 - 1; idx >= 0 && idx < l.abstractList.ItemCount() {
			topY, topOK := l.itemYFromIndex(context, idx)
			bottomY, bottomOK := l.itemYFromIndex(context, idx+1)
			if topOK && bottomOK {
				guigui.DispatchEvent(l, listEventScrollYEnsureVisible, float64(-topY), float64(-bottomY))
			}
			l.indexToEnsureVisiblePlus1 = 0
		}
		l.jumpTick = 0
	}
	return nil
}

// itemYFromIndex returns the Y position of the item at the given index relative to the top of the List widget.
// itemYFromIndex returns the same value whatever the List position is.
//
// itemYFromIndex is available after Build is called, so do not use this from a parent widget.
func (l *listContent[T]) itemYFromIndex(context *guigui.Context, index int) (int, bool) {
	if index < 0 || index > len(l.itemBoundsForLayoutFromIndex) || len(l.itemBoundsForLayoutFromIndex) == 0 {
		return 0, false
	}

	baseY := l.itemBoundsForLayoutFromIndex[0].Min.Y
	head := RoundedCornerRadius(context)

	var itemRelY int
	if index == len(l.itemBoundsForLayoutFromIndex) {
		itemRelY = l.itemBoundsForLayoutFromIndex[index-1].Max.Y - baseY
		var padding guigui.Padding
		if item, ok := l.abstractList.ItemByIndex(index - 1); ok {
			padding = item.Padding
		}
		return itemRelY + head + padding.Bottom, true
	}

	itemRelY = l.itemBoundsForLayoutFromIndex[index].Min.Y - baseY
	var padding guigui.Padding
	if item, ok := l.abstractList.ItemByIndex(index); ok {
		padding = item.Padding
	}
	return itemRelY + head - padding.Top, true
}

// itemYFromIndexForMenu returns the Y position of the item at the given index relative to the top of the List widget.
// itemYFromIndexForMenu returns the same value whatever the List position is.
//
// itemYFromIndexForMenu is available anytime even before Build is called.
func (l *listContent[T]) itemYFromIndexForMenu(context *guigui.Context, index int) (int, bool) {
	y := RoundedCornerRadius(context)
	for i := range l.visibleItems() {
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

func (l *listContent[T]) selectedItemColor(context *guigui.Context) color.Color {
	if l.SelectedItemIndex() < 0 || l.SelectedItemIndex() >= l.abstractList.ItemCount() {
		return nil
	}
	if l.style == ListStyleMenu {
		return nil
	}
	if context.IsFocusedOrHasFocusedChild(l) || l.style == ListStyleSidebar {
		return draw.Color(context.ColorMode(), draw.ColorTypeAccent, 0.5)
	}
	if !context.IsEnabled(l) {
		return draw.Color2(context.ColorMode(), draw.ColorTypeBase, 0.7, 0.2)
	}
	return draw.Color2(context.ColorMode(), draw.ColorTypeBase, 0.7, 0.5)
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
		for i := range l.content.visibleItems() {
			count++
			if count%2 == 1 {
				continue
			}
			bounds := l.content.itemBounds(context, i)
			// Reset the X position to ignore indentation.
			item, _ := l.content.abstractList.ItemByIndex(i)
			bounds.Min.X -= listItemIndentSize(context, item.IndentLevel)
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

	content *listContent[T]
}

func (l *listBackground2[T]) setListContent(content *listContent[T]) {
	l.content = content
}

func (l *listBackground2[T]) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	vb := widgetBounds.VisibleBounds()

	// Draw the selected item background.
	if clr := l.content.selectedItemColor(context); clr != nil {
		// TODO: Improve the performance.
		indexToVisibleItemIndex := map[int]int{}
		var visibleItemIndexToIndex []int
		var count int
		for index := range l.content.visibleItems() {
			indexToVisibleItemIndex[index] = count
			visibleItemIndexToIndex = append(visibleItemIndexToIndex, index)
			count++
		}
		for _, index := range l.content.AppendSelectedItemIndices(nil) {
			if !l.content.isItemVisible(index) {
				continue
			}
			bounds := l.content.itemBounds(context, index)
			if l.content.style == ListStyleMenu {
				bounds.Max.X = bounds.Min.X + widgetBounds.Bounds().Dx() - 2*RoundedCornerRadius(context)
			}
			if bounds.Overlaps(vb) {
				item, _ := l.content.ItemByIndex(index)
				var corners basicwidgetdraw.Corners
				vi := indexToVisibleItemIndex[index]
				// If prev visible item is adjacent to this item, don't draw the top corner.
				if item.Padding.Top == 0 && vi-1 >= 0 && vi-1 < len(visibleItemIndexToIndex) {
					prevIndex := visibleItemIndexToIndex[vi-1]
					if prevItem, ok := l.content.ItemByIndex(prevIndex); ok && prevItem.Padding.Bottom == 0 {
						if l.content.IsSelectedItemIndex(prevIndex) {
							corners.TopStart = prevItem.IndentLevel <= item.IndentLevel &&
								prevItem.Padding.Start == item.Padding.Start
							corners.TopEnd = prevItem.Padding.End == item.Padding.End
						}
					}
				}
				// If next visible item is adjacent to this item, don't draw the bottom corner.
				if item.Padding.Bottom == 0 && vi+1 >= 0 && vi+1 < len(visibleItemIndexToIndex) {
					nextIndex := visibleItemIndexToIndex[vi+1]
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
	if ok && l.content.isHoveringVisible() && hoveredItemIndex >= 0 && hoveredItemIndex < l.content.abstractList.ItemCount() && !hoveredItem.Unselectable && l.content.isItemVisible(hoveredItemIndex) {
		bounds := l.content.itemBounds(context, hoveredItemIndex)
		if l.content.style == ListStyleMenu {
			bounds.Max.X = bounds.Min.X + widgetBounds.Bounds().Dx() - 2*RoundedCornerRadius(context)
		}
		if bounds.Overlaps(vb) {
			clr := draw.Color(context.ColorMode(), draw.ColorTypeBase, 0.9)
			if l.content.style == ListStyleMenu {
				clr = draw.Color(context.ColorMode(), draw.ColorTypeAccent, 0.5)
			}
			basicwidgetdraw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
		}
	}

	// Draw a drag indicator.
	if context.IsEnabled(l) && l.content.dragSrcIndexPlus1 == 0 {
		if item, ok := l.content.abstractList.ItemByIndex(hoveredItemIndex); ok && item.Movable {
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
	if l.content.dragDstIndexPlus1 > 0 {
		p := widgetBounds.Bounds().Min
		x0 := float32(p.X) + float32(RoundedCornerRadius(context))
		cw := widgetBounds.Bounds().Dx()
		if l.content.contentWidthPlus1 > 0 {
			cw = l.content.contentWidthPlus1 - 1
		}
		x1 := x0 + float32(cw)
		x1 -= 2 * float32(RoundedCornerRadius(context))
		y := float32(p.Y)
		if itemY, ok := l.content.itemYFromIndex(context, l.content.dragDstIndexPlus1-1); ok {
			y += float32(itemY)
			vector.StrokeLine(dst, x0, y, x1, y, 2*float32(context.Scale()), draw.Color(context.ColorMode(), draw.ColorTypeAccent, 0.5), false)
		}
	}
}

type listFrame struct {
	guigui.DefaultWidget

	headerHeight int
	footerHeight int
	style        ListStyle
}

func (l *listFrame) SetHeaderHeight(height int) {
	if l.headerHeight == height {
		return
	}
	l.headerHeight = height
	guigui.RequestRebuild(l)
}

func (l *listFrame) SetFooterHeight(height int) {
	if l.footerHeight == height {
		return
	}
	l.footerHeight = height
	guigui.RequestRebuild(l)
}

func (l *listFrame) SetStyle(style ListStyle) {
	if l.style == style {
		return
	}
	l.style = style
	guigui.RequestRebuild(l)
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
		clr := draw.Color2(context.ColorMode(), draw.ColorTypeBase, 0.9, 0.4)
		if !context.IsEnabled(l) {
			clr = draw.Color2(context.ColorMode(), draw.ColorTypeBase, 0.8, 0.3)
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
		clr := draw.Color2(context.ColorMode(), draw.ColorTypeBase, 0.9, 0.4)
		if !context.IsEnabled(l) {
			clr = draw.Color2(context.ColorMode(), draw.ColorTypeBase, 0.8, 0.3)
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

func listItemIndentSize(context *guigui.Context, level int) int {
	if level == 0 {
		return 0
	}
	return LineHeight(context) + LineHeight(context)/2*(level-1)
}

func listBorderWidth(context *guigui.Context) float32 {
	return float32(1 * context.Scale())
}
