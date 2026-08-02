// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
)

// TextStyles is a set of ranged style overrides for a [Text] value. The
// zero value is an empty set ready for use. A set built with the Set*
// methods is applied to a widget with [TextStyleEditor.SetStyles];
// [TextStyleEditor.ReadStyles] reads a widget's current overrides back.
type TextStyles struct {
	runs textstyle.Runs
}

// SetVariationInRange overrides the OpenType variation axis tag in
// [startInBytes, endInBytes) with value.
func (s *TextStyles) SetVariationInRange(startInBytes, endInBytes int, tag text.Tag, value float32) {
	s.runs.SetVariation(startInBytes, endInBytes, tag, value)
}

// UnsetVariationInRange removes the override of the OpenType variation axis
// tag in [startInBytes, endInBytes).
func (s *TextStyles) UnsetVariationInRange(startInBytes, endInBytes int, tag text.Tag) {
	s.runs.UnsetVariation(startInBytes, endInBytes, tag)
}

// SetFeatureInRange overrides the OpenType feature tag in
// [startInBytes, endInBytes) with value.
func (s *TextStyles) SetFeatureInRange(startInBytes, endInBytes int, tag text.Tag, value uint32) {
	s.runs.SetFeature(startInBytes, endInBytes, tag, value)
}

// UnsetFeatureInRange removes the override of the OpenType feature tag in
// [startInBytes, endInBytes).
func (s *TextStyles) UnsetFeatureInRange(startInBytes, endInBytes int, tag text.Tag) {
	s.runs.UnsetFeature(startInBytes, endInBytes, tag)
}

// SetWeightInRange overrides the font weight in [startInBytes, endInBytes)
// by setting the wght variation axis.
func (s *TextStyles) SetWeightInRange(startInBytes, endInBytes int, weight text.Weight) {
	s.runs.SetVariation(startInBytes, endInBytes, font.TagWght, float32(weight))
}

// UnsetWeightInRange removes the font weight override in
// [startInBytes, endInBytes).
func (s *TextStyles) UnsetWeightInRange(startInBytes, endInBytes int) {
	s.runs.UnsetVariation(startInBytes, endInBytes, font.TagWght)
}

// SetFontFamilyInRange overrides the font family in
// [startInBytes, endInBytes). A nil family renders the range with the
// registered face source stack alone.
func (s *TextStyles) SetFontFamilyInRange(startInBytes, endInBytes int, family *FontFamily) {
	var f *font.Family
	if family != nil {
		f = family.f
	}
	s.runs.SetFamily(startInBytes, endInBytes, f)
}

// UnsetFontFamilyInRange removes the font family override in
// [startInBytes, endInBytes).
func (s *TextStyles) UnsetFontFamilyInRange(startInBytes, endInBytes int) {
	s.runs.UnsetFamily(startInBytes, endInBytes)
}

// SetItalicInRange overrides the italic face selection in
// [startInBytes, endInBytes).
func (s *TextStyles) SetItalicInRange(startInBytes, endInBytes int, italic bool) {
	s.runs.SetItalic(startInBytes, endInBytes, italic)
}

// UnsetItalicInRange removes the italic override in
// [startInBytes, endInBytes).
func (s *TextStyles) UnsetItalicInRange(startInBytes, endInBytes int) {
	s.runs.UnsetItalic(startInBytes, endInBytes)
}

// SetScaleInRange overrides the multiplier of the font size in
// [startInBytes, endInBytes). scale must be positive.
func (s *TextStyles) SetScaleInRange(startInBytes, endInBytes int, scale float64) {
	s.runs.SetScale(startInBytes, endInBytes, scale)
}

// UnsetScaleInRange removes the font size multiplier override in
// [startInBytes, endInBytes).
func (s *TextStyles) UnsetScaleInRange(startInBytes, endInBytes int) {
	s.runs.UnsetScale(startInBytes, endInBytes)
}

// SetLangInRange overrides the language used for rendering in
// [startInBytes, endInBytes).
func (s *TextStyles) SetLangInRange(startInBytes, endInBytes int, lang language.Tag) {
	s.runs.SetLang(startInBytes, endInBytes, lang)
}

// UnsetLangInRange removes the language override in
// [startInBytes, endInBytes).
func (s *TextStyles) UnsetLangInRange(startInBytes, endInBytes int) {
	s.runs.UnsetLang(startInBytes, endInBytes)
}

// SetColorInRange overrides the text color in [startInBytes, endInBytes).
func (s *TextStyles) SetColorInRange(startInBytes, endInBytes int, clr color.Color) {
	s.runs.SetColor(startInBytes, endInBytes, clr)
}

// UnsetColorInRange removes the text color override in
// [startInBytes, endInBytes).
func (s *TextStyles) UnsetColorInRange(startInBytes, endInBytes int) {
	s.runs.UnsetColor(startInBytes, endInBytes)
}

