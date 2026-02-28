// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"image"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

type FormItem struct {
	PrimaryWidget   guigui.Widget
	SecondaryWidget guigui.Widget
}

type Form struct {
	guigui.DefaultWidget

	items []FormItem

	cachedItemBounds    []image.Rectangle
	cachedContentBounds map[guigui.Widget]image.Rectangle

	cachedItemBoundsForMeasure    []image.Rectangle
	cachedContentBoundsForMeasure map[guigui.Widget]image.Rectangle
}

func formItemPadding(context *guigui.Context) image.Point {
	return image.Pt(UnitSize(context)/2, UnitSize(context)/4)
}

func (f *Form) SetItems(items []FormItem) {
	f.items = slices.Delete(f.items, 0, len(f.items))
	f.items = append(f.items, items...)
}

func (f *Form) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	for _, item := range f.items {
		if item.PrimaryWidget != nil {
			adder.AddWidget(item.PrimaryWidget)
		}
		if item.SecondaryWidget != nil {
			adder.AddWidget(item.SecondaryWidget)
		}
	}
	return nil
}

func (f *Form) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	f.cachedItemBounds = slices.Delete(f.cachedItemBounds, 0, len(f.cachedItemBounds))
	clear(f.cachedContentBounds)
	pt := widgetBounds.Bounds().Min
	f.cachedItemBounds, f.cachedContentBounds = f.appendItemBounds(context, f.cachedItemBounds, f.cachedContentBounds, pt, widgetBounds.Bounds().Dx())

	for widget, bounds := range f.cachedContentBounds {
		layouter.LayoutWidget(widget, bounds)
	}
}

func (f *Form) isItemOmitted(context *guigui.Context, item FormItem) bool {
	return (item.PrimaryWidget == nil || !context.IsVisible(item.PrimaryWidget)) &&
		(item.SecondaryWidget == nil || !context.IsVisible(item.SecondaryWidget))
}

func (f *Form) appendItemBounds(context *guigui.Context, itemBounds []image.Rectangle, contentBounds map[guigui.Widget]image.Rectangle, point image.Point, width int) ([]image.Rectangle, map[guigui.Widget]image.Rectangle) {
	if contentBounds == nil {
		contentBounds = map[guigui.Widget]image.Rectangle{}
	}

	paddingS := formItemPadding(context)

	var y int
	for _, item := range f.items {
		if f.isItemOmitted(context, item) {
			itemBounds = append(itemBounds, image.Rectangle{})
			continue
		}

		u := UnitSize(context)

		var primaryS image.Point
		var secondaryS image.Point
		if item.PrimaryWidget != nil {
			primaryS = item.PrimaryWidget.Measure(context, guigui.Constraints{})
			if primaryS.X > width-2*paddingS.X {
				primaryS = item.PrimaryWidget.Measure(context, guigui.FixedWidthConstraints(width-2*paddingS.X))
			}
		}
		if item.SecondaryWidget != nil {
			secondaryS = item.SecondaryWidget.Measure(context, guigui.Constraints{})
			if secondaryS.X > width-2*paddingS.X {
				secondaryS = item.SecondaryWidget.Measure(context, guigui.FixedWidthConstraints(width-2*paddingS.X))
			}
		}
		newLine := item.PrimaryWidget != nil && primaryS.X+u/4+secondaryS.X+2*paddingS.X > width
		var baseH int
		if newLine {
			baseH = max(primaryS.Y+paddingS.Y+secondaryS.Y, minFormItemHeight(context)) + 2*paddingS.Y
		} else {
			baseH = max(primaryS.Y, secondaryS.Y, minFormItemHeight(context)) + 2*paddingS.Y
		}
		b := image.Rectangle{
			Min: point.Add(image.Pt(0, y)),
			Max: point.Add(image.Pt(width, y+baseH)),
		}
		itemBounds = append(itemBounds, b)

		maxPaddingY := paddingS.Y + (u-LineHeight(context))/2
		if item.PrimaryWidget != nil {
			bounds := b
			bounds.Min.X += paddingS.X
			bounds.Max.X = bounds.Min.X + primaryS.X
			pY := min((baseH-primaryS.Y)/2, maxPaddingY)
			bounds.Min.Y += pY
			bounds.Max.Y += pY
			contentBounds[item.PrimaryWidget] = image.Rectangle{
				Min: bounds.Min,
				Max: bounds.Min.Add(primaryS),
			}
		}
		if item.SecondaryWidget != nil {
			bounds := b
			bounds.Min.X = bounds.Max.X - secondaryS.X - paddingS.X
			bounds.Max.X -= paddingS.X
			if newLine {
				bounds.Min.Y += paddingS.Y + primaryS.Y + paddingS.Y
				bounds.Max.Y += paddingS.Y + primaryS.Y + paddingS.Y
			} else {
				pY := min((baseH-secondaryS.Y)/2, maxPaddingY)
				bounds.Min.Y += pY
				bounds.Max.Y += pY
			}
			contentBounds[item.SecondaryWidget] = image.Rectangle{
				Min: bounds.Min,
				Max: bounds.Min.Add(secondaryS),
			}
		}

		y += baseH
	}

	return itemBounds, contentBounds
}

