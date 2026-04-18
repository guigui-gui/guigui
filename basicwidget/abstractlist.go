// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
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
	visible() bool
}

type abstractList[Value comparable, Item valuer[Value]] struct {
	items           []Item
	selectedIndices map[int]struct{}
	multiSelection  bool

	onItemSelected  func(index int)
	onItemsSelected func(indices []int)

	tmpIndexSlice []int
	tmpIndexMap   map[int]struct{}

	anchorIndex          int
	lastExtendIndexPlus1 int

	// selectionString is a comparable fingerprint of the selected-indices set,
	// refreshed only when [abstractList.selectItemsByIndices] actually changes
	// the selection. Callers include it in their [Widget.BuildKey] so selection
	// changes trigger automatic rebuilds without explicit [guigui.RequestRebuild]
	// calls — and without the false positives a bump counter would produce.
	selectionString string
}

type abstractListBuildKey struct {
	selection string
}

func (a *abstractList[Value, Item]) buildKey() abstractListBuildKey {
	return abstractListBuildKey{
		selection: a.selectionString,
	}
}

// refreshSelectionString encodes the current selectedIndices as a sorted,
// comma-separated decimal string. Called only at the points where the
// selection is actually mutated.
func (a *abstractList[Value, Item]) refreshSelectionString() {
	a.tmpIndexSlice = a.tmpIndexSlice[:0]
	for idx := range a.selectedIndices {
		a.tmpIndexSlice = append(a.tmpIndexSlice, idx)
	}
	slices.Sort(a.tmpIndexSlice)
	var sb strings.Builder
	for i, idx := range a.tmpIndexSlice {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.Itoa(idx))
	}
	a.selectionString = sb.String()
}

func (a *abstractList[Value, Item]) isItemIndexSelectable(index int) bool {
	if index < 0 || index >= len(a.items) {
		return false
	}
	return a.items[index].selectable()
}

func (a *abstractList[Value, Item]) OnItemSelected(f func(index int)) {
	a.onItemSelected = f
}

