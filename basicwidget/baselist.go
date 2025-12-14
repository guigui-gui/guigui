// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"fmt"
	"image"
	"image/color"
	"iter"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

const (
	baseListEventItemSelected        = "itemSelected"
	baseListEventItemsMoved          = "itemsMoved"
	baseListEventItemExpanderToggled = "itemExpanderToggled"
)

type ListStyle int

const (
	ListStyleNormal ListStyle = iota
	ListStyleSidebar
	ListStyleMenu
)

type baseListItem[T comparable] struct {
	Content     guigui.Widget
	Selectable  bool
	Movable     bool
	Value       T
	IndentLevel int
	Padding     guigui.Padding
	Collapsed   bool
}

func (b baseListItem[T]) value() T {
	return b.Value
}

func (b baseListItem[T]) selectable() bool {
	return b.Selectable
}

func DefaultActiveListItemTextColor(context *guigui.Context) color.Color {
	return draw.Color2(context.ColorMode(), draw.ColorTypeBase, 1, 1)
}

type baseList[T comparable] struct {
	guigui.DefaultWidget

	background1 baseListBackground1[T]
	content     baseListContent[T]
	frame       baseListFrame

	headerHeight int
	footerHeight int
}

func (b *baseList[T]) SetBackground(widget guigui.Widget) {
	b.content.SetBackground(widget)
}

func (b *baseList[T]) SetOnItemSelected(f func(context *guigui.Context, index int)) {
	b.content.SetOnItemSelected(f)
}

func (b *baseList[T]) SetOnItemsMoved(f func(context *guigui.Context, from, count, to int)) {
	b.content.SetOnItemsMoved(f)
}

func (b *baseList[T]) SetOnItemExpanderToggled(f func(context *guigui.Context, index int, expanded bool)) {
	b.content.SetOnItemExpanderToggled(f)
}

func (b *baseList[T]) SetCheckmarkIndex(index int) {
	b.content.SetCheckmarkIndex(index)
}

func (b *baseList[T]) SetHeaderHeight(height int) {
	if b.headerHeight == height {
		return
	}
	b.headerHeight = height
	b.frame.SetHeaderHeight(height)
	guigui.RequestRedraw(b)
}

func (b *baseList[T]) SetFooterHeight(height int) {
	if b.footerHeight == height {
		return
	}
	b.footerHeight = height
	b.frame.SetFooterHeight(height)
	guigui.RequestRedraw(b)
}

func (b *baseList[T]) SetContentWidth(width int) {
	b.content.SetContentWidth(width)
}

func (b *baseList[T]) Style() ListStyle {
	return b.content.Style()
}

func (b *baseList[T]) SetStyle(style ListStyle) {
	b.content.SetStyle(style)
	b.frame.SetStyle(style)
}

func (b *baseList[T]) ScrollOffset() (float64, float64) {
	return b.content.ScrollOffset()
}

func (b *baseList[T]) SetItems(items []baseListItem[T]) {
	b.content.SetItems(items)
}

func (b *baseList[T]) SelectedItemIndex() int {
	return b.content.SelectedItemIndex()
}

func (b *baseList[T]) SelectItemByIndex(index int) {
	b.content.SelectItemByIndex(index)
}

func (b *baseList[T]) SelectItemByValue(value T) {
	b.content.SelectItemByValue(value)
}

func (b *baseList[T]) JumpToItemByIndex(index int) {
	b.content.JumpToItemByIndex(index)
}

func (b *baseList[T]) EnsureItemVisibleByIndex(index int) {
	b.content.EnsureItemVisibleByIndex(index)
}

func (b *baseList[T]) SetStripeVisible(visible bool) {
	b.content.SetStripeVisible(visible)
}

func (b *baseList[T]) IsHoveringVisible() bool {
	return b.content.IsHoveringVisible()
}

func (b *baseList[T]) HoveredItemIndex() int {
	return b.content.hoveredItemIndexPlus1 - 1
}

func (b *baseList[T]) ItemBounds(index int) image.Rectangle {
	return b.content.ItemBounds(index)
}

func (b *baseList[T]) ItemYFromIndexForMenu(context *guigui.Context, index int) (int, bool) {
	return b.content.itemYFromIndexForMenu(context, index)
}

func (b *baseList[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&b.background1)
	adder.AddChild(&b.content)
	adder.AddChild(&b.frame)
	context.SetContainer(b, true)

	b.background1.setListContent(&b.content)
	return nil
}

