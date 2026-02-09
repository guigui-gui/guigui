// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"maps"
	"slices"
)

func fastFirstIndex(indices map[int]struct{}) int {
	minIdx := -1
	for idx := range indices {
		if minIdx == -1 || idx < minIdx {
			minIdx = idx
		}
	}
	return minIdx
}

type valuer[Value comparable] interface {
	value() Value
	selectable() bool
}

type abstractList[Value comparable, Item valuer[Value]] struct {
	items           []Item
	selectedIndices map[int]struct{}
	multiSelection  bool

	onItemSelected  func(index int)
	onItemsSelected func(indices []int)

	tmpIndices []int
}

func (a *abstractList[Value, Item]) isItemIndexSelectable(index int) bool {
	if index < 0 || index >= len(a.items) {
		return false
	}
	return a.items[index].selectable()
}

func (a *abstractList[Value, Item]) SetOnItemSelected(f func(index int)) {
	a.onItemSelected = f
}

func (a *abstractList[Value, Item]) SetOnItemsSelected(f func(indices []int)) {
	a.onItemsSelected = f
}

func (a *abstractList[Value, Item]) MultiSelection() bool {
	return a.multiSelection
}

func (a *abstractList[Value, Item]) SetMultiSelection(multi bool) {
	if a.multiSelection == multi {
		return
	}
	a.multiSelection = multi
	if !multi && len(a.selectedIndices) > 1 {
		// Keep the smallest index.
		a.selectItemsByIndices(map[int]struct{}{
			fastFirstIndex(a.selectedIndices): {},
		}, false)
	}
}

func (a *abstractList[Value, Item]) SetItems(items []Item) {
	a.items = adjustSliceSize(items, len(items))
	copy(a.items, items)

	maps.DeleteFunc(a.selectedIndices, func(idx int, _ struct{}) bool {
		return !a.isItemIndexSelectable(idx)
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

func (a *abstractList[Value, Item]) ToggleItemSelectionByIndex(index int, forceFireEvents bool) bool {
	if index < 0 || index >= len(a.items) {
		return false
	}

	// If the item is already selected, deselect it.
	if _, ok := a.selectedIndices[index]; ok {
		m := maps.Clone(a.selectedIndices)
		delete(m, index)
		return a.selectItemsByIndices(m, forceFireEvents)
	}

	// If the item is not selected, select it.
	if a.multiSelection {
		m := maps.Clone(a.selectedIndices)
		if m == nil {
			m = map[int]struct{}{}
		}
		m[index] = struct{}{}
		return a.selectItemsByIndices(m, forceFireEvents)
	}

	// In single selection mode, replace the selection.
	return a.selectItemsByIndices(map[int]struct{}{index: {}}, forceFireEvents)
}

func (a *abstractList[Value, Item]) SelectItemsByIndices(indices []int, forceFireEvents bool) bool {
	var m map[int]struct{}
	if len(indices) > 0 {
		m = make(map[int]struct{}, len(indices))
		for _, idx := range indices {
			m[idx] = struct{}{}
		}
	}
	return a.selectItemsByIndices(m, forceFireEvents)
}

func (a *abstractList[Value, Item]) selectItemsByIndices(indices map[int]struct{}, forceFireEvents bool) bool {
	maps.DeleteFunc(indices, func(idx int, _ struct{}) bool {
		return !a.isItemIndexSelectable(idx)
	})
	if !a.multiSelection && len(indices) > 1 {
		indices = map[int]struct{}{
			fastFirstIndex(indices): {},
		}
	}

	if maps.Equal(a.selectedIndices, indices) {
		if forceFireEvents {
			if a.onItemsSelected != nil {
				s := slices.Collect(maps.Keys(indices))
				slices.Sort(s)
				a.onItemsSelected(s)
			}
			if a.onItemSelected != nil && len(indices) > 0 {
				a.onItemSelected(fastFirstIndex(indices))
			}
		}
		return false
	}

	oldFirstIndex := a.SelectedItemIndex()

	clear(a.selectedIndices)
	if a.selectedIndices == nil {
		a.selectedIndices = map[int]struct{}{}
	}
	maps.Copy(a.selectedIndices, indices)

	newFirstIndex := fastFirstIndex(indices)

	if a.onItemsSelected != nil {
		s := slices.Collect(maps.Keys(indices))
		slices.Sort(s)
		a.onItemsSelected(s)
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
	idx := a.SelectedItemIndex()
	if idx == -1 {
		var item Item
		return item, false
	}
	return a.items[idx], true
}

func (a *abstractList[Value, Item]) AppendSelectedItems(items []Item) []Item {
	a.tmpIndices = a.AppendSelectedItemIndices(a.tmpIndices[:0])
	for _, idx := range a.tmpIndices {
		items = append(items, a.items[idx])
	}
	return items
}

func (a *abstractList[Value, Item]) SelectedItemIndex() int {
	minIdx := -1
	for idx := range a.selectedIndices {
		if !a.isItemIndexSelectable(idx) {
			continue
		}
		if minIdx == -1 || idx < minIdx {
			minIdx = idx
		}
	}
	return minIdx
}

func (a *abstractList[Value, Item]) IsSelectedItemIndex(index int) bool {
	if !a.isItemIndexSelectable(index) {
		return false
	}
	_, ok := a.selectedIndices[index]
	return ok
}

func (a *abstractList[Value, Item]) AppendSelectedItemIndices(indices []int) []int {
	origLen := len(indices)
	for idx := range a.selectedIndices {
		if !a.isItemIndexSelectable(idx) {
			continue
		}
		indices = append(indices, idx)
	}
	slices.Sort(indices[origLen:])
	return indices
}
