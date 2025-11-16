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
	iconLayout  guigui.Layout
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

func (b *Button) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	var yOffset int
	if b.button.isPressed(context, widgetBounds) {
		yOffset = int(0.5 * context.Scale())
	} else {
		yOffset = -int(0.5 * context.Scale())
	}

	layouter.LayoutWidget(&b.button, widgetBounds.Bounds())
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
			r := b.button.radius(context, widgetBounds)
			width = max(width, widgetBounds.Bounds().Dx()-2*r)
			height = max(height, widgetBounds.Bounds().Dy()-2*r)
		}

		// TODO: Cache the layout like this condition
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
