// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"
	"slices"

	"github.com/guigui-gui/guigui"
)

const (
	selectEventItemSelected = "itemSelected"
)

type SelectItem[T comparable] struct {
	Text         string
	TextColor    color.Color
	Header       bool
	Content      guigui.Widget
	Unselectable bool
	Border       bool
	Disabled     bool
	Value        T
}

type Select[T comparable] struct {
	guigui.DefaultWidget

	button        Button
	buttonContent selectButtonContent
	popupMenu     PopupMenu[T]

	items                 []SelectItem[T]
	popupMenuItems        []PopupMenuItem[T]
	popupMenuItemContents []selectItemContent

	indexAtOpen int

	onDown                  func()
	onPopupMenuItemSelected func(index int)
}

func (s *Select[T]) SetOnItemSelected(f func(index int)) {
	guigui.RegisterEventHandler(s, selectEventItemSelected, f)
}

func (s *Select[T]) updatePopupMenuitems() {
	s.popupMenuItems = adjustSliceSize(s.popupMenuItems, len(s.items))
	s.popupMenuItemContents = adjustSliceSize(s.popupMenuItemContents, len(s.items))
	for i, item := range s.items {
		pmItem := PopupMenuItem[T](item)
		if s.popupMenu.IsOpen() && pmItem.Content != nil {
			s.popupMenuItemContents[i].SetContent(pmItem.Content)
			pmItem.Content = &s.popupMenuItemContents[i]
		} else {
			s.popupMenuItemContents[i].SetContent(nil)
			pmItem.Content = nil
		}
		s.popupMenuItems[i] = pmItem
	}
	s.popupMenu.SetItems(s.popupMenuItems)
}

func (s *Select[T]) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&s.button)
	adder.AddChild(&s.popupMenu)
}

func (s *Select[T]) Update(context *guigui.Context) error {
	s.updatePopupMenuitems()
	if index := s.popupMenu.SelectedItemIndex(); index >= 0 {
		if content := s.items[index].Content; content != nil {
			if s.popupMenu.IsOpen() {
				s.buttonContent.SetContentSize(content.Measure(context, guigui.Constraints{}))
			} else {
				s.buttonContent.SetContent(content)
			}
			s.buttonContent.SetText("")
		} else {
			s.buttonContent.SetContent(nil)
			s.buttonContent.SetText(s.items[index].Text)
		}
	} else {
		s.buttonContent.SetContent(nil)
		s.buttonContent.SetText("")
	}
	s.button.SetContent(&s.buttonContent)

	if s.onDown == nil {
		s.onDown = func() {
			s.popupMenu.SetOpen(true)
			s.indexAtOpen = s.popupMenu.SelectedItemIndex()
		}
	}
	s.button.SetOnDown(s.onDown)
	s.button.setKeepPressed(s.popupMenu.IsOpen())
	s.button.SetIconAlign(IconAlignEnd)

	if s.onPopupMenuItemSelected == nil {
		s.onPopupMenuItemSelected = func(index int) {
			guigui.DispatchEventHandler(s, selectEventItemSelected, index)
		}
	}
	s.popupMenu.SetOnItemSelected(s.onPopupMenuItemSelected)
	s.popupMenu.SetCheckmarkIndex(s.indexAtOpen)

	return nil
}

func (s *Select[T]) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	p := widgetBounds.Bounds().Min
	layouter.LayoutWidget(&s.button, image.Rectangle{
		Min: p,
		Max: p.Add(s.button.Measure(context, guigui.Constraints{})),
	})

	p = widgetBounds.Bounds().Min
	p.X -= listItemCheckmarkSize(context) + listItemTextAndImagePadding(context)
	p.X = max(p.X, 0)
	// TODO: The item content in a button and a select might have different heights. Handle this case properly.
	if y, ok := s.popupMenu.itemYFromIndexForMenu(context, max(0, s.popupMenu.SelectedItemIndex())); ok {
		p.Y -= y
	}
	p.Y = max(p.Y, 0)
	layouter.LayoutWidget(&s.popupMenu, image.Rectangle{
		Min: p,
		Max: p.Add(s.popupMenu.Measure(context, guigui.Constraints{})),
	})
}

func (s *Select[T]) SetItems(items []SelectItem[T]) {
	s.items = adjustSliceSize(s.items, len(items))
	copy(s.items, items)
	s.updatePopupMenuitems()
}

func (s *Select[T]) SetItemsByStrings(items []string) {
	s.items = adjustSliceSize(s.items, len(items))
	for i, str := range items {
		s.items[i] = SelectItem[T]{
			Text: str,
		}
	}
	s.updatePopupMenuitems()
}

