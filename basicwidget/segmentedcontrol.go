// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"fmt"
	"image"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
	"github.com/hajimehoshi/ebiten/v2"
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

type SegmentedControl[T comparable] struct {
	guigui.DefaultWidget

	abstractList abstractList[T, SegmentedControlItem[T]]
	buttons      []Button

	direction   SegmentedControlDirection
	layoutItems []guigui.LinearLayoutItem
}

func (s *SegmentedControl[T]) SetDirection(direction SegmentedControlDirection) {
	if s.direction == direction {
		return
	}
	s.direction = direction
	guigui.RequestRedraw(s)
}

func (s *SegmentedControl[T]) SetOnItemSelected(f func(index int)) {
	s.abstractList.SetOnItemSelected(s, f)
}

func (s *SegmentedControl[T]) SetItems(items []SegmentedControlItem[T]) {
	s.abstractList.SetItems(s, items)
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
	if s.abstractList.SelectItemByIndex(s, index, false) {
		guigui.RequestRedraw(s)
	}
}

func (s *SegmentedControl[T]) SelectItemByValue(value T) {
	if s.abstractList.SelectItemByValue(s, value, false) {
		guigui.RequestRedraw(s)
	}
}

func (s *SegmentedControl[T]) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	for i := range s.buttons {
		adder.AddChild(&s.buttons[i])
	}
}

func (s *SegmentedControl[T]) Update(context *guigui.Context) error {
	s.updateButtons(context)
	return nil
}

func (s *SegmentedControl[T]) updateButtons(context *guigui.Context) {
	s.buttons = adjustSliceSize(s.buttons, s.abstractList.ItemCount())

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
					s.buttons[i].setSharpenCorners(draw.SharpenCorners{
						UpperEnd: true,
						LowerEnd: true,
					})
				case SegmentedControlDirectionVertical:
					s.buttons[i].setSharpenCorners(draw.SharpenCorners{
						LowerStart: true,
						LowerEnd:   true,
					})
				}
			case s.abstractList.ItemCount() - 1:
				switch s.direction {
				case SegmentedControlDirectionHorizontal:
					s.buttons[i].setSharpenCorners(draw.SharpenCorners{
						UpperStart: true,
						LowerStart: true,
					})
				case SegmentedControlDirectionVertical:
					s.buttons[i].setSharpenCorners(draw.SharpenCorners{
						UpperEnd:   true,
						UpperStart: true,
					})
				}
			default:
				s.buttons[i].setSharpenCorners(draw.SharpenCorners{
					UpperStart: true,
					LowerStart: true,
					UpperEnd:   true,
					LowerEnd:   true,
				})
			}
		}
		context.SetEnabled(&s.buttons[i], !item.Disabled)
		s.buttons[i].setKeepPressed(s.abstractList.SelectedItemIndex() == i)
		s.buttons[i].SetOnDown(func() {
			s.SelectItemByIndex(i)
		})
	}
}

func (s *SegmentedControl[T]) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
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
	s.updateButtons(context)
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
