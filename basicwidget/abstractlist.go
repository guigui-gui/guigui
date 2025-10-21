// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"slices"

	"github.com/guigui-gui/guigui"
)

const (
	abstractListEventItemSelected = "itemSelected"
)

type valuer[Value comparable] interface {
	value() Value
}

type abstractList[Value comparable, Item valuer[Value]] struct {
	items               []Item
	selectedIndices     []int
	nextSelectedIndices []int
}

func (a *abstractList[Value, Item]) SetOnItemSelected(widget guigui.Widget, f func(index int)) {
	guigui.RegisterEventHandler(widget, abstractListEventItemSelected, f)
}

func (a *abstractList[Value, Item]) SetItems(widget guigui.Widget, items []Item) {
	a.items = adjustSliceSize(items, len(items))
	copy(a.items, items)

	if len(a.nextSelectedIndices) > 0 {
		if len(a.nextSelectedIndices) != 1 {
			panic("basicwidget: nextSelectedIndices must have length 0 or 1 so far")
		}
		index := a.nextSelectedIndices[0]
		a.selectedIndices = adjustSliceSize(a.selectedIndices, 1)
		a.selectedIndices[0] = index
		guigui.DispatchEventHandler(widget, abstractListEventItemSelected, index)
	}
}

func (a *abstractList[Value, Item]) ItemCount() int {
	return len(a.items)
}

func (a *abstractList[Value, Item]) ItemByIndex(index int) (Item, bool) {
	if index < 0 || index >= len(a.items) {
		var item Item
		return item, false
	}
	return a.items[index], true
}

func (a *abstractList[Value, Item]) SelectItemByIndex(widget guigui.Widget, index int, forceFireEvents bool) bool {
	if index < 0 {
		if len(a.selectedIndices) == 0 {
			return false
		}
		a.selectedIndices = a.selectedIndices[:0]
		return true
	}

	if index >= len(a.items) {
		a.selectedIndices = a.selectedIndices[:0]
		if len(a.nextSelectedIndices) == 1 && a.nextSelectedIndices[0] == index {
			return false
		}
		a.nextSelectedIndices = adjustSliceSize(a.nextSelectedIndices, 1)
		a.nextSelectedIndices[0] = index
		return true
	}

	if len(a.selectedIndices) == 1 && a.selectedIndices[0] == index && !forceFireEvents {
		return false
	}

	selected := slices.Contains(a.selectedIndices, index)
	a.selectedIndices = adjustSliceSize(a.selectedIndices, 1)
	a.selectedIndices[0] = index
	a.nextSelectedIndices = a.nextSelectedIndices[:0]
	if !selected || forceFireEvents {
		guigui.DispatchEventHandler(widget, abstractListEventItemSelected, index)
	}
	return true
}

func (a *abstractList[Value, Item]) SelectItemByValue(widget guigui.Widget, value Value, forceFireEvents bool) bool {
	idx := slices.IndexFunc(a.items, func(item Item) bool {
		return item.value() == value
	})
	return a.SelectItemByIndex(widget, idx, forceFireEvents)
}

func (a *abstractList[Value, Item]) SelectedItem() (Item, bool) {
	if len(a.selectedIndices) == 0 {
		var item Item
		return item, false
	}
	return a.items[a.selectedIndices[0]], true
}

func (a *abstractList[Value, Item]) SelectedItemIndex() int {
	if len(a.selectedIndices) == 0 {
		return -1
	}
	return a.selectedIndices[0]
}