func (s *Select[T]) SelectedItem() (SelectItem[T], bool) {
	item, ok := s.popupMenu.SelectedItem()
	if !ok {
		return SelectItem[T]{}, false
	}
	return SelectItem[T](item), true
}

func (s *Select[T]) ItemByIndex(index int) (SelectItem[T], bool) {
	item, ok := s.popupMenu.ItemByIndex(index)
	if !ok {
		return SelectItem[T]{}, false
	}
	return SelectItem[T](item), true
}

func (s *Select[T]) SelectedItemIndex() int {
	return s.popupMenu.SelectedItemIndex()
}

func (s *Select[T]) SelectItemByIndex(index int) {
	s.popupMenu.SelectItemByIndex(index)
}

func (s *Select[T]) SelectItemByValue(value T) {
	s.popupMenu.SelectItemByValue(value)
}

func (s *Select[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return s.button.Measure(context, constraints)
}

func (s *Select[T]) ItemTextColor(context *guigui.Context, index int) color.Color {
	return s.popupMenu.ItemTextColor(context, index)
}

func (s *Select[T]) IsPopupOpen() bool {
	return s.popupMenu.IsOpen()
}

type selectButtonContent struct {
	guigui.DefaultWidget

	content      guigui.Widget
	dummyContent guigui.WidgetWithSize[*guigui.DefaultWidget]

	contentSizePlus1 image.Point
	text             Text
	icon             Image

	layoutItems []guigui.LinearLayoutItem
}

func (s *selectButtonContent) SetContent(content guigui.Widget) {
	s.content = content
	s.contentSizePlus1 = image.Point{}
}

func (s *selectButtonContent) SetContentSize(size image.Point) {
	s.content = nil
	s.contentSizePlus1 = size.Add(image.Point{1, 1})
}

func (s *selectButtonContent) SetText(text string) {
	s.text.SetValue(text)
}

func (s *selectButtonContent) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	if s.content != nil {
		adder.AddChild(s.content)
	}
	adder.AddChild(&s.dummyContent)
	adder.AddChild(&s.text)
	adder.AddChild(&s.icon)
}

func (s *selectButtonContent) Update(context *guigui.Context) error {
	s.text.SetVerticalAlign(VerticalAlignMiddle)

	img, err := theResourceImages.Get("unfold_more", context.ColorMode())
	if err != nil {
		return err
	}
	s.icon.SetImage(img)
	return nil
}

func (s *selectButtonContent) layout(context *guigui.Context) guigui.LinearLayout {
	s.layoutItems = slices.Delete(s.layoutItems, 0, len(s.layoutItems))

	if s.contentSizePlus1.X != 0 || s.contentSizePlus1.Y != 0 {
		s.dummyContent.SetFixedSize(s.contentSizePlus1.Sub(image.Pt(1, 1)))
		s.layoutItems = append(s.layoutItems,
			guigui.LinearLayoutItem{
				Widget: &s.dummyContent,
			})
	} else if s.content != nil {
		s.layoutItems = append(s.layoutItems,
			guigui.LinearLayoutItem{
				Widget: s.content,
			})
	} else {
		s.layoutItems = append(s.layoutItems,
			guigui.LinearLayoutItem{
				Widget: &s.text,
			})
	}

	iconSize := defaultIconSize(context)
	s.layoutItems = append(s.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &s.icon,
			Size:   guigui.FixedSize(iconSize),
		})

	// Add paddings. Paddings are calculated as if the content is a text widget.
	// Even if the content is not a text widget, this padding should look good enough.
	padding := defaultButtonSize(context).Y - int(LineHeight(context))
	paddingTop := padding / 2
	paddingBottom := padding - paddingTop

	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Gap:       buttonTextAndImagePadding(context),
		Items:     s.layoutItems,
		Padding: guigui.Padding{
			Start:  buttonEdgeAndTextPadding(context),
			Top:    paddingTop,
			End:    buttonTextAndImagePadding(context),
			Bottom: paddingBottom,
		},
	}
}

func (s *selectButtonContent) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	s.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (s *selectButtonContent) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return s.layout(context).Measure(context, constraints)
}

type selectItemContent struct {
	guigui.DefaultWidget

	content guigui.Widget
}

func (s *selectItemContent) SetContent(content guigui.Widget) {
	s.content = content
}

func (s *selectItemContent) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(s.content)
}

func (s *selectItemContent) layout(context *guigui.Context) guigui.LinearLayout {
	u := UnitSize(context)
	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: s.content,
			},
		},
		Padding: guigui.Padding{
			Start:  u / 4,
			Top:    int(context.Scale()),
			End:    u / 4,
			Bottom: int(context.Scale()),
		},
	}
}

func (s *selectItemContent) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	s.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (s *selectItemContent) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return s.layout(context).Measure(context, constraints)
}
