// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidgetdraw

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

type RoundedRectBorderType int

const (
	RoundedRectBorderTypeRegular RoundedRectBorderType = RoundedRectBorderType(draw.RoundedRectBorderTypeRegular)
	RoundedRectBorderTypeInset   RoundedRectBorderType = RoundedRectBorderType(draw.RoundedRectBorderTypeInset)
	RoundedRectBorderTypeOutset  RoundedRectBorderType = RoundedRectBorderType(draw.RoundedRectBorderTypeOutset)
)

type Corners struct {
	TopStart    bool
	TopEnd      bool
	BottomStart bool
	BottomEnd   bool
}

func DrawRoundedRect(context *guigui.Context, dst *ebiten.Image, bounds image.Rectangle, clr color.Color, radius int) {
	draw.DrawRoundedRect(context, dst, bounds, clr, radius)
}

func DrawRoundedRectWithSharpCorners(context *guigui.Context, dst *ebiten.Image, bounds image.Rectangle, clr color.Color, radius int, sharpCorners Corners) {
	draw.DrawRoundedRectWithSharpCorners(context, dst, bounds, clr, radius, draw.Corners(sharpCorners))
}

func DrawRoundedRectBorder(context *guigui.Context, dst *ebiten.Image, bounds image.Rectangle, clr1, clr2 color.Color, radius int, borderWidth float32, borderType RoundedRectBorderType) {
	draw.DrawRoundedRectBorder(context, dst, bounds, clr1, clr2, radius, borderWidth, draw.RoundedRectBorderType(borderType))
}

func DrawRoundedRectBorderWithSharpCorners(context *guigui.Context, dst *ebiten.Image, bounds image.Rectangle, clr1, clr2 color.Color, radius int, borderWidth float32, borderType RoundedRectBorderType, sharpCorners Corners) {
	draw.DrawRoundedRectBorderWithSharpCorners(context, dst, bounds, clr1, clr2, radius, borderWidth, draw.RoundedRectBorderType(borderType), draw.Corners(sharpCorners))
}
