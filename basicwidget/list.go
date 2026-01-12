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

const (
	listEventItemSelected         = "itemSelected"
	listEventItemsMoved           = "itemsMoved"
	listEventItemExpanderToggled  = "itemExpanderToggled"
	listEventScrollY              = "scrollY"
	listEventScrollYEnsureVisible = "scrollYEnsureVisible"
	listEventScrollDeltaY         = "scrollDeltaY"
)

// TODO: Clean up functions for colors.
func DefaultActiveListItemTextColor(context *guigui.Context) color.Color {
	return draw.Color2(context.ColorMode(), draw.ColorTypeBase, 1, 1)
}

type List[T comparable] struct {
	guigui.DefaultWidget

	abstractListItems []abstractListItem[T]
	listItemWidgets   []listItemWidget[T]
	background1       listBackground1[T]
	content           listContent[T]
	frame             listFrame

	listItemHeightPlus1 int
	headerHeight        int
	footerHeight        int
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

func (l *List[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&l.background1)
	adder.AddChild(&l.content)
	adder.AddChild(&l.frame)
	context.SetContainer(l, true)

	l.background1.setListContent(&l.content)

	for i := range l.listItemWidgets {
		item := &l.listItemWidgets[i]
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
	layouter.LayoutWidget(&l.content, bounds)
	layouter.LayoutWidget(&l.frame, widgetBounds.Bounds())
}

func (l *List[T]) hoveredItemIndex() int {
	return l.content.hoveredItemIndexPlus1 - 1
}

func (l *List[T]) HighlightedItemIndex(context *guigui.Context) int {
	index := -1
	switch l.content.Style() {
	case ListStyleNormal, ListStyleSidebar:
		index = l.content.SelectedItemIndex()
	case ListStyleMenu:
		if !l.content.IsHoveringVisible() {
			return -1
		}
		index = l.hoveredItemIndex()
	}
	if index < 0 || index >= len(l.listItemWidgets) {
		return -1
	}
	if !l.abstractListItems[index].selectable() {
		return -1
	}
	if !context.IsEnabled(&l.listItemWidgets[index]) {
		return -1
	}
	return index
}

func (l *List[T]) ItemTextColor(context *guigui.Context, index int) color.Color {
	if l.HighlightedItemIndex(context) == index {
		return DefaultActiveListItemTextColor(context)
	}
	item := &l.listItemWidgets[index]
	if clr := item.textColor(); clr != nil {
		return clr
	}
	return basicwidgetdraw.TextColor(context.ColorMode(), context.IsEnabled(item))
}

func (l *List[T]) SelectedItemIndex() int {
	return l.content.SelectedItemIndex()
}

func (l *List[T]) SelectedItem() (ListItem[T], bool) {
	return l.ItemByIndex(l.content.SelectedItemIndex())
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
	l.abstractListItems = adjustSliceSize(l.abstractListItems, len(items))
	l.listItemWidgets = adjustSliceSize(l.listItemWidgets, len(items))

	for i, item := range items {
		l.listItemWidgets[i].setListItem(item)
		l.listItemWidgets[i].setHeight(l.listItemHeightPlus1 - 1)
		l.listItemWidgets[i].setStyle(l.content.Style())
		l.abstractListItems[i].Content = &l.listItemWidgets[i]
		l.abstractListItems[i].Unselectable = !item.selectable()
		l.abstractListItems[i].Movable = item.Movable
		l.abstractListItems[i].Value = item.Value
		l.abstractListItems[i].IndentLevel = item.IndentLevel
		l.abstractListItems[i].Padding = item.Padding
		l.abstractListItems[i].Collapsed = item.Collapsed
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

func (l *List[T]) SelectItemByValue(value T) {
	l.content.SelectItemByValue(value)
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
	l.listItemWidgets[index].setText(str)
}

func (l *List[T]) setContentWidth(width int) {
	l.content.SetContentWidth(width)
}

func (l *List[T]) scrollOffset() (float64, float64) {
	return l.content.ScrollOffset()
}

func (l *List[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return l.content.Measure(context, constraints)
}

type listItemWidget[T comparable] struct {
	guigui.DefaultWidget

	text    Text
	keyText Text

	item        ListItem[T]
	heightPlus1 int
	style       ListStyle

	layoutItems []guigui.LinearLayoutItem
}

func (l *listItemWidget[T]) setListItem(listItem ListItem[T]) {
	l.item = listItem
	// TODO: Should this call guigui.RequestRedraw(l) when the item changes?
}

func (l *listItemWidget[T]) setHeight(height int) {
	if l.heightPlus1 == height+1 {
		return
	}
	l.heightPlus1 = height + 1
	guigui.RequestRebuild(l)
}

func (l *listItemWidget[T]) setStyle(style ListStyle) {
	if l.style == style {
		return
	}
	l.style = style
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
}

func (a abstractListItem[T]) value() T {
	return a.Value
}

func (a abstractListItem[T]) selectable() bool {
	return !a.Unselectable
}

type listContent[T comparable] struct {
	guigui.DefaultWidget

	customBackground guigui.Widget
	background2      listBackground2[T]
	checkmark        Image
	expanderImages   []Image
	scrollOverlay    scrollOverlay

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

	widgetBoundsForLayout        map[guigui.Widget]image.Rectangle
	itemBoundsForLayoutFromIndex []image.Rectangle

	treeItemCollapsedImage *ebiten.Image
	treeItemExpandedImage  *ebiten.Image

	prevWidth int

	onItemSelected         func(index int)
	onScrollY              func(context *guigui.Context, offsetY float64)
	onScrollYEnsureVisible func(context *guigui.Context, offsetYTop, offsetYBottom float64)
	onScrollDeltaY         func(context *guigui.Context, deltaY float64)

	// TODO: Remove these members by introducing Panel.
	scrollOffsetYMinus1       float64
	scrollOffsetDeltaY        float64
	scrollOffsetYTopMinus1    float64
	scrollOffsetYBottomMinus1 float64
}

func (l *listContent[T]) SetBackground(widget guigui.Widget) {
	l.customBackground = widget
}

func (l *listContent[T]) SetOnItemSelected(f func(context *guigui.Context, index int)) {
	guigui.SetEventHandler(l, listEventItemSelected, f)
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

func (l *listContent[T]) visibleItems() iter.Seq2[int, abstractListItem[T]] {
	return func(yield func(int, abstractListItem[T]) bool) {
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
			if !yield(i, item) {
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
	l.expanderImages = adjustSliceSize(l.expanderImages, l.abstractList.ItemCount())
	for i := range l.visibleItems() {
		item, _ := l.abstractList.ItemByIndex(i)
		if l.checkmarkIndexPlus1 == i+1 {
			adder.AddChild(&l.checkmark)
		}
		if item.IndentLevel > 0 {
			adder.AddChild(&l.expanderImages[i])
		}
		adder.AddChild(item.Content)
	}
	adder.AddChild(&l.scrollOverlay)

	if l.onItemSelected == nil {
		l.onItemSelected = func(index int) {
			guigui.DispatchEvent(l, listEventItemSelected, index)
		}
	}
	l.abstractList.SetOnItemSelected(l.onItemSelected)

	if l.onScrollY == nil {
		l.onScrollY = func(context *guigui.Context, offsetY float64) {
			l.scrollOffsetYMinus1 = offsetY - 1
		}
	}
	guigui.SetEventHandler(l, listEventScrollY, l.onScrollY)

	if l.onScrollYEnsureVisible == nil {
		l.onScrollYEnsureVisible = func(context *guigui.Context, offsetYTop, offsetYBottom float64) {
			l.scrollOffsetYTopMinus1 = offsetYTop - 1
			l.scrollOffsetYBottomMinus1 = offsetYBottom - 1
		}
	}
	guigui.SetEventHandler(l, listEventScrollYEnsureVisible, l.onScrollYEnsureVisible)

	if l.onScrollDeltaY == nil {
		l.onScrollDeltaY = func(context *guigui.Context, deltaY float64) {
			l.scrollOffsetDeltaY = deltaY
		}
	}
	guigui.SetEventHandler(l, listEventScrollDeltaY, l.onScrollDeltaY)

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
	// Record the current position of the selected item.
	var headToSelectedItem int
	if idx := l.SelectedItemIndex(); idx >= 0 {
		if y0, ok := l.itemYFromIndex(context, idx); ok {
			_, offsetY := l.scrollOverlay.Offset()
			y := int(-offsetY)
			headToSelectedItem = y0 - y
			if headToSelectedItem < 0 || headToSelectedItem >= widgetBounds.Bounds().Dy() {
				headToSelectedItem = 0
			}
		}
	}

	cw := widgetBounds.Bounds().Dx()
	if l.contentWidthPlus1 > 0 {
		cw = l.contentWidthPlus1 - 1
	}

	p := widgetBounds.Bounds().Min
	offsetX, offsetY := l.scrollOverlay.Offset()
	p.X += RoundedCornerRadius(context) + int(offsetX)
	p.Y += RoundedCornerRadius(context) + int(offsetY)

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
			l.expanderImages[i].SetImage(img)
			expanderP := p
			expanderP.X += listItemIndentSize(context, item.IndentLevel) - int(LineHeight(context))
			// Adjust the position a bit for better appearance.
			expanderP.Y += UnitSize(context) / 16
			expanderP.Y += item.Padding.Top
			s := image.Pt(
				int(LineHeight(context)),
				contentH,
			)
			l.widgetBoundsForLayout[&l.expanderImages[i]] = image.Rectangle{
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

	// TODO: Now scrollOverlay's widgetBounds doens't match with List's widgetBounds.
	// Separate a content part and use Panel.
	cs := l.measure(context, cw)
	l.scrollOverlay.SetContentSize(context, widgetBounds, cs)

	// Adjust the scroll offset to show the selected item if needed.
	if l.prevWidth != widgetBounds.Bounds().Dx() && headToSelectedItem != 0 {
		if y0, ok := l.itemYFromIndex(context, l.SelectedItemIndex()); ok {
			newOffsetY := -float64(y0 - headToSelectedItem)
			guigui.DispatchEvent(l, listEventScrollY, context, newOffsetY)
		}
	}
	l.prevWidth = widgetBounds.Bounds().Dx()

	if l.customBackground != nil {
		layouter.LayoutWidget(l.customBackground, widgetBounds.Bounds())
	}
	layouter.LayoutWidget(&l.background2, widgetBounds.Bounds())
	layouter.LayoutWidget(&l.scrollOverlay, widgetBounds.Bounds())
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
	return l.measure(context, width)
}

func (l *listContent[T]) measure(context *guigui.Context, width int) image.Point {
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
			guigui.FixedWidthConstraints(itemW)
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

func (l *listContent[T]) SelectedItemIndex() int {
	return l.abstractList.SelectedItemIndex()
}

func (l *listContent[T]) SetItems(items []abstractListItem[T]) {
	l.abstractList.SetItems(items)
}

func (l *listContent[T]) SelectItemByIndex(index int) {
	l.selectItemByIndex(index, false)
}

func (l *listContent[T]) selectItemByIndex(index int, forceFireEvents bool) {
	if l.abstractList.SelectItemByIndex(index, forceFireEvents) {
		guigui.RequestRebuild(l)
	}
}

func (l *listContent[T]) SelectItemByValue(value T) {
	if l.abstractList.SelectItemByValue(value, false) {
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

func (l *listContent[T]) IsHoveringVisible() bool {
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

func (l *listContent[T]) ScrollOffset() (float64, float64) {
	return l.scrollOverlay.Offset()
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
	if l.hoveredItemIndexPlus1 == 0 {
		return
	}
	l.hoveredItemIndexPlus1 = 0
	guigui.RequestRebuild(l)
}

func (l *listContent[T]) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	l.hoveredItemIndexPlus1 = 0
	if widgetBounds.IsHitAtCursor() {
		cp := image.Pt(ebiten.CursorPosition())
		listBounds := widgetBounds.Bounds()
		for i := range l.visibleItems() {
			bounds := l.itemBounds(context, i)
			bounds.Min.X = listBounds.Min.X
			bounds.Max.X = listBounds.Max.X
			hovered := cp.In(bounds)
			if hovered {
				l.hoveredItemIndexPlus1 = i + 1
			}
		}
	}

	colorMode := context.ColorMode()
	if l.hoveredItemIndexPlus1 == l.checkmarkIndexPlus1 {
		colorMode = guigui.ColorModeDark
	}
	checkImg, err := theResourceImages.Get("check", colorMode)
	if err != nil {
		panic(fmt.Sprintf("basicwidget: failed to get check image: %v", err))
	}
	l.checkmark.SetImage(checkImg)

	if l.IsHoveringVisible() || l.hasMovableItems() {
		if l.lastHoveredItemIndexPlus1 != l.hoveredItemIndexPlus1 {
			l.lastHoveredItemIndexPlus1 = l.hoveredItemIndexPlus1
			guigui.RequestRebuild(l)
		}
	}

	// Process dragging.
	if l.dragSrcIndexPlus1 > 0 {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			_, y := ebiten.CursorPosition()
			p := widgetBounds.Bounds().Min
			h := widgetBounds.Bounds().Dy()
			var dy float64
			if upperY := p.Y + UnitSize(context); y < upperY {
				dy = float64(upperY-y) / 4
			}
			if lowerY := p.Y + h - UnitSize(context); y >= lowerY {
				dy = float64(lowerY-y) / 4
			}
			guigui.DispatchEvent(l, listEventScrollDeltaY, dy)
			if i := l.calcDropDstIndex(context); l.dragDstIndexPlus1-1 != i {
				l.dragDstIndexPlus1 = i + 1
				guigui.RequestRedraw(l)
				return guigui.HandleInputByWidget(l)
			}
			return guigui.AbortHandlingInputByWidget(l)
		}
		if l.dragDstIndexPlus1 > 0 {
			// TODO: Implement multiple items drop.
			guigui.DispatchEvent(l, listEventItemsMoved, l.dragSrcIndexPlus1-1, 1, l.dragDstIndexPlus1-1)
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
		case (left || right) && l.scrollOverlay.isWidgetHitAtCursor(context, widgetBounds):
			item, _ := l.abstractList.ItemByIndex(index)
			if c.X < l.itemBoundsForLayoutFromIndex[index].Min.X {
				if left {
					expanded := !item.Collapsed
					guigui.DispatchEvent(l, listEventItemExpanderToggled, index, !expanded)
				}
				return guigui.AbortHandlingInputByWidget(l)
			}
			if item.Unselectable {
				return guigui.AbortHandlingInputByWidget(l)
			}

			wasFocused := context.IsFocusedOrHasFocusedChild(l)
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
			if l.SelectedItemIndex() != index || !wasFocused || l.style == ListStyleMenu {
				l.selectItemByIndex(index, true)
			}
			l.pressStartPlus1 = c.Add(image.Pt(1, 1))
			l.startPressingIndexPlus1 = index + 1
			if left {
				return guigui.HandleInputByWidget(l)
			}
			// For the right click, give a chance to a parent widget to handle the right click e.g. to open a context menu.
			// TODO: This behavior seems a little ad-hoc. Consider a better way.
			return guigui.HandleInputResult{}

		case ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && l.scrollOverlay.isWidgetHitAtCursor(context, widgetBounds):
			item, _ := l.abstractList.ItemByIndex(index)
			if item.Movable && l.SelectedItemIndex() == index && l.startPressingIndexPlus1-1 == index && (l.pressStartPlus1 != c.Add(image.Pt(1, 1))) {
				l.dragSrcIndexPlus1 = index + 1
				return guigui.HandleInputByWidget(l)
			}
			return guigui.AbortHandlingInputByWidget(l)

		case inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft):
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
	cw := widgetBounds.Bounds().Dx()
	if l.contentWidthPlus1 > 0 {
		cw = l.contentWidthPlus1 - 1
	}
	cs := l.measure(context, cw)
	if l.scrollOffsetYMinus1 != 0 {
		offsetX, _ := l.scrollOverlay.Offset()
		l.scrollOverlay.SetOffset(context, widgetBounds, cs, offsetX, l.scrollOffsetYMinus1+1)
		l.scrollOffsetYMinus1 = 0
	}
	if l.scrollOffsetYTopMinus1 != 0 || l.scrollOffsetYBottomMinus1 != 0 {
		// Adjust the bottom first.
		if l.scrollOffsetYBottomMinus1 != 0 {
			y := l.scrollOffsetYBottomMinus1 + 1
			y += float64(widgetBounds.Bounds().Dy())
			y -= float64(RoundedCornerRadius(context))
			if offsetX, offsetY := l.scrollOverlay.Offset(); y < offsetY {
				l.scrollOverlay.SetOffset(context, widgetBounds, cs, offsetX, y)
			}
		}
		// Then adjust the top.
		if l.scrollOffsetYTopMinus1 != 0 {
			y := l.scrollOffsetYTopMinus1 + 1
			y += float64(RoundedCornerRadius(context))
			// Reget the offset as it may be changed by the above bottom adjustment.
			if offsetX, offsetY := l.scrollOverlay.Offset(); y > offsetY {
				l.scrollOverlay.SetOffset(context, widgetBounds, cs, offsetX, y)
			}
		}
		l.scrollOffsetYTopMinus1 = 0
		l.scrollOffsetYBottomMinus1 = 0
	}
	if l.scrollOffsetDeltaY != 0 {
		l.scrollOverlay.SetOffsetByDelta(context, widgetBounds, cs, 0, l.scrollOffsetDeltaY)
		l.scrollOffsetDeltaY = 0
	}

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
		clr = basicwidgetdraw.SecondaryControlColor(context.ColorMode(), context.IsEnabled(l))
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
			clr := basicwidgetdraw.SecondaryControlColor(context.ColorMode(), context.IsEnabled(l))
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
	if clr := l.content.selectedItemColor(context); clr != nil && l.content.SelectedItemIndex() >= 0 && l.content.SelectedItemIndex() < l.content.abstractList.ItemCount() && l.content.isItemVisible(l.content.SelectedItemIndex()) {
		bounds := l.content.itemBounds(context, l.content.SelectedItemIndex())
		if l.content.style == ListStyleMenu {
			bounds.Max.X = bounds.Min.X + widgetBounds.Bounds().Dx() - 2*RoundedCornerRadius(context)
		}
		if bounds.Overlaps(vb) {
			basicwidgetdraw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
		}
	}

	hoveredItemIndex := l.content.hoveredItemIndexPlus1 - 1
	hoveredItem, ok := l.content.abstractList.ItemByIndex(hoveredItemIndex)
	if ok && l.content.IsHoveringVisible() && hoveredItemIndex >= 0 && hoveredItemIndex < l.content.abstractList.ItemCount() && !hoveredItem.Unselectable && l.content.isItemVisible(hoveredItemIndex) {
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
		offsetX, _ := l.content.scrollOverlay.Offset()
		x0 := float32(p.X) + float32(RoundedCornerRadius(context))
		x0 += float32(offsetX)
		cw := widgetBounds.Bounds().Dx()
		if l.content.contentWidthPlus1 > 0 {
			cw = l.content.contentWidthPlus1 - 1
		}
		x1 := x0 + float32(cw)
		x1 -= 2 * float32(RoundedCornerRadius(context))
		y := float32(p.Y)
		if itemY, ok := l.content.itemYFromIndex(context, l.content.dragDstIndexPlus1-1); ok {
			y += float32(itemY)
			_, offsetY := l.content.scrollOverlay.Offset()
			y += float32(offsetY)
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
	clr1, clr2 := basicwidgetdraw.BorderColors(context.ColorMode(), basicwidgetdraw.RoundedRectBorderType(border), false)
	borderWidth := float32(1 * context.Scale())
	basicwidgetdraw.DrawRoundedRectBorder(context, dst, bounds, clr1, clr2, RoundedCornerRadius(context), borderWidth, border)
}

func listItemCheckmarkSize(context *guigui.Context) int {
	return int(LineHeight(context) * 3 / 4)
}

func listItemTextAndImagePadding(context *guigui.Context) int {
	return UnitSize(context) / 8
}

func listItemIndentSize(context *guigui.Context, level int) int {
	if level == 0 {
		return 0
	}
	return int(LineHeight(context) + LineHeight(context)/2*float64(level-1))
}
