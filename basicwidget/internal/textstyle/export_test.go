// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textstyle

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
)

// The With* methods build Style values for test expectations.

func (s Style) WithFamily(family *font.Family) Style {
	s.family = opt(family)
	return s
}

func (s Style) WithItalic(italic bool) Style {
	s.italic = opt(italic)
	return s
}

func (s Style) WithScale(scale float64) Style {
	s.scale = opt(scale)
	return s
}

func (s Style) WithColor(clr color.Color) Style {
	s.color = opt(clr)
	return s
}

func (s Style) WithBackgroundColor(clr color.Color) Style {
	s.backgroundColor = opt(clr)
	return s
}

func (s Style) WithUnderline(underline bool) Style {
	s.underline = opt(underline)
	return s
}

func (s Style) WithStrikethrough(strikethrough bool) Style {
	s.strikethrough = opt(strikethrough)
	return s
}

func (s Style) WithLang(lang language.Tag) Style {
	s.lang = opt(lang)
	return s
}

func (s Style) WithFeature(tag text.Tag, value uint32) Style {
	s.features = mergeTagged(s.features, []font.Feature{{Tag: tag, Value: value}}, func(f font.Feature) text.Tag {
		return f.Tag
	})
	return s
}

func (s Style) WithVariation(tag text.Tag, value float32) Style {
	s.variations = mergeTagged(s.variations, []font.Variation{{Tag: tag, Value: value}}, func(v font.Variation) text.Tag {
		return v.Tag
	})
	return s
}
