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
	onItemsSelected func(indices []int)

	tmpIndices []int
}

func (a *abstractList[Value, Item]) SetOnItemSelected(f func(index int)) {
	a.onItemSelected = f
}

func (a *abstractList[Value, Item]) SetOnItemsSelected(f func(indices []int)) {
	a.onItemsSelected = f
}

func (a *abstractList[Value, Item]) SetItems(items []Item) {
	a.items = adjustSliceSize(items, len(items))
	copy(a.items, items)
	a.selectedIndices = slices.DeleteFunc(a.selectedIndices, func(index int) bool {
		return index < 0 || index >= len(a.items) || !a.items[index].selectable()
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
	if index < 0 || index >= len(a.items) {
		return a.SelectItemsByIndices(nil, forceFireEvents)
	}
	a.tmpIndices = append(a.tmpIndices[:0], index)
	return a.SelectItemsByIndices(a.tmpIndices, forceFireEvents)
}

func (a *abstractList[Value, Item]) SelectItemsByIndices(indices []int, forceFireEvents bool) bool {
	indices = slices.DeleteFunc(indices, func(index int) bool {
		return index < 0 || index >= len(a.items) || !a.items[index].selectable()
	})
	slices.Sort(indices)

	if slices.Equal(a.selectedIndices, indices) {
		if forceFireEvents {
			if a.onItemsSelected != nil {
				a.onItemsSelected(indices)
			}
			if a.onItemSelected != nil && len(indices) > 0 {
				a.onItemSelected(indices[0])
			}
		}
		return false
	}

	oldFirstIndex := -1
	if len(a.selectedIndices) > 0 {
		oldFirstIndex = a.selectedIndices[0]
	}

	a.selectedIndices = adjustSliceSize(a.selectedIndices, len(indices))
	copy(a.selectedIndices, indices)

	newFirstIndex := -1
	if len(a.selectedIndices) > 0 {
		newFirstIndex = a.selectedIndices[0]
	}

	if a.onItemsSelected != nil {
		a.onItemsSelected(indices)
	}
	if newFirstIndex >= 0 && (oldFirstIndex != newFirstIndex || forceFireEvents) {
		if a.onItemSelected != nil {
			a.onItemSelected(newFirstIndex)
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

func (a *abstractList[Value, Item]) SelectItemsByValues(values []Value, forceFireEvents bool) bool {
	a.tmpIndices = a.tmpIndices[:0]
	for i, item := range a.items {
		if slices.Contains(values, item.value()) {
			a.tmpIndices = append(a.tmpIndices, i)
		}
	}
	return a.SelectItemsByIndices(a.tmpIndices, forceFireEvents)
}

func (a *abstractList[Value, Item]) SelectedItemCount() int {
	return len(a.selectedIndices)
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

func (a *abstractList[Value, Item]) AppendSelectedItems(items []Item) []Item {
	for _, idx := range a.selectedIndices {
		if idx < 0 || idx >= len(a.items) {
			continue
		}
		items = append(items, a.items[idx])
	}
	return items
}

func (a *abstractList[Value, Item]) SelectedItemIndex() int {
	if len(a.selectedIndices) > 0 {
		return a.selectedIndices[0]
	}
	return -1
}

func (a *abstractList[Value, Item]) AppendSelectedItemIndices(indices []int) []int {
	for _, idx := range a.selectedIndices {
		if idx < 0 || idx >= len(a.items) {
			continue
		}
		indices = append(indices, idx)
	}
	return indices
}
