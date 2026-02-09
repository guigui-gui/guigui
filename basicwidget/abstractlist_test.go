// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget_test

import (
	"slices"
	"testing"

	"github.com/guigui-gui/guigui/basicwidget"
)

func TestAbstractList(t *testing.T) {
	type Item = basicwidget.AbstractListTestItem[string]
	var l basicwidget.AbstractList[string, Item]

	items := []Item{
		{Value: "foo", Selectable: true, Visible: true},
		{Value: "bar", Selectable: true, Visible: true},
		{Value: "baz", Selectable: false, Visible: true},
		{Value: "qux", Selectable: true, Visible: true},
	}
	l.SetItems(items)

	if got, want := l.ItemCount(), 4; got != want {
		t.Errorf("got %d, want %d", got, want)
	}

	// Test ItemByIndex
	for i, item := range items {
		got, ok := l.ItemByIndex(i)
		if !ok {
			t.Errorf("ItemByIndex(%d) returned false", i)
		}
		if got != item {
			t.Errorf("ItemByIndex(%d) = %v, want %v", i, got, item)
		}
	}
	if _, ok := l.ItemByIndex(-1); ok {
		t.Error("ItemByIndex(-1) returned true")
	}
	if _, ok := l.ItemByIndex(4); ok {
		t.Error("ItemByIndex(4) returned true")
	}

	// Test SelectItemByIndex
	l.SelectItemByIndex(0, false)
	if got, want := l.SelectedItemIndex(), 0; got != want {
		t.Errorf("SelectedItemIndex() = %d, want %d", got, want)
	}
	if got, want := l.SelectedItemCount(), 1; got != want {
		t.Errorf("SelectedItemCount() = %d, want %d", got, want)
	}
	if item, ok := l.SelectedItem(); !ok || item.Value != "foo" {
		t.Errorf("SelectedItem() = %v, %v; want {1, true}, true", item, ok)
	}
	l.SelectItemByIndex(2, false) // Index 2 is not selectable.
	if got, want := l.SelectedItemIndex(), -1; got != want {
		t.Errorf("SelectedItemIndex() after SelectItemByIndex(2) = %d, want %d", got, want)
	}
	if got, want := l.SelectedItemCount(), 0; got != want {
		t.Errorf("SelectedItemCount() after SelectItemByIndex(2) = %d, want %d", got, want)
	}
	if item, ok := l.SelectedItem(); ok {
		t.Errorf("SelectedItem() after SelectItemByIndex(2) = %v, %v; want nil, false", item, ok)
	}
	l.SelectItemByIndex(99, false) // Index 99 is out of bounds.
	if got, want := l.SelectedItemIndex(), -1; got != want {
		t.Errorf("SelectedItemIndex() after SelectItemByIndex(99) = %d, want %d", got, want)
	}
	if got, want := l.SelectedItemCount(), 0; got != want {
		t.Errorf("SelectedItemCount() after SelectItemByIndex(99) = %d, want %d", got, want)
	}
	if item, ok := l.SelectedItem(); ok {
		t.Errorf("SelectedItem() after SelectItemByIndex(99) = %v, %v; want nil, false", item, ok)
	}

	// Test SelectItemByValue
	l.SelectItemByValue("qux", false)
	if got, want := l.SelectedItemIndex(), 3; got != want {
		t.Errorf("SelectedItemIndex() after SelectItemByValue(4) = %d, want %d", got, want)
	}
	l.SelectItemByValue("baz", false) // "baz" is not selectable.
	if got, want := l.SelectedItemIndex(), -1; got != want {
		t.Errorf("SelectedItemIndex() after SelectItemByValue(baz) = %d, want %d", got, want)
	}
	l.SelectItemByValue("xyz", false) // "xyz" does not exist.
	if got, want := l.SelectedItemIndex(), -1; got != want {
		t.Errorf("SelectedItemIndex() after SelectItemByValue(baz) = %d, want %d", got, want)
	}

	// Test SelectItemsByValues with single selection (default)
	l.SelectItemsByValues([]string{"foo", "qux"}, false)
	if got, want := l.SelectedItemIndex(), 0; got != want {
		t.Errorf("got %d, want %d", got, want)
	}
	if got, want := l.SelectedItemCount(), 1; got != want {
		t.Errorf("got %d, want %d", got, want)
	}

	// Test SetMultiSelection
	l.SetMultiSelection(true)
	l.SelectItemsByValues([]string{"foo", "qux"}, false)
	if got, want := l.SelectedItemCount(), 2; got != want {
		t.Errorf("SelectedItemCount() = %d, want %d", got, want)
	}
	indices := l.AppendSelectedItemIndices(nil)
	if got, want := indices, []int{0, 3}; !slices.Equal(got, want) {
		t.Errorf("AppendSelectedItemIndices = %v, want %v", got, want)
	}

	// Test disabling SetMultiSelection
	l.SetMultiSelection(false)
	if got, want := l.SelectedItemCount(), 1; got != want {
		t.Errorf("SelectedItemCount() = %d, want %d", got, want)
	}
	if got, want := l.SelectedItemIndex(), 0; got != want {
		t.Errorf("SelectedItemIndex() = %d, want %d", got, want)
	}

	// Test SelectItemsByIndices with multi-selection disabled
	l.SelectItemsByIndices([]int{0, 2, 3}, false) // Index 2 is not selectable.
	if got, want := l.SelectedItemCount(), 1; got != want {
		t.Errorf("SelectedItemCount() = %d, want %d", got, want)
	}
	if got, want := l.SelectedItemIndex(), 0; got != want {
		t.Errorf("SelectedItemIndex() = %d, want %d", got, want)
	}

	// Test SelectItemsByValues
	l.SetMultiSelection(true)
	l.SelectItemsByIndices([]int{0, 2, 3}, false) // Index 2 is not selectable.
	if got, want := l.SelectedItemCount(), 2; got != want {
		t.Errorf("SelectedItemCount() = %d, want %d", got, want)
	}
	indices = l.AppendSelectedItemIndices(nil)
	if !slices.Equal(indices, []int{0, 3}) {
		t.Errorf("AppendSelectedItemIndices = %v, want [0, 3]", indices)
	}
	l.SelectItemsByValues([]string{"foo", "qux"}, false)
	if got, want := l.SelectedItemCount(), 2; got != want {
		t.Errorf("SelectedItemCount() = %d, want %d", got, want)
	}
	indices = l.AppendSelectedItemIndices(nil)
	if !slices.Equal(indices, []int{0, 3}) {
		t.Errorf("AppendSelectedItemIndices = %v, want [0, 3]", indices)
	}
	l.SelectItemsByValues([]string{"foo", "baz", "qux"}, false) // "baz" is not selectable.
	if got, want := l.SelectedItemCount(), 2; got != want {
		t.Errorf("SelectedItemCount() = %d, want %d", got, want)
	}
	indices = l.AppendSelectedItemIndices(nil)
	if !slices.Equal(indices, []int{0, 3}) {
		t.Errorf("AppendSelectedItemIndices = %v, want [0, 3]", indices)
	}
	l.SelectItemsByValues([]string{"foo", "bar", "xyz"}, false) // "xyz" does not exist.
	if got, want := l.SelectedItemCount(), 2; got != want {
		t.Errorf("SelectedItemCount() = %d, want %d", got, want)
	}
	indices = l.AppendSelectedItemIndices(nil)
	if !slices.Equal(indices, []int{0, 1}) {
		t.Errorf("AppendSelectedItemIndices = %v, want [0, 1]", indices)
	}

	// Test Callbacks
	var selectedIndex int = -1
	var selectedIndices []int
	l.SetOnItemSelected(func(index int) {
		selectedIndex = index
	})
	l.SetOnItemsSelected(func(indices []int) {
		selectedIndices = slices.Clone(indices)
	})

	l.SelectItemByIndex(1, true)
	if selectedIndex != 1 {
		t.Errorf("OnItemSelected not called: got %d, want 1", selectedIndex)
	}
	if !slices.Equal(selectedIndices, []int{1}) {
		t.Errorf("OnItemsSelected not called: got %v, want [1]", selectedIndices)
	}

	l.SelectItemsByIndices([]int{1, 2, 3, 99}, true)
	if selectedIndex != 1 {
		t.Errorf("OnItemSelected not called: got %d, want 1", selectedIndex)
	}
	if !slices.Equal(selectedIndices, []int{1, 3}) {
		t.Errorf("OnItemsSelected not called: got %v, want [1, 3]", selectedIndices)
	}
}