// SetBackgroundColorInRange overrides the background color in
// [startInBytes, endInBytes).
func (s *TextStyles) SetBackgroundColorInRange(startInBytes, endInBytes int, clr color.Color) {
	s.runs.SetBackgroundColor(startInBytes, endInBytes, clr)
}

// UnsetBackgroundColorInRange removes the background color override in
// [startInBytes, endInBytes).
func (s *TextStyles) UnsetBackgroundColorInRange(startInBytes, endInBytes int) {
	s.runs.UnsetBackgroundColor(startInBytes, endInBytes)
}

// SetUnderlineInRange overrides whether an underline is drawn in
// [startInBytes, endInBytes).
func (s *TextStyles) SetUnderlineInRange(startInBytes, endInBytes int, underline bool) {
	s.runs.SetUnderline(startInBytes, endInBytes, underline)
}

// UnsetUnderlineInRange removes the underline override in
// [startInBytes, endInBytes).
func (s *TextStyles) UnsetUnderlineInRange(startInBytes, endInBytes int) {
	s.runs.UnsetUnderline(startInBytes, endInBytes)
}

// SetStrikethroughInRange overrides whether a strikethrough is drawn in
// [startInBytes, endInBytes).
func (s *TextStyles) SetStrikethroughInRange(startInBytes, endInBytes int, strikethrough bool) {
	s.runs.SetStrikethrough(startInBytes, endInBytes, strikethrough)
}

// UnsetStrikethroughInRange removes the strikethrough override in
// [startInBytes, endInBytes).
func (s *TextStyles) UnsetStrikethroughInRange(startInBytes, endInBytes int) {
	s.runs.UnsetStrikethrough(startInBytes, endInBytes)
}

// ResetStylesInRange removes all style overrides in
// [startInBytes, endInBytes).
func (s *TextStyles) ResetStylesInRange(startInBytes, endInBytes int) {
	s.runs.Reset(startInBytes, endInBytes)
}

// VariationInRange returns the value of the OpenType variation axis tag
// every byte of [startInBytes, endInBytes) takes and whether the value is
// uniform. A byte with an override of the axis takes the override's value;
// every other byte takes fallback. An empty range is uniform with fallback.
func (s *TextStyles) VariationInRange(startInBytes, endInBytes int, tag text.Tag, fallback float32) (value float32, uniform bool) {
	return s.runs.UniformVariation(startInBytes, endInBytes, tag, fallback)
}

// WeightInRange returns the font weight every byte of
// [startInBytes, endInBytes) takes and whether the weight is uniform. A byte
// with a wght variation override takes the override's value; every other
// byte takes fallback. An empty range is uniform with fallback.
func (s *TextStyles) WeightInRange(startInBytes, endInBytes int, fallback text.Weight) (weight text.Weight, uniform bool) {
	v, uniform := s.runs.UniformVariation(startInBytes, endInBytes, font.TagWght, float32(fallback))
	return text.Weight(v), uniform
}

// ItalicInRange returns the italic state every byte of
// [startInBytes, endInBytes) takes and whether the state is uniform. A byte
// with an italic override takes the override's value; every other byte takes
// fallback. An empty range is uniform with fallback.
func (s *TextStyles) ItalicInRange(startInBytes, endInBytes int, fallback bool) (italic, uniform bool) {
	return s.runs.UniformItalic(startInBytes, endInBytes, fallback)
}

// ScaleInRange returns the font size multiplier every byte of
// [startInBytes, endInBytes) takes and whether the multiplier is uniform. A
// byte with a scale override takes the override's value; every other byte
// takes fallback. An empty range is uniform with fallback.
func (s *TextStyles) ScaleInRange(startInBytes, endInBytes int, fallback float64) (scale float64, uniform bool) {
	return s.runs.UniformScale(startInBytes, endInBytes, fallback)
}

// ColorInRange returns the text color every byte of
// [startInBytes, endInBytes) takes and whether the color is uniform. A byte
// with a color override takes the override's value; every other byte takes
// fallback. An empty range is uniform with fallback.
func (s *TextStyles) ColorInRange(startInBytes, endInBytes int, fallback color.Color) (clr color.Color, uniform bool) {
	return s.runs.UniformColor(startInBytes, endInBytes, fallback)
}

// UnderlineInRange returns the underline state every byte of
// [startInBytes, endInBytes) takes and whether the state is uniform. A byte
// with an underline override takes the override's value; every other byte
// takes fallback. An empty range is uniform with fallback.
func (s *TextStyles) UnderlineInRange(startInBytes, endInBytes int, fallback bool) (underline, uniform bool) {
	return s.runs.UniformUnderline(startInBytes, endInBytes, fallback)
}

// StrikethroughInRange returns the strikethrough state every byte of
// [startInBytes, endInBytes) takes and whether the state is uniform. A byte
// with a strikethrough override takes the override's value; every other byte
// takes fallback. An empty range is uniform with fallback.
func (s *TextStyles) StrikethroughInRange(startInBytes, endInBytes int, fallback bool) (strikethrough, uniform bool) {
	return s.runs.UniformStrikethrough(startInBytes, endInBytes, fallback)
}
