// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
)

// TextStyleEditor reads and writes the ranged style overrides of a widget's
// text value. The overrides follow the text through edits. An editor is
// obtained with [Text.StyleEditor] or [TextInput.StyleEditor] and stays
// valid as long as the widget.
type TextStyleEditor struct {
	text *Text
}

// SetVariationInRange overrides the OpenType variation axis tag in
// [startInBytes, endInBytes) with value. The override lasts until the value
// changes.
func (e TextStyleEditor) SetVariationInRange(startInBytes, endInBytes int, tag text.Tag, value float32) {
	e.text.core.SetVariationInRange(startInBytes, endInBytes, tag, value)
}

// UnsetVariationInRange removes the override of the OpenType variation axis
// tag in [startInBytes, endInBytes).
func (e TextStyleEditor) UnsetVariationInRange(startInBytes, endInBytes int, tag text.Tag) {
	e.text.core.UnsetVariationInRange(startInBytes, endInBytes, tag)
}

// SetFeatureInRange overrides the OpenType feature tag in
// [startInBytes, endInBytes) with value. The override lasts until the value
// changes.
func (e TextStyleEditor) SetFeatureInRange(startInBytes, endInBytes int, tag text.Tag, value uint32) {
	e.text.core.SetFeatureInRange(startInBytes, endInBytes, tag, value)
}

// UnsetFeatureInRange removes the override of the OpenType feature tag in
// [startInBytes, endInBytes).
func (e TextStyleEditor) UnsetFeatureInRange(startInBytes, endInBytes int, tag text.Tag) {
	e.text.core.UnsetFeatureInRange(startInBytes, endInBytes, tag)
}

// SetWeightInRange overrides the font weight in [startInBytes, endInBytes)
// by setting the wght variation axis. The override lasts until the value
// changes.
func (e TextStyleEditor) SetWeightInRange(startInBytes, endInBytes int, weight text.Weight) {
	e.text.core.SetVariationInRange(startInBytes, endInBytes, font.TagWght, float32(weight))
}

// UnsetWeightInRange removes the font weight override in
// [startInBytes, endInBytes).
func (e TextStyleEditor) UnsetWeightInRange(startInBytes, endInBytes int) {
	e.text.core.UnsetVariationInRange(startInBytes, endInBytes, font.TagWght)
}

// SetBoldInRange makes every byte of [startInBytes, endInBytes) bold or
// not bold by adjusting the ranged wght variation overrides over the base
// style. Overrides that would restate the base style are removed instead
// of set; making a range of a bold base style not bold overrides it with
// the default weight. An empty range is a no-op.
func (e TextStyleEditor) SetBoldInRange(startInBytes, endInBytes int, bold bool) {
	e.text.core.ApplyBoldInRange(startInBytes, endInBytes, bold)
}

// SetFontFamilyInRange overrides the font family in
// [startInBytes, endInBytes). A nil family renders the range with the
// registered face source stack alone.
func (e TextStyleEditor) SetFontFamilyInRange(startInBytes, endInBytes int, family *FontFamily) {
	var f *font.Family
	if family != nil {
		f = family.f
	}
	e.text.core.SetFontFamilyInRange(startInBytes, endInBytes, f)
}

// UnsetFontFamilyInRange removes the font family override in
// [startInBytes, endInBytes).
func (e TextStyleEditor) UnsetFontFamilyInRange(startInBytes, endInBytes int) {
	e.text.core.UnsetFontFamilyInRange(startInBytes, endInBytes)
}

// SetItalicInRange makes every byte of [startInBytes, endInBytes) render
// with an italic face or a regular face by adjusting the ranged italic
// overrides over the base style. Overrides that would restate the base
// style are removed instead of set. When the font family has no italic
// face, the range renders with a regular face. An empty range is a no-op.
func (e TextStyleEditor) SetItalicInRange(startInBytes, endInBytes int, italic bool) {
	e.text.core.ApplyItalicInRange(startInBytes, endInBytes, italic)
}

// UnsetItalicInRange removes the italic face selection override in
// [startInBytes, endInBytes).
func (e TextStyleEditor) UnsetItalicInRange(startInBytes, endInBytes int) {
	e.text.core.UnsetItalicInRange(startInBytes, endInBytes)
}

// SetScaleInRange overrides the font size in [startInBytes, endInBytes) as a
// multiplier applied to the base font size. The override lasts until the
// value changes. The line height is unaffected; the range renders on the
// line's baseline, and glyphs scaled past the line height may overlap
// adjacent lines.
func (e TextStyleEditor) SetScaleInRange(startInBytes, endInBytes int, scale float64) {
	e.text.core.SetScaleInRange(startInBytes, endInBytes, scale)
}

// UnsetScaleInRange removes the font size override in
// [startInBytes, endInBytes).
func (e TextStyleEditor) UnsetScaleInRange(startInBytes, endInBytes int) {
	e.text.core.UnsetScaleInRange(startInBytes, endInBytes)
}

// SetLangInRange overrides the language used to select the face and its
// features when shaping [startInBytes, endInBytes). The override lasts until
// the value changes.
func (e TextStyleEditor) SetLangInRange(startInBytes, endInBytes int, lang language.Tag) {
	e.text.core.SetLangInRange(startInBytes, endInBytes, lang)
}

// UnsetLangInRange removes the language override in
// [startInBytes, endInBytes).
func (e TextStyleEditor) UnsetLangInRange(startInBytes, endInBytes int) {
	e.text.core.UnsetLangInRange(startInBytes, endInBytes)
}

