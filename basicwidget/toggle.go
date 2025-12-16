// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

const (
	toggleEventValueChanged = "valueChanged"
)

type Toggle struct {
	guigui.DefaultWidget

	pressed      bool
	value        bool
	onceRendered bool
	prevHovered  bool

	count int
}

func (t *Toggle) SetOnValueChanged(f func(context *guigui.Context, value bool)) {
	guigui.SetEventHandler(t, toggleEventValueChanged, f)
}

func (t *Toggle) Value() bool {
	return t.value
}

func (t *Toggle) SetValue(value bool) {
	if t.value == value {
		return
	}

	t.value = value
	if t.onceRendered {
		t.count = toggleMaxCount() - t.count
	}
	guigui.RequestRedraw(t)

	guigui.DispatchEvent(t, toggleEventValueChanged, value)
}

func toggleMaxCount() int {
	return ebiten.TPS() / 12
}

func (t *Toggle) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if context.IsEnabled(t) && widgetBounds.IsHitAtCursor() && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		context.SetFocused(t, true)
		t.pressed = true
		t.SetValue(!t.value)
		return guigui.HandleInputByWidget(t)
	}
	if !context.IsEnabled(t) || !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		t.pressed = false
	}
	return guigui.HandleInputResult{}
}

func (t *Toggle) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if t.count > 0 {
		t.count--
		guigui.RequestRedraw(t)
	}
	if hovered := widgetBounds.IsHitAtCursor(); t.prevHovered != hovered {
		t.prevHovered = hovered
		guigui.RequestRedraw(t)
	}
	return nil
}

func (t *Toggle) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	if t.canPress(context, widgetBounds) || t.pressed {
		return ebiten.CursorShapePointer, true
	}
	return 0, true
}

func (t *Toggle) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	rate := 1 - float64(t.count)/float64(toggleMaxCount())

	bounds := widgetBounds.Bounds()

	cm := context.ColorMode()
	backgroundColor := draw.Color(context.ColorMode(), draw.ColorTypeBase, 0.8)
	thumbColor := basicwidgetdraw.ThumbColor(context.ColorMode(), context.IsEnabled(t))
	if t.isActive(context, widgetBounds) {
		thumbColor = draw.Color2(cm, draw.ColorTypeBase, 0.95, 0.55)
	} else if t.canPress(context, widgetBounds) {
		thumbColor = draw.Color2(cm, draw.ColorTypeBase, 0.975, 0.575)
	}

	// Background
	bgColorOff := backgroundColor
	bgColorOn := draw.Color(context.ColorMode(), draw.ColorTypeAccent, 0.5)
	var bgColor color.Color
	if t.value && context.IsEnabled(t) {
		bgColor = draw.MixColors(bgColorOff, bgColorOn, rate)
	} else {
		bgColor = draw.MixColors(bgColorOn, bgColorOff, rate)
	}
	r := bounds.Dy() / 2
	basicwidgetdraw.DrawRoundedRect(context, dst, bounds, bgColor, r)

	// Border (upper)
	b := bounds
	halfHeight := b.Dy() / 2
	b.Max.Y = b.Min.Y + halfHeight
	strokeWidth := float32(1 * context.Scale())
	borderClr1, borderClr2 := basicwidgetdraw.BorderColors(context.ColorMode(), basicwidgetdraw.RoundedRectBorderTypeInset, t.value && context.IsEnabled(t))
	basicwidgetdraw.DrawRoundedRectBorder(context, dst.SubImage(b).(*ebiten.Image), bounds, borderClr1, borderClr2, r, strokeWidth, basicwidgetdraw.RoundedRectBorderTypeInset)

	// Thumb
	cxOff := float64(bounds.Min.X) + float64(r)
	cxOn := float64(bounds.Max.X) - float64(r)
	var cx int
	if t.value {
		cx = int((1-rate)*cxOff + rate*cxOn)
	} else {
		cx = int((1-rate)*cxOn + rate*cxOff)
	}
	cy := bounds.Min.Y + r
	thumbClr1, thumbClr2 := basicwidgetdraw.BorderColors(context.ColorMode(), basicwidgetdraw.RoundedRectBorderTypeOutset, false)
	thumbBounds := image.Rect(cx-r, cy-r, cx+r, cy+r)
	basicwidgetdraw.DrawRoundedRect(context, dst, thumbBounds, thumbColor, r)
	basicwidgetdraw.DrawRoundedRectBorder(context, dst, thumbBounds, thumbClr1, thumbClr2, r, strokeWidth, basicwidgetdraw.RoundedRectBorderTypeOutset)

	// Border (lower)
	b = bounds
	b.Min.Y += halfHeight
	basicwidgetdraw.DrawRoundedRectBorder(context, dst.SubImage(b).(*ebiten.Image), bounds, borderClr1, borderClr2, r, strokeWidth, basicwidgetdraw.RoundedRectBorderTypeInset)

	t.onceRendered = true
}

func (t *Toggle) canPress(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(t) && widgetBounds.IsHitAtCursor() && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
}

func (t *Toggle) isActive(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(t) && widgetBounds.IsHitAtCursor() && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && t.pressed
}

func (t *Toggle) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return image.Pt(int(LineHeight(context)*1.75), int(LineHeight(context)))
}