func (f *Form) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bgClr := draw.ScaleAlpha(draw.Color(context.ResolvedColorMode(), draw.ColorTypeBase, 0), 1/32.0)
	borderClr := draw.ScaleAlpha(draw.Color(context.ResolvedColorMode(), draw.ColorTypeBase, 0), 2/32.0)

	bounds := widgetBounds.Bounds()
	basicwidgetdraw.DrawRoundedRect(context, dst, bounds, bgClr, RoundedCornerRadius(context))

	// Render borders between items.
	if len(f.items) > 0 {
		paddingS := formItemPadding(context)
		for i := range f.items[:len(f.items)-1] {
			x0 := float32(bounds.Min.X + paddingS.X)
			x1 := float32(bounds.Max.X - paddingS.X)
			y := float32(f.cachedItemBounds[i].Max.Y)
			width := 1 * float32(context.Scale())
			vector.StrokeLine(dst, x0, y, x1, y, width, borderClr, false)
		}
	}

	basicwidgetdraw.DrawRoundedRectBorder(context, dst, bounds, borderClr, borderClr, RoundedCornerRadius(context), 1*float32(context.Scale()), basicwidgetdraw.RoundedRectBorderTypeRegular)
}

func (f *Form) measureWithoutConstraints(context *guigui.Context) image.Point {
	// Measure without size constraints should return the default size rather than an actual size.
	// Do not use itemBounds, primaryBounds, or secondaryBounds here.

	paddingS := formItemPadding(context)
	gapX := UnitSize(context)

	var s image.Point
	for _, item := range f.items {
		if f.isItemOmitted(context, item) {
			continue
		}
		var primaryS image.Point
		var secondaryS image.Point
		if item.PrimaryWidget != nil {
			primaryS = item.PrimaryWidget.Measure(context, guigui.Constraints{})
		}
		if item.SecondaryWidget != nil {
			secondaryS = item.SecondaryWidget.Measure(context, guigui.Constraints{})
		}

		s.X = max(s.X, primaryS.X+secondaryS.X+2*paddingS.X+gapX)
		h := max(primaryS.Y, secondaryS.Y, minFormItemHeight(context)) + 2*paddingS.Y
		s.Y += h
	}
	return s
}

func (f *Form) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	s := f.measureWithoutConstraints(context)
	w, ok := constraints.FixedWidth()
	if !ok {
		return s
	}
	if s.X <= w {
		return image.Pt(w, s.Y)
	}

	f.cachedItemBoundsForMeasure = slices.Delete(f.cachedItemBoundsForMeasure, 0, len(f.cachedItemBoundsForMeasure))
	clear(f.cachedContentBoundsForMeasure)
	// As only the size matters, the point can be zero.
	f.cachedItemBoundsForMeasure, f.cachedContentBoundsForMeasure = f.appendItemBounds(context, f.cachedItemBoundsForMeasure, f.cachedContentBoundsForMeasure, image.Point{}, w)
	if len(f.cachedItemBoundsForMeasure) == 0 {
		return image.Pt(w, 0)
	}
	return f.cachedItemBoundsForMeasure[len(f.cachedItemBoundsForMeasure)-1].Max.Sub(f.cachedItemBoundsForMeasure[0].Min)
}

func minFormItemHeight(context *guigui.Context) int {
	return UnitSize(context)
}