func (a *abstractList[Value, Item]) OnItemsSelected(f func(indices []int)) {
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
		idx := fastFirstIndex(a.selectedIndices)
		a.selectItemsByIndices(map[int]struct{}{
			idx: {},
		}, idx, true, false)
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

func (a *abstractList[Value, Item]) SelectItemByIndex(index int, forceFireEvents bool) {
	if index < 0 || index >= len(a.items) {
		a.SelectItemsByIndices(nil, forceFireEvents)
		return
	}
	a.tmpIndexSlice = append(a.tmpIndexSlice[:0], index)
	a.SelectItemsByIndices(a.tmpIndexSlice, forceFireEvents)
}

func (a *abstractList[Value, Item]) ExtendItemSelectionByIndex(index int, forceFireEvents bool) {
	if index < 0 || index >= len(a.items) {
		return
	}

	if !a.multiSelection {
		a.SelectItemByIndex(index, forceFireEvents)
		return
	}

	newIndices := maps.Clone(a.selectedIndices)
	if newIndices == nil {
		newIndices = map[int]struct{}{}
	}

	// If there was a previous extension, clear that range first (excluding the anchor itself,
	// unless the new range covers it, but the new range is re-added anyway).
	if a.lastExtendIndexPlus1 > 0 {
		start, end := a.anchorIndex, a.lastExtendIndexPlus1-1
		if start > end {
			start, end = end, start
		}
		for i := start; i <= end; i++ {
			delete(newIndices, i)
		}
	}

	// Select the new range from anchorIndex to index.
	start, end := a.anchorIndex, index
	if start > end {
		start, end = end, start
	}
	for i := start; i <= end; i++ {
		if a.isItemIndexSelectable(i) {
			newIndices[i] = struct{}{}
		}
	}

	a.lastExtendIndexPlus1 = index + 1

	a.selectItemsByIndices(newIndices, 0, false, forceFireEvents)
}

func (a *abstractList[Value, Item]) ToggleItemSelectionByIndex(index int, forceFireEvents bool) {
	if index < 0 || index >= len(a.items) {
		return
	}

	// If the item is already selected, deselect it.
	if _, ok := a.selectedIndices[index]; ok {
		m := maps.Clone(a.selectedIndices)
		delete(m, index)
		a.selectItemsByIndices(m, fastFirstIndex(m), true, forceFireEvents)
		return
	}

	// If the item is not selected, select it.
	if a.multiSelection {
		m := maps.Clone(a.selectedIndices)
		if m == nil {
			m = map[int]struct{}{}
		}
		m[index] = struct{}{}
		a.selectItemsByIndices(m, index, true, forceFireEvents)
		return
	}

	// In single selection mode, replace the selection.
	a.selectItemsByIndices(map[int]struct{}{
		index: {},
	}, index, true, forceFireEvents)
}

func (a *abstractList[Value, Item]) SelectItemsByIndices(indices []int, forceFireEvents bool) {
	clear(a.tmpIndexMap)
	newAnchor := math.MaxInt
	if len(indices) > 0 {
		if a.tmpIndexMap == nil {
			a.tmpIndexMap = make(map[int]struct{}, len(indices))
		}
		for _, idx := range indices {
			a.tmpIndexMap[idx] = struct{}{}
			if !a.isItemIndexSelectable(idx) {
				continue
			}
			newAnchor = min(newAnchor, idx)
		}
	}
	if newAnchor == math.MaxInt {
		newAnchor = 0
	}
	a.selectItemsByIndices(a.tmpIndexMap, newAnchor, true, forceFireEvents)
}

func (a *abstractList[Value, Item]) SelectAllItems(forceFireEvents bool) {
	clear(a.tmpIndexMap)
	if a.tmpIndexMap == nil {
		a.tmpIndexMap = make(map[int]struct{}, len(a.items))
	}
	newAnchor := math.MaxInt
	for i := range a.items {
		a.tmpIndexMap[i] = struct{}{}
		if !a.isItemIndexSelectable(i) {
			continue
		}
		newAnchor = min(newAnchor, i)
	}
	if newAnchor == math.MaxInt {
		newAnchor = 0
	}
	a.selectItemsByIndices(a.tmpIndexMap, newAnchor, true, forceFireEvents)
}

func (a *abstractList[Value, Item]) selectItemsByIndices(indices map[int]struct{}, newAnchorCandidate int, updateAnchor bool, forceFireEvents bool) {
	// maps.DeleteFunc changes the indices directly.
	// It is caller's responsibility to make a copy if needed.
	maps.DeleteFunc(indices, func(idx int, _ struct{}) bool {
		return !a.isItemIndexSelectable(idx)
	})
	if !a.multiSelection && len(indices) > 1 {
		indices = map[int]struct{}{
			fastFirstIndex(indices): {},
		}
	}

	if updateAnchor {
		defer func() {
			if a.isItemIndexSelectable(newAnchorCandidate) {
				a.anchorIndex = newAnchorCandidate
			} else if idx := fastFirstIndex(indices); idx >= 0 {
				a.anchorIndex = idx
			} else {
				a.anchorIndex = 0
			}
			a.lastExtendIndexPlus1 = 0
		}()
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
		return
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
	a.refreshSelectionString()
}

func (a *abstractList[Value, Item]) SelectItemByValue(value Value, forceFireEvents bool) {
	idx := slices.IndexFunc(a.items, func(item Item) bool {
		return item.value() == value
	})
	a.SelectItemByIndex(idx, forceFireEvents)
}

func (a *abstractList[Value, Item]) SelectItemsByValues(values []Value, forceFireEvents bool) {
	a.tmpIndexSlice = a.tmpIndexSlice[:0]
	for i, item := range a.items {
		if slices.Contains(values, item.value()) {
			a.tmpIndexSlice = append(a.tmpIndexSlice, i)
		}
	}
	a.SelectItemsByIndices(a.tmpIndexSlice, forceFireEvents)
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
	a.tmpIndexSlice = a.AppendSelectedItemIndices(a.tmpIndexSlice[:0])
	for _, idx := range a.tmpIndexSlice {
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

// SelectGroupAt selects the group that contains the given index.
// If index is not selected, select it and deselect all other items.
// If index is selected, keep it. And if the index is in a group, select all items in the group and deselect all other items.
// A group is a contiguous range of selected visible items.
func (a *abstractList[Value, Item]) SelectGroupAt(index int, forceFireEvents bool) {
	if !a.multiSelection {
		a.SelectItemByIndex(index, forceFireEvents)
		return
	}

	if index < 0 || index >= len(a.items) {
		a.SelectItemsByIndices(nil, forceFireEvents)
		return
	}

	if _, ok := a.selectedIndices[index]; !ok {
		a.SelectItemByIndex(index, forceFireEvents)
		return
	}

	// Use tmpIndexMap to collect group indices.
	clear(a.tmpIndexMap)
	if a.tmpIndexMap == nil {
		a.tmpIndexMap = map[int]struct{}{}
	}
	a.tmpIndexMap[index] = struct{}{}

	// Search backwards
	for i := index - 1; i >= 0; i-- {
		if !a.items[i].visible() {
			continue
		}
		if _, ok := a.selectedIndices[i]; !ok {
			break
		}
		a.tmpIndexMap[i] = struct{}{}
	}

	// Search forwards
	for i := index + 1; i < len(a.items); i++ {
		if !a.items[i].visible() {
			continue
		}
		if _, ok := a.selectedIndices[i]; !ok {
			break
		}
		a.tmpIndexMap[i] = struct{}{}
	}

	a.selectItemsByIndices(a.tmpIndexMap, index, true, forceFireEvents)
}
