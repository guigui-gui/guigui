// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package guigui

import (
	"image"
)

type Layout interface {
	WidgetBounds(context *Context, bounds image.Rectangle, widget Widget) image.Rectangle
	Measure(context *Context, constraints Constraints) image.Point
}

type Size struct {
	typ   sizeType
	value int
}

type sizeType int

const (
	sizeTypeDefault sizeType = iota
	sizeTypeFixed
	sizeTypeFlexible
)

func FixedSize(value int) Size {
	return Size{
		typ:   sizeTypeFixed,
		value: value,
	}
}

func FlexibleSize(value int) Size {
	return Size{
		typ:   sizeTypeFlexible,
		value: value,
	}
}

type LayoutDirection int

const (
	LayoutDirectionHorizontal LayoutDirection = iota
	LayoutDirectionVertical
)

type Padding struct {
	Start  int
	Top    int
	End    int
	Bottom int
}

type LinearLayout struct {
	Direction LayoutDirection
	Items     []LinearLayoutItem
	Gap       int
	Padding   Padding

	tmpSizes     []int
	tmpBoundsArr []image.Rectangle
}

type LinearLayoutItem struct {
	Widget Widget
	Size   Size
	Layout Layout
}

func (l LinearLayout) WidgetBounds(context *Context, bounds image.Rectangle, widget Widget) image.Rectangle {
	l.tmpBoundsArr = l.appendWidgetBounds(l.tmpBoundsArr[:0], context, bounds)

	for i, item := range l.Items {
		if item.Widget == nil {
			continue
		}
		if item.Widget.widgetState() == widget.widgetState() {
			return l.tmpBoundsArr[i]
		}
	}

	for i, item := range l.Items {
		if item.Layout == nil {
			continue
		}
		if r := item.Layout.WidgetBounds(context, l.tmpBoundsArr[i], widget); !r.Empty() {
			return r
		}
	}

	return image.Rectangle{}
}

func (l LinearLayout) AppendItemBounds(boundsArr []image.Rectangle, context *Context, bounds image.Rectangle) []image.Rectangle {
	return l.appendWidgetBounds(boundsArr, context, bounds)
}

func (l LinearLayout) ItemBounds(context *Context, bounds image.Rectangle, index int) image.Rectangle {
	l.tmpBoundsArr = l.appendWidgetBounds(l.tmpBoundsArr[:0], context, bounds)
	return l.tmpBoundsArr[index]
}

func (l *LinearLayout) alongSize(bounds image.Rectangle) int {
	switch l.Direction {
	case LayoutDirectionHorizontal:
		return bounds.Dx() - l.Padding.Start - l.Padding.End
	case LayoutDirectionVertical:
		return bounds.Dy() - l.Padding.Top - l.Padding.Bottom
	}
	return 0
}

func (l *LinearLayout) acrossSize(bounds image.Rectangle) int {
	switch l.Direction {
	case LayoutDirectionHorizontal:
		return bounds.Dy() - l.Padding.Top - l.Padding.Bottom
	case LayoutDirectionVertical:
		return bounds.Dx() - l.Padding.Start - l.Padding.End
	}
	return 0
}

func linearLayoutItemDefaultAlongSize(context *Context, direction LayoutDirection, item *LinearLayoutItem, acrossSize int) int {
	if item.Widget != nil {
		switch direction {
		case LayoutDirectionHorizontal:
			if acrossSize <= 0 {
				return item.Widget.Measure(context, Constraints{}).X
			}
			return item.Widget.Measure(context, FixedHeightConstraints(acrossSize)).X
		case LayoutDirectionVertical:
			if acrossSize <= 0 {
				return item.Widget.Measure(context, Constraints{}).Y
			}
			return item.Widget.Measure(context, FixedWidthConstraints(acrossSize)).Y
		}
	} else if item.Layout != nil {
		switch direction {
		case LayoutDirectionHorizontal:
			if acrossSize <= 0 {
				return item.Layout.Measure(context, Constraints{}).X
			}
			return item.Layout.Measure(context, FixedHeightConstraints(acrossSize)).X
		case LayoutDirectionVertical:
			if acrossSize <= 0 {
				return item.Layout.Measure(context, Constraints{}).Y
			}
			return item.Layout.Measure(context, FixedWidthConstraints(acrossSize)).Y
		}
	}
	return 0
}

func (l *LinearLayout) appendSizesInPixels(sizesInPixels []int, context *Context, alongSize, acrossSize int) []int {
	rest := alongSize
	rest -= (len(l.Items) - 1) * l.Gap
	var denom int

	origLen := len(sizesInPixels)
	for i, item := range l.Items {
		switch item.Size.typ {
		case sizeTypeDefault:
			sizesInPixels = append(sizesInPixels, linearLayoutItemDefaultAlongSize(context, l.Direction, &item, acrossSize))
		case sizeTypeFixed:
			sizesInPixels = append(sizesInPixels, item.Size.value)
		case sizeTypeFlexible:
			sizesInPixels = append(sizesInPixels, 0)
			denom += item.Size.value
		}
		rest -= sizesInPixels[origLen+i]
	}

	rest = max(rest, 0)

	if denom > 0 && rest > 0 {
		origRest := rest
		for i, item := range l.Items {
			if item.Size.typ != sizeTypeFlexible {
				continue
			}
			w := int(float64(origRest) * float64(item.Size.value) / float64(denom))
			sizesInPixels[origLen+i] = w
			rest -= w
		}
		// TODO: Use a better algorithm to distribute the rest.
		for rest > 0 {
			for i := len(sizesInPixels) - origLen - 1; i >= 0; i-- {
				if l.Items[i].Size.typ != sizeTypeFlexible {
					continue
				}
				sizesInPixels[origLen+i]++
				rest--
				if rest <= 0 {
					break
				}
			}
			if rest <= 0 {
				break
			}
		}
	}

	return sizesInPixels
}

