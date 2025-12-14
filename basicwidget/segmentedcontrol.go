// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"fmt"
	"image"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	segmentedControlEventItemSelected = "itemSelected"
)

type SegmentedControlDirection int

const (
	SegmentedControlDirectionHorizontal SegmentedControlDirection = iota
	SegmentedControlDirectionVertical
)

type SegmentedControlItem[T comparable] struct {
	Text      string
	Icon      *ebiten.Image
	IconAlign IconAlign
	Disabled  bool
	Value     T
}

func (s SegmentedControlItem[T]) value() T {
	return s.Value
}

func (s SegmentedControlItem[T]) selectable() bool {
	return !s.Disabled
}

type SegmentedControl[T comparable] struct {
	guigui.DefaultWidget

	abstractList abstractList[T, SegmentedControlItem[T]]
	buttons      []Button

	direction   SegmentedControlDirection
	layoutItems []guigui.LinearLayoutItem

	onItemSelected func(index int)

	onButtonDowns []func(context *guigui.Context)
}

func (s *SegmentedControl[T]) SetDirection(direction SegmentedControlDirection) {
	if s.direction == direction {
		return
	}
	s.direction = direction
	guigui.RequestRedraw(s)
}

func (s *SegmentedControl[T]) SetOnItemSelected(f func(context *guigui.Context, index int)) {
	guigui.RegisterEventHandler(s, segmentedControlEventItemSelected, f)
}

func (s *SegmentedControl[T]) SetItems(items []SegmentedControlItem[T]) {
	s.abstractList.SetItems(items)
}

func (s *SegmentedControl[T]) SelectedItem() (SegmentedControlItem[T], bool) {
	return s.abstractList.SelectedItem()
}

func (s *SegmentedControl[T]) SelectedItemIndex() int {
	return s.abstractList.SelectedItemIndex()
}

func (s *SegmentedControl[T]) ItemByIndex(index int) (SegmentedControlItem[T], bool) {
	return s.abstractList.ItemByIndex(index)
}

func (s *SegmentedControl[T]) SelectItemByIndex(index int) {
	if s.abstractList.SelectItemByIndex(index, false) {
		guigui.RequestRedraw(s)
	}
}

func (s *SegmentedControl[T]) SelectItemByValue(value T) {
	if s.abstractList.SelectItemByValue(value, false) {
		guigui.RequestRedraw(s)
	}
}

func (s *SegmentedControl[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	for i := range s.buttons {
		adder.AddChild(&s.buttons[i])
	}

	if s.onItemSelected == nil {
		s.onItemSelected = func(index int) {
			guigui.DispatchEventHandler(s, segmentedControlEventItemSelected, index)
		}
	}
	s.abstractList.SetOnItemSelected(s.onItemSelected)

	s.buttons = adjustSliceSize(s.buttons, s.abstractList.ItemCount())
	s.onButtonDowns = adjustSliceSize(s.onButtonDowns, s.abstractList.ItemCount())

	for i := range s.abstractList.ItemCount() {
		item, _ := s.abstractList.ItemByIndex(i)
		s.buttons[i].SetText(item.Text)
		s.buttons[i].SetIcon(item.Icon)
		s.buttons[i].SetIconAlign(item.IconAlign)
		s.buttons[i].SetTextBold(s.abstractList.SelectedItemIndex() == i)
		s.buttons[i].setUseAccentColor(true)
		if s.abstractList.ItemCount() > 1 {
			switch i {
			case 0:
				switch s.direction {
				case SegmentedControlDirectionHorizontal:
					s.buttons[i].setSharpCorners(basicwidgetdraw.Corners{
						TopEnd:    true,
						BottomEnd: true,
					})
				case SegmentedControlDirectionVertical:
					s.buttons[i].setSharpCorners(basicwidgetdraw.Corners{
						BottomStart: true,
						BottomEnd:   true,
					})
				}
			case s.abstractList.ItemCount() - 1:
				switch s.direction {
				case SegmentedControlDirectionHorizontal:
					s.buttons[i].setSharpCorners(basicwidgetdraw.Corners{
						TopStart:    true,
						BottomStart: true,
					})
				case SegmentedControlDirectionVertical:
					s.buttons[i].setSharpCorners(basicwidgetdraw.Corners{
						TopEnd:   true,
						TopStart: true,
					})
				}
			default:
				s.buttons[i].setSharpCorners(basicwidgetdraw.Corners{
					TopStart:    true,
					BottomStart: true,
					TopEnd:      true,
					BottomEnd:   true,
				})
			}
		}
		context.SetEnabled(&s.buttons[i], !item.Disabled)
		s.buttons[i].setKeepPressed(s.abstractList.SelectedItemIndex() == i)
		if s.onButtonDowns[i] == nil {
			s.onButtonDowns[i] = func(context *guigui.Context) {
				s.SelectItemByIndex(i)
			}
		}
		s.buttons[i].SetOnDown(s.onButtonDowns[i])
	}
	return nil
}

func (s *SegmentedControl[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	s.layoutItems = adjustSliceSize(s.layoutItems, s.abstractList.ItemCount())
	for i := range s.abstractList.ItemCount() {
		s.layoutItems[i] = guigui.LinearLayoutItem{
			Widget: &s.buttons[i],
			Size:   guigui.FlexibleSize(1),
		}
	}

	var direction guigui.LayoutDirection
	switch s.direction {
	case SegmentedControlDirectionHorizontal:
		direction = guigui.LayoutDirectionHorizontal
	case SegmentedControlDirectionVertical:
		direction = guigui.LayoutDirectionVertical
	}
	(guigui.LinearLayout{
		Direction: direction,
		Items:     s.layoutItems,
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (s *SegmentedControl[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	if s.abstractList.ItemCount() == 0 {
		return image.Pt(0, 0)
	}

	var w, h int
	for i := range s.buttons {
		switch s.direction {
		case SegmentedControlDirectionHorizontal:
			if fixedWidth, ok := constraints.FixedWidth(); ok {
				constraints = guigui.FixedWidthConstraints(fixedWidth / s.abstractList.ItemCount())
			}
		case SegmentedControlDirectionVertical:
			if fixedHeight, ok := constraints.FixedHeight(); ok {
				constraints = guigui.FixedHeightConstraints(fixedHeight / s.abstractList.ItemCount())
			}
		}
		size := s.buttons[i].measure(context, constraints, true)
		w = max(w, size.X)
		h = max(h, size.Y)
	}
	switch s.direction {
	case SegmentedControlDirectionHorizontal:
		return image.Pt(w*len(s.buttons), h)
	case SegmentedControlDirectionVertical:
		return image.Pt(w, h*len(s.buttons))
	default:
		panic(fmt.Sprintf("basicwidget: unknown direction %d", s.direction))
	}
}
