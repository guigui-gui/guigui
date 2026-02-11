// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

var (
	buttonEventDown   guigui.EventKey = guigui.GenerateEventKey()
	buttonEventUp     guigui.EventKey = guigui.GenerateEventKey()
	buttonEventRepeat guigui.EventKey = guigui.GenerateEventKey()
)

type Corners struct {
	TopStart    bool
	TopEnd      bool
	BottomStart bool
	BottomEnd   bool
}

type IconAlign int

const (
	IconAlignStart IconAlign = iota
	IconAlignEnd
)

type ButtonType int

const (
	ButtonTypeNormal ButtonType = iota
	ButtonTypePrimary
	buttonTypeActiveSegmentControlButton
)

type Button struct {
	guigui.DefaultWidget

	content   guigui.Widget
	text      Text
	icon      Image
	iconAlign IconAlign

	typ       ButtonType
	textColor color.Color
	textBold  bool

	layoutItems []guigui.LinearLayoutItem
	iconLayout  guigui.LinearLayout

	pressed         bool
	keepPressed     bool
	borderInvisible bool
	prevPressed     bool
	sharpCorners    Corners
	pairedButton    *Button
	prevCanPress    bool
}

func (b *Button) OnDown(f func(context *guigui.Context)) {
	guigui.AddEventHandler(b, buttonEventDown, f)
}

func (b *Button) OnUp(f func(context *guigui.Context)) {
	guigui.AddEventHandler(b, buttonEventUp, f)
}

func (b *Button) setOnRepeat(f func(context *guigui.Context)) {
	guigui.AddEventHandler(b, buttonEventRepeat, f)
}

func (b *Button) setPairedButton(pair *Button) {
	b.pairedButton = pair
}

func (b *Button) setPressed(pressed bool) {
	if b.pressed == pressed {
		return
	}
	b.pressed = pressed
	guigui.RequestRebuild(b)
}

func (b *Button) SetContent(content guigui.Widget) {
	b.content = content
}

func (b *Button) SetText(text string) {
	b.text.SetValue(text)
}

func (b *Button) SetTextBold(bold bool) {
	if b.textBold == bold {
		return
	}
	b.textBold = bold
	guigui.RequestRebuild(b)
}

func (b *Button) SetIcon(icon *ebiten.Image) {
	b.icon.SetImage(icon)
}

func (b *Button) SetIconAlign(align IconAlign) {
	if b.iconAlign == align {
		return
	}
	b.iconAlign = align
	guigui.RequestRebuild(b)
}

func (b *Button) SetType(typ ButtonType) {
	if b.typ == typ {
		return
	}
	b.typ = typ
	guigui.RequestRebuild(b)
}

func (b *Button) SetTextColor(clr color.Color) {
	if draw.EqualColor(b.textColor, clr) {
		return
	}
	b.textColor = clr
	guigui.RequestRebuild(b)
}

func (b *Button) setKeepPressed(keep bool) {
	if b.keepPressed == keep {
		return
	}
	b.keepPressed = keep
	guigui.RequestRebuild(b)
}

func (b *Button) SetSharpCorners(sharpCorners Corners) {
	if b.sharpCorners == sharpCorners {
		return
	}
	b.sharpCorners = sharpCorners
	guigui.RequestRebuild(b)
}

func (b *Button) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if b.content != nil {
		adder.AddChild(b.content)
	}
	adder.AddChild(&b.text)
	adder.AddChild(&b.icon)

	if b.textColor != nil {
		b.text.SetColor(b.textColor)
	} else if !context.IsEnabled(b) {
		b.text.SetColor(basicwidgetdraw.TextColor(context.ColorMode(), false))
	} else {
		switch b.typ {
		case ButtonTypePrimary:
			b.text.SetColor(basicwidgetdraw.TextColor(guigui.ColorModeDark, true))
		default:
			b.text.SetColor(basicwidgetdraw.TextColor(context.ColorMode(), true))
		}
	}
	if b.textBold {
		b.text.SetBold(true)
	} else if b.typ == ButtonTypePrimary {
		b.text.SetBold(true)
	} else {
		b.text.SetBold(false)
	}
	b.text.SetHorizontalAlign(HorizontalAlignCenter)
	b.text.SetVerticalAlign(VerticalAlignMiddle)
	return nil
}

