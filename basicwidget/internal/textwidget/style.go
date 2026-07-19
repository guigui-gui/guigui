// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
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

// ensureStyleRuns clears the ranged style overrides if the store's
// renderable content has been mutated since they were applied, and returns
// the runs.
func (t *Text) ensureStyleRuns() *textstyle.Runs {
	if gen := t.store.Generation(); t.styleRunsValidGeneration != gen {
		t.styleRuns.Clear()
		t.styleRunsValidGeneration = gen
	}
	return &t.styleRuns
}

// SetColorInRange overrides the text color in [startInBytes, endInBytes).
// The override lasts until the value changes.
func (t *Text) SetColorInRange(startInBytes, endInBytes int, clr color.Color) {
	t.ensureStyleRuns().SetColor(startInBytes, endInBytes, clr)
}

// UnsetColorInRange removes the text color override in
// [startInBytes, endInBytes).
func (t *Text) UnsetColorInRange(startInBytes, endInBytes int) {
	t.ensureStyleRuns().UnsetColor(startInBytes, endInBytes)
}

// SetBackgroundColorInRange overrides the background color in
// [startInBytes, endInBytes). The override lasts until the value changes.
func (t *Text) SetBackgroundColorInRange(startInBytes, endInBytes int, clr color.Color) {
	t.ensureStyleRuns().SetBackgroundColor(startInBytes, endInBytes, clr)
}

// UnsetBackgroundColorInRange removes the background color override in
// [startInBytes, endInBytes).
func (t *Text) UnsetBackgroundColorInRange(startInBytes, endInBytes int) {
	t.ensureStyleRuns().UnsetBackgroundColor(startInBytes, endInBytes)
}

// SetUnderlineInRange overrides whether an underline is drawn in
// [startInBytes, endInBytes). The override lasts until the value changes.
func (t *Text) SetUnderlineInRange(startInBytes, endInBytes int, underline bool) {
	t.ensureStyleRuns().SetUnderline(startInBytes, endInBytes, underline)
}

// UnsetUnderlineInRange removes the underline override in
// [startInBytes, endInBytes).
func (t *Text) UnsetUnderlineInRange(startInBytes, endInBytes int) {
	t.ensureStyleRuns().UnsetUnderline(startInBytes, endInBytes)
}

// SetStrikethroughInRange overrides whether a strikethrough is drawn in
// [startInBytes, endInBytes). The override lasts until the value changes.
func (t *Text) SetStrikethroughInRange(startInBytes, endInBytes int, strikethrough bool) {
	t.ensureStyleRuns().SetStrikethrough(startInBytes, endInBytes, strikethrough)
}

// UnsetStrikethroughInRange removes the strikethrough override in
// [startInBytes, endInBytes).
func (t *Text) UnsetStrikethroughInRange(startInBytes, endInBytes int) {
	t.ensureStyleRuns().UnsetStrikethrough(startInBytes, endInBytes)
}

// ResetStylesInRange removes all style overrides in
// [startInBytes, endInBytes).
func (t *Text) ResetStylesInRange(startInBytes, endInBytes int) {
	t.ensureStyleRuns().Reset(startInBytes, endInBytes)
}

// writeStyleRunsStateKey writes the ranged style overrides into the state
// key.
func (t *Text) writeStyleRunsStateKey(w *guigui.StateKeyWriter) {
	for run := range t.ensureStyleRuns().All() {
		w.WriteInt(run.Start)
		w.WriteInt(run.End)
		clr, ok := run.Style.Color()
		w.WriteBool(ok)
		if ok {
			writeColor(w, clr)
		}
		clr, ok = run.Style.BackgroundColor()
		w.WriteBool(ok)
		if ok {
			writeColor(w, clr)
		}
		underline, ok := run.Style.Underline()
		w.WriteBool(ok)
		w.WriteBool(underline)
		strikethrough, ok := run.Style.Strikethrough()
		w.WriteBool(ok)
		w.WriteBool(strikethrough)
	}
}
