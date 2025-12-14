// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"slices"
)

type valuer[Value comparable] interface {
	value() Value
	selectable() bool
}

type abstractList[Value comparable, Item valuer[Value]] struct {
	items           []Item
	selectedIndices []int
	onItemSelected  func(index int)
}

func (a *abstractList[Value, Item]) SetOnItemSelected(f func(index int)) {
	a.onItemSelected = f
}

func (a *abstractList[Value, Item]) SetItems(items []Item) {
	a.items = adjustSliceSize(items, len(items))
	copy(a.items, items)
	a.selectedIndices = slices.DeleteFunc(a.selectedIndices, func(index int) bool {
		return index >= 0 && index < len(a.items) && !a.items[index].selectable()
	})
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

func (a *abstractList[Value, Item]) SelectItemByIndex(index int, forceFireEvents bool) bool {
	if index < 0 {
		if len(a.selectedIndices) == 0 {
			return false
		}
		a.selectedIndices = a.selectedIndices[:0]
		return true
	}

	if index >= 0 && index < len(a.items) && !a.items[index].selectable() {
		a.selectedIndices = a.selectedIndices[:0]
		return true
	}

	if len(a.selectedIndices) == 1 && a.selectedIndices[0] == index && !forceFireEvents {
		return false
	}

	selected := slices.Contains(a.selectedIndices, index)
	a.selectedIndices = adjustSliceSize(a.selectedIndices, 1)
	a.selectedIndices[0] = index
	if !selected || forceFireEvents {
		if a.onItemSelected != nil {
			a.onItemSelected(index)
		}
	}
	return true
}

func (a *abstractList[Value, Item]) SelectItemByValue(value Value, forceFireEvents bool) bool {
	idx := slices.IndexFunc(a.items, func(item Item) bool {
		return item.value() == value
	})
	return a.SelectItemByIndex(idx, forceFireEvents)
}

func (a *abstractList[Value, Item]) SelectedItem() (Item, bool) {
	if len(a.selectedIndices) > 0 {
		idx := a.selectedIndices[0]
		if idx < 0 || idx >= len(a.items) {
			var item Item
			return item, false
		}
		return a.items[idx], true
	}
	var item Item
	return item, false
}

func (a *abstractList[Value, Item]) SelectedItemIndex() int {
	if len(a.selectedIndices) > 0 {
		return a.selectedIndices[0]
	}
	return -1
}