func (b *Button) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	var yOffset int
	if b.isPressed(context, widgetBounds) {
		yOffset = int(0.5 * context.Scale())
	} else {
		yOffset = -int(0.5 * context.Scale())
	}

	if b.content != nil {
		layouter.LayoutWidget(b.content, widgetBounds.Bounds().Add(image.Pt(0, yOffset)))
	}

	b.layoutItems = slices.Delete(b.layoutItems, 0, len(b.layoutItems))
	b.layoutItems = append(b.layoutItems,
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		})
	var iconLayoutItem guigui.LinearLayoutItem
	if b.icon.HasImage() {
		width := min(defaultIconSize(context), widgetBounds.Bounds().Dx())
		height := min(defaultIconSize(context), widgetBounds.Bounds().Dy())
		if b.text.Value() == "" {
			// The bounds for Button and baseButton are the same, so it's ok to pass widgetBounds here.
			r := b.radius(context, widgetBounds)
			width = max(width, widgetBounds.Bounds().Dx()-2*r)
			height = max(height, widgetBounds.Bounds().Dy()-2*r)
		}

		var toCreateIconLayout bool
		if len(b.iconLayout.Items) == 0 {
			toCreateIconLayout = true
		} else {
			// The address of b.icon can be changed anytime, so the cahched layout must be updated accordingly.
			iconItem := b.iconLayout.Items[1]
			toCreateIconLayout = iconItem.Widget != &b.icon || iconItem.Size != guigui.FixedSize(height)
		}
		if toCreateIconLayout {
			b.iconLayout = guigui.LinearLayout{
				Direction: guigui.LayoutDirectionVertical,
				Items: []guigui.LinearLayoutItem{
					{
						Size: guigui.FlexibleSize(1),
					},
					{
						Widget: &b.icon,
						Size:   guigui.FixedSize(height),
					},
					{
						Size: guigui.FlexibleSize(1),
					},
				},
			}
		}
		iconLayoutItem = guigui.LinearLayoutItem{
			Layout: b.iconLayout,
			Size:   guigui.FixedSize(width),
		}
	}

	if b.icon.HasImage() && b.iconAlign == IconAlignStart {
		b.layoutItems = append(b.layoutItems, iconLayoutItem)
		if b.text.Value() != "" {
			b.layoutItems = append(b.layoutItems,
				guigui.LinearLayoutItem{
					Size: guigui.FixedSize(buttonTextAndImagePadding(context)),
				})
		}
	}
	if b.text.Value() != "" {
		b.layoutItems = append(b.layoutItems,
			guigui.LinearLayoutItem{
				Widget: &b.text,
			})
	}
	if b.icon.HasImage() && b.iconAlign == IconAlignEnd {
		if b.text.Value() != "" {
			b.layoutItems = append(b.layoutItems,
				guigui.LinearLayoutItem{
					Size: guigui.FixedSize(buttonTextAndImagePadding(context)),
				})
		}
		b.layoutItems = append(b.layoutItems, iconLayoutItem)
	}

	b.layoutItems = append(b.layoutItems,
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		})

	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     b.layoutItems,
	}).LayoutWidgets(context, widgetBounds.Bounds().Add(image.Pt(0, yOffset)), layouter)
}

func (b *Button) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return b.measure(context, constraints, false)
}

func (b *Button) measure(context *guigui.Context, constraints guigui.Constraints, forceBold bool) image.Point {
	h := defaultButtonSize(context).Y
	var w int
	if b.text.Value() != "" {
		w += buttonEdgeAndTextPadding(context)
		if forceBold {
			w += b.text.boldTextSize(context, guigui.Constraints{}).X
		} else {
			w += b.text.Measure(context, guigui.Constraints{}).X
		}
	}
	if b.icon.HasImage() {
		if w == 0 {
			w += buttonEdgeAndImagePadding(context)
		}
		if b.text.Value() != "" {
			w += buttonTextAndImagePadding(context)
		}
		w += defaultIconSize(context)
		w += buttonEdgeAndImagePadding(context)
	} else {
		w += buttonEdgeAndTextPadding(context)
	}

	if b.content != nil {
		s := b.content.Measure(context, constraints)
		w = max(w, s.X)
		h = max(h, s.Y)
	}

	if fixedWidth, ok := constraints.FixedWidth(); ok {
		w = min(w, fixedWidth)
	}
	if fixedHeight, ok := constraints.FixedHeight(); ok {
		h = min(h, fixedHeight)
	}

	return image.Pt(w, h)
}

func (b *Button) checkPressed(context *guigui.Context, widgetBounds *guigui.WidgetBounds) {
	if hovered := b.isPressed(context, widgetBounds); b.prevPressed != hovered {
		b.prevPressed = hovered
		guigui.RequestRebuild(b)
	}
}

func (b *Button) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	b.checkPressed(context, widgetBounds)

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
			guigui.DispatchEvent(b, buttonEventDown)
			if isMouseButtonRepeating(ebiten.MouseButtonLeft) {
				guigui.DispatchEvent(b, buttonEventRepeat)
			}
			justPressedOrReleased = true
		}
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && b.pressed {
			if b.keepPressed {
				return guigui.AbortHandlingInputByWidget(b)
			}
			b.setPressed(false)
			guigui.DispatchEvent(b, buttonEventUp)
			guigui.RequestRebuild(b)
			justPressedOrReleased = true
		}
		if justPressedOrReleased {
			return guigui.HandleInputByWidget(b)
		}
		if (b.pressed || b.pairedButton != nil && b.pairedButton.pressed) && isMouseButtonRepeating(ebiten.MouseButtonLeft) {
			guigui.DispatchEvent(b, buttonEventRepeat)
			return guigui.HandleInputByWidget(b)
		}
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		b.setPressed(false)
	}
	return guigui.HandleInputResult{}
}

