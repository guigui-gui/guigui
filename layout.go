// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package guigui

import (
	"image"
	"slices"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
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

	tmpSizes []int
}

// linearLayoutItemCacheIdentity represents the identity of a cache.
// If and only if two linearLayoutItemCacheIdentity values are equal, the cache can be reused.
type linearLayoutItemCacheIdentity struct {
	defaultSize int

	// widgetState is needed to make widgetIndices valid.
	widgetState *widgetState

	size Size
}

func (l LinearLayout) WidgetBounds(context *Context, bounds image.Rectangle, widget Widget) image.Rectangle {
	if b, ok := theCachedLinearLayouts.widgetBounds(context, &l, bounds, widget); ok {
		return b
	}
	for i, item := range l.Items {
		if item.Layout == nil {
			continue
		}
		b := theCachedLinearLayouts.itemBounds(context, &l, bounds, i)
		if r := item.Layout.WidgetBounds(context, b, widget); !r.Empty() {
			return r
		}
	}
	return image.Rectangle{}
}

func (l LinearLayout) ItemBounds(context *Context, bounds image.Rectangle, index int) image.Rectangle {
	return theCachedLinearLayouts.itemBounds(context, &l, bounds, index)
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

type positionAndSize struct {
	position int
	size     int
}

func (l *LinearLayout) appendWidgetAlongPositionAndSizes(widgetAlongPositions []positionAndSize, context *Context, alongSize, acrossSize int) []positionAndSize {
	l.tmpSizes = l.appendSizesInPixels(l.tmpSizes[:0], context, alongSize, acrossSize)

	var progress int
	for i := range l.Items {
		widgetAlongPositions = append(widgetAlongPositions, positionAndSize{
			position: progress,
			size:     l.tmpSizes[i],
		})
		progress += l.tmpSizes[i] + l.Gap
	}

	return widgetAlongPositions
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
	if rest < 0 {
		rest = 0
	}
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

	if denom > 0 {
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

type LinearLayoutItem struct {
	Widget Widget
	Size   Size
	Layout Layout
}

func (l *LinearLayoutItem) cacheIdentity(context *Context, direction LayoutDirection, acrossSize int) linearLayoutItemCacheIdentity {
	identity := linearLayoutItemCacheIdentity{
		size: l.Size,
	}
	if l.Widget != nil {
		identity.widgetState = l.Widget.widgetState()
		if l.Size.typ == sizeTypeDefault {
			identity.defaultSize = linearLayoutItemDefaultAlongSize(context, direction, l, acrossSize)
		}
	}
	return identity
}

type cachedLinearLayoutValues struct {
	itemAlongPositionAndSizes []positionAndSize
	widgetIndices             map[Widget]int

	direction  LayoutDirection
	alongSize  int
	acrossSize int
	items      []linearLayoutItemCacheIdentity
	gap        int

	atime int64
}

func (c *cachedLinearLayoutValues) matches(context *Context, linearLayout *LinearLayout, alongSize, acrossSize int) bool {
	if c.alongSize != alongSize {
		return false
	}
	if c.acrossSize != acrossSize {
		return false
	}
	if c.direction != linearLayout.Direction {
		return false
	}
	if len(c.items) != len(linearLayout.Items) {
		return false
	}
	for i, item := range linearLayout.Items {
		if c.items[i] != item.cacheIdentity(context, linearLayout.Direction, acrossSize) {
			return false
		}
	}
	if c.gap != linearLayout.Gap {
		return false
	}
	return true
}

type cachedLinearLayouts struct {
	values []*cachedLinearLayoutValues

	m sync.Mutex
}

var theCachedLinearLayouts cachedLinearLayouts

func (c *cachedLinearLayouts) itemBounds(context *Context, linearLayout *LinearLayout, bounds image.Rectangle, index int) image.Rectangle {
	c.m.Lock()
	defer c.m.Unlock()

	v := c.get(context, linearLayout, bounds)
	ps := v.itemAlongPositionAndSizes[index]
	return positionAndSizeToBounds(linearLayout, bounds, ps)
}

func (c *cachedLinearLayouts) widgetBounds(context *Context, linearLayout *LinearLayout, bounds image.Rectangle, widget Widget) (image.Rectangle, bool) {
	c.m.Lock()
	defer c.m.Unlock()

	v := c.get(context, linearLayout, bounds)
	idx, ok := v.widgetIndices[widget]
	if !ok {
		return image.Rectangle{}, false
	}
	ps := v.itemAlongPositionAndSizes[idx]
	return positionAndSizeToBounds(linearLayout, bounds, ps), true
}

func positionAndSizeToBounds(linearLayout *LinearLayout, bounds image.Rectangle, ps positionAndSize) image.Rectangle {
	pt := bounds.Min.Add(image.Pt(linearLayout.Padding.Start, linearLayout.Padding.Top))
	acrossSize := linearLayout.acrossSize(bounds)
	switch linearLayout.Direction {
	case LayoutDirectionHorizontal:
		pt.X += ps.position
		return image.Rectangle{
			Min: pt,
			Max: pt.Add(image.Pt(ps.size, acrossSize)),
		}
	case LayoutDirectionVertical:
		pt.Y += ps.position
		return image.Rectangle{
			Min: pt,
			Max: pt.Add(image.Pt(acrossSize, ps.size)),
		}
	}
	return image.Rectangle{}
}

func (c *cachedLinearLayouts) get(context *Context, linearLayout *LinearLayout, bounds image.Rectangle) *cachedLinearLayoutValues {
	alongSize := linearLayout.alongSize(bounds)
	acrossSize := linearLayout.acrossSize(bounds)

	for _, v := range c.values {
		if !v.matches(context, linearLayout, alongSize, acrossSize) {
			continue
		}
		v.atime = ebiten.Tick()
		return v
	}

	// GC old results.
	now := ebiten.Tick()
	for i := len(c.values) - 1; i >= 0; i-- {
		if now-c.values[i].atime > int64(ebiten.TPS()) {
			c.values = slices.Delete(c.values, i, i+1)
		}
	}

	v := &cachedLinearLayoutValues{
		alongSize:  alongSize,
		acrossSize: acrossSize,
		direction:  linearLayout.Direction,
		gap:        linearLayout.Gap,
		atime:      now,
	}

	if len(linearLayout.Items) > 0 {
		v.items = make([]linearLayoutItemCacheIdentity, len(linearLayout.Items))
		for i, item := range linearLayout.Items {
			v.items[i] = item.cacheIdentity(context, linearLayout.Direction, alongSize)
			if item.Widget != nil {
				if v.widgetIndices == nil {
					v.widgetIndices = map[Widget]int{}
				}
				v.widgetIndices[item.Widget] = i
			}
		}

		v.itemAlongPositionAndSizes = make([]positionAndSize, 0, len(linearLayout.Items))
		v.itemAlongPositionAndSizes = linearLayout.appendWidgetAlongPositionAndSizes(v.itemAlongPositionAndSizes, context, alongSize, acrossSize)
	}
	c.values = append(c.values, v)

	return v
}
