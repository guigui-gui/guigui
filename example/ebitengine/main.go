// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var _ ebiten.Game = (*Game)(nil)

// Game is an Ebitengine game that rotates a gopher image.
type Game struct {
	gophersImage *ebiten.Image
	count        int
}

func NewGame() (*Game, error) {
	img, _, err := image.Decode(bytes.NewReader(images.Gophers_jpg))
	if err != nil {
		return nil, err
	}
	return &Game{
		gophersImage: ebiten.NewImageFromImage(img),
	}, nil
}

func (g *Game) Update() error {
	g.count++
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.gophersImage == nil {
		return
	}

	s := g.gophersImage.Bounds().Size()

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(s.X)/2, -float64(s.Y)/2)
	op.GeoM.Rotate(float64(g.count%360) * 2 * math.Pi / 360)
	op.GeoM.Translate(screenWidth/2, screenHeight/2)
	op.Filter = ebiten.FilterLinear

	screen.DrawImage(g.gophersImage, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// gameWidget is a Guigui widget that wraps an Ebitengine Game.
type gameWidget struct {
	guigui.DefaultWidget

	game      *Game
	paused    bool
	offscreen *ebiten.Image
}

func (w *gameWidget) Build(context *guigui.Context, childAdder *guigui.ChildAdder) error {
	if w.game == nil {
		g, err := NewGame()
		if err != nil {
			return err
		}
		w.game = g
	}
	return nil
}

func (w *gameWidget) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if !w.paused {
		w.game.Update()
		guigui.RequestRedraw(w)
	}
	return nil
}

func (w *gameWidget) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if w.game == nil {
		return
	}
	b := widgetBounds.Bounds()
	sw, sh := w.game.Layout(b.Dx(), b.Dy())
	if w.offscreen == nil || w.offscreen.Bounds().Dx() != sw || w.offscreen.Bounds().Dy() != sh {
		if w.offscreen != nil {
			w.offscreen.Deallocate()
		}
		w.offscreen = ebiten.NewImage(sw, sh)
	}
	w.offscreen.Clear()
	w.game.Draw(w.offscreen)

	scale := min(float64(b.Dx())/float64(sw), float64(b.Dy())/float64(sh))

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(b.Min.X)+(float64(b.Dx())-float64(sw)*scale)/2,
		float64(b.Min.Y)+(float64(b.Dy())-float64(sh)*scale)/2)
	op.Filter = ebiten.FilterPixelated
	dst.DrawImage(w.offscreen, op)
}

type Root struct {
	guigui.DefaultWidget

	background        basicwidget.Background
	gameWidget        gameWidget
	pauseResumeButton basicwidget.Button
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.gameWidget)
	adder.AddWidget(&r.pauseResumeButton)

	if r.gameWidget.paused {
		r.pauseResumeButton.SetText("Play")
	} else {
		r.pauseResumeButton.SetText("Pause")
	}
	r.pauseResumeButton.OnDown(func(context *guigui.Context) {
		r.gameWidget.paused = !r.gameWidget.paused
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
				Widget: &r.gameWidget,
				Size:   guigui.FlexibleSize(1),
			},
			{
				Widget: &r.pauseResumeButton,
			},
		},
		Gap: u / 2,
		Padding: guigui.Padding{
			Start:  u,
			Top:    u,
			End:    u,
			Bottom: u,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func main() {
	op := &guigui.RunOptions{
		Title:         "Ebitengine",
		WindowMinSize: image.Pt(640, 480),
	}
	if err := guigui.Run(&Root{}, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
