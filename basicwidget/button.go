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
)

type Button struct {
	guigui.DefaultWidget

	content   guigui.Widget
	text      Text
	icon      Image
	iconAlign IconAlign

	typ      ButtonType
	tint     color.Color
	textBold bool

	layoutItems     []guigui.LinearLayoutItem
	iconLayout      guigui.LinearLayout
	iconLayoutItems []guigui.LinearLayoutItem

	pressedByInput  bool
	pressedByMethod bool
	toggleable      bool
	borderInvisible bool
	prevPressed     bool
	sharpCorners    Corners
	pairedButton    *Button
	prevCanPress    bool
}

func (b *Button) OnDown(f func(context *guigui.Context)) {
	guigui.SetEventHandler(b, buttonEventDown, f)
}

func (b *Button) OnUp(f func(context *guigui.Context)) {
	guigui.SetEventHandler(b, buttonEventUp, f)
}

func (b *Button) setOnRepeat(f func(context *guigui.Context)) {
	guigui.SetEventHandler(b, buttonEventRepeat, f)
}

func (b *Button) setPairedButton(pair *Button) {
	b.pairedButton = pair
}

func (b *Button) WriteStateKey(context *guigui.Context, w *guigui.StateKeyWriter) {
	w.WriteBool(b.pressedByInput)
	w.WriteBool(b.pressedByMethod)
	w.WriteBool(b.toggleable)
	w.WriteBool(b.prevPressed)
	w.WriteBool(b.textBold)
	w.WriteUint64(uint64(b.iconAlign))
	w.WriteUint64(uint64(b.typ))
	writeColor(w, b.tint)
	w.WriteBool(b.sharpCorners.TopStart)
	w.WriteBool(b.sharpCorners.TopEnd)
	w.WriteBool(b.sharpCorners.BottomStart)
	w.WriteBool(b.sharpCorners.BottomEnd)
}

func (b *Button) SetContent(content guigui.Widget) {
	b.content = content
}

func (b *Button) SetText(text string) {
	b.text.SetValue(text)
}

func (b *Button) SetTextBold(bold bool) {
	b.textBold = bold
}

func (b *Button) SetIcon(icon *ebiten.Image) {
	b.icon.SetImage(icon)
}

func (b *Button) SetIconAlign(align IconAlign) {
	b.iconAlign = align
}

func (b *Button) SetType(typ ButtonType) {
	b.typ = typ
}

// SetTintColor sets the color the button derives its appearance from.
// A nil tint restores the default appearance. A primary button ignores the
// tint.
func (b *Button) SetTintColor(tint color.Color) {
	b.tint = tint
}

// SetPressed sets whether the button stays pressed.
//
// A pressed button ignores clicks unless the button is toggleable.
func (b *Button) SetPressed(pressed bool) {
	b.pressedByMethod = pressed
}

// SetToggleable sets whether the button works as a toggle button.
//
// A toggleable button accepts clicks even while it is pressed.
func (b *Button) SetToggleable(toggleable bool) {
	b.toggleable = toggleable
}

func (b *Button) SetSharpCorners(sharpCorners Corners) {
	b.sharpCorners = sharpCorners
}

func (b *Button) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if b.content != nil {
		adder.AddWidget(b.content)
	}
	adder.AddWidget(&b.text)
	adder.AddWidget(&b.icon)

	var style TextStyle
	switch {
	case !context.IsEnabled(b):
		style.SetColor(draw.TextColor(context.ColorMode(), false))
	case b.typ == ButtonTypePrimary:
		style.SetColor(draw.TextOnAccentColor(context.ColorMode()))
	case b.tint != nil:
		style.SetColor(draw.TextColorFromTint(context.ColorMode(), b.tint))
	default:
		style.SetColor(draw.TextColor(context.ColorMode(), true))
	}
	style.SetBold(b.textBold || b.typ == ButtonTypePrimary || b.showsPressedState())
	b.text.SetBaseStyle(&style)
	b.text.SetHorizontalAlign(HorizontalAlignCenter)
	b.text.SetVerticalAlign(VerticalAlignMiddle)
	return nil
}

