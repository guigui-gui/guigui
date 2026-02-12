// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Settings struct {
	guigui.DefaultWidget

	form                      basicwidget.Form
	colorModeText             basicwidget.Text
	colorModeSegmentedControl basicwidget.SegmentedControl[string]
	localeText                textWithSubText
	localeSelect              basicwidget.Select[language.Tag]
	scaleText                 basicwidget.Text
	scaleSegmentedControl     basicwidget.SegmentedControl[float64]
}

var hongKongChinese = language.MustParse("zh-HK")

func (s *Settings) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&s.form)

	lightModeImg, err := theImageCache.GetMonochrome("light_mode", context.ColorMode())
	if err != nil {
		return err
	}
	darkModeImg, err := theImageCache.GetMonochrome("dark_mode", context.ColorMode())
	if err != nil {
		return err
	}

	s.colorModeText.SetValue("Color mode")
	s.colorModeSegmentedControl.SetItems([]basicwidget.SegmentedControlItem[string]{
		{
			Text:  "Auto",
			Value: "",
		},
		{
			Icon:  lightModeImg,
			Value: "light",
		},
		{
			Icon:  darkModeImg,
			Value: "dark",
		},
	})
	s.colorModeSegmentedControl.OnItemSelected(func(context *guigui.Context, index int) {
		item, ok := s.colorModeSegmentedControl.ItemByIndex(index)
		if !ok {
			context.SetColorMode(ebiten.ColorModeLight)
			return
		}
		switch item.Value {
		case "light":
			context.SetColorMode(ebiten.ColorModeLight)
		case "dark":
			context.SetColorMode(ebiten.ColorModeDark)
		default:
			context.UseAutoColorMode()
		}
	})
	if context.IsAutoColorModeUsed() {
		s.colorModeSegmentedControl.SelectItemByValue("")
	} else {
		switch context.ColorMode() {
		case ebiten.ColorModeLight:
			s.colorModeSegmentedControl.SelectItemByValue("light")
		case ebiten.ColorModeDark:
			s.colorModeSegmentedControl.SelectItemByValue("dark")
		default:
			s.colorModeSegmentedControl.SelectItemByValue("")
		}
	}

	s.localeText.text.SetValue("Locale")
	s.localeText.subText.SetValue("The locale affects the glyphs for Chinese characters.")

	s.localeSelect.SetItems([]basicwidget.SelectItem[language.Tag]{
		{
			Text:  "(Default)",
			Value: language.Und,
		},
		{
			Text:  "English",
			Value: language.English,
		},
		{
			Text:  "Japanese",
			Value: language.Japanese,
		},
		{
			Text:  "Korean",
			Value: language.Korean,
		},
		{
			Text:  "Simplified Chinese",
			Value: language.SimplifiedChinese,
		},
		{
			Text:  "Traditional Chinese",
			Value: language.TraditionalChinese,
		},
		{
			Text:  "Hong Kong Chinese",
			Value: hongKongChinese,
		},
	})
	s.localeSelect.OnItemSelected(func(context *guigui.Context, index int) {
		item, ok := s.localeSelect.ItemByIndex(index)
		if !ok {
			context.SetAppLocales(nil)
			return
		}
		if item.Value == language.Und {
			context.SetAppLocales(nil)
			return
		}
		context.SetAppLocales([]language.Tag{item.Value})
	})
	if !s.localeSelect.IsPopupOpen() {
		if locales := context.AppendAppLocales(nil); len(locales) > 0 {
			s.localeSelect.SelectItemByValue(locales[0])
		} else {
			s.localeSelect.SelectItemByValue(language.Und)
		}
	}

	s.scaleText.SetValue("Scale")
	s.scaleSegmentedControl.SetItems([]basicwidget.SegmentedControlItem[float64]{
		{
			Text:  "80%",
			Value: 0.8,
		},
		{
			Text:  "100%",
			Value: 1,
		},
		{
			Text:  "120%",
			Value: 1.2,
		},
	})
	s.scaleSegmentedControl.OnItemSelected(func(context *guigui.Context, index int) {
		item, ok := s.scaleSegmentedControl.ItemByIndex(index)
		if !ok {
			context.SetAppScale(1)
			return
		}
		context.SetAppScale(item.Value)
	})
	s.scaleSegmentedControl.SelectItemByValue(context.AppScale())

	s.form.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &s.colorModeText,
			SecondaryWidget: &s.colorModeSegmentedControl,
		},
		{
			PrimaryWidget:   &s.localeText,
			SecondaryWidget: &s.localeSelect,
		},
		{
			PrimaryWidget:   &s.scaleText,
			SecondaryWidget: &s.scaleSegmentedControl,
		},
	})

	return nil
}

func (s *Settings) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &s.form,
			},
		},
		Gap: u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

type textWithSubText struct {
	guigui.DefaultWidget

	text    basicwidget.Text
	subText basicwidget.Text
}

func (t *textWithSubText) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&t.text)
	adder.AddChild(&t.subText)
	t.subText.SetScale(0.875)
	t.subText.SetMultiline(true)
	t.subText.SetAutoWrap(true)
	t.subText.SetOpacity(0.675)
	return nil
}

func (t *textWithSubText) layout() guigui.LinearLayout {
	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &t.text,
			},
			{
				Widget: &t.subText,
			},
		},
	}
}

func (t *textWithSubText) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	t.layout().LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (t *textWithSubText) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.layout().Measure(context, constraints)
}