func (l LinearLayout) Measure(context *Context, constraints Constraints) image.Point {
	var contentAlongSize int
	var contentAcrossSize int
	switch l.Direction {
	case LayoutDirectionHorizontal:
		if fixedW, ok := constraints.FixedWidth(); ok {
			contentAlongSize = fixedW - l.Padding.Start - l.Padding.End
			contentAlongSize = max(contentAlongSize, 0)
		}
		if fixedH, ok := constraints.FixedHeight(); ok {
			contentAcrossSize = fixedH - l.Padding.Top - l.Padding.Bottom
			contentAcrossSize = max(contentAcrossSize, 0)
		}
	case LayoutDirectionVertical:
		if fixedW, ok := constraints.FixedWidth(); ok {
			contentAcrossSize = fixedW - l.Padding.Start - l.Padding.End
			contentAcrossSize = max(contentAcrossSize, 0)
		}
		if fixedH, ok := constraints.FixedHeight(); ok {
			contentAlongSize = fixedH - l.Padding.Top - l.Padding.Bottom
			contentAlongSize = max(contentAlongSize, 0)
		}
	}

	var autoAlongSize int
	var autoAcrossSize int
	l.tmpSizes = l.appendSizesInPixels(l.tmpSizes[:0], context, contentAlongSize, contentAcrossSize)
	for i, item := range l.Items {
		s := l.tmpSizes[i]
		autoAlongSize += s
		if item.Widget != nil {
			switch l.Direction {
			case LayoutDirectionHorizontal:
				if s <= 0 {
					autoAcrossSize = max(autoAcrossSize, item.Widget.Measure(context, Constraints{}).Y)
				} else {
					autoAcrossSize = max(autoAcrossSize, item.Widget.Measure(context, FixedWidthConstraints(s)).Y)
				}
			case LayoutDirectionVertical:
				if s <= 0 {
					autoAcrossSize = max(autoAcrossSize, item.Widget.Measure(context, Constraints{}).X)
				} else {
					autoAcrossSize = max(autoAcrossSize, item.Widget.Measure(context, FixedHeightConstraints(s)).X)
				}
			}
		} else if item.Layout != nil {
			switch l.Direction {
			case LayoutDirectionHorizontal:
				if s <= 0 {
					autoAcrossSize = max(autoAcrossSize, item.Layout.Measure(context, Constraints{}).Y)
				} else {
					autoAcrossSize = max(autoAcrossSize, item.Layout.Measure(context, FixedWidthConstraints(s)).Y)
				}
			case LayoutDirectionVertical:
				if s <= 0 {
					autoAcrossSize = max(autoAcrossSize, item.Layout.Measure(context, Constraints{}).X)
				} else {
					autoAcrossSize = max(autoAcrossSize, item.Layout.Measure(context, FixedHeightConstraints(s)).X)
				}
			}
		}
	}

	if len(l.Items) > 0 {
		autoAlongSize += (len(l.Items) - 1) * l.Gap
	}

	switch l.Direction {
	case LayoutDirectionHorizontal:
		alongSize := autoAlongSize + l.Padding.Start + l.Padding.End
		acrossSize := autoAcrossSize + l.Padding.Top + l.Padding.Bottom
		if fixedWidth, ok := constraints.FixedWidth(); ok {
			alongSize = fixedWidth
		}
		if fixedHeight, ok := constraints.FixedHeight(); ok {
			acrossSize = fixedHeight
		}
		return image.Pt(alongSize, acrossSize)
	case LayoutDirectionVertical:
		alongSize := autoAlongSize + l.Padding.Top + l.Padding.Bottom
		acrossSize := autoAcrossSize + l.Padding.Start + l.Padding.End
		if fixedWidth, ok := constraints.FixedWidth(); ok {
			acrossSize = fixedWidth
		}
		if fixedHeight, ok := constraints.FixedHeight(); ok {
			alongSize = fixedHeight
		}
		return image.Pt(acrossSize, alongSize)
	}
	return image.Point{}
}

func (l *LinearLayout) appendWidgetBounds(boundsArr []image.Rectangle, context *Context, bounds image.Rectangle) []image.Rectangle {
	alongSize := l.alongSize(bounds)
	acrossSize := l.acrossSize(bounds)
	l.tmpSizes = l.appendSizesInPixels(l.tmpSizes[:0], context, alongSize, acrossSize)
	var progress int
	for i := range l.Items {
		boundsArr = append(boundsArr, l.positionAndSizeToBounds(bounds, progress, l.tmpSizes[i]))
		progress += l.tmpSizes[i] + l.Gap
	}
	return boundsArr
}

func (l *LinearLayout) positionAndSizeToBounds(bounds image.Rectangle, position int, size int) image.Rectangle {
	pt := bounds.Min.Add(image.Pt(l.Padding.Start, l.Padding.Top))
	acrossSize := l.acrossSize(bounds)
	switch l.Direction {
	case LayoutDirectionHorizontal:
		pt.X += position
		return image.Rectangle{
			Min: pt,
			Max: pt.Add(image.Pt(size, acrossSize)),
		}
	case LayoutDirectionVertical:
		pt.Y += position
		return image.Rectangle{
			Min: pt,
			Max: pt.Add(image.Pt(acrossSize, size)),
		}
	}
	return image.Rectangle{}
}