func TestAbstractListSelectionSliceCopy(t *testing.T) {
	type Item = basicwidget.AbstractListTestItem[string]
	var l basicwidget.AbstractList[string, Item]

	items := []Item{
		{Value: "foo", Selectable: true, Visible: true},
		{Value: "bar", Selectable: true, Visible: true},
		{Value: "baz", Selectable: false, Visible: true},
		{Value: "qux", Selectable: true, Visible: true},
	}
	l.SetItems(items)
	l.SetMultiSelection(true)

	var receivedIndices []int
	l.SetOnItemsSelected(func(indices []int) {
		// Modify the received slice to verify it's a copy
		if len(indices) > 0 {
			indices[0] = 999
		}
		receivedIndices = indices
	})

	// Select items.
	inputIndices := []int{1, 0}
	l.SelectItemsByIndices(inputIndices, true)

	if receivedIndices == nil {
		t.Fatal("onItemsSelected was not called")
	}
	if !slices.Equal(receivedIndices, []int{999, 1}) {
		t.Errorf("receivedIndices: got %v, want [999, 1]", receivedIndices)
	}
	if !slices.Equal(inputIndices, []int{1, 0}) {
		t.Errorf("inputIndices: got %v, want [1, 0]", inputIndices)
	}

	// Select the same items again.
	inputIndices2 := []int{1, 0}
	receivedIndices = nil
	l.SelectItemsByIndices(inputIndices2, true)

	if receivedIndices == nil {
		t.Fatal("onItemsSelected was not called (forceFireEvents)")
	}
	if !slices.Equal(receivedIndices, []int{999, 1}) {
		t.Errorf("receivedIndices: got %v, want [999, 1]", receivedIndices)
	}
	if !slices.Equal(inputIndices2, []int{1, 0}) {
		t.Errorf("inputIndices2: got %v, want [1, 0]", inputIndices2)
	}
}

