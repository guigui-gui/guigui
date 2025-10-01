// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget/internal/draw"
)

const (
	baseListEventItemsMoved = "itemsMoved"
)

type ListStyle int

const (
	ListStyleNormal ListStyle = iota
	ListStyleSidebar
	ListStyleMenu
)

type baseListItem[T comparable] struct {
	Content    guigui.Widget
	Selectable bool
	Movable    bool
	Value      T
}

func (b baseListItem[T]) value() T {
	return b.Value
}

func DefaultActiveListItemTextColor(context *guigui.Context) color.Color {
	return draw.Color2(context.ColorMode(), draw.ColorTypeBase, 1, 1)
}

type baseList[T comparable] struct {
	guigui.DefaultWidget

	checkmark     Image
	listFrame     listFrame[T]
	scrollOverlay ScrollOverlay

	abstractList               abstractList[T, baseListItem[T]]
	stripeVisible              bool
	style                      ListStyle
	checkmarkIndexPlus1        int
	lastHoverredItemIndexPlus1 int

	indexToJumpPlus1        int
	dragSrcIndexPlus1       int
	dragDstIndexPlus1       int
	pressStartPlus1         image.Point
	startPressingIndexPlus1 int
	headerHeight            int
	footerHeight            int
	contentWidthPlus1       int
	contentHeight           int

	cachedDefaultWidth         int
	cachedDefaultContentHeight int

	itemBoundsForLayoutFromWidget map[guigui.Widget]image.Rectangle
	itemBoundsForLayoutFromIndex  []image.Rectangle
}

func listItemPadding(context *guigui.Context) int {
	return RoundedCornerRadius(context) + UnitSize(context)/4
}

func (b *baseList[T]) SetOnItemSelected(f func(index int)) {
	b.abstractList.SetOnItemSelected(b, f)
}

func (b *baseList[T]) SetOnItemsMoved(f func(from, count, to int)) {
	guigui.RegisterEventHandler(b, baseListEventItemsMoved, f)
}

func (b *baseList[T]) SetCheckmarkIndex(index int) {
	if index < 0 {
		index = -1
	}
	if b.checkmarkIndexPlus1 == index+1 {
		return
	}
	b.checkmarkIndexPlus1 = index + 1
	guigui.RequestRedraw(b)
}

func (b *baseList[T]) SetHeaderHeight(height int) {
	if b.headerHeight == height {
		return
	}
	b.headerHeight = height
	guigui.RequestRedraw(b)
}

func (b *baseList[T]) SetFooterHeight(height int) {
	if b.footerHeight == height {
		return
	}
	b.footerHeight = height
	guigui.RequestRedraw(b)
}

func (b *baseList[T]) SetContentWidth(width int) {
	if b.contentWidthPlus1 == width+1 {
		return
	}
	b.contentWidthPlus1 = width + 1
	guigui.RequestRedraw(b)
}

func (b *baseList[T]) contentWidth(context *guigui.Context) int {
	if b.contentWidthPlus1 > 0 {
		return b.contentWidthPlus1 - 1
	}
	return context.Bounds(b).Dx()
}

func (b *baseList[T]) contentSize(context *guigui.Context) image.Point {
	w := b.contentWidth(context)
	return image.Pt(w, b.contentHeight)
}

func (b *baseList[T]) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	for i := range b.abstractList.ItemCount() {
		item, _ := b.abstractList.ItemByIndex(i)
		if b.checkmarkIndexPlus1 == i+1 {
			adder.AddChild(&b.checkmark)
		}
		adder.AddChild(item.Content)
	}
	if b.style != ListStyleSidebar && b.style != ListStyleMenu {
		adder.AddChild(&b.listFrame)
	}
	adder.AddChild(&b.scrollOverlay)
}