// SetColorInRange overrides the text color in [startInBytes, endInBytes). A
// nil color selects the default color.
func (e TextStyleEditor) SetColorInRange(startInBytes, endInBytes int, clr color.Color) {
	e.text.core.SetColorInRange(startInBytes, endInBytes, clr)
}

// UnsetColorInRange removes the text color override in
// [startInBytes, endInBytes).
func (e TextStyleEditor) UnsetColorInRange(startInBytes, endInBytes int) {
	e.text.core.UnsetColorInRange(startInBytes, endInBytes)
}

// SetBackgroundColorInRange overrides the background color in
// [startInBytes, endInBytes).
func (e TextStyleEditor) SetBackgroundColorInRange(startInBytes, endInBytes int, clr color.Color) {
	e.text.core.SetBackgroundColorInRange(startInBytes, endInBytes, clr)
}

// UnsetBackgroundColorInRange removes the background color override in
// [startInBytes, endInBytes).
func (e TextStyleEditor) UnsetBackgroundColorInRange(startInBytes, endInBytes int) {
	e.text.core.UnsetBackgroundColorInRange(startInBytes, endInBytes)
}

// SetUnderlineInRange sets whether an underline is drawn over every byte
// of [startInBytes, endInBytes). Underline false removes the underline
// overrides in the range. An empty range is a no-op.
func (e TextStyleEditor) SetUnderlineInRange(startInBytes, endInBytes int, underline bool) {
	e.text.core.ApplyUnderlineInRange(startInBytes, endInBytes, underline)
}

// UnsetUnderlineInRange removes the underline override in
// [startInBytes, endInBytes).
func (e TextStyleEditor) UnsetUnderlineInRange(startInBytes, endInBytes int) {
	e.text.core.UnsetUnderlineInRange(startInBytes, endInBytes)
}

// SetStrikethroughInRange sets whether a strikethrough is drawn over
// every byte of [startInBytes, endInBytes). Strikethrough false removes
// the strikethrough overrides in the range. An empty range is a no-op.
func (e TextStyleEditor) SetStrikethroughInRange(startInBytes, endInBytes int, strikethrough bool) {
	e.text.core.ApplyStrikethroughInRange(startInBytes, endInBytes, strikethrough)
}

// UnsetStrikethroughInRange removes the strikethrough override in
// [startInBytes, endInBytes).
func (e TextStyleEditor) UnsetStrikethroughInRange(startInBytes, endInBytes int) {
	e.text.core.UnsetStrikethroughInRange(startInBytes, endInBytes)
}

// ResetStylesInRange removes all style overrides in
// [startInBytes, endInBytes).
func (e TextStyleEditor) ResetStylesInRange(startInBytes, endInBytes int) {
	e.text.core.ResetStylesInRange(startInBytes, endInBytes)
}

// IsBoldInRange reports whether the effective font weight, the ranged wght
// variation overrides applied over the base style, is the bold weight over
// every byte of [startInBytes, endInBytes). For an empty range, it reports
// the state that text typed at startInBytes would adopt.
func (e TextStyleEditor) IsBoldInRange(startInBytes, endInBytes int) bool {
	return e.text.core.IsBoldInRange(startInBytes, endInBytes)
}

// IsItalicInRange reports whether the effective italic state, the ranged
// italic overrides applied over the base style, selects an italic face over
// every byte of [startInBytes, endInBytes). For an empty range, it reports
// the state that text typed at startInBytes would adopt.
func (e TextStyleEditor) IsItalicInRange(startInBytes, endInBytes int) bool {
	return e.text.core.IsItalicInRange(startInBytes, endInBytes)
}

// IsUnderlineInRange reports whether an underline is drawn over every byte
// of [startInBytes, endInBytes). For an empty range, it reports the state
// that text typed at startInBytes would adopt.
func (e TextStyleEditor) IsUnderlineInRange(startInBytes, endInBytes int) bool {
	return e.text.core.IsUnderlineInRange(startInBytes, endInBytes)
}

// IsStrikethroughInRange reports whether a strikethrough is drawn over
// every byte of [startInBytes, endInBytes). For an empty range, it reports
// the state that text typed at startInBytes would adopt.
func (e TextStyleEditor) IsStrikethroughInRange(startInBytes, endInBytes int) bool {
	return e.text.core.IsStrikethroughInRange(startInBytes, endInBytes)
}

// ColorInRange returns the text color override shared by every byte of
// [startInBytes, endInBytes) and whether the range is uniform. A nil color
// with uniform true means no byte has a color override, so the range
// renders in the base text color. For an empty range, it returns the
// override that text typed at startInBytes would adopt.
func (e TextStyleEditor) ColorInRange(startInBytes, endInBytes int) (clr color.Color, uniform bool) {
	return e.text.core.ColorInRange(startInBytes, endInBytes)
}

// ScaleInRange returns the font size multiplier shared by every byte of
// [startInBytes, endInBytes) and whether the range is uniform. A byte
// without a scale override takes the multiplier 1. For an empty range, it
// returns the multiplier that text typed at startInBytes would adopt.
func (e TextStyleEditor) ScaleInRange(startInBytes, endInBytes int) (scale float64, uniform bool) {
	return e.text.core.ScaleInRange(startInBytes, endInBytes)
}

// ReadStyles replaces styles with a detached copy of the ranged style
// overrides, reflecting the adjustments made for edits since the overrides
// were set. Base styles are not included.
func (e TextStyleEditor) ReadStyles(styles *TextStyles) {
	e.text.core.ReadStyleRuns(&styles.runs)
}

// SetStyles replaces the ranged style overrides with styles.
func (e TextStyleEditor) SetStyles(styles TextStyles) {
	e.text.core.CopyStyleRunsFrom(&styles.runs)
}