func TestAbstractListToggleItemSelectionByIndex(t *testing.T) {
	type Item = basicwidget.AbstractListTestItem[string]
	var l basicwidget.AbstractList[string, Item]

	items := []Item{
		{Value: "foo", Selectable: true, Visible: true},
		{Value: "bar", Selectable: true, Visible: true},
		{Value: "baz", Selectable: false, Visible: true},
		{Value: "qux", Selectable: true, Visible: true},
	}
	l.SetItems(items)

	// Test Single Selection Mode (default)
	l.ToggleItemSelectionByIndex(0, false)
	if got, want := l.SelectedItemIndex(), 0; got != want {
		t.Errorf("Single: Toggle(0) index = %d, want %d", got, want)
	}
	if got, want := l.SelectedItemCount(), 1; got != want {
		t.Errorf("Single: Toggle(0) count = %d, want %d", got, want)
	}
	l.ToggleItemSelectionByIndex(0, false)
	if got, want := l.SelectedItemIndex(), -1; got != want {
		t.Errorf("Single: Toggle(0) again index = %d, want %d", got, want)
	}
	if got, want := l.SelectedItemCount(), 0; got != want {
		t.Errorf("Single: Toggle(0) count = %d, want %d", got, want)
	}

	l.ToggleItemSelectionByIndex(0, false)
	l.ToggleItemSelectionByIndex(1, false)
	if got, want := l.SelectedItemIndex(), 1; got != want {
		t.Errorf("Single: Toggle(1) after (0) index = %d, want %d", got, want)
	}
	if got, want := l.SelectedItemCount(), 1; got != want {
		t.Errorf("Single: Toggle(1) after (0) count = %d, want %d", got, want)
	}

	// Test Multi Selection Mode
	l.SetMultiSelection(true)
	l.SelectItemsByIndices(nil, false)

	l.ToggleItemSelectionByIndex(0, false)
	if got, want := l.SelectedItemIndex(), 0; got != want {
		t.Errorf("Multi: Toggle(0) index = %d, want %d", got, want)
	}
	if got, want := l.SelectedItemCount(), 1; got != want {
		t.Errorf("Multi: Toggle(0) count = %d, want %d", got, want)
	}
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{0}) {
		t.Errorf("Multi: Indices = %v, want [0]", indices)
	}

	l.ToggleItemSelectionByIndex(3, false)
	if got, want := l.SelectedItemCount(), 2; got != want {
		t.Errorf("Multi: Toggle(3) count = %d, want %d", got, want)
	}
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{0, 3}) {
		t.Errorf("Multi: Indices = %v, want [0, 3]", indices)
	}

	l.ToggleItemSelectionByIndex(0, false)
	if got, want := l.SelectedItemCount(), 1; got != want {
		t.Errorf("Multi: Toggle(0) again count = %d, want %d", got, want)
	}
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{3}) {
		t.Errorf("Multi: Indices = %v, want [3]", indices)
	}

	l.ToggleItemSelectionByIndex(2, false) // Non-selectable item
	if got, want := l.SelectedItemCount(), 1; got != want {
		t.Errorf("Multi: Toggle(2) count = %d, want %d", got, want)
	}
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{3}) {
		t.Errorf("Multi: Indices = %v, want [3]", indices)
	}

	l.ToggleItemSelectionByIndex(99, false) // Out of bounds
	if got, want := l.SelectedItemCount(), 1; got != want {
		t.Errorf("Multi: Toggle(99) count = %d, want %d", got, want)
	}
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{3}) {
		t.Errorf("Multi: Indices = %v, want [3]", indices)
	}
}

