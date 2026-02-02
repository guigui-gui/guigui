// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"math"

	"github.com/guigui-gui/guigui"
)

const baseUnitSize = 24

func FontSize(context *guigui.Context) float64 {
	return baseUnitSize * context.Scale() * 1 / 2
}

func LineHeight(context *guigui.Context) int {
	return int(math.Ceil(baseUnitSize * context.Scale() * 3 / 4))
}

func UnitSize(context *guigui.Context) int {
	return int(baseUnitSize * context.Scale())
}

func RoundedCornerRadius(context *guigui.Context) int {
	return int(baseUnitSize * context.Scale() / 4)
}
