// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

type IconAlign int

const (
	IconAlignStart IconAlign = iota
	IconAlignEnd
)

type Button struct {
	guigui.DefaultWidget

	button    baseButton
	content   guigui.Widget
	text      Text
	icon      Image
	iconAlign IconAlign

	textColor color.Color

	layoutItems []guigui.LinearLayoutItem
}

func (b *Button) SetOnDown(f func()) {
	b.button.SetOnDown(f)
}

func (b *Button) SetOnUp(f func()) {
	b.button.SetOnUp(f)
}

func (b *Button) setOnRepeat(f func()) {
	b.button.setOnRepeat(f)
}

func (b *Button) SetContent(content guigui.Widget) {
	b.content = content
}

func (b *Button) SetText(text string) {
	b.text.SetValue(text)
}

func (b *Button) SetTextBold(bold bool) {
	b.text.SetBold(bold)
}

func (b *Button) SetIcon(icon *ebiten.Image) {
	b.icon.SetImage(icon)
}

func (b *Button) SetIconAlign(align IconAlign) {
	if b.iconAlign == align {
		return
	}
	b.iconAlign = align
	guigui.RequestRedraw(b)
}

func (b *Button) SetTextColor(clr color.Color) {
	if draw.EqualColor(b.textColor, clr) {
		return
	}
	b.textColor = clr
	guigui.RequestRedraw(b)
}

func (b *Button) setPairedButton(pair *Button) {
	b.button.setPairedButton(&pair.button)
}

func (b *Button) setKeepPressed(keep bool) {
	b.button.setKeepPressed(keep)
}

func (b *Button) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&b.button)
	if b.content != nil {
		adder.AddChild(b.content)
	}
	adder.AddChild(&b.text)
	adder.AddChild(&b.icon)
}

func (b *Button) Update(context *guigui.Context) error {
	if b.textColor != nil {
		b.text.SetColor(b.textColor)
	} else {
		b.text.SetColor(draw.TextColor(context.ColorMode(), context.IsEnabled(b)))
	}
	b.text.SetHorizontalAlign(HorizontalAlignCenter)
	b.text.SetVerticalAlign(VerticalAlignMiddle)
	return nil
}

func (b *Button) Layout(context *guigui.Context, widget guigui.Widget) image.Rectangle {
	var yOffset int
	if b.button.isPressed(context) {
		yOffset = int(0.5 * context.Scale())
	} else {
		yOffset = -int(0.5 * context.Scale())
	}

	switch widget {
	case &b.button:
		return context.Bounds(b)
	case b.content:
		return context.Bounds(b).Add(image.Pt(0, yOffset))
	}

	r := b.button.radius(context)

	b.layoutItems = slices.Delete(b.layoutItems, 0, len(b.layoutItems))
	b.layoutItems = append(b.layoutItems,
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		})
	if b.icon.HasImage() {
		var width guigui.Size
		if b.text.Value() != "" {
			width = guigui.FixedSize(defaultIconSize(context))
		} else {
			width = guigui.FixedSize(context.Bounds(b).Dx() - 2*r)
		}
		var height guigui.Size
		if b.text.Value() != "" {
			height = guigui.FixedSize(defaultIconSize(context))
		} else {
			height = guigui.FixedSize(max(defaultIconSize(context), context.Bounds(b).Dy()-2*r))
		}
		b.layoutItems = append(b.layoutItems,
			guigui.LinearLayoutItem{
				Layout: guigui.LinearLayout{
					Direction: guigui.LayoutDirectionVertical,
					Items: []guigui.LinearLayoutItem{
						{
							Size: guigui.FlexibleSize(1),
						},
						{
							Widget: &b.icon,
							Size:   height,
						},
						{
							Size: guigui.FlexibleSize(1),
						},
					},
				},
				Size: width,
			})
	}
	if b.icon.HasImage() && b.text.Value() != "" {
		b.layoutItems = append(b.layoutItems,
			guigui.LinearLayoutItem{
				Size: guigui.FixedSize(buttonTextAndImagePadding(context)),
			})
	}
	if b.text.Value() != "" {
		b.layoutItems = append(b.layoutItems,
			guigui.LinearLayoutItem{
				Widget: &b.text,
			})
	}
	b.layoutItems = append(b.layoutItems,
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		})

	return (guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     b.layoutItems,
	}).WidgetBounds(context, context.Bounds(b).Add(image.Pt(0, yOffset)), widget)
}

func (b *Button) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return b.measure(context, constraints, false)
}

func (b *Button) measure(context *guigui.Context, constraints guigui.Constraints, forceBold bool) image.Point {
	h := defaultButtonSize(context).Y
	var textAndImageW int
	if b.text.Value() != "" {
		textAndImageW += buttonEdgeAndTextPadding(context)
		if forceBold {
			textAndImageW += b.text.boldTextSize(context, guigui.Constraints{}).X
		} else {
			textAndImageW += b.text.Measure(context, guigui.Constraints{}).X
		}
	}
	if b.icon.HasImage() {
		if textAndImageW == 0 {
			textAndImageW += buttonEdgeAndImagePadding(context)
		}
		if b.text.Value() != "" {
			textAndImageW += buttonTextAndImagePadding(context)
		}
		textAndImageW += defaultIconSize(context)
		textAndImageW += buttonEdgeAndImagePadding(context)
	} else {
		textAndImageW += buttonEdgeAndTextPadding(context)
	}

	var contentW int
	if b.content != nil {
		contentW = b.content.Measure(context, constraints).X
	}

	return image.Pt(max(textAndImageW, contentW), h)
}

func (b *Button) setSharpenCorners(sharpenCorners draw.SharpenCorners) {
	b.button.setSharpenCorners(sharpenCorners)
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

func (b *Button) setUseAccentColor(use bool) {
	b.button.setUseAccentColor(use)
}