func (b *baseList[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	bounds := widgetBounds.Bounds()
	bounds.Min.Y += b.headerHeight
	bounds.Max.Y -= b.footerHeight
	layouter.LayoutWidget(&b.background1, widgetBounds.Bounds())
	layouter.LayoutWidget(&b.content, bounds)
	layouter.LayoutWidget(&b.frame, widgetBounds.Bounds())
}

func (b *baseList[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return b.content.Measure(context, constraints)
}

type baseListContent[T comparable] struct {
	guigui.DefaultWidget

	customBackground guigui.Widget
	background2      baseListBackground2[T]
	checkmark        Image
	expanderImages   []Image
	scrollOverlay    scrollOverlay

	abstractList              abstractList[T, baseListItem[T]]
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
	contentHeight             int

	widgetBoundsForLayout        map[guigui.Widget]image.Rectangle
	itemBoundsForLayoutFromIndex []image.Rectangle

	treeItemCollapsedImage *ebiten.Image
	treeItemExpandedImage  *ebiten.Image

	prevWidth int

	onItemSelected func(index int)
}

func (b *baseListContent[T]) SetBackground(widget guigui.Widget) {
	b.customBackground = widget
}

func (b *baseListContent[T]) SetOnItemSelected(f func(context *guigui.Context, index int)) {
	guigui.RegisterEventHandler(b, baseListEventItemSelected, f)
}

func (b *baseListContent[T]) SetOnItemsMoved(f func(context *guigui.Context, from, count, to int)) {
	guigui.RegisterEventHandler(b, baseListEventItemsMoved, f)
}

func (b *baseListContent[T]) SetOnItemExpanderToggled(f func(context *guigui.Context, index int, expanded bool)) {
	guigui.RegisterEventHandler(b, baseListEventItemExpanderToggled, f)
}

func (b *baseListContent[T]) SetCheckmarkIndex(index int) {
	if index < 0 {
		index = -1
	}
	if b.checkmarkIndexPlus1 == index+1 {
		return
	}
	b.checkmarkIndexPlus1 = index + 1
	guigui.RequestRedraw(b)
}

func (b *baseListContent[T]) SetContentWidth(width int) {
	if b.contentWidthPlus1 == width+1 {
		return
	}
	b.contentWidthPlus1 = width + 1
	guigui.RequestRedraw(b)
}

func (b *baseListContent[T]) contentWidth(context *guigui.Context, widgetBounds *guigui.WidgetBounds) int {
	if b.contentWidthPlus1 > 0 {
		return b.contentWidthPlus1 - 1
	}
	return widgetBounds.Bounds().Dx()
}

func (b *baseListContent[T]) contentSize(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Point {
	w := b.contentWidth(context, widgetBounds)
	return image.Pt(w, b.contentHeight)
}

func (b *baseListContent[T]) ItemBounds(index int) image.Rectangle {
	return b.itemBoundsForLayoutFromIndex[index]
}

func (b *baseListContent[T]) visibleItems() iter.Seq2[int, baseListItem[T]] {
	return func(yield func(int, baseListItem[T]) bool) {
		var lastCollapsedIndentLevel int
		for i := range b.abstractList.ItemCount() {
			item, _ := b.abstractList.ItemByIndex(i)
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

func (b *baseListContent[T]) isItemVisible(index int) bool {
	item, ok := b.abstractList.ItemByIndex(index)
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
		item, ok := b.abstractList.ItemByIndex(index)
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

func (b *baseListContent[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if b.customBackground != nil {
		adder.AddChild(b.customBackground)
	}
	adder.AddChild(&b.background2)
	b.expanderImages = adjustSliceSize(b.expanderImages, b.abstractList.ItemCount())
	for i := range b.visibleItems() {
		item, _ := b.abstractList.ItemByIndex(i)
		if b.checkmarkIndexPlus1 == i+1 {
			adder.AddChild(&b.checkmark)
		}
		if item.IndentLevel > 0 {
			adder.AddChild(&b.expanderImages[i])
		}
		adder.AddChild(item.Content)
	}
	adder.AddChild(&b.scrollOverlay)

	if b.onItemSelected == nil {
		b.onItemSelected = func(index int) {
			guigui.DispatchEventHandler(b, baseListEventItemSelected, index)
		}
	}
	b.abstractList.SetOnItemSelected(b.onItemSelected)

	b.background2.setListContent(b)

	var err error
	b.treeItemCollapsedImage, err = theResourceImages.Get("keyboard_arrow_right", context.ColorMode())
	if err != nil {
		return err
	}
	b.treeItemExpandedImage, err = theResourceImages.Get("keyboard_arrow_down", context.ColorMode())
	if err != nil {
		return err
	}
	return nil
}

func (b *baseListContent[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	// Record the current position of the selected item.
	var headToSelectedItem int
	if idx := b.SelectedItemIndex(); idx >= 0 {
		if y0, ok := b.itemYFromIndex(context, idx); ok {
			_, offsetY := b.scrollOverlay.Offset()
			y := int(-offsetY)
			headToSelectedItem = y0 - y
			if headToSelectedItem < 0 || headToSelectedItem >= widgetBounds.Bounds().Dy() {
				headToSelectedItem = 0
			}
		}
	}

	cw := b.contentWidth(context, widgetBounds)

	p := widgetBounds.Bounds().Min
	offsetX, offsetY := b.scrollOverlay.Offset()
	p.X += RoundedCornerRadius(context) + int(offsetX)
	p.Y += RoundedCornerRadius(context) + int(offsetY)
	origY := p.Y
	clear(b.widgetBoundsForLayout)
	if b.widgetBoundsForLayout == nil {
		b.widgetBoundsForLayout = map[guigui.Widget]image.Rectangle{}
	}

	b.itemBoundsForLayoutFromIndex = adjustSliceSize(b.itemBoundsForLayoutFromIndex, b.abstractList.ItemCount())
	clear(b.itemBoundsForLayoutFromIndex)

	origP := p
	for i := range b.visibleItems() {
		item, _ := b.abstractList.ItemByIndex(i)
		itemW := cw - 2*RoundedCornerRadius(context)
		itemW -= listItemIndentSize(context, item.IndentLevel)
		itemW -= item.Padding.Start + item.Padding.End
		contentH := item.Content.Measure(context, guigui.FixedWidthConstraints(itemW)).Y

		if b.checkmarkIndexPlus1 == i+1 {
			imgSize := listItemCheckmarkSize(context)
			imgP := p
			imgP.X += listItemIndentSize(context, item.IndentLevel)
			imgP.X += UnitSize(context) / 4
			itemH := contentH
			imgP.Y += (itemH - imgSize) / 2
			// Adjust the position a bit for better appearance.
			imgP.Y += UnitSize(context) / 16
			imgP.Y += item.Padding.Top
			imgP.Y = b.adjustItemY(context, imgP.Y)
			b.widgetBoundsForLayout[&b.checkmark] = image.Rectangle{
				Min: imgP,
				Max: imgP.Add(image.Pt(imgSize, imgSize)),
			}
		}

		if item.IndentLevel > 0 {
			var img *ebiten.Image
			var hasChild bool
			if nextItem, ok := b.abstractList.ItemByIndex(i + 1); ok {
				hasChild = nextItem.IndentLevel > item.IndentLevel
			}
			if hasChild {
				if item.Collapsed {
					img = b.treeItemCollapsedImage
				} else {
					img = b.treeItemExpandedImage
				}
			}
			b.expanderImages[i].SetImage(img)
			expanderP := p
			expanderP.X += listItemIndentSize(context, item.IndentLevel) - int(LineHeight(context))
			// Adjust the position a bit for better appearance.
			expanderP.Y += UnitSize(context) / 16
			expanderP.Y += item.Padding.Top
			s := image.Pt(
				int(LineHeight(context)),
				contentH,
			)
			b.widgetBoundsForLayout[&b.expanderImages[i]] = image.Rectangle{
				Min: expanderP,
				Max: expanderP.Add(s),
			}
		}

		itemP := p
		if b.checkmarkIndexPlus1 > 0 {
			itemP.X += listItemCheckmarkSize(context) + listItemTextAndImagePadding(context)
		}
		itemP.X += listItemIndentSize(context, item.IndentLevel)
		itemP.X += item.Padding.Start
		itemP.Y = b.adjustItemY(context, itemP.Y)
		itemP.Y += item.Padding.Top
		r := image.Rectangle{
			Min: itemP,
			Max: itemP.Add(image.Pt(itemW, contentH)),
		}
		b.widgetBoundsForLayout[item.Content] = r
		b.itemBoundsForLayoutFromIndex[i] = r

		p.Y += contentH + item.Padding.Top + item.Padding.Bottom
	}

	b.contentHeight = p.Y - origY + 2*RoundedCornerRadius(context)
	cs := image.Pt(cw, b.contentHeight)
	// TODO: Now scrollOverlay's widgetBounds doens't match with baseList's widgetBounds.
	// Separate a content part and use Panel.
	b.scrollOverlay.SetContentSize(context, widgetBounds, cs)

	// Adjust the scroll offset to show the selected item if needed.
	if b.prevWidth != widgetBounds.Bounds().Dx() && headToSelectedItem != 0 {
		if y0, ok := b.itemYFromIndex(context, b.SelectedItemIndex()); ok {
			newOffsetY := -float64(y0 - headToSelectedItem)
			offsetX, _ := b.scrollOverlay.Offset()
			b.scrollOverlay.SetOffset(context, widgetBounds, cs, offsetX, newOffsetY)
		}
	}
	b.prevWidth = widgetBounds.Bounds().Dx()

	if b.customBackground != nil {
		layouter.LayoutWidget(b.customBackground, image.Rectangle{
			Min: origP,
			Max: origP.Add(cs).Sub(image.Pt(2*RoundedCornerRadius(context), 2*RoundedCornerRadius(context))),
		})
	}
	layouter.LayoutWidget(&b.background2, widgetBounds.Bounds())
	layouter.LayoutWidget(&b.scrollOverlay, widgetBounds.Bounds())
	for widget, bounds := range b.widgetBoundsForLayout {
		layouter.LayoutWidget(widget, bounds)
	}
}

func (b *baseListContent[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	// Measure is mainly for a menu list.
	var itemConstraints guigui.Constraints
	if fixedWidth, ok := constraints.FixedWidth(); ok {
		itemConstraints = guigui.FixedWidthConstraints(fixedWidth - 2*RoundedCornerRadius(context))
	}
	var size image.Point
	for i := range b.abstractList.ItemCount() {
		item, _ := b.abstractList.ItemByIndex(i)
		s := item.Content.Measure(context, itemConstraints)
		size.X = max(size.X, s.X+listItemIndentSize(context, item.IndentLevel))
		size.Y += s.Y
	}

	if b.checkmarkIndexPlus1 > 0 {
		size.X += listItemCheckmarkSize(context) + listItemTextAndImagePadding(context)
	}
	size.X += 2 * RoundedCornerRadius(context)
	size.Y += 2 * RoundedCornerRadius(context)
	return size
}

func (b *baseListContent[T]) hasMovableItems() bool {
	for i := range b.visibleItems() {
		item, ok := b.abstractList.ItemByIndex(i)
		if !ok {
			continue
		}
		if item.Movable {
			return true
		}
	}
	return false
}

func (b *baseListContent[T]) ItemByIndex(index int) (baseListItem[T], bool) {
	return b.abstractList.ItemByIndex(index)
}

func (b *baseListContent[T]) SelectedItemIndex() int {
	return b.abstractList.SelectedItemIndex()
}

func (b *baseListContent[T]) SetItems(items []baseListItem[T]) {
	b.abstractList.SetItems(b, items)
}

func (b *baseListContent[T]) SelectItemByIndex(index int) {
	b.selectItemByIndex(index, false)
}

func (b *baseListContent[T]) selectItemByIndex(index int, forceFireEvents bool) {
	if b.abstractList.SelectItemByIndex(b, index, forceFireEvents) {
		guigui.RequestRedraw(b)
	}
}

func (b *baseListContent[T]) SelectItemByValue(value T) {
	if b.abstractList.SelectItemByValue(b, value, false) {
		guigui.RequestRedraw(b)
	}
}

func (b *baseListContent[T]) JumpToItemByIndex(index int) {
	if index < 0 {
		return
	}
	b.indexToJumpPlus1 = index + 1
	b.indexToEnsureVisiblePlus1 = 0
	b.jumpTick = ebiten.Tick() + 1
}

func (b *baseListContent[T]) EnsureItemVisibleByIndex(index int) {
	if index < 0 {
		return
	}
	b.indexToEnsureVisiblePlus1 = index + 1
	b.indexToJumpPlus1 = 0
	b.jumpTick = ebiten.Tick() + 1
}

func (b *baseListContent[T]) SetStripeVisible(visible bool) {
	if b.stripeVisible == visible {
		return
	}
	b.stripeVisible = visible
	guigui.RequestRedraw(b)
}

func (b *baseListContent[T]) IsHoveringVisible() bool {
	return b.style == ListStyleMenu
}

func (b *baseListContent[T]) Style() ListStyle {
	return b.style
}

func (b *baseListContent[T]) SetStyle(style ListStyle) {
	if b.style == style {
		return
	}
	b.style = style
	guigui.RequestRedraw(b)
}

func (b *baseListContent[T]) ScrollOffset() (float64, float64) {
	return b.scrollOverlay.Offset()
}

func (b *baseListContent[T]) calcDropDstIndex(context *guigui.Context) int {
	_, y := ebiten.CursorPosition()
	for i := range b.visibleItems() {
		if b := b.itemBounds(context, i); y < (b.Min.Y+b.Max.Y)/2 {
			return i
		}
	}
	return b.abstractList.ItemCount()
}

func (b *baseListContent[T]) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if r := b.scrollOverlay.handlePointingInput(context, widgetBounds); r != (guigui.HandleInputResult{}) {
		return r
	}

	b.hoveredItemIndexPlus1 = 0
	if widgetBounds.IsHitAtCursor() {
		cp := image.Pt(ebiten.CursorPosition())
		listBounds := widgetBounds.Bounds()
		for i := range b.visibleItems() {
			bounds := b.itemBounds(context, i)
			bounds.Min.X = listBounds.Min.X
			bounds.Max.X = listBounds.Max.X
			hovered := cp.In(bounds)
			if hovered {
				b.hoveredItemIndexPlus1 = i + 1
			}
		}
	}

	colorMode := context.ColorMode()
	if b.hoveredItemIndexPlus1 == b.checkmarkIndexPlus1 {
		colorMode = guigui.ColorModeDark
	}
	checkImg, err := theResourceImages.Get("check", colorMode)
	if err != nil {
		panic(fmt.Sprintf("basicwidget: failed to get check image: %v", err))
	}
	b.checkmark.SetImage(checkImg)

	if b.IsHoveringVisible() || b.hasMovableItems() {
		if b.lastHoveredItemIndexPlus1 != b.hoveredItemIndexPlus1 {
			b.lastHoveredItemIndexPlus1 = b.hoveredItemIndexPlus1
			guigui.RequestRedraw(b)
		}
	}

	// Process dragging.
	if b.dragSrcIndexPlus1 > 0 {
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
			b.scrollOverlay.SetOffsetByDelta(context, widgetBounds, b.contentSize(context, widgetBounds), 0, dy)
			if i := b.calcDropDstIndex(context); b.dragDstIndexPlus1-1 != i {
				b.dragDstIndexPlus1 = i + 1
				guigui.RequestRedraw(b)
				return guigui.HandleInputByWidget(b)
			}
			return guigui.AbortHandlingInputByWidget(b)
		}
		if b.dragDstIndexPlus1 > 0 {
			// TODO: Implement multiple items drop.
			guigui.DispatchEventHandler(b, baseListEventItemsMoved, b.dragSrcIndexPlus1-1, 1, b.dragDstIndexPlus1-1)
			b.dragDstIndexPlus1 = 0
		}
		b.dragSrcIndexPlus1 = 0
		guigui.RequestRedraw(b)
		return guigui.HandleInputByWidget(b)
	}

	if index := b.hoveredItemIndexPlus1 - 1; index >= 0 && index < b.abstractList.ItemCount() {
		c := image.Pt(ebiten.CursorPosition())

		left := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
		right := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
		switch {
		case (left || right) && b.scrollOverlay.isWidgetHitAtCursor(context, widgetBounds):
			item, _ := b.abstractList.ItemByIndex(index)
			if c.X < b.itemBoundsForLayoutFromIndex[index].Min.X {
				if left {
					expanded := !item.Collapsed
					guigui.DispatchEventHandler(b, baseListEventItemExpanderToggled, index, !expanded)
				}
				return guigui.AbortHandlingInputByWidget(b)
			}
			if !item.Selectable {
				return guigui.AbortHandlingInputByWidget(b)
			}

			wasFocused := context.IsFocusedOrHasFocusedChild(b)
			// A popup menu should not take a focus.
			// For example, a context menu for a text field should not take a focus from the text field.
			// TODO: It might be better to distinguish a menu and a popup menu in the future.
			if b.style != ListStyleMenu {
				if item, ok := b.abstractList.ItemByIndex(index); ok {
					context.SetFocused(item.Content, true)
				} else {
					context.SetFocused(b, true)
				}
			}
			if b.SelectedItemIndex() != index || !wasFocused || b.style == ListStyleMenu {
				b.selectItemByIndex(index, true)
			}
			b.pressStartPlus1 = c.Add(image.Pt(1, 1))
			b.startPressingIndexPlus1 = index + 1
			if left {
				return guigui.HandleInputByWidget(b)
			}
			// For the right click, give a chance to a parent widget to handle the right click e.g. to open a context menu.
			// TODO: This behavior seems a little ad-hoc. Consider a better way.
			return guigui.HandleInputResult{}

		case ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && b.scrollOverlay.isWidgetHitAtCursor(context, widgetBounds):
			item, _ := b.abstractList.ItemByIndex(index)
			if item.Movable && b.SelectedItemIndex() == index && b.startPressingIndexPlus1-1 == index && (b.pressStartPlus1 != c.Add(image.Pt(1, 1))) {
				b.dragSrcIndexPlus1 = index + 1
				return guigui.HandleInputByWidget(b)
			}
			return guigui.AbortHandlingInputByWidget(b)

		case inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft):
			b.pressStartPlus1 = image.Point{}
			b.startPressingIndexPlus1 = 0
			return guigui.AbortHandlingInputByWidget(b)
		}
	}

	b.dragSrcIndexPlus1 = 0
	b.pressStartPlus1 = image.Point{}
	return guigui.HandleInputResult{}
}

func (b *baseListContent[T]) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	// Jump to the item if requested.
	// This is done in Tick to wait for the list items are updated, or an item cannot be measured correctly.
	if b.jumpTick > 0 && ebiten.Tick() >= b.jumpTick {
		cs := b.contentSize(context, widgetBounds)
		if idx := b.indexToJumpPlus1 - 1; idx >= 0 && idx < b.abstractList.ItemCount() {
			if y, ok := b.itemYFromIndex(context, idx); ok {
				y -= RoundedCornerRadius(context)
				b.scrollOverlay.SetOffset(context, widgetBounds, cs, 0, float64(-y))
			}
			b.indexToJumpPlus1 = 0
		}
		if idx := b.indexToEnsureVisiblePlus1 - 1; idx >= 0 && idx < b.abstractList.ItemCount() {
			if y, ok := b.itemYFromIndex(context, idx+1); ok {
				y -= widgetBounds.Bounds().Dy()
				y += RoundedCornerRadius(context)
				if offsetX, offsetY := b.scrollOverlay.Offset(); float64(y) > -offsetY {
					b.scrollOverlay.SetOffset(context, widgetBounds, cs, offsetX, float64(-y))
				}
			}
			if y, ok := b.itemYFromIndex(context, idx); ok {
				y -= RoundedCornerRadius(context)
				if offsetX, offsetY := b.scrollOverlay.Offset(); float64(y) < -offsetY {
					b.scrollOverlay.SetOffset(context, widgetBounds, cs, offsetX, float64(-y))
				}
			}
			b.indexToEnsureVisiblePlus1 = 0
		}
		b.jumpTick = 0
	}
	return nil
}

// itemYFromIndex returns the Y position of the item at the given index relative to the top of the baseList widget.
// itemYFromIndex returns the same value whatever the baseList position is.
//
// itemYFromIndex is available after Build is called, so do not use this from a parent widget.
func (b *baseListContent[T]) itemYFromIndex(context *guigui.Context, index int) (int, bool) {
	if index < 0 || index > len(b.itemBoundsForLayoutFromIndex) || len(b.itemBoundsForLayoutFromIndex) == 0 {
		return 0, false
	}

	baseY := b.itemBoundsForLayoutFromIndex[0].Min.Y
	var itemRelY int
	if index == len(b.itemBoundsForLayoutFromIndex) {
		itemRelY = b.itemBoundsForLayoutFromIndex[index-1].Max.Y - baseY
	} else {
		itemRelY = b.itemBoundsForLayoutFromIndex[index].Min.Y - baseY
	}
	head := RoundedCornerRadius(context)
	var padding guigui.Padding
	if item, ok := b.abstractList.ItemByIndex(index); ok {
		padding = item.Padding
	}
	return itemRelY + head - padding.Top, true
}

// itemYFromIndexForMenu returns the Y position of the item at the given index relative to the top of the baseList widget.
// itemYFromIndexForMenu returns the same value whatever the baseList position is.
//
// itemYFromIndexForMenu is available anytime even before Build is called.
func (b *baseListContent[T]) itemYFromIndexForMenu(context *guigui.Context, index int) (int, bool) {
	y := RoundedCornerRadius(context)
	for i := range b.visibleItems() {
		if i == index {
			return y, true
		}
		if i > index {
			break
		}
		item, _ := b.abstractList.ItemByIndex(i)
		// Use a free constraints to measure the item height for menu.
		y += item.Content.Measure(context, guigui.Constraints{}).Y
	}

	return 0, false
}

func (b *baseListContent[T]) adjustItemY(context *guigui.Context, y int) int {
	// Adjust the bounds based on the list style (inset or outset).
	switch b.style {
	case ListStyleNormal:
		y += int(0.5 * context.Scale())
	case ListStyleMenu:
		y += int(-0.5 * context.Scale())
	}
	return y
}

func (b *baseListContent[T]) itemBounds(context *guigui.Context, index int) image.Rectangle {
	if index < 0 || index >= len(b.itemBoundsForLayoutFromIndex) {
		return image.Rectangle{}
	}
	r := b.itemBoundsForLayoutFromIndex[index]
	if b.checkmarkIndexPlus1 > 0 {
		r.Min.X -= listItemCheckmarkSize(context) + listItemTextAndImagePadding(context)
	}
	return r
}

func (b *baseListContent[T]) selectedItemColor(context *guigui.Context) color.Color {
	if b.SelectedItemIndex() < 0 || b.SelectedItemIndex() >= b.abstractList.ItemCount() {
		return nil
	}
	if b.style == ListStyleMenu {
		return nil
	}
	if context.IsFocusedOrHasFocusedChild(b) || b.style == ListStyleSidebar {
		return draw.Color(context.ColorMode(), draw.ColorTypeAccent, 0.5)
	}
	if !context.IsEnabled(b) {
		return draw.Color2(context.ColorMode(), draw.ColorTypeBase, 0.7, 0.2)
	}
	return draw.Color2(context.ColorMode(), draw.ColorTypeBase, 0.7, 0.5)
}

type baseListBackground1[T comparable] struct {
	guigui.DefaultWidget

	content *baseListContent[T]
}

func (b *baseListBackground1[T]) setListContent(content *baseListContent[T]) {
	b.content = content
}

func (b *baseListBackground1[T]) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	var clr color.Color
	switch b.content.style {
	case ListStyleSidebar:
	case ListStyleNormal:
		clr = basicwidgetdraw.ControlColor(context.ColorMode(), context.IsEnabled(b))
	case ListStyleMenu:
		clr = basicwidgetdraw.SecondaryControlColor(context.ColorMode(), context.IsEnabled(b))
	}
	if clr != nil {
		bounds := widgetBounds.Bounds()
		basicwidgetdraw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
	}

	if b.content.stripeVisible && b.content.abstractList.ItemCount() > 0 {
		vb := widgetBounds.VisibleBounds()
		// Draw item stripes.
		// TODO: Get indices of items that are visible.
		var count int
		for i := range b.content.visibleItems() {
			count++
			if count%2 == 1 {
				continue
			}
			bounds := b.content.itemBounds(context, i)
			// Reset the X position to ignore indentation.
			item, _ := b.content.abstractList.ItemByIndex(i)
			bounds.Min.X -= listItemIndentSize(context, item.IndentLevel)
			if bounds.Min.Y > vb.Max.Y {
				break
			}
			if !bounds.Overlaps(vb) {
				continue
			}
			clr := basicwidgetdraw.SecondaryControlColor(context.ColorMode(), context.IsEnabled(b))
			basicwidgetdraw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
		}
	}
}

type baseListBackground2[T comparable] struct {
	guigui.DefaultWidget

	content *baseListContent[T]
}

func (b *baseListBackground2[T]) setListContent(content *baseListContent[T]) {
	b.content = content
}

func (b *baseListBackground2[T]) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	vb := widgetBounds.VisibleBounds()

	// Draw the selected item background.
	if clr := b.content.selectedItemColor(context); clr != nil && b.content.SelectedItemIndex() >= 0 && b.content.SelectedItemIndex() < b.content.abstractList.ItemCount() && b.content.isItemVisible(b.content.SelectedItemIndex()) {
		bounds := b.content.itemBounds(context, b.content.SelectedItemIndex())
		if b.content.style == ListStyleMenu {
			bounds.Max.X = bounds.Min.X + widgetBounds.Bounds().Dx() - 2*RoundedCornerRadius(context)
		}
		if bounds.Overlaps(vb) {
			basicwidgetdraw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
		}
	}

	hoveredItemIndex := b.content.hoveredItemIndexPlus1 - 1
	hoveredItem, ok := b.content.abstractList.ItemByIndex(hoveredItemIndex)
	if ok && b.content.IsHoveringVisible() && hoveredItemIndex >= 0 && hoveredItemIndex < b.content.abstractList.ItemCount() && hoveredItem.Selectable && b.content.isItemVisible(hoveredItemIndex) {
		bounds := b.content.itemBounds(context, hoveredItemIndex)
		if b.content.style == ListStyleMenu {
			bounds.Max.X = bounds.Min.X + widgetBounds.Bounds().Dx() - 2*RoundedCornerRadius(context)
		}
		if bounds.Overlaps(vb) {
			clr := draw.Color(context.ColorMode(), draw.ColorTypeBase, 0.9)
			if b.content.style == ListStyleMenu {
				clr = draw.Color(context.ColorMode(), draw.ColorTypeAccent, 0.5)
			}
			basicwidgetdraw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
		}
	}

	// Draw a drag indicator.
	if context.IsEnabled(b) && b.content.dragSrcIndexPlus1 == 0 {
		if item, ok := b.content.abstractList.ItemByIndex(hoveredItemIndex); ok && item.Movable {
			img, err := theResourceImages.Get("drag_indicator", context.ColorMode())
			if err != nil {
				panic(fmt.Sprintf("basicwidget: failed to get drag indicator image: %v", err))
			}
			op := &ebiten.DrawImageOptions{}
			s := float64(2*RoundedCornerRadius(context)) / float64(img.Bounds().Dy())
			op.GeoM.Scale(s, s)
			bounds := b.content.itemBounds(context, hoveredItemIndex)
			p := bounds.Min
			p.X = widgetBounds.Bounds().Min.X
			op.GeoM.Translate(float64(p.X), float64(p.Y)+(float64(bounds.Dy())-float64(img.Bounds().Dy())*s)/2)
			op.ColorScale.ScaleAlpha(0.5)
			op.Filter = ebiten.FilterLinear
			dst.DrawImage(img, op)
		}
	}

	// Draw a dragging guideline.
	if b.content.dragDstIndexPlus1 > 0 {
		p := widgetBounds.Bounds().Min
		offsetX, _ := b.content.scrollOverlay.Offset()
		x0 := float32(p.X) + float32(RoundedCornerRadius(context))
		x0 += float32(offsetX)
		x1 := x0 + float32(b.content.contentSize(context, widgetBounds).X)
		x1 -= 2 * float32(RoundedCornerRadius(context))
		y := float32(p.Y)
		if itemY, ok := b.content.itemYFromIndex(context, b.content.dragDstIndexPlus1-1); ok {
			y += float32(itemY)
			_, offsetY := b.content.scrollOverlay.Offset()
			y += float32(offsetY)
			vector.StrokeLine(dst, x0, y, x1, y, 2*float32(context.Scale()), draw.Color(context.ColorMode(), draw.ColorTypeAccent, 0.5), false)
		}
	}
}

type baseListFrame struct {
	guigui.DefaultWidget

	headerHeight int
	footerHeight int
	style        ListStyle
}

func (b *baseListFrame) SetHeaderHeight(height int) {
	if b.headerHeight == height {
		return
	}
	b.headerHeight = height
	guigui.RequestRedraw(b)
}

func (b *baseListFrame) SetFooterHeight(height int) {
	if b.footerHeight == height {
		return
	}
	b.footerHeight = height
	guigui.RequestRedraw(b)
}

func (b *baseListFrame) SetStyle(style ListStyle) {
	if b.style == style {
		return
	}
	b.style = style
	guigui.RequestRedraw(b)
}

func (b *baseListFrame) headerBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	bounds := widgetBounds.Bounds()
	bounds.Max.Y = bounds.Min.Y + b.headerHeight
	return bounds
}

func (b *baseListFrame) footerBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	bounds := widgetBounds.Bounds()
	bounds.Min.Y = bounds.Max.Y - b.footerHeight
	return bounds
}

func (b *baseListFrame) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if b.style == ListStyleSidebar || b.style == ListStyleMenu {
		return
	}

	// Draw a header.
	if b.headerHeight > 0 {
		bounds := b.headerBounds(context, widgetBounds)
		basicwidgetdraw.DrawRoundedRectWithSharpCorners(context, dst, bounds, basicwidgetdraw.ControlColor(context.ColorMode(), context.IsEnabled(b)), RoundedCornerRadius(context), basicwidgetdraw.Corners{
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
		if !context.IsEnabled(b) {
			clr = draw.Color2(context.ColorMode(), draw.ColorTypeBase, 0.8, 0.3)
		}
		vector.StrokeLine(dst, x0, y0, x1, y1, float32(context.Scale()), clr, false)
	}

	// Draw a footer.
	if b.footerHeight > 0 {
		bounds := b.footerBounds(context, widgetBounds)
		basicwidgetdraw.DrawRoundedRectWithSharpCorners(context, dst, bounds, basicwidgetdraw.ControlColor(context.ColorMode(), context.IsEnabled(b)), RoundedCornerRadius(context), basicwidgetdraw.Corners{
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
		if !context.IsEnabled(b) {
			clr = draw.Color2(context.ColorMode(), draw.ColorTypeBase, 0.8, 0.3)
		}
		vector.StrokeLine(dst, x0, y0, x1, y1, float32(context.Scale()), clr, false)
	}

	bounds := widgetBounds.Bounds()
	border := basicwidgetdraw.RoundedRectBorderTypeInset
	if b.style != ListStyleNormal {
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
