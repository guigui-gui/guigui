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
		{Value: "foo", Selectable: true},
		{Value: "bar", Selectable: true},
		{Value: "baz", Selectable: false},
		{Value: "qux", Selectable: true},
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
		{Value: "foo", Selectable: true},
		{Value: "bar", Selectable: true},
		{Value: "baz", Selectable: false},
		{Value: "qux", Selectable: true},
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
