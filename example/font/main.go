// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"slices"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	_ "github.com/guigui-gui/guigui/basicwidget/cjkfont"
)

const sampleText = "The quick brown fox / 日本語 / 中文 / 한국어"

type Root struct {
	guigui.DefaultWidget

	background basicwidget.Background

	defaultLabel  basicwidget.Text
	defaultText   basicwidget.Text
	goLabel       basicwidget.Text
	goText        basicwidget.Text
	goStrictLabel basicwidget.Text
	goStrictText  basicwidget.Text

	goFont       *basicwidget.FontFamily
	goStrictFont *basicwidget.FontFamily

	layoutItems []guigui.LinearLayoutItem
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.defaultLabel)
	adder.AddWidget(&r.defaultText)
	adder.AddWidget(&r.goLabel)
	adder.AddWidget(&r.goText)
	adder.AddWidget(&r.goStrictLabel)
	adder.AddWidget(&r.goStrictText)

	r.defaultLabel.SetValue("Default font:")
	r.defaultLabel.SetBold(true)
	r.defaultLabel.SetScale(1.2)
	r.defaultText.SetValue(sampleText)
	r.defaultText.SetScale(1.5)

	r.goLabel.SetValue("Go Regular with fallback:")
	r.goLabel.SetBold(true)
	r.goLabel.SetScale(1.2)
	r.goText.SetValue(sampleText)
	r.goText.SetScale(1.5)
	r.goText.SetFontFamily(r.goFont)

	r.goStrictLabel.SetValue("Go Regular without fallback:")
	r.goStrictLabel.SetBold(true)
	r.goStrictLabel.SetScale(1.2)
	r.goStrictText.SetValue(sampleText)
	r.goStrictText.SetScale(1.5)
	r.goStrictText.SetFontFamily(r.goStrictFont)

	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	b := widgetBounds.Bounds()
	layouter.LayoutWidget(&r.background, b)

	u := basicwidget.UnitSize(context)
	r.layoutItems = slices.Delete(r.layoutItems, 0, len(r.layoutItems))
	r.layoutItems = append(r.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &r.defaultLabel,
			Size:   guigui.FixedSize(u),
		},
		guigui.LinearLayoutItem{
			Widget: &r.defaultText,
			Size:   guigui.FixedSize(2 * u),
		},
		guigui.LinearLayoutItem{
			Widget: &r.goLabel,
			Size:   guigui.FixedSize(u),
		},
		guigui.LinearLayoutItem{
			Widget: &r.goText,
			Size:   guigui.FixedSize(2 * u),
		},
		guigui.LinearLayoutItem{
			Widget: &r.goStrictLabel,
			Size:   guigui.FixedSize(u),
		},
		guigui.LinearLayoutItem{
			Widget: &r.goStrictText,
			Size:   guigui.FixedSize(2 * u),
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     r.layoutItems,
		Gap:       u / 2,
		Padding: guigui.Padding{
			Start:  u,
			Top:    u,
			End:    u,
			Bottom: u,
		},
	}).LayoutWidgets(context, b, layouter)
}

func newGoFonts() (*basicwidget.FontFamily, *basicwidget.FontFamily, error) {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return nil, nil, err
	}
	entries := []basicwidget.FaceSourceEntry{
		{FaceSource: src},
	}
	withFallback := basicwidget.NewFontFamily(entries, nil)
	withoutFallback := basicwidget.NewFontFamily(entries, &basicwidget.FontFamilyOptions{
		DisableFallback: true,
	})
	return withFallback, withoutFallback, nil
}

func main() {
	withFallback, withoutFallback, err := newGoFonts()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	root := &Root{
		goFont:       withFallback,
		goStrictFont: withoutFallback,
	}
	op := &guigui.RunOptions{
		Title:         "Font",
		WindowMinSize: image.Pt(700, 480),
	}
	if err := guigui.Run(root, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