func (b *Button) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	b.checkPressed(context, widgetBounds)
	if pressed := b.canPress(context, widgetBounds); pressed != b.prevCanPress {
		b.prevCanPress = pressed
		guigui.RequestRedraw(b)
	}
	return nil
}

func (b *Button) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	if (b.canPress(context, widgetBounds) || b.pressed || b.pairedButton != nil && b.pairedButton.pressed) && !b.keepPressed {
		return ebiten.CursorShapePointer, true
	}
	return 0, true
}

func (b *Button) radius(context *guigui.Context, widgetBounds *guigui.WidgetBounds) int {
	size := widgetBounds.Bounds().Size()
	return min(RoundedCornerRadius(context), size.X/4, size.Y/4)
}

func (b *Button) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	cm := context.ColorMode()
	backgroundColor := basicwidgetdraw.ControlColor(context.ColorMode(), context.IsEnabled(b))
	if context.IsEnabled(b) {
		switch b.typ {
		case ButtonTypePrimary:
			backgroundColor = draw.Color2(cm, draw.ColorTypeAccent, 0.5, 0.5)
			if b.isPressed(context, widgetBounds) {
				backgroundColor = draw.Color2(cm, draw.ColorTypeAccent, 0.475, 0.475)
			} else if b.canPress(context, widgetBounds) {
				backgroundColor = draw.Color2(cm, draw.ColorTypeAccent, 0.45, 0.45)
			}
		case buttonTypeActiveSegmentControlButton:
			if b.isPressed(context, widgetBounds) {
				backgroundColor = draw.Color2(cm, draw.ColorTypeAccent, 0.875, 0.5)
			} else if b.canPress(context, widgetBounds) {
				backgroundColor = draw.Color2(cm, draw.ColorTypeBase, 0.975, 0.275)
			}
		default:
			if b.isPressed(context, widgetBounds) {
				backgroundColor = draw.Color2(cm, draw.ColorTypeBase, 0.95, 0.25)
			} else if b.canPress(context, widgetBounds) {
				backgroundColor = draw.Color2(cm, draw.ColorTypeBase, 0.975, 0.275)
			}
		}
	}

	r := b.radius(context, widgetBounds)
	border := !b.borderInvisible
	if context.IsEnabled(b) && (widgetBounds.IsHitAtCursor() || b.keepPressed) {
		border = true
	}
	bounds := widgetBounds.Bounds()
	if border || b.isPressed(context, widgetBounds) {
		basicwidgetdraw.DrawRoundedRectWithSharpCorners(context, dst, bounds, backgroundColor, r, basicwidgetdraw.Corners(b.sharpCorners))
	}

	if border {
		borderType := basicwidgetdraw.RoundedRectBorderTypeOutset
		if b.isPressed(context, widgetBounds) {
			borderType = basicwidgetdraw.RoundedRectBorderTypeInset
		}
		clr1, clr2 := basicwidgetdraw.BorderColors(context.ColorMode(), basicwidgetdraw.RoundedRectBorderType(borderType))
		if context.IsEnabled(b) {
			switch b.typ {
			case ButtonTypePrimary:
				clr1, clr2 = basicwidgetdraw.BorderAccentColors(context.ColorMode(), basicwidgetdraw.RoundedRectBorderType(borderType))
			case buttonTypeActiveSegmentControlButton:
				if b.isPressed(context, widgetBounds) {
					clr1, clr2 = basicwidgetdraw.BorderAccentSecondaryColors(context.ColorMode(), basicwidgetdraw.RoundedRectBorderType(borderType))
				} else {
					clr1, clr2 = basicwidgetdraw.BorderColors(context.ColorMode(), basicwidgetdraw.RoundedRectBorderType(borderType))
				}
			}
		}

		basicwidgetdraw.DrawRoundedRectBorderWithSharpCorners(context, dst, bounds, clr1, clr2, r, float32(1*context.Scale()), borderType, basicwidgetdraw.Corners(b.sharpCorners))
	}
}

func (b *Button) canPress(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(b) && widgetBounds.IsHitAtCursor() && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && !b.keepPressed
}

func (b *Button) isActive(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(b) && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && widgetBounds.IsHitAtCursor() && (b.pressed || b.pairedButton != nil && b.pairedButton.pressed)
}

func (b *Button) isPressed(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(b) && b.isActive(context, widgetBounds) || b.keepPressed
}

func defaultButtonSize(context *guigui.Context) image.Point {
	return image.Pt(6*UnitSize(context), UnitSize(context))
}

func buttonTextAndImagePadding(context *guigui.Context) int {
	return UnitSize(context) / 4
}

func buttonEdgeAndTextPadding(context *guigui.Context) int {
	return UnitSize(context) / 2
}

func buttonEdgeAndImagePadding(context *guigui.Context) int {
	return UnitSize(context) / 4
}
