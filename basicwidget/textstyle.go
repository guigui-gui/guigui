// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
)

// TextStyle is a set of text style properties where each property is
// individually set or unset. The zero value has every property unset.
// A TextStyle carries a widget's base style through [Text.ReadBaseStyle]
// and [Text.SetBaseStyle], and an effective style through
// [Text.ReadEffectiveStyleAt].
type TextStyle struct {
	s textstyle.Style
}

// writeStateKey writes the style's properties into w.
func (s *TextStyle) writeStateKey(w *guigui.StateKeyWriter) {
	var familyID uint64
	if family, ok := s.s.Family(); ok && family != nil {
		familyID = family.ID()
	}
	w.WriteUint64(familyID)

	italic, _ := s.s.Italic()
	w.WriteBool(italic)
	scale, _ := s.s.Scale()
	w.WriteFloat64(scale)
	lang, _ := s.s.Lang()
	w.WriteString(lang.String())
	underline, _ := s.s.Underline()
	w.WriteBool(underline)
	strikethrough, _ := s.s.Strikethrough()
	w.WriteBool(strikethrough)

	clr, _ := s.s.Color()
	writeColor(w, clr)
	backgroundColor, _ := s.s.BackgroundColor()
	writeColor(w, backgroundColor)

	variations := s.s.Variations()
	w.WriteInt(len(variations))
	for _, v := range variations {
		w.WriteUint32(uint32(v.Tag))
		w.WriteFloat64(float64(v.Value))
	}
	features := s.s.Features()
	w.WriteInt(len(features))
	for _, f := range features {
		w.WriteUint32(uint32(f.Tag))
		w.WriteUint32(f.Value)
	}
}

// SetFontFamily sets the font family. A nil family renders with the
// registered face source stack alone.
func (s *TextStyle) SetFontFamily(family *FontFamily) {
	var f *font.Family
	if family != nil {
		f = family.f
	}
	s.s = s.s.WithFamily(f)
}

// UnsetFontFamily unsets the font family.
func (s *TextStyle) UnsetFontFamily() {
	s.s = s.s.WithoutFamily()
}

// FontFamily returns the font family and whether it is set.
func (s *TextStyle) FontFamily() (*FontFamily, bool) {
	f, ok := s.s.Family()
	if !ok || f == nil {
		return nil, ok
	}
	return &FontFamily{f: f}, true
}

// SetItalic sets the italic face selection.
func (s *TextStyle) SetItalic(italic bool) {
	s.s = s.s.WithItalic(italic)
}

// UnsetItalic unsets the italic face selection.
func (s *TextStyle) UnsetItalic() {
	s.s = s.s.WithoutItalic()
}

// Italic returns the italic face selection and whether it is set.
func (s *TextStyle) Italic() (italic, ok bool) {
	return s.s.Italic()
}

// SetWeight sets the font weight by setting the wght variation axis.
func (s *TextStyle) SetWeight(weight text.Weight) {
	s.s = s.s.WithVariation(font.TagWght, float32(weight))
}

// UnsetWeight unsets the font weight by unsetting the wght variation axis.
func (s *TextStyle) UnsetWeight() {
	s.s = s.s.WithoutVariation(font.TagWght)
}

// Weight returns the font weight, the value of the wght variation axis, and
// whether it is set.
func (s *TextStyle) Weight() (text.Weight, bool) {
	v, ok := s.s.Variation(font.TagWght)
	return text.Weight(v), ok
}

// SetBold sets the font weight to the bold or the medium weight by setting
// the wght variation axis. Use [TextStyle.UnsetWeight] to unset the weight
// and [TextStyle.Weight] to read it back.
func (s *TextStyle) SetBold(bold bool) {
	weight := text.WeightMedium
	if bold {
		weight = text.WeightBold
	}
	s.s = s.s.WithVariation(font.TagWght, float32(weight))
}

// SetVariation sets the OpenType variation axis tag to value.
func (s *TextStyle) SetVariation(tag text.Tag, value float32) {
	s.s = s.s.WithVariation(tag, value)
}

// UnsetVariation unsets the OpenType variation axis tag.
func (s *TextStyle) UnsetVariation(tag text.Tag) {
	s.s = s.s.WithoutVariation(tag)
}

// Variation returns the value of the OpenType variation axis tag and whether
// it is set.
func (s *TextStyle) Variation(tag text.Tag) (float32, bool) {
	return s.s.Variation(tag)
}

