// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"fmt"
	"image"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	"github.com/guigui-gui/guigui/ebitenginewidget"
)

// defaultPackage is the Ebitengine application built and shown at startup.
const defaultPackage = "github.com/hajimehoshi/ebiten/v2/examples/rotate"

// buildResult is the outcome of an asynchronous guest build.
type buildResult struct {
	pkg string
	bin string
	err error
}

type Root struct {
	guigui.DefaultWidget

	background       basicwidget.Background
	ebitengineWidget ebitenginewidget.Ebitengine
	form             basicwidget.Form
	packageText      basicwidget.Text
	packageField     packageField
	tpsText          basicwidget.Text
	tpsField         tpsField
	statusText       basicwidget.Text

	initialized    bool
	startedInitial bool

	// dir is the temporary directory guest binaries are built into.
	dir string

	// buildResults receives the outcomes of the asynchronous guest builds.
	buildResults chan buildResult

	// building reports whether a build is in flight.
	building bool

	// buildGen numbers the builds, giving each guest binary its own path.
	buildGen int

	// builtBinary is the most recently built guest binary, handed to the widget.
	builtBinary string

	// launchedPackage is the package of the most recently adopted build, which the widget runs. The
	// text field's buffer may have been edited since, so status messages use this instead.
	launchedPackage string

	// requestedTPS is the rate the current guest's game requests, 0 until reported.
	requestedTPS int

	tps    int
	status string

	layoutItems []guigui.LinearLayoutItem
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.ebitengineWidget)
	adder.AddWidget(&r.form)

	if !r.initialized {
		r.initialized = true
		r.tps = ebiten.DefaultTPS
		r.packageField.textInput.SetValue(defaultPackage)
		r.status = "Building " + defaultPackage + " ..."
	}

	r.packageText.SetValue("Package")
	r.packageField.textInput.SetPlaceholder("Import path or ./local path")
	r.packageField.textInput.OnHandleButtonInput(func(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
			r.launch()
			return guigui.HandleInputByWidget(&r.packageField.textInput)
		}
		return guigui.HandleInputResult{}
	})

	r.packageField.launchButton.SetText("Launch")
	context.SetEnabled(&r.packageField.launchButton, !r.building)
	r.packageField.launchButton.OnDown(func(context *guigui.Context) {
		r.launch()
	})

	r.tpsText.SetValue(fmt.Sprintf("Guest TPS: %d", r.tps))
	r.tpsField.slider.SetMinimumValue(0)
	r.tpsField.slider.SetMaximumValue(300)
	r.tpsField.slider.SetValue(r.tps)
	r.tpsField.slider.OnValueChanged(func(context *guigui.Context, value int) {
		r.tps = value
	})

	r.tpsField.resetButton.SetText("Reset")
	context.SetEnabled(&r.tpsField.resetButton, r.requestedTPS > 0 && r.tps != r.requestedTPS)
	r.tpsField.resetButton.OnDown(func(context *guigui.Context) {
		r.tps = r.requestedTPS
	})

	if r.builtBinary != "" {
		r.ebitengineWidget.SetBinaryPath(r.builtBinary)
	}
	r.ebitengineWidget.SetTPS(r.tps)
	r.ebitengineWidget.OnLaunched(func(context *guigui.Context) {
		r.status = "Running " + r.launchedPackage
	})
	// Pace the guest at the rate its own game requests, and reflect it in the slider.
	r.ebitengineWidget.OnTPSRequested(func(context *guigui.Context, tps int) {
		r.requestedTPS = tps
		r.tps = tps
	})
	r.ebitengineWidget.OnExited(func(context *guigui.Context) {
		r.status = r.launchedPackage + " exited"
	})
	r.ebitengineWidget.OnError(func(context *guigui.Context, err error) {
		r.status = "Error: " + err.Error()
	})

	r.statusText.SetValue(r.status)

	r.form.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &r.packageText,
			SecondaryWidget: &r.packageField,
		},
		{
			PrimaryWidget:   &r.tpsText,
			SecondaryWidget: &r.tpsField,
		},
		{
			PrimaryWidget: &r.statusText,
		},
	})

	return nil
}

