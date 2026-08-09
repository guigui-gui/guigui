// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"os"
	"runtime"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	"github.com/guigui-gui/guigui/clipboard"
)

var placeholderMessage = fmt.Sprintf("No image yet.\n\n"+
	"Copy an image, then press %s or the Paste button.\n\n"+
	"The clipboard image is read as PNG. Most apps offer their image as PNG, such as a screenshot copied on macOS or Copy Image in a browser. "+
	"An app that offers an image only in another format, such as TIFF or a bitmap, has nothing to show here.", hotkey("V"))

type Root struct {
	guigui.DefaultWidget

	background  basicwidget.Background
	image       basicwidget.Image
	placeholder basicwidget.Text
	status      basicwidget.Text
	pasteButton basicwidget.Button

	pastedImage *ebiten.Image
	statusText  string

	bottomRowLayout guigui.LinearLayout
	bottomRowItems  []guigui.LinearLayoutItem
	layoutItems     []guigui.LinearLayoutItem
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	if r.pastedImage != nil {
		adder.AddWidget(&r.image)
	} else {
		adder.AddWidget(&r.placeholder)
	}
	adder.AddWidget(&r.status)
	adder.AddWidget(&r.pasteButton)

	r.image.SetImage(r.pastedImage)

	r.placeholder.SetMultiline(true)
	r.placeholder.SetWrapMode(basicwidget.WrapModeNormal)
	r.placeholder.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)
	r.placeholder.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	r.placeholder.SetValue(placeholderMessage)

	r.status.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	r.status.SetValue(r.statusText)

	r.pasteButton.SetText("Paste")
	r.pasteButton.OnDown(func(context *guigui.Context) {
		r.paste()
	})

	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())

	u := basicwidget.UnitSize(context)
	r.bottomRowItems = slices.Delete(r.bottomRowItems, 0, len(r.bottomRowItems))
	r.bottomRowItems = append(r.bottomRowItems,
		guigui.LinearLayoutItem{
			Widget: &r.status,
			Size:   guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: &r.pasteButton,
			Size:   guigui.FixedSize(6 * u),
		},
	)
	r.bottomRowLayout = guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     r.bottomRowItems,
		Gap:       u / 2,
	}

	var content guigui.Widget = &r.placeholder
	if r.pastedImage != nil {
		content = &r.image
	}
	r.layoutItems = slices.Delete(r.layoutItems, 0, len(r.layoutItems))
	r.layoutItems = append(r.layoutItems,
		guigui.LinearLayoutItem{
			Widget: content,
			Size:   guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Size:   guigui.FixedSize(u),
			Layout: &r.bottomRowLayout,
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     r.layoutItems,
		Gap:       u,
		Padding: guigui.Padding{
			Start:  u,
			Top:    u,
			End:    u,
			Bottom: u,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

// HandleButtonInput handles the paste shortcut. It lives on the root so that
// it works wherever the focus is.
func (r *Root) HandleButtonInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if !cmdPressed() || !inpututil.IsKeyJustPressed(ebiten.KeyV) {
		return guigui.HandleInputResult{}
	}
	r.paste()
	return guigui.HandleInputByWidget(r)
}

func (r *Root) paste() {
	contents, err := clipboard.Read()
	if err != nil {
		slog.Error("failed to read the clipboard", "error", err)
		r.statusText = "Failed to read the clipboard."
		return
	}
	if len(contents.PNG) == 0 {
		r.pastedImage = nil
		r.statusText = "The clipboard has no PNG image."
		return
	}
	img, err := png.Decode(bytes.NewReader(contents.PNG))
	if err != nil {
		slog.Error("failed to decode the clipboard image", "error", err)
		r.pastedImage = nil
		r.statusText = "Failed to decode the clipboard image."
		return
	}
	r.pastedImage = ebiten.NewImageFromImage(img)
	size := r.pastedImage.Bounds().Size()
	r.statusText = fmt.Sprintf("Pasted a %d×%d image.", size.X, size.Y)
}

// cmdPressed reports whether the platform's primary command modifier is
// pressed: Command on macOS, Control elsewhere.
func cmdPressed() bool {
	if runtime.GOOS == "darwin" {
		return ebiten.IsKeyPressed(ebiten.KeyMeta)
	}
	return ebiten.IsKeyPressed(ebiten.KeyControl)
}

// hotkey returns the platform-conventional display label of a shortcut with
// the primary command modifier.
func hotkey(key string) string {
	if runtime.GOOS == "darwin" {
		return "⌘" + key
	}
	return "Ctrl+" + key
}

func main() {
	op := &guigui.RunOptions{
		Title:         "Image Clipboard",
		WindowMinSize: image.Pt(600, 400),
	}
	if err := guigui.Run(&Root{}, op); err != nil {
		slog.Error("guigui.Run", "err", err)
		os.Exit(1)
	}
}