func TestAbstractListExtendItemSelectionByIndex(t *testing.T) {
	type Item = basicwidget.AbstractListTestItem[string]
	var l basicwidget.AbstractList[string, Item]

	items := []Item{
		{Value: "0", Selectable: true, Visible: true},
		{Value: "1", Selectable: true, Visible: true},
		{Value: "2", Selectable: true, Visible: true},
		{Value: "3", Selectable: true, Visible: true},
		{Value: "4", Selectable: true, Visible: true},
		{Value: "5", Selectable: false, Visible: true},
		{Value: "6", Selectable: true, Visible: true},
	}
	l.SetItems(items)
	l.SetMultiSelection(true)

	// Extend selection from the anchor (0) to 2.
	// As the anchor is 0, the items 0, 1, and 2 should be selected.
	l.ExtendItemSelectionByIndex(2, false)
	if got, want := l.SelectedItemCount(), 3; got != want {
		t.Errorf("SelectedItemCount = %d, want %d", got, want)
	}
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{0, 1, 2}) {
		t.Errorf("Indices = %v, want [0, 1, 2]", indices)
	}

	// Extend selection from the anchor (0) to 4.
	// As the anchor is 0, the items 0, 1, 2, 3, and 4 should be selected.
	// The previous extension (0, 1, 2) should be cleared.
	l.ExtendItemSelectionByIndex(4, false)
	if got, want := l.SelectedItemCount(), 5; got != want {
		t.Errorf("SelectedItemCount = %d, want %d", got, want)
	}
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{0, 1, 2, 3, 4}) {
		t.Errorf("Indices = %v, want [0, 1, 2, 3, 4]", indices)
	}

	// Extend selection from the anchor (0) to 1.
	// As the anchor is 0, the items 0 and 1 should be selected.
	// The previous extension (0, 1, 2, 3, 4) should be cleared.
	l.ExtendItemSelectionByIndex(1, false)
	if got, want := l.SelectedItemCount(), 2; got != want {
		t.Errorf("SelectedItemCount = %d, want %d", got, want)
	}
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{0, 1}) {
		t.Errorf("Indices = %v, want [0, 1]", indices)
	}

	// Select 3. The anchor should be updated to 3.
	// The existing selection should be cleared.
	l.SelectItemByIndex(3, false)
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{3}) {
		t.Errorf("Indices = %v, want [3]", indices)
	}

	// Extend selection from the anchor (3) to 1.
	// As the anchor is 3, the items 1, 2, and 3 should be selected.
	l.ExtendItemSelectionByIndex(1, false)
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{1, 2, 3}) {
		t.Errorf("Indices = %v, want [1, 2, 3]", indices)
	}

	// Extend selection from the anchor (3) to 6.
	// As the anchor is 3, the items 3, 4, and 6 should be selected.
	// Item 5 is not selectable.
	// The previous extension (1, 2, 3) should be cleared.
	l.ExtendItemSelectionByIndex(6, false)
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{3, 4, 6}) {
		t.Errorf("Indices = %v, want [3, 4, 6]", indices)
	}

	// Toggle 0. The anchor should be updated to 0.
	// Item 0 should be added to the selection.
	l.ToggleItemSelectionByIndex(0, false)
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{0, 3, 4, 6}) {
		t.Errorf("Indices = %v, want [0, 3, 4, 6]", indices)
	}

	// Extend selection from the anchor (0) to 2.
	// As the anchor is 0, the items 0, 1, and 2 should be selected.
	// This extension should be added to the existing selection (3, 4, 6).
	// There is no previous extension to clear as ToggleItemSelectionByIndex resets the last extension.
	l.ExtendItemSelectionByIndex(2, false)
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{0, 1, 2, 3, 4, 6}) {
		t.Errorf("Indices = %v, want [0, 1, 2, 3, 4, 6]", indices)
	}

	// Toggle 0. The anchor should be updated to 1 (the minimum index of the remaining selection).
	// Item 0 should be removed from the selection.
	l.ToggleItemSelectionByIndex(0, false)
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{1, 2, 3, 4, 6}) {
		t.Errorf("Indices = %v, want [1, 2, 3, 4, 6]", indices)
	}

	// Extend selection from the anchor (1) to 3.
	// As the anchor is 1, the items 1, 2, and 3 should be selected.
	// This extension should be added to the existing selection (4, 6).
	// There is no previous extension to clear as ToggleItemSelectionByIndex resets the last extension.
	l.ExtendItemSelectionByIndex(3, false)
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{1, 2, 3, 4, 6}) {
		t.Errorf("Indices = %v, want [1, 2, 3, 4, 6]", indices)
	}

	// Extend selection from the anchor (1) to 4.
	// As the anchor is 1, the items 1, 2, 3, and 4 should be selected.
	// The previous extension (1, 2, 3) should be cleared.
	// The item 6 should be kept selected.
	l.ExtendItemSelectionByIndex(4, false)
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{1, 2, 3, 4, 6}) {
		t.Errorf("Indices = %v, want [1, 2, 3, 4, 6]", indices)
	}

	// Switch to single selection mode.
	// The selection should be updated to the anchor (1).
	l.SetMultiSelection(false)
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{1}) {
		t.Errorf("Single selection: Indices = %v, want [1]", indices)
	}

	l.ExtendItemSelectionByIndex(3, false)
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{3}) {
		t.Errorf("Single selection: Indices = %v, want [3]", indices)
	}
}

