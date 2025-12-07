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
	items           []Item
	selectedIndices []int
}

func (a *abstractList[Value, Item]) SetOnItemSelected(widget guigui.Widget, f func(index int)) {
	guigui.RegisterEventHandler(widget, abstractListEventItemSelected, f)
}

func (a *abstractList[Value, Item]) SetItems(widget guigui.Widget, items []Item) {
	a.items = adjustSliceSize(items, len(items))
	copy(a.items, items)
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

	if len(a.selectedIndices) == 1 && a.selectedIndices[0] == index && !forceFireEvents {
		return false
	}

	selected := slices.Contains(a.selectedIndices, index)
	a.selectedIndices = adjustSliceSize(a.selectedIndices, 1)
	a.selectedIndices[0] = index
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
