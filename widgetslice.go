// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package guigui

import "slices"

// WidgetSlice is a collection of widgets.
//
// As Widget implementation (DefaultWidget) must not be copied by value,
// a plain slice of widgets is very risky to use. Use this instead.
type WidgetSlice[T Widget] struct {
	s []lazyWidget[T]
}

// At returns the widget at the specified index.
func (w *WidgetSlice[T]) At(index int) T {
	return w.s[index].Widget()
}

// Len returns the number of widgets.
func (w *WidgetSlice[T]) Len() int {
	return len(w.s)
}

// SetLen sets the length of the slice.
//
// If the length is increased, the new elements are zero-cleared values. The existing elements are kept.
//
// If the length is decreased, the elements are dropped. The remaining elements are kept.
func (w *WidgetSlice[T]) SetLen(l int) {
	if len(w.s) == l {
		return
	}
	if len(w.s) < l {
		w.s = slices.Grow(w.s, l-len(w.s))[:l]
		return
	}
	w.s = slices.Delete(w.s, l, len(w.s))
}