func TestAbstractListSelectGroupAt(t *testing.T) {
	type Item = basicwidget.AbstractListTestItem[string]
	var l basicwidget.AbstractList[string, Item]

	items := []Item{
		{Value: "0", Selectable: true, Visible: true},
		{Value: "1", Selectable: true, Visible: true},
		{Value: "2", Selectable: true, Visible: true},
		{Value: "3", Selectable: true, Visible: true},
		{Value: "4", Selectable: true, Visible: true},
		{Value: "5", Selectable: true, Visible: false},
		{Value: "6", Selectable: true, Visible: true},
		{Value: "7", Selectable: true, Visible: true},
	}
	l.SetItems(items)
	l.SetMultiSelection(true)

	// Click on unselected item (2).
	l.SelectGroupAt(2, false)
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{2}) {
		t.Errorf("Indices = %v, want [2]", indices)
	}

	// Click on selected item (2) which is part of a group (0, 1, 2, 3).
	l.SelectItemsByIndices([]int{0, 1, 2, 3, 7}, false)
	l.SelectGroupAt(2, false)
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{0, 1, 2, 3}) {
		t.Errorf("Indices = %v, want [0, 1, 2, 3]", indices)
	}

	// Verify hidden items are skipped.
	l.SelectItemsByIndices([]int{3, 4, 5, 6}, false)
	l.SelectGroupAt(4, false)
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{3, 4, 6}) {
		t.Errorf("Indices = %v, want [3, 4, 6]", indices)
	}

	// Single selection mode fallback.
	l.SetMultiSelection(false)
	l.SelectItemsByIndices([]int{0, 2}, false)
	l.SelectGroupAt(2, false)
	if indices := l.AppendSelectedItemIndices(nil); !slices.Equal(indices, []int{2}) {
		t.Errorf("Indices = %v, want [2]", indices)
	}

	// Out of bounds index should deselect everything.
	l.SetMultiSelection(true)
	l.SelectItemsByIndices([]int{0, 1}, false)
	l.SelectGroupAt(-1, false)
	if got := l.SelectedItemCount(); got != 0 {
		t.Errorf("SelectedItemCount = %d, want 0", got)
	}
}
