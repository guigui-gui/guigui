// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

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

var (
	checkboxEventValueChanged guigui.EventKey = guigui.GenerateEventKey()
)

type Checkbox struct {
	guigui.DefaultWidget

	image Image

	pressed     bool
	value       bool
	prevHovered bool
}

func (c *Checkbox) OnValueChanged(f func(context *guigui.Context, value bool)) {
	guigui.SetEventHandler(c, checkboxEventValueChanged, f)
}

func (c *Checkbox) Value() bool {
	return c.value
}

func (c *Checkbox) SetValue(value bool) {
	if c.value == value {
		return
	}

	c.value = value
	guigui.RequestRebuild(c)

	guigui.DispatchEvent(c, checkboxEventValueChanged, value)
}

func (c *Checkbox) setPressed(pressed bool) {
	if c.pressed == pressed {
		return
	}
	c.pressed = pressed
	guigui.RequestRedraw(c)
}

func (c *Checkbox) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if context.IsEnabled(c) && widgetBounds.IsHitAtCursor() {
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			c.SetValue(!c.value)
			c.setPressed(false)
			return guigui.HandleInputByWidget(c)
		}
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			context.SetFocused(c, true)
			c.setPressed(true)
			return guigui.HandleInputByWidget(c)
		}
	}
	if !context.IsEnabled(c) || !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		c.setPressed(false)
	}
	return guigui.HandleInputResult{}
}

func (c *Checkbox) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if hovered := widgetBounds.IsHitAtCursor(); c.prevHovered != hovered {
		c.prevHovered = hovered
		guigui.RequestRedraw(c)
	}
	return nil
}

func (c *Checkbox) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	if c.canPress(context, widgetBounds) || c.pressed {
		return ebiten.CursorShapePointer, true
	}
	return 0, true
}

func (c *Checkbox) canPress(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(c) && widgetBounds.IsHitAtCursor() && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
}

func (c *Checkbox) isActive(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(c) && widgetBounds.IsHitAtCursor() && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && c.pressed
}

func (c *Checkbox) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if c.value {
		adder.AddWidget(&c.image)
	}

	imageCM := ebiten.ColorModeDark
	if context.ResolvedColorMode() == ebiten.ColorModeLight && !context.IsEnabled(c) {
		imageCM = ebiten.ColorModeLight
	}
	checkImg, err := theResourceImages.Get("check", imageCM)
	if err != nil {
		return err
	}
	c.image.SetImage(checkImg)

	return nil
}

func (c *Checkbox) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	bounds := widgetBounds.Bounds()
	checkboxSize := LineHeight(context)
	markSize := float64(checkboxSize) * 0.8

	pt := bounds.Min
	pt.X += int((float64(checkboxSize) - markSize) / 2)
	// Adjust the position a bit for better appearance.
	pt.Y += int((float64(checkboxSize)-markSize)/2 + 0.5*context.Scale() + float64(UnitSize(context))/16)
	imgBounds := image.Rectangle{
		Min: pt,
		Max: pt.Add(image.Pt(int(markSize), int(markSize))),
	}
	layouter.LayoutWidget(&c.image, imgBounds)
}

func (c *Checkbox) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	pt := widgetBounds.Bounds().Min
	bounds := image.Rectangle{
		Min: pt,
		Max: pt.Add(image.Pt(LineHeight(context), LineHeight(context))),
	}
	cm := context.ResolvedColorMode()
	r := UnitSize(context) / 8

	var backgroundColor color.Color
	if context.IsEnabled(c) {
		if c.value {
			if c.isActive(context, widgetBounds) {
				backgroundColor = draw.Color2(cm, draw.ColorTypeAccent, 0.45, 0.45)
			} else {
				backgroundColor = draw.Color(cm, draw.ColorTypeAccent, 0.5)
			}
		} else {
			if c.isActive(context, widgetBounds) {
				backgroundColor = basicwidgetdraw.ControlSecondaryColor(cm, true)
			} else {
				backgroundColor = basicwidgetdraw.ControlColor(cm, true)
			}
		}
	} else {
		backgroundColor = basicwidgetdraw.ControlColor(cm, false)
	}
	basicwidgetdraw.DrawRoundedRect(context, dst, bounds, backgroundColor, r)

	// Border
	strokeWidth := float32(1 * context.Scale())
	var borderClr1, borderClr2 color.Color
	if c.value && context.IsEnabled(c) {
		borderClr1, borderClr2 = basicwidgetdraw.BorderAccentSecondaryColors(cm, basicwidgetdraw.RoundedRectBorderTypeInset)
	} else {
		borderClr1, borderClr2 = basicwidgetdraw.BorderColors(cm, basicwidgetdraw.RoundedRectBorderTypeInset)
	}
	basicwidgetdraw.DrawRoundedRectBorder(context, dst, bounds, borderClr1, borderClr2, r, strokeWidth, basicwidgetdraw.RoundedRectBorderTypeInset)
}

func (c *Checkbox) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	h := LineHeight(context)
	return image.Pt(h, h)
}
