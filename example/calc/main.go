// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"fmt"
	"image"
	"os"

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

	rows := make([]guigui.LinearLayoutItem, 0, 6)

	// Display row.
	rows = append(rows, guigui.LinearLayoutItem{
		Widget: &r.displayText,
		Size:   guigui.FlexibleSize(2),
	})

	// Button rows (4 rows of 4 buttons).
	for row := range 4 {
		items := make([]guigui.LinearLayoutItem, 4)
		for col := range 4 {
			idx := row*4 + col
			items[col] = guigui.LinearLayoutItem{
				Widget: &r.buttons[idx],
				Size:   guigui.FlexibleSize(1),
			}
		}
		rows = append(rows, guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
			Layout: guigui.LinearLayout{
				Direction: guigui.LayoutDirectionHorizontal,
				Items:     items,
				Gap:       u / 4,
			},
		})
	}

	// Last row: 0 (wide), dot, equals.
	// Use 4 cells to align with above rows, but merge the first two for the "0" button.
	lastRowLayout := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items: []guigui.LinearLayoutItem{
			{
				Size: guigui.FlexibleSize(1),
			},
			{
				Size: guigui.FlexibleSize(1),
			},
			{
				Widget: &r.buttons[17],
				Size:   guigui.FlexibleSize(1),
			},
			{
				Widget: &r.buttons[18],
				Size:   guigui.FlexibleSize(1),
			},
		},
		Gap: u / 4,
	}
	rows = append(rows, guigui.LinearLayoutItem{
		Size:   guigui.FlexibleSize(1),
		Layout: lastRowLayout,
	})

	outerLayout := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     rows,
		Gap:       u / 4,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}
	outerLayout.LayoutWidgets(context, widgetBounds.Bounds(), layouter)

	// Layout the "0" button spanning the first two cells of the last row.
	lastRowBounds := outerLayout.ItemBoundsAt(len(rows)-1, context, widgetBounds.Bounds())
	b0 := lastRowLayout.ItemBoundsAt(0, context, lastRowBounds)
	b1 := lastRowLayout.ItemBoundsAt(1, context, lastRowBounds)
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
