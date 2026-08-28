// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"image"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
)

var palettePopupContentEventSelected = guigui.GenerateEventKey()

// palettePopupContent is a palette popup's content: a row of color buttons
// followed by a default entry that clears the property.
type palettePopupContent struct {
	guigui.DefaultWidget

	colors        []color.Color
	colorButtons  guigui.WidgetSlice[*basicwidget.Button]
	colorSamples  guigui.WidgetSlice[*colorSample]
	defaultButton basicwidget.Button

	layoutItems []guigui.LinearLayoutItem
}

func (p *palettePopupContent) SetColors(colors []color.Color) {
	p.colors = slices.Clone(colors)
}

// OnSelected sets the handler invoked when an entry is chosen. ok is false
// for the default entry.
func (p *palettePopupContent) OnSelected(f func(context *guigui.Context, clr color.Color, ok bool)) {
	guigui.SetEventHandler(p, palettePopupContentEventSelected, f)
}

func (p *palettePopupContent) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	p.colorButtons.SetLen(len(p.colors))
	p.colorSamples.SetLen(len(p.colors))
	for _, button := range p.colorButtons.All() {
		adder.AddWidget(button)
	}
	adder.AddWidget(&p.defaultButton)

	for i, button := range p.colorButtons.All() {
		clr := p.colors[i]
		content := p.colorSamples.At(i)
		content.SetColor(clr)
		context.SetPassthrough(content, true)
		button.SetContent(content)
		button.OnDown(func(context *guigui.Context) {
			guigui.DispatchEvent(p, palettePopupContentEventSelected, clr, true)
		})
	}

	p.defaultButton.SetText("Default")
	p.defaultButton.OnDown(func(context *guigui.Context) {
		guigui.DispatchEvent(p, palettePopupContentEventSelected, color.RGBA{}, false)
	})

	return nil
}

func (p *palettePopupContent) layout(context *guigui.Context) guigui.LinearLayout {
	u := basicwidget.UnitSize(context)

	p.layoutItems = slices.Delete(p.layoutItems, 0, len(p.layoutItems))
	for _, button := range p.colorButtons.All() {
		p.layoutItems = append(p.layoutItems, guigui.LinearLayoutItem{
			Widget: button,
			Size:   guigui.FixedSize(u),
		})
	}
	p.layoutItems = append(p.layoutItems, guigui.LinearLayoutItem{
		Widget: &p.defaultButton,
	})

	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     p.layoutItems,
		Gap:       u / 8,
		Padding: guigui.Padding{
			Start:  u / 4,
			Top:    u / 4,
			End:    u / 4,
			Bottom: u / 4,
		},
	}
}

func (p *palettePopupContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	p.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (p *palettePopupContent) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return p.layout(context).Measure(context, constraints)
}

// colorSample draws a color as a filled rounded rectangle inside a palette
// button.
type colorSample struct {
	guigui.DefaultWidget

	color color.Color
}

func (c *colorSample) SetColor(clr color.Color) {
	c.color = clr
}

func (c *colorSample) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	u := basicwidget.UnitSize(context)
	return image.Pt(u, u)
}

func (c *colorSample) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if c.color == nil {
		return
	}
	u := basicwidget.UnitSize(context)
	bounds := widgetBounds.Bounds().Inset(u / 4)
	basicwidgetdraw.DrawRoundedRect(context, dst, bounds, c.color, u/8)
}