// SetFeature sets the OpenType feature tag to value.
func (s *TextStyle) SetFeature(tag text.Tag, value uint32) {
	s.s = s.s.WithFeature(tag, value)
}

// UnsetFeature unsets the OpenType feature tag.
func (s *TextStyle) UnsetFeature(tag text.Tag) {
	s.s = s.s.WithoutFeature(tag)
}

// Feature returns the value of the OpenType feature tag and whether it is
// set.
func (s *TextStyle) Feature(tag text.Tag) (uint32, bool) {
	return s.s.Feature(tag)
}

// SetTabular sets whether the digits are rendered with a uniform advance by
// setting the tnum OpenType feature.
func (s *TextStyle) SetTabular(tabular bool) {
	var value uint32
	if tabular {
		value = 1
	}
	s.s = s.s.WithFeature(font.TagTnum, value)
}

// UnsetTabular unsets whether the digits are rendered with a uniform advance
// by unsetting the tnum OpenType feature.
func (s *TextStyle) UnsetTabular() {
	s.s = s.s.WithoutFeature(font.TagTnum)
}

// Tabular returns whether the digits are rendered with a uniform advance, the
// value of the tnum OpenType feature, and whether it is set.
func (s *TextStyle) Tabular() (tabular, ok bool) {
	v, ok := s.s.Feature(font.TagTnum)
	return v != 0, ok
}

// SetScale sets the multiplier of the font size. scale must be positive.
func (s *TextStyle) SetScale(scale float64) {
	s.s = s.s.WithScale(scale)
}

// UnsetScale unsets the multiplier of the font size.
func (s *TextStyle) UnsetScale() {
	s.s = s.s.WithoutScale()
}

// Scale returns the multiplier of the font size and whether it is set.
func (s *TextStyle) Scale() (float64, bool) {
	return s.s.Scale()
}

// SetLang sets the language used for rendering.
func (s *TextStyle) SetLang(lang language.Tag) {
	s.s = s.s.WithLang(lang)
}

// UnsetLang unsets the language used for rendering.
func (s *TextStyle) UnsetLang() {
	s.s = s.s.WithoutLang()
}

// Lang returns the language used for rendering and whether it is set.
func (s *TextStyle) Lang() (language.Tag, bool) {
	return s.s.Lang()
}

// SetColor sets the text color.
func (s *TextStyle) SetColor(clr color.Color) {
	s.s = s.s.WithColor(clr)
}

// UnsetColor unsets the text color.
func (s *TextStyle) UnsetColor() {
	s.s = s.s.WithoutColor()
}

// Color returns the text color and whether it is set.
func (s *TextStyle) Color() (color.Color, bool) {
	return s.s.Color()
}

// SetBackgroundColor sets the background color.
func (s *TextStyle) SetBackgroundColor(clr color.Color) {
	s.s = s.s.WithBackgroundColor(clr)
}

// UnsetBackgroundColor unsets the background color.
func (s *TextStyle) UnsetBackgroundColor() {
	s.s = s.s.WithoutBackgroundColor()
}

// BackgroundColor returns the background color and whether it is set.
func (s *TextStyle) BackgroundColor() (color.Color, bool) {
	return s.s.BackgroundColor()
}

// SetUnderline sets whether an underline is drawn.
func (s *TextStyle) SetUnderline(underline bool) {
	s.s = s.s.WithUnderline(underline)
}

// UnsetUnderline unsets whether an underline is drawn.
func (s *TextStyle) UnsetUnderline() {
	s.s = s.s.WithoutUnderline()
}

// Underline returns whether an underline is drawn and whether the property
// is set.
func (s *TextStyle) Underline() (underline, ok bool) {
	return s.s.Underline()
}

// SetStrikethrough sets whether a strikethrough is drawn.
func (s *TextStyle) SetStrikethrough(strikethrough bool) {
	s.s = s.s.WithStrikethrough(strikethrough)
}

// UnsetStrikethrough unsets whether a strikethrough is drawn.
func (s *TextStyle) UnsetStrikethrough() {
	s.s = s.s.WithoutStrikethrough()
}

// Strikethrough returns whether a strikethrough is drawn and whether the
// property is set.
func (s *TextStyle) Strikethrough() (strikethrough, ok bool) {
	return s.s.Strikethrough()
}
