// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package main

import (
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Root struct {
	guigui.DefaultWidget
	description basicwidget.Text
	background  basicwidget.Background
	resetButton basicwidget.Button
	totalMoved  basicwidget.Text
	game        GameWidget
}

type GameWidget struct {
	guigui.DefaultWidget
	background *ebiten.Image
	gopher     *ebiten.Image
	posX       int
	posY       int
	totalMoved int
}

func (g *GameWidget) Tick(context *guigui.Context, bounds *guigui.WidgetBounds) error {
	keys := []ebiten.Key{}
	keys = inpututil.AppendPressedKeys(keys)
	for _, key := range keys {
		if key == ebiten.KeyLeft {
			g.posX -= 1
		}
		if key == ebiten.KeyRight {
			g.posX += 1
		}
		if key == ebiten.KeyUp {
			g.posY -= 1
		}
		if key == ebiten.KeyDown {
			g.posY += 1
		}
	}
	g.totalMoved += len(keys)
	return nil
}

func (g *GameWidget) Draw(context *guigui.Context, bounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if g.gopher == nil {
		img, _, err := ebitenutil.NewImageFromFile(filepath.Join("resources", "gopher_center.png"))
		if err != nil {
			panic(err)
		}
		g.gopher = img
	}
	opts := ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(g.posX+bounds.Bounds().Min.X), float64(g.posY+bounds.Bounds().Min.Y))
	dst.DrawImage(g.gopher, &opts)
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&r.background)
	adder.AddChild(&r.description)
	adder.AddChild(&r.game)
	adder.AddChild(&r.resetButton)
	adder.AddChild(&r.totalMoved)
	r.description.ForceSetValue("Use the arrow keys to move the gopher around")
	r.resetButton.SetText("Reset")
	r.resetButton.SetOnUp(func(context *guigui.Context) {
		r.game.posY = 0
		r.game.posX = 0
		r.game.totalMoved = 0
	})
	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())
	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &r.description,
				Size:   guigui.FixedSize(u),
			},
			{
				Widget: &r.game,
				Size:   guigui.FlexibleSize(1),
			},
			{
				Size: guigui.FixedSize(u),
				Layout: guigui.LinearLayout{
					Direction: guigui.LayoutDirectionHorizontal,
					Items: []guigui.LinearLayoutItem{
						{
							Widget: &r.totalMoved,
							Size:   guigui.FixedSize(6 * u),
						},
						{
							Size: guigui.FlexibleSize(1),
						},
						{
							Widget: &r.resetButton,
							Size:   guigui.FixedSize(6 * u),
						},
					},
					Gap: u / 2,
				},
			},
		},
		Gap: u,
		Padding: guigui.Padding{
			Start:  u,
			Top:    u,
			End:    u,
			Bottom: u,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (r *Root) Tick(context *guigui.Context, bounds *guigui.WidgetBounds) error {
	guigui.RequestRedraw(&r.game)
	r.totalMoved.ForceSetValue(fmt.Sprintf("Total moved: %d", r.game.totalMoved))
	return nil
}

func main() {
	op := &guigui.RunOptions{
		Title:         "Game embed",
		WindowMinSize: image.Pt(600, 300),
		RunGameOptions: &ebiten.RunGameOptions{
			ApplePressAndHoldEnabled: true,
		},
	}
	if err := guigui.Run(&Root{}, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