func (b *baseList[T]) Update(context *guigui.Context) error {
	cw := b.contentWidth(context)

	// TODO: Do not call HoveredItemIndex in Build (#52).
	hoveredItemIndex := b.hoveredItemIndex(context)
	p := context.Bounds(b).Min
	offsetX, offsetY := b.scrollOverlay.Offset()
	p.X += listItemPadding(context) + int(offsetX)
	p.Y += RoundedCornerRadius(context) + b.headerHeight + int(offsetY)
	origY := p.Y
	clear(b.itemBoundsForLayoutFromWidget)
	if b.itemBoundsForLayoutFromWidget == nil {
		b.itemBoundsForLayoutFromWidget = map[guigui.Widget]image.Rectangle{}
	}
	b.itemBoundsForLayoutFromIndex = adjustSliceSize(b.itemBoundsForLayoutFromIndex, b.abstractList.ItemCount())
	for i := range b.abstractList.ItemCount() {
		item, _ := b.abstractList.ItemByIndex(i)
		itemW := cw - 2*listItemPadding(context)

		if b.checkmarkIndexPlus1 == i+1 {
			mode := context.ColorMode()
			if b.checkmarkIndexPlus1 == hoveredItemIndex+1 {
				mode = guigui.ColorModeDark
			}
			img, err := theResourceImages.Get("check", mode)
			if err != nil {
				return err
			}
			b.checkmark.SetImage(img)

			imgSize := listItemCheckmarkSize(context)
			imgP := p
			itemH := item.Content.Measure(context, guigui.FixedWidthConstraints(itemW)).Y
			imgP.Y += (itemH - imgSize) * 3 / 4
			imgP.Y = b.adjustItemY(context, imgP.Y)
			b.itemBoundsForLayoutFromWidget[&b.checkmark] = image.Rectangle{
				Min: imgP,
				Max: imgP.Add(image.Pt(imgSize, imgSize)),
			}
		}

		itemP := p
		if b.checkmarkIndexPlus1 > 0 {
			itemP.X += listItemCheckmarkSize(context) + listItemTextAndImagePadding(context)
		}
		itemP.Y = b.adjustItemY(context, itemP.Y)
		s := item.Content.Measure(context, guigui.FixedWidthConstraints(itemW))
		r := image.Rectangle{
			Min: itemP,
			Max: itemP.Add(s),
		}
		b.itemBoundsForLayoutFromWidget[item.Content] = r
		b.itemBoundsForLayoutFromIndex[i] = r

		p.Y += s.Y
	}

	if b.style != ListStyleSidebar && b.style != ListStyleMenu {
		b.listFrame.list = b
	}

	b.contentHeight = p.Y - origY + 2*RoundedCornerRadius(context)
	cs := image.Pt(cw, b.contentHeight)
	b.scrollOverlay.SetContentSize(context, cs)

	if idx := b.indexToJumpPlus1 - 1; idx >= 0 {
		y := b.itemYFromIndex(context, idx) - b.headerHeight - RoundedCornerRadius(context)
		b.scrollOverlay.SetOffset(context, cs, 0, float64(-y))
		b.indexToJumpPlus1 = 0
	}

	return nil
}

func (b *baseList[T]) Layout(context *guigui.Context, widget guigui.Widget) image.Rectangle {
	switch widget {
	case &b.listFrame:
		return context.Bounds(b)
	case &b.scrollOverlay:
		bounds := context.Bounds(b)
		bounds.Min.Y += b.headerHeight
		bounds.Max.Y -= b.footerHeight
		return bounds
	}
	if r, ok := b.itemBoundsForLayoutFromWidget[widget]; ok {
		return r
	}
	return image.Rectangle{}
}

