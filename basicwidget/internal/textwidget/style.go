// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

// textStyle holds a [Text]'s render-style configuration: alignment, scale,
// font selection, and the concrete colors set by a wrapping widget.
type textStyle struct {
	// hAlign is the horizontal alignment of the value.
	hAlign textutil.HorizontalAlign

	// vAlign is the vertical alignment of the value.
	vAlign textutil.VerticalAlign

	// scaleMinus1 is the text scale minus 1, so the zero value means scale 1.
	scaleMinus1 float64

	// bold renders the value in a bold weight.
	bold bool

	// tabular enables tabular figures.
	tabular bool

	// tabWidth is the tab width in pixels. A non-positive value selects the
	// default width.
	tabWidth float64

	// fontFamily is the resolved font family used to render the value, or nil
	// to render with the registered face source stack alone.
	fontFamily *font.Family

	// baseFontSize is the font size at scale 1; the rendered size is
	// baseFontSize multiplied by the widget scale.
	baseFontSize float64

	// baseLineHeight is the line height at scale 1; the rendered line height
	// is baseLineHeight multiplied by the widget scale.
	baseLineHeight float64

	// lang is the language used to select the face and its features when
	// shaping the value.
	lang language.Tag

	// langString is lang's string form, cached for [Text.WriteStateKey].
	langString string

	// textColor is the concrete color the value is drawn in.
	textColor color.Color

	// selectionColor is the concrete color of the selection highlight.
	selectionColor color.Color

	// inactiveCompositionColor is the concrete color of the inactive part of
	// an IME composition's underline.
	inactiveCompositionColor color.Color

	// activeCompositionColor is the concrete color of the active part of an
	// IME composition's underline.
	activeCompositionColor color.Color

	// caretColor is the concrete color of the caret.
	caretColor color.Color
}

// scale returns the text scale.
func (s *textStyle) scale() float64 {
	return s.scaleMinus1 + 1
}

// lineHeight returns the line height in pixels, with the scale applied.
func (s *textStyle) lineHeight() float64 {
	return s.baseLineHeight * s.scale()
}

// fontFamilyID returns fontFamily's ID, or 0 for a nil family.
func (s *textStyle) fontFamilyID() uint64 {
	if s.fontFamily == nil {
		return 0
	}
	return s.fontFamily.ID()
}

// faceAttributes returns the font attributes to shape the value with. liga
// sets whether ligatures are enabled.
func (s *textStyle) faceAttributes(forceBold bool, liga bool) font.Attributes {
	size := s.baseFontSize * s.scale()
	weight := text.WeightMedium
	if s.bold || forceBold {
		weight = text.WeightBold
	}
	return font.Attributes{
		Size:   size,
		Weight: weight,
		Liga:   liga,
		Tnum:   s.tabular,
		Lang:   s.lang,
	}
}
