// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

var (
	radioButtonGroupEventItemSelected guigui.EventKey = guigui.GenerateEventKey()
)

// RadioButtonGroup is a group of radio buttons.
//
// RadioButtonGroup holds [RadioButton] widgets, but doesn't add them to the widget tree.
// The user must add the [RadioButton] widgets to the widget tree manually.
type RadioButtonGroup[T comparable] struct {
	guigui.DefaultWidget

	buttons guigui.WidgetSlice[*RadioButton[T]]

	indexPlus1 int
	values     []T
}

func (r *RadioButtonGroup[T]) SelectedIndex() int {
	if r.indexPlus1 <= 0 {
		return -1
	}
	return r.indexPlus1 - 1
}

func (r *RadioButtonGroup[T]) SelectItemByIndex(index int) {
	if r.indexPlus1 == index+1 {
		return
	}
	r.indexPlus1 = index + 1
	for i := range r.buttons.Len() {
		guigui.RequestRebuild(r.buttons.At(i))
	}
	guigui.DispatchEvent(r, radioButtonGroupEventItemSelected, r.indexPlus1-1)
}

func (r *RadioButtonGroup[T]) SelectItemByValue(value T) {
	idx := slices.Index(r.values, value)
	r.SelectItemByIndex(idx)
}

func (r *RadioButtonGroup[T]) SelectedValue() (T, bool) {
	if r.indexPlus1 <= 0 || r.indexPlus1-1 >= len(r.values) {
		var zero T
		return zero, false
	}
	return r.values[r.indexPlus1-1], true
}

func (r *RadioButtonGroup[T]) OnItemSelected(f func(context *guigui.Context, index int)) {
	guigui.SetEventHandler(r, radioButtonGroupEventItemSelected, f)
}

// SetValues sets the values of the radio buttons.
// The length of the values slice determines the number of radio buttons.
func (r *RadioButtonGroup[T]) SetValues(values []T) {
	if slices.Equal(r.values, values) {
		return
	}
	r.values = adjustSliceSize(r.values, len(values))
	copy(r.values, values)
	r.buttons.SetLen(len(values))
	for i := range r.buttons.Len() {
		r.buttons.At(i).setGroupAndIndex(r, i)
	}
	guigui.RequestRebuild(r)
}

// RadioButton returns the radio button at the specified index.
func (r *RadioButtonGroup[T]) RadioButton(index int) *RadioButton[T] {
	return r.buttons.At(index)
}

// RadioButton is a radio button to choose one value from a group of values.
//
// RadioButton should not be created directly. Use [RadioButtonGroup] instead.
type RadioButton[T comparable] struct {
	guigui.DefaultWidget

	group *RadioButtonGroup[T]
	index int

	pressed     bool
	prevHovered bool
}

func (r *RadioButton[T]) setGroupAndIndex(group *RadioButtonGroup[T], index int) {
	r.group = group
	r.index = index
}

func (r *RadioButton[T]) setPressed(pressed bool) {
	if r.pressed == pressed {
		return
	}
	r.pressed = pressed
	guigui.RequestRedraw(r)
}

func (r *RadioButton[T]) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if context.IsEnabled(r) && widgetBounds.IsHitAtCursor() {
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			r.group.SelectItemByIndex(r.index)
			r.setPressed(false)
			return guigui.HandleInputByWidget(r)
		}
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			context.SetFocused(r, true)
			r.setPressed(true)
			return guigui.HandleInputByWidget(r)
		}
	}
	if !context.IsEnabled(r) || !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		r.setPressed(false)
	}
	return guigui.HandleInputResult{}
}

func (r *RadioButton[T]) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if hovered := widgetBounds.IsHitAtCursor(); r.prevHovered != hovered {
		r.prevHovered = hovered
		guigui.RequestRedraw(r)
	}
	return nil
}

func (r *RadioButton[T]) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	if r.canPress(context, widgetBounds) || r.pressed {
		return ebiten.CursorShapePointer, true
	}
	return 0, true
}

func (r *RadioButton[T]) canPress(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(r) && widgetBounds.IsHitAtCursor() && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
}

func (r *RadioButton[T]) isActive(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(r) && widgetBounds.IsHitAtCursor() && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && r.pressed
}

func (r *RadioButton[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	return nil
}

func (r *RadioButton[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
}

func (r *RadioButton[T]) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	pt := widgetBounds.Bounds().Min
	bounds := image.Rectangle{
		Min: pt,
		Max: pt.Add(image.Pt(LineHeight(context), LineHeight(context))),
	}
	cm := context.ResolvedColorMode()
	radius := LineHeight(context) / 2

	var backgroundColor color.Color
	isSelected := r.group.SelectedIndex() == r.index
	if context.IsEnabled(r) {
		if isSelected {
			if r.isActive(context, widgetBounds) {
				backgroundColor = draw.Color2(cm, draw.ColorTypeAccent, 0.45, 0.45)
			} else {
				backgroundColor = draw.Color(cm, draw.ColorTypeAccent, 0.5)
			}
		} else {
			if r.isActive(context, widgetBounds) {
				backgroundColor = basicwidgetdraw.ControlSecondaryColor(cm, true)
			} else {
				backgroundColor = basicwidgetdraw.ControlColor(cm, true)
			}
		}
	} else {
		backgroundColor = basicwidgetdraw.ControlColor(cm, false)
	}
	basicwidgetdraw.DrawRoundedRect(context, dst, bounds, backgroundColor, radius)

	// Border
	strokeWidth := float32(1 * context.Scale())
	var borderClr1, borderClr2 color.Color
	if isSelected && context.IsEnabled(r) {
		borderClr1, borderClr2 = basicwidgetdraw.BorderAccentSecondaryColors(cm, basicwidgetdraw.RoundedRectBorderTypeInset)
	} else {
		borderClr1, borderClr2 = basicwidgetdraw.BorderColors(cm, basicwidgetdraw.RoundedRectBorderTypeInset)
	}
	basicwidgetdraw.DrawRoundedRectBorder(context, dst, bounds, borderClr1, borderClr2, radius, strokeWidth, basicwidgetdraw.RoundedRectBorderTypeInset)

	if isSelected {
		innerBounds := bounds
		innerRadius := int(float64(min(bounds.Dx(), bounds.Dy())) * 0.175)
		innerBounds.Min.X += bounds.Dx()/2 - innerRadius
		innerBounds.Min.Y += bounds.Dy()/2 - innerRadius + int(0.5*context.Scale())
		innerBounds.Max.X = innerBounds.Min.X + innerRadius*2
		innerBounds.Max.Y = innerBounds.Min.Y + innerRadius*2

		imageCM := ebiten.ColorModeDark
		if cm == ebiten.ColorModeLight && !context.IsEnabled(r) {
			imageCM = ebiten.ColorModeLight
		}
		innerColor := draw.Color(imageCM, draw.ColorTypeBase, 0)
		if !context.IsEnabled(r) {
			innerColor = draw.ScaleAlpha(innerColor, 0.5)
		}
		basicwidgetdraw.DrawRoundedRect(context, dst, innerBounds, innerColor, innerRadius)
	}
}

func (r *RadioButton[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	h := LineHeight(context)
	return image.Pt(h, h)
}