func (b *Button) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	var yOffset int
	switch {
	case b.isDeeplyPressed(context, widgetBounds):
		yOffset = int(1 * context.Scale())
	case b.isPressed(context, widgetBounds):
		yOffset = int(0.5 * context.Scale())
	default:
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
			b.iconLayoutItems = slices.Delete(b.iconLayoutItems, 0, len(b.iconLayoutItems))
			b.iconLayoutItems = append(b.iconLayoutItems,
				guigui.LinearLayoutItem{
					Size: guigui.FlexibleSize(1),
				},
				guigui.LinearLayoutItem{
					Widget: &b.icon,
					Size:   guigui.FixedSize(height),
				},
				guigui.LinearLayoutItem{
					Size: guigui.FlexibleSize(1),
				})
			b.iconLayout = guigui.LinearLayout{
				Direction: guigui.LayoutDirectionVertical,
				Items:     b.iconLayoutItems,
			}
		}
		iconLayoutItem = guigui.LinearLayoutItem{
			Layout: &b.iconLayout,
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
	h := defaultButtonSize(context).Y
	var w int
	if b.text.Value() != "" {
		// Measure the text as bold so that the size doesn't depend on whether the button is pressed.
		w += buttonEdgeAndTextPadding(context)
		w += b.text.boldTextSize(context, guigui.Constraints{}).X
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
	b.prevPressed = b.isPressed(context, widgetBounds)
}

func (b *Button) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	b.checkPressed(context, widgetBounds)

	if widgetBounds.IsHitAtCursor() {
		// IsMouseButtonJustPressed and IsMouseButtonJustReleased can be true at the same time as of Ebitengine v2.9.
		// Check both.
		var justPressedOrReleased bool
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			if b.pressedByMethod && !b.toggleable {
				return guigui.AbortHandlingInputByWidget(b)
			}
			context.SetFocused(b, true)
			b.pressedByInput = true
			guigui.DispatchEvent(b, buttonEventDown)
			if isMouseButtonRepeating(ebiten.MouseButtonLeft) {
				guigui.DispatchEvent(b, buttonEventRepeat)
			}
			justPressedOrReleased = true
		}
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && b.pressedByInput {
			b.pressedByInput = false
			if b.pressedByMethod && !b.toggleable {
				return guigui.AbortHandlingInputByWidget(b)
			}
			guigui.DispatchEvent(b, buttonEventUp)
			justPressedOrReleased = true
		}
		if justPressedOrReleased {
			return guigui.HandleInputByWidget(b)
		}
		if (b.pressedByInput || b.pairedButton != nil && b.pairedButton.pressedByInput) && isMouseButtonRepeating(ebiten.MouseButtonLeft) {
			guigui.DispatchEvent(b, buttonEventRepeat)
			return guigui.HandleInputByWidget(b)
		}
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		b.pressedByInput = false
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
	if b.toggleable && context.IsEnabled(b) && widgetBounds.IsHitAtCursor() {
		return ebiten.CursorShapePointer, true
	}
	if (b.canPress(context, widgetBounds) || b.pressedByInput || b.pairedButton != nil && b.pairedButton.pressedByInput) && (!b.pressedByMethod || b.toggleable) {
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
	backgroundColor := draw.ControlColor(context.ColorMode(), context.IsEnabled(b))
	if context.IsEnabled(b) {
		switch {
		case b.typ == ButtonTypePrimary:
			backgroundColor = draw.PrimaryButtonBackgroundColor(cm, b.isPressed(context, widgetBounds), b.canPress(context, widgetBounds))
		case b.showsPressedState():
			// Keep the hovered color while the button is being pressed, so that pressing it never lightens it.
			hovered := b.canPress(context, widgetBounds) || b.isBeingPressed(context, widgetBounds)
			backgroundColor = draw.PressedButtonBackgroundColor(cm, hovered)
		default:
			backgroundColor = draw.ButtonBackgroundColorFromTint(cm, b.tint, b.isPressed(context, widgetBounds), b.canPress(context, widgetBounds))
		}
	}

	r := b.radius(context, widgetBounds)
	border := !b.borderInvisible
	if context.IsEnabled(b) && (widgetBounds.IsHitAtCursor() || b.pressedByMethod) {
		border = true
	}
	bounds := widgetBounds.Bounds()
	if border || b.isPressed(context, widgetBounds) {
		draw.DrawRoundedRectWithSharpCorners(context, dst, bounds, backgroundColor, r, draw.Corners(b.sharpCorners))
	}

	if border {
		borderType := draw.RoundedRectBorderTypeOutset
		if b.isPressed(context, widgetBounds) {
			borderType = draw.RoundedRectBorderTypeInset
		}
		clr1, clr2 := draw.BorderColors(context.ColorMode(), draw.RoundedRectBorderType(borderType))
		if context.IsEnabled(b) {
			switch {
			case b.typ == ButtonTypePrimary:
				clr1, clr2 = draw.BorderAccentColors(context.ColorMode(), draw.RoundedRectBorderType(borderType))
			case b.showsPressedState():
				clr1, clr2 = draw.BorderAccentSecondaryColors(context.ColorMode(), draw.RoundedRectBorderType(borderType))
			}
		}

		borderWidth := float32(1 * context.Scale())
		if b.isDeeplyPressed(context, widgetBounds) {
			borderWidth = float32(1.5 * context.Scale())
		}
		draw.DrawRoundedRectBorderWithSharpCorners(context, dst, bounds, clr1, clr2, r, borderWidth, borderType, draw.Corners(b.sharpCorners))
	}
}

func (b *Button) canPress(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(b) && widgetBounds.IsHitAtCursor() && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && (!b.pressedByMethod || b.toggleable)
}

// isDeeplyPressed reports whether the button should look deeper pressed than its pressed state alone.
func (b *Button) isDeeplyPressed(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return b.showsPressedState() && b.isBeingPressed(context, widgetBounds)
}

// showsPressedState reports whether the button shows its pressed state rather than a transient press.
// A toggleable button shows it while it is being clicked,
// so that unpressing it by a click doesn't change the appearance until the release.
func (b *Button) showsPressedState() bool {
	return b.pressedByMethod || b.toggleable && b.pressedByInput
}

func (b *Button) isBeingPressed(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(b) && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && widgetBounds.IsHitAtCursor() && (b.pressedByInput || b.pairedButton != nil && b.pairedButton.pressedByInput)
}

func (b *Button) isPressed(context *guigui.Context, widgetBounds *guigui.WidgetBounds) bool {
	return context.IsEnabled(b) && b.isBeingPressed(context, widgetBounds) || b.pressedByMethod
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