func (r *Root) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if !r.startedInitial {
		r.startedInitial = true
		r.startBuild(defaultPackage)
	}

	// Adopt an asynchronously-built guest binary once it is ready.
	select {
	case res := <-r.buildResults:
		r.building = false
		if res.err != nil {
			slog.Error(res.err.Error())
			r.status = "Build failed (see console)"
		} else {
			r.builtBinary = res.bin
			r.launchedPackage = res.pkg
			r.status = "Launching " + res.pkg
		}
		guigui.RequestRebuild()
	default:
	}
	return nil
}

// launch builds the package currently in the text field.
func (r *Root) launch() {
	r.startBuild(r.packageField.textInput.Value())
}

// startBuild kicks off an asynchronous build of pkg into a fresh guest binary, unless one is already in
// flight. The result is adopted in [Root.Tick].
func (r *Root) startBuild(pkg string) {
	if pkg == "" || r.building {
		return
	}
	if r.buildResults == nil {
		r.buildResults = make(chan buildResult, 1)
	}
	if r.dir == "" {
		dir, err := os.MkdirTemp("", "guigui-ebitengine-example")
		if err != nil {
			r.status = "Error: " + err.Error()
			guigui.RequestRebuild()
			return
		}
		r.dir = dir
	}

	r.building = true
	r.buildGen++
	r.status = "Building " + pkg + " ..."

	// A new path per build: an old guest's binary may still be running (and locked, on Windows) while the
	// next one builds.
	bin := filepath.Join(r.dir, fmt.Sprintf("guest-%d", r.buildGen))
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	go func() {
		// The ebitenginevm build tag makes the guest dial EBITENGINE_VM_ENDPOINT instead of opening a
		// window.
		err := buildGuest(r.dir, bin, pkg)
		r.buildResults <- buildResult{pkg: pkg, bin: bin, err: err}
	}()
	guigui.RequestRebuild()
}

// packageField is a form field with a package text input and a launch button.
type packageField struct {
	guigui.DefaultWidget

	textInput    basicwidget.TextInput
	launchButton basicwidget.Button

	layoutItems []guigui.LinearLayoutItem
}

func (p *packageField) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&p.textInput)
	adder.AddWidget(&p.launchButton)

	p.textInput.SetHorizontalAlign(basicwidget.HorizontalAlignEnd)

	return nil
}

func (p *packageField) layout(context *guigui.Context) guigui.LinearLayout {
	u := basicwidget.UnitSize(context)
	p.layoutItems = slices.Delete(p.layoutItems, 0, len(p.layoutItems))
	p.layoutItems = append(p.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &p.textInput,
			Size:   guigui.FixedSize(15 * u),
		},
		guigui.LinearLayoutItem{
			Widget: &p.launchButton,
			Size:   guigui.FixedSize(3 * u),
		},
	)
	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     p.layoutItems,
		Gap:       u / 4,
	}
}

func (p *packageField) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	p.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (p *packageField) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return p.layout(context).Measure(context, constraints)
}

// tpsField is a form field with a TPS slider and a button resetting it to the guest's requested rate.
type tpsField struct {
	guigui.DefaultWidget

	slider      basicwidget.Slider
	resetButton basicwidget.Button

	layoutItems []guigui.LinearLayoutItem
}

func (t *tpsField) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.slider)
	adder.AddWidget(&t.resetButton)
	return nil
}

func (t *tpsField) layout(context *guigui.Context) guigui.LinearLayout {
	u := basicwidget.UnitSize(context)
	t.layoutItems = slices.Delete(t.layoutItems, 0, len(t.layoutItems))
	t.layoutItems = append(t.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &t.slider,
			Size:   guigui.FixedSize(12 * u),
		},
		guigui.LinearLayoutItem{
			Widget: &t.resetButton,
			Size:   guigui.FixedSize(3 * u),
		},
	)
	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     t.layoutItems,
		Gap:       u / 4,
	}
}

func (t *tpsField) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	t.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (t *tpsField) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.layout(context).Measure(context, constraints)
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())

	u := basicwidget.UnitSize(context)

	r.layoutItems = slices.Delete(r.layoutItems, 0, len(r.layoutItems))
	r.layoutItems = append(r.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &r.ebitengineWidget,
			Size:   guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: &r.form,
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     r.layoutItems,
		Gap:       u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
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