func (b *baseList[T]) hasMovableItems() bool {
	for i := range b.abstractList.ItemCount() {
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

func (b *baseList[T]) ItemByIndex(index int) (baseListItem[T], bool) {
	return b.abstractList.ItemByIndex(index)
}

func (b *baseList[T]) SelectedItemIndex() int {
	return b.abstractList.SelectedItemIndex()
}

func (b *baseList[T]) hoveredItemIndex(context *guigui.Context) int {
	if !context.IsWidgetHitAtCursor(b) {
		return -1
	}
	_, y := ebiten.CursorPosition()
	_, offsetY := b.scrollOverlay.Offset()
	y -= RoundedCornerRadius(context) + b.headerHeight
	y -= context.Bounds(b).Min.Y
	y -= int(offsetY)
	index := -1
	var cy int
	for i := range b.abstractList.ItemCount() {
		item, _ := b.abstractList.ItemByIndex(i)
		h := context.Bounds(item.Content).Dy()
		if cy <= y && y < cy+h {
			index = i
			break
		}
		cy += h
	}
	return index
}

func (b *baseList[T]) SetItems(items []baseListItem[T]) {
	b.abstractList.SetItems(items)
	b.cachedDefaultWidth = 0
	b.cachedDefaultContentHeight = 0
}

func (b *baseList[T]) SelectItemByIndex(index int) {
	b.selectItemByIndex(index, false)
}

func (b *baseList[T]) selectItemByIndex(index int, forceFireEvents bool) {
	if b.abstractList.SelectItemByIndex(b, index, forceFireEvents) {
		guigui.RequestRedraw(b)
	}
}

func (b *baseList[T]) SelectItemByValue(value T) {
	if b.abstractList.SelectItemByValue(b, value, false) {
		guigui.RequestRedraw(b)
	}
}

func (b *baseList[T]) JumpToItemIndex(index int) {
	if index < 0 || index >= b.abstractList.ItemCount() {
		return
	}
	b.indexToJumpPlus1 = index + 1
}

func (b *baseList[T]) SetStripeVisible(visible bool) {
	if b.stripeVisible == visible {
		return
	}
	b.stripeVisible = visible
	guigui.RequestRedraw(b)
}

func (b *baseList[T]) isHoveringVisible() bool {
	return b.style == ListStyleMenu
}

func (b *baseList[T]) Style() ListStyle {
	return b.style
}

func (b *baseList[T]) SetStyle(style ListStyle) {
	if b.style == style {
		return
	}
	b.style = style
	guigui.RequestRedraw(b)
}

func (b *baseList[T]) ScrollOffset() (float64, float64) {
	return b.scrollOverlay.Offset()
}

func (b *baseList[T]) calcDropDstIndex(context *guigui.Context) int {
	_, y := ebiten.CursorPosition()
	for i := range b.abstractList.ItemCount() {
		if b := b.itemBounds(context, i); y < (b.Min.Y+b.Max.Y)/2 {
			return i
		}
	}
	return b.abstractList.ItemCount()
}

func (b *baseList[T]) HandlePointingInput(context *guigui.Context) guigui.HandleInputResult {
	if b.isHoveringVisible() || b.hasMovableItems() {
		if hoveredItemIndex := b.hoveredItemIndex(context); b.lastHoverredItemIndexPlus1 != hoveredItemIndex+1 {
			b.lastHoverredItemIndexPlus1 = hoveredItemIndex + 1
			guigui.RequestRedraw(b)
		}
	}

	// Process dragging.
	if b.dragSrcIndexPlus1 > 0 {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			_, y := ebiten.CursorPosition()
			p := context.Bounds(b).Min
			h := context.Bounds(b).Dy() - (b.headerHeight + b.footerHeight)
			var dy float64
			if upperY := p.Y + UnitSize(context); y < upperY {
				dy = float64(upperY-y) / 4
			}
			if lowerY := p.Y + h - UnitSize(context); y >= lowerY {
				dy = float64(lowerY-y) / 4
			}
			b.scrollOverlay.SetOffsetByDelta(context, b.contentSize(context), 0, dy)
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

	index := b.hoveredItemIndex(context)
	if index >= 0 && index < b.abstractList.ItemCount() {
		c := image.Pt(ebiten.CursorPosition())

		switch {
		case (inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)) &&
			context.IsWidgetHitAtCursor(b):
			item, _ := b.abstractList.ItemByIndex(index)
			if !item.Selectable {
				return guigui.AbortHandlingInputByWidget(b)
			}

			wasFocused := context.IsFocusedOrHasFocusedChild(b)
			if item, ok := b.abstractList.ItemByIndex(index); ok {
				context.SetFocused(item.Content, true)
			} else {
				context.SetFocused(b, true)
			}
			if b.SelectedItemIndex() != index || !wasFocused || b.style == ListStyleMenu {
				b.selectItemByIndex(index, true)
			}
			b.pressStartPlus1 = c.Add(image.Pt(1, 1))
			b.startPressingIndexPlus1 = index + 1
			return guigui.HandleInputByWidget(b)

		case ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft):
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

	if context.IsWidgetHitAtCursor(b) {
		return guigui.HandleInputResult{}
	}

	b.dragSrcIndexPlus1 = 0
	b.pressStartPlus1 = image.Point{}

	return guigui.HandleInputResult{}
}

func (b *baseList[T]) itemYFromIndex(context *guigui.Context, index int) int {
	y := RoundedCornerRadius(context) + b.headerHeight
	for i := range b.abstractList.ItemCount() {
		if i == index {
			break
		}
		item, _ := b.abstractList.ItemByIndex(i)
		y += context.Bounds(item.Content).Dy()
	}
	y = b.adjustItemY(context, y)
	return y
}

func (b *baseList[T]) adjustItemY(context *guigui.Context, y int) int {
	// Adjust the bounds based on the list style (inset or outset).
	switch b.style {
	case ListStyleNormal:
		y += int(0.5 * context.Scale())
	case ListStyleMenu:
		y += int(-0.5 * context.Scale())
	}
	return y
}

func (b *baseList[T]) itemBounds(context *guigui.Context, index int) image.Rectangle {
	if index < 0 || index >= len(b.itemBoundsForLayoutFromIndex) {
		return image.Rectangle{}
	}
	r := b.itemBoundsForLayoutFromIndex[index]
	if b.checkmarkIndexPlus1 > 0 {
		r.Min.X -= listItemCheckmarkSize(context) + listItemTextAndImagePadding(context)
	}
	return r
}

func (b *baseList[T]) selectedItemColor(context *guigui.Context) color.Color {
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

func (b *baseList[T]) Draw(context *guigui.Context, dst *ebiten.Image) {
	var clr color.Color
	switch b.style {
	case ListStyleSidebar:
	case ListStyleNormal:
		clr = draw.ControlColor(context.ColorMode(), context.IsEnabled(b))
	case ListStyleMenu:
		clr = draw.SecondaryControlColor(context.ColorMode(), context.IsEnabled(b))
	}
	if clr != nil {
		bounds := context.Bounds(b)
		draw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
	}

	vb := context.VisibleBounds(b)

	if b.stripeVisible && b.abstractList.ItemCount() > 0 {
		// Draw item stripes.
		// TODO: Get indices of items that are visible.
		for i := range b.abstractList.ItemCount() {
			if i%2 == 0 {
				continue
			}
			bounds := b.itemBounds(context, i)
			if bounds.Min.Y > vb.Max.Y {
				break
			}
			bounds.Min.X -= RoundedCornerRadius(context)
			bounds.Max.X += RoundedCornerRadius(context)
			if !bounds.Overlaps(vb) {
				continue
			}
			clr := draw.SecondaryControlColor(context.ColorMode(), context.IsEnabled(b))
			draw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
		}
	}

	// Draw the selected item background.
	if clr := b.selectedItemColor(context); clr != nil && b.SelectedItemIndex() >= 0 && b.SelectedItemIndex() < b.abstractList.ItemCount() {
		bounds := b.itemBounds(context, b.SelectedItemIndex())
		bounds.Min.X -= RoundedCornerRadius(context)
		bounds.Max.X += RoundedCornerRadius(context)
		if b.style == ListStyleMenu {
			bounds.Max.X = bounds.Min.X + context.Bounds(b).Dx() - 2*RoundedCornerRadius(context)
		}
		if bounds.Overlaps(vb) {
			draw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
		}
	}

	hoveredItemIndex := b.hoveredItemIndex(context)
	hoveredItem, ok := b.abstractList.ItemByIndex(hoveredItemIndex)
	if ok && b.isHoveringVisible() && hoveredItemIndex >= 0 && hoveredItemIndex < b.abstractList.ItemCount() && hoveredItem.Selectable {
		bounds := b.itemBounds(context, hoveredItemIndex)
		bounds.Min.X -= RoundedCornerRadius(context)
		bounds.Max.X += RoundedCornerRadius(context)
		if b.style == ListStyleMenu {
			bounds.Max.X = bounds.Min.X + context.Bounds(b).Dx() - 2*RoundedCornerRadius(context)
		}
		if bounds.Overlaps(vb) {
			clr := draw.Color(context.ColorMode(), draw.ColorTypeBase, 0.9)
			if b.style == ListStyleMenu {
				clr = draw.Color(context.ColorMode(), draw.ColorTypeAccent, 0.5)
			}
			draw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
		}
	}

	// Draw a drag indicator.
	if context.IsEnabled(b) && b.dragSrcIndexPlus1 == 0 {
		if item, ok := b.abstractList.ItemByIndex(hoveredItemIndex); ok && item.Movable {
			img, err := theResourceImages.Get("drag_indicator", context.ColorMode())
			if err != nil {
				panic(fmt.Sprintf("basicwidget: failed to get drag indicator image: %v", err))
			}
			op := &ebiten.DrawImageOptions{}
			s := float64(2*RoundedCornerRadius(context)) / float64(img.Bounds().Dy())
			op.GeoM.Scale(s, s)
			bounds := b.itemBounds(context, hoveredItemIndex)
			p := bounds.Min
			p.X = context.Bounds(b).Min.X + listItemPadding(context)
			op.GeoM.Translate(float64(p.X-2*RoundedCornerRadius(context)), float64(p.Y)+(float64(bounds.Dy())-float64(img.Bounds().Dy())*s)/2)
			op.ColorScale.ScaleAlpha(0.5)
			op.Filter = ebiten.FilterLinear
			dst.DrawImage(img, op)
		}
	}

	// Draw a dragging guideline.
	if b.dragDstIndexPlus1 > 0 {
		p := context.Bounds(b).Min
		offsetX, _ := b.scrollOverlay.Offset()
		x0 := float32(p.X) + float32(RoundedCornerRadius(context))
		x0 += float32(offsetX)
		x1 := x0 + float32(b.contentSize(context).X)
		x1 -= 2 * float32(RoundedCornerRadius(context))
		y := float32(p.Y)
		y += float32(b.itemYFromIndex(context, b.dragDstIndexPlus1-1))
		_, offsetY := b.scrollOverlay.Offset()
		y += float32(offsetY)
		vector.StrokeLine(dst, x0, y, x1, y, 2*float32(context.Scale()), draw.Color(context.ColorMode(), draw.ColorTypeAccent, 0.5), false)
	}
}

func (b *baseList[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	// Measure is mainly for a menu list.
	if len(b.itemBoundsForLayoutFromIndex) == 0 {
		return image.Point{}
	}
	var w int
	for _, r := range b.itemBoundsForLayoutFromIndex {
		w = max(w, r.Dx())
	}
	if b.checkmarkIndexPlus1 > 0 {
		w += listItemCheckmarkSize(context) + listItemTextAndImagePadding(context)
	}
	w += 2 * listItemPadding(context)

	b0 := b.itemBoundsForLayoutFromIndex[0]
	b1 := b.itemBoundsForLayoutFromIndex[len(b.itemBoundsForLayoutFromIndex)-1]
	h := b1.Max.Y - b0.Min.Y
	h += 2 * RoundedCornerRadius(context)
	return image.Pt(w, h)
}

type listFrame[T comparable] struct {
	guigui.DefaultWidget

	list *baseList[T]
}

func (l *listFrame[T]) headerBounds(context *guigui.Context) image.Rectangle {
	bounds := context.Bounds(l)
	bounds.Max.Y = bounds.Min.Y + l.list.headerHeight
	return bounds
}

func (l *listFrame[T]) footerBounds(context *guigui.Context) image.Rectangle {
	bounds := context.Bounds(l)
	bounds.Min.Y = bounds.Max.Y - l.list.footerHeight
	return bounds
}

func (l *listFrame[T]) Draw(context *guigui.Context, dst *ebiten.Image) {
	// Draw a header.
	if l.list.headerHeight > 0 {
		bounds := l.headerBounds(context)
		draw.DrawRoundedRectWithSharpenCorners(context, dst, bounds, draw.ControlColor(context.ColorMode(), context.IsEnabled(l)), RoundedCornerRadius(context), draw.SharpenCorners{
			UpperStart: false,
			UpperEnd:   false,
			LowerStart: true,
			LowerEnd:   true,
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
	if l.list.footerHeight > 0 {
		bounds := l.footerBounds(context)
		draw.DrawRoundedRectWithSharpenCorners(context, dst, bounds, draw.ControlColor(context.ColorMode(), context.IsEnabled(l)), RoundedCornerRadius(context), draw.SharpenCorners{
			UpperStart: true,
			UpperEnd:   true,
			LowerStart: false,
			LowerEnd:   false,
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

	bounds := context.Bounds(l)
	border := draw.RoundedRectBorderTypeInset
	if l.list.style != ListStyleNormal {
		border = draw.RoundedRectBorderTypeOutset
	}
	clr1, clr2 := draw.BorderColors(context.ColorMode(), border, false)
	borderWidth := float32(1 * context.Scale())
	draw.DrawRoundedRectBorder(context, dst, bounds, clr1, clr2, RoundedCornerRadius(context), borderWidth, border)
}

func listItemCheckmarkSize(context *guigui.Context) int {
	return int(LineHeight(context) * 3 / 4)
}

func listItemTextAndImagePadding(context *guigui.Context) int {
	return UnitSize(context) / 8
}
