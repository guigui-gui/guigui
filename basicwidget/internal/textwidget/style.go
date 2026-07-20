// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"encoding/binary"
	"hash"
	"hash/fnv"
	"image/color"
	"math"

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
	weight := text.WeightMedium
	if s.bold || forceBold {
		weight = text.WeightBold
	}
	a := font.Attributes{
		Size: s.baseFontSize * s.scale(),
		Lang: s.lang,
	}
	a = a.WithVariation(font.TagWght, float32(weight))
	a = a.WithFeature(font.TagLiga, boolToFeatureValue(liga))
	a = a.WithFeature(font.TagTnum, boolToFeatureValue(s.tabular))
	return a
}

func boolToFeatureValue(b bool) uint32 {
	if b {
		return 1
	}
	return 0
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

// SetVariationInRange overrides the OpenType variation axis tag in
// [startInBytes, endInBytes) with value. The override lasts until the value
// changes.
func (t *Text) SetVariationInRange(startInBytes, endInBytes int, tag text.Tag, value float32) {
	t.ensureStyleRuns().SetVariation(startInBytes, endInBytes, tag, value)
}

// UnsetVariationInRange removes the override of the OpenType variation axis
// tag in [startInBytes, endInBytes).
func (t *Text) UnsetVariationInRange(startInBytes, endInBytes int, tag text.Tag) {
	t.ensureStyleRuns().UnsetVariation(startInBytes, endInBytes, tag)
}

// SetFeatureInRange overrides the OpenType feature tag in
// [startInBytes, endInBytes) with value. The override lasts until the value
// changes.
func (t *Text) SetFeatureInRange(startInBytes, endInBytes int, tag text.Tag, value uint32) {
	t.ensureStyleRuns().SetFeature(startInBytes, endInBytes, tag, value)
}

// UnsetFeatureInRange removes the override of the OpenType feature tag in
// [startInBytes, endInBytes).
func (t *Text) UnsetFeatureInRange(startInBytes, endInBytes int, tag text.Tag) {
	t.ensureStyleRuns().UnsetFeature(startInBytes, endInBytes, tag)
}

// hasMetricStyleRuns reports whether any ranged style override affects glyph
// metrics (currently the variation axes and feature settings).
func (t *Text) hasMetricStyleRuns() bool {
	return t.ensureStyleRuns().HasMetricProperties()
}

// metricHashWriter adapts an FNV-1a hash to [textstyle.Writer] for
// fingerprinting the metric style properties.
type metricHashWriter struct {
	h   hash.Hash64
	buf [8]byte
}

func (w *metricHashWriter) writeUint64(v uint64) {
	binary.LittleEndian.PutUint64(w.buf[:], v)
	_, _ = w.h.Write(w.buf[:])
}

func (w *metricHashWriter) WriteBool(v bool) {
	if v {
		w.writeUint64(1)
	} else {
		w.writeUint64(0)
	}
}

func (w *metricHashWriter) WriteInt(v int) {
	w.writeUint64(uint64(v))
}

func (w *metricHashWriter) WriteUint16(v uint16) {
	w.writeUint64(uint64(v))
}

func (w *metricHashWriter) WriteUint32(v uint32) {
	w.writeUint64(uint64(v))
}

func (w *metricHashWriter) WriteFloat64(v float64) {
	w.writeUint64(math.Float64bits(v))
}

// metricStyleRunsFingerprint fingerprints the metric properties of the
// ranged style overrides.
func (t *Text) metricStyleRunsFingerprint() uint64 {
	h := fnv.New64a()
	w := metricHashWriter{h: h}
	t.ensureStyleRuns().WriteMetricStateKey(&w)
	return h.Sum64()
}

// invalidateSizeCacheForMetricStyleRuns resets the cached text sizes when
// the metric style overrides have changed since the last measurement.
func (t *Text) invalidateSizeCacheForMetricStyleRuns() {
	if fp := t.metricStyleRunsFingerprint(); fp != t.lastMetricStyleRunsFingerprint {
		t.lastMetricStyleRunsFingerprint = fp
		t.resetCachedTextSize()
	}
}

// appendFaceRunsForStyle appends the face runs derived from the ranged style
// overrides' metric properties to runs and returns the extended slice, with
// byte offsets into the committed text. Masked values and values with an
// active IME composition append no face runs.
func (t *Text) appendFaceRunsForStyle(runs []textutil.FaceRun, context *guigui.Context, forceBold bool) []textutil.FaceRun {
	if t.masking() {
		return runs
	}
	if t.store.UncommittedTextLengthInBytes() > 0 {
		return runs
	}
	for run := range t.ensureStyleRuns().All() {
		variations := run.Style.Variations()
		features := run.Style.Features()
		if len(variations) == 0 && len(features) == 0 {
			continue
		}
		attrs := t.faceAttributes(forceBold)
		for _, v := range variations {
			attrs = attrs.WithVariation(v.Tag, v.Value)
		}
		for _, f := range features {
			attrs = attrs.WithFeature(f.Tag, f.Value)
		}
		runs = append(runs, textutil.FaceRun{
			Start: run.Start,
			End:   run.End,
			Face:  font.NewFace(context, t.style.fontFamily, attrs),
		})
	}
	return runs
}
