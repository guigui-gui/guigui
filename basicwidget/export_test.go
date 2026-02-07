// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

func ReplaceNewLinesWithSpace(text string, start, end int) (string, int, int) {
	return replaceNewLinesWithSpace(text, start, end)
}

type AbstractListValuer[T comparable] interface {
	valuer[T]
}

type AbstractList[Value comparable, Item AbstractListValuer[Value]] struct {
	abstractList[Value, Item]
}

type AbstractListTestItem[T comparable] struct {
	Value      T
	Selectable bool
}

func (a AbstractListTestItem[T]) value() T {
	return a.Value
}

func (a AbstractListTestItem[T]) selectable() bool {
	return a.Selectable
}
