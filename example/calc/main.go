// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"fmt"
	"image"
	"os"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	"github.com/guigui-gui/guigui/example/calc/internal/calc"
)

type buttonLabel struct {
	guigui.DefaultWidget

	text basicwidget.Text
}

func (b *buttonLabel) SetText(text string) {
	b.text.SetValue(text)
}

func (b *buttonLabel) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&b.text)
	b.text.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)
	b.text.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	b.text.SetScale(1.5)
	return nil
}

func (b *buttonLabel) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&b.text, widgetBounds.Bounds())
}

func (b *buttonLabel) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return b.text.Measure(context, constraints)
}

var buttonLabels = [...]calc.ButtonLabel{
	calc.ButtonLabelClear, calc.ButtonLabelNegate, calc.ButtonLabelPercent, calc.ButtonLabelDivide,
	calc.ButtonLabel7, calc.ButtonLabel8, calc.ButtonLabel9, calc.ButtonLabelMultiply,
	calc.ButtonLabel4, calc.ButtonLabel5, calc.ButtonLabel6, calc.ButtonLabelSubtract,
	calc.ButtonLabel1, calc.ButtonLabel2, calc.ButtonLabel3, calc.ButtonLabelAdd,
	calc.ButtonLabel0, calc.ButtonLabelDot, calc.ButtonLabelEquals,
}

const buttonCount = len(buttonLabels)

type Root struct {
	guigui.DefaultWidget

	calc calc.Calc

	background         basicwidget.Background
	displayText        basicwidget.Text
	buttons            [buttonCount]basicwidget.Button
	buttonLabelWidgets [buttonCount]buttonLabel

	rowItems      [4][]guigui.LinearLayoutItem
	rowLayouts    []guigui.LinearLayout
	lastRowItems  []guigui.LinearLayoutItem
	lastRowLayout guigui.LinearLayout
	outerItems    []guigui.LinearLayoutItem
	outerLayout   guigui.LinearLayout
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.displayText)

	r.displayText.SetSelectable(true)
	r.displayText.SetBold(true)
	r.displayText.SetTabular(true)
	r.displayText.SetHorizontalAlign(basicwidget.HorizontalAlignEnd)
	r.displayText.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	r.displayText.SetScale(3)
	r.displayText.SetValue(r.calc.Display())

	for i := range r.buttons {
		r.buttonLabelWidgets[i].SetText(string(buttonLabels[i]))
		r.buttons[i].SetContent(&r.buttonLabelWidgets[i])
		adder.AddWidget(&r.buttons[i])
		r.buttons[i].OnDown(func(context *guigui.Context) {
			r.calc.PressButton(buttonLabels[i])
		})
	}

	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())

	u := basicwidget.UnitSize(context)

	r.outerItems = slices.Delete(r.outerItems, 0, len(r.outerItems))

	// Display row.
	r.outerItems = append(r.outerItems, guigui.LinearLayoutItem{
		Widget: &r.displayText,
		Size:   guigui.FlexibleSize(2),
	})

	// Button rows (4 rows of 4 buttons).
	r.rowLayouts = slices.Delete(r.rowLayouts, 0, len(r.rowLayouts))
	for row := range 4 {
		r.rowItems[row] = slices.Delete(r.rowItems[row], 0, len(r.rowItems[row]))
		for col := range 4 {
			idx := row*4 + col
			r.rowItems[row] = append(r.rowItems[row], guigui.LinearLayoutItem{
				Widget: &r.buttons[idx],
				Size:   guigui.FlexibleSize(1),
			})
		}
		r.rowLayouts = append(r.rowLayouts, guigui.LinearLayout{
			Direction: guigui.LayoutDirectionHorizontal,
			Items:     r.rowItems[row],
			Gap:       u / 4,
		})
		r.outerItems = append(r.outerItems, guigui.LinearLayoutItem{
			Size:   guigui.FlexibleSize(1),
			Layout: &r.rowLayouts[row],
		})
	}

	// Last row: 0 (wide), dot, equals.
	// Use 4 cells to align with above rows, but merge the first two for the "0" button.
	r.lastRowItems = slices.Delete(r.lastRowItems, 0, len(r.lastRowItems))
	r.lastRowItems = append(r.lastRowItems,
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: &r.buttons[17],
			Size:   guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: &r.buttons[18],
			Size:   guigui.FlexibleSize(1),
		},
	)
	r.lastRowLayout = guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     r.lastRowItems,
		Gap:       u / 4,
	}
	r.outerItems = append(r.outerItems, guigui.LinearLayoutItem{
		Size:   guigui.FlexibleSize(1),
		Layout: &r.lastRowLayout,
	})

	r.outerLayout = guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     r.outerItems,
		Gap:       u / 4,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}
	r.outerLayout.LayoutWidgets(context, widgetBounds.Bounds(), layouter)

	// Layout the "0" button spanning the first two cells of the last row.
	lastRowBounds := r.outerLayout.ItemBoundsAt(len(r.outerItems)-1, context, widgetBounds.Bounds())
	b0 := r.lastRowLayout.ItemBoundsAt(0, context, lastRowBounds)
	b1 := r.lastRowLayout.ItemBoundsAt(1, context, lastRowBounds)
	layouter.LayoutWidget(&r.buttons[16], b0.Union(b1))
}

func main() {
	op := &guigui.RunOptions{
		Title:         "Calculator",
		WindowMinSize: image.Pt(300, 400),
		WindowSize:    image.Pt(320, 480),
		RunGameOptions: &ebiten.RunGameOptions{
			ApplePressAndHoldEnabled: true,
		},
	}
	if err := guigui.Run(&Root{}, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
