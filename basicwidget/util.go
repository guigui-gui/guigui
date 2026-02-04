// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
)

func adjustSliceSize[T any](slice []T, size int) []T {
	if len(slice) == size {
		return slice
	}
	if len(slice) < size {
		return slices.Grow(slice, size-len(slice))[:size]
	}
	return slices.Delete(slice, size, len(slice))
}

func MoveItemsInSlice[T any](slice []T, from int, count int, to int) int {
	if count < 0 {
		panic("basicwidget: count must be non-negative")
	}
	if count == 0 {
		return from
	}
	if from <= to && to <= from+count {
		return from
	}

	slices.Reverse(slice[from : from+count])
	if from < to {
		slices.Reverse(slice[from+count : to])
		slices.Reverse(slice[from:to])
	} else {
		slices.Reverse(slice[to:from])
		slices.Reverse(slice[to : from+count])
	}

	if from < to {
		return to - count
	}
	return to
}

func doubleClickLimitInTicks() int {
	return ebiten.TPS() / 2
}

func defaultIconSize(context *guigui.Context) int {
	return LineHeight(context)
}
