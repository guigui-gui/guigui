// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

const (
	baseButtonEventDown   = "down"
	baseButtonEventUp     = "up"
	baseButtonEventRepeat = "repeat"
)

type baseButton struct {
	guigui.DefaultWidget

	pressed         bool
	keepPressed     bool
	useAccentColor  bool
	borderInvisible bool
	prevHovered     bool
	sharpenCorners  basicwidgetdraw.Corners
	pairedButton    *baseButton
}

func (b *baseButton) SetOnDown(f func(context *guigui.Context)) {
	guigui.RegisterEventHandler(b, baseButtonEventDown, f)
}

func (b *baseButton) SetOnUp(f func(context *guigui.Context)) {
	guigui.RegisterEventHandler(b, baseButtonEventUp, f)
}

func (b *baseButton) setOnRepeat(f func(context *guigui.Context)) {
	guigui.RegisterEventHandler(b, baseButtonEventRepeat, f)
}

func (b *baseButton) setPairedButton(pair *baseButton) {
	b.pairedButton = pair
}

func (b *baseButton) setPressed(pressed bool) {
	if b.pressed == pressed {
		return
	}
	b.pressed = pressed
	guigui.RequestRedraw(b)
}

func (b *baseButton) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if hovered := widgetBounds.IsHitAtCursor(); b.prevHovered != hovered {
		b.prevHovered = hovered
		guigui.RequestRedraw(b)
	}
	return nil
}

func (b *baseButton) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if widgetBounds.IsHitAtCursor() {
		// IsMouseButtonJustPressed and IsMouseButtonJustReleased can be true at the same time as of Ebitengine v2.9.
		// Check both.
		var justPressedOrReleased bool
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			if b.keepPressed {
				return guigui.AbortHandlingInputByWidget(b)
			}
			context.SetFocused(b, true)
			b.setPressed(true)
			guigui.DispatchEventHandler(b, baseButtonEventDown)
			if isMouseButtonRepeating(ebiten.MouseButtonLeft) {
				guigui.DispatchEventHandler(b, baseButtonEventRepeat)
			}
			justPressedOrReleased = true
		}
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && b.pressed {
			if b.keepPressed {
				return guigui.AbortHandlingInputByWidget(b)
			}
			b.setPressed(false)
			guigui.DispatchEventHandler(b, baseButtonEventUp)
			guigui.RequestRedraw(b)
			justPressedOrReleased = true
		}
		if justPressedOrReleased {
			return guigui.HandleInputByWidget(b)
		}
		if (b.pressed || b.pairedButton != nil && b.pairedButton.pressed) && isMouseButtonRepeating(ebiten.MouseButtonLeft) {
			guigui.DispatchEventHandler(b, baseButtonEventRepeat)
			return guigui.HandleInputByWidget(b)
		}
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		b.setPressed(false)
	}
	return guigui.HandleInputResult{}
}

func (b *baseButton) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	if (b.canPress(context, widgetBounds) || b.pressed || b.pairedButton != nil && b.pairedButton.pressed) && !b.keepPressed {
		return ebiten.CursorShapePointer, true
	}
	return 0, true
}

func (b *baseButton) radius(context *guigui.Context, widgetBounds *guigui.WidgetBounds) int {
	size := widgetBounds.Bounds().Size()
	return min(RoundedCornerRadius(context), size.X/4, size.Y/4)
}

func (b *baseButton) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	cm := context.ColorMode()
	backgroundColor := basicwidgetdraw.ControlColor(context.ColorMode(), context.IsEnabled(b))
	if context.IsEnabled(b) {
		if b.isPressed(context, widgetBounds) {
			if b.useAccentColor {
				backgroundColor = draw.Color2(cm, draw.ColorTypeAccent, 0.875, 0.5)
			} else {
				backgroundColor = draw.Color2(cm, draw.ColorTypeBase, 0.95, 0.25)
			}
		} else if b.canPress(context, widgetBounds) {
			backgroundColor = draw.Color2(cm, draw.ColorTypeBase, 0.975, 0.275)
		}
	}

	r := b.radius(context, widgetBounds)
	border := !b.borderInvisible
	if context.IsEnabled(b) && (widgetBounds.IsHitAtCursor() || b.keepPressed) {
		border = true
	}
	bounds := widgetBounds.Bounds()
	if border || b.isPressed(context, widgetBounds) {
		basicwidgetdraw.DrawRoundedRectWithSharpenCorners(context, dst, bounds, backgroundColor, r, b.sharpenCorners)
	}

	if border {
		borderType := basicwidgetdraw.RoundedRectBorderTypeOutset
		if b.isPressed(context, widgetBounds) {
			borderType = basicwidgetdraw.RoundedRectBorderTypeInset
		}
		clr1, clr2 := basicwidgetdraw.BorderColors(context.ColorMode(), basicwidgetdraw.RoundedRectBorderType(borderType), b.useAccentColor && b.isPressed(context, widgetBounds) && context.IsEnabled(b))
		basicwidgetdraw.DrawRoundedRectBorderWithSharpenCorners(context, dst, bounds, clr1, clr2, r, float32(1*context.Scale()), borderType, b.sharpenCorners)
	}
}

func (b *baseButton) canPress(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(b) && widgetBounds.IsHitAtCursor() && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && !b.keepPressed
}

func (b *baseButton) isActive(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(b) && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && widgetBounds.IsHitAtCursor() && (b.pressed || b.pairedButton != nil && b.pairedButton.pressed)
}

func (b *baseButton) isPressed(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(b) && b.isActive(context, widgetBounds) || b.keepPressed
}

func (b *baseButton) setKeepPressed(keep bool) {
	if b.keepPressed == keep {
		return
	}
	b.keepPressed = keep
	guigui.RequestRedraw(b)
}

func defaultButtonSize(context *guigui.Context) image.Point {
	return image.Pt(6*UnitSize(context), UnitSize(context))
}

func (b *baseButton) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return defaultButtonSize(context)
}

func (b *baseButton) setSharpenCorners(sharpenCorners basicwidgetdraw.Corners) {
	if b.sharpenCorners == sharpenCorners {
		return
	}
	b.sharpenCorners = sharpenCorners
	guigui.RequestRedraw(b)
}

func (b *baseButton) setUseAccentColor(use bool) {
	if b.useAccentColor == use {
		return
	}
	b.useAccentColor = use
	guigui.RequestRedraw(b)
}
