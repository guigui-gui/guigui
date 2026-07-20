// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"cmp"
	"encoding/binary"
	"hash"
	"hash/fnv"
	"image/color"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/font"
	"github.com/guigui-gui/guigui/basicwidget/internal/textstyle"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

// textStyle holds a [Text]'s base render-style configuration: alignment,
// scale, font selection, and the concrete colors set by a wrapping widget.
// The base style applies to the whole value and persists across value
// changes; the ranged style overrides in [Text.ensureStyleRuns] apply on top
// and are cleared when the value changes.
type textStyle struct {
	// hAlign is the horizontal alignment of the value.
	hAlign textutil.HorizontalAlign

	// vAlign is the vertical alignment of the value.
	vAlign textutil.VerticalAlign

	// scaleMinus1 is the base text scale minus 1, so the zero value means
	// scale 1.
	scaleMinus1 float64

	// variations and features are the base OpenType settings, sorted by tag
	// with at most one entry per tag. Ranged style overrides apply on top.
	variations []font.Variation
	features   []font.Feature

	// italic selects an italic face from the font family.
	italic bool

	// tabWidth is the tab width in pixels. A non-positive value selects the
	// default width.
	tabWidth float64

	// fontFamily is the resolved font family used to render the value, or nil
	// to render with the registered face source stack alone.
	fontFamily *font.Family

	// fontSize is the font size at scale 1; the rendered size is fontSize
	// multiplied by the widget scale.
	fontSize float64

	// lineHeight is the line height at scale 1; the rendered line height is
	// lineHeight multiplied by the widget scale.
	lineHeight float64

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

// scaledLineHeight returns the line height in pixels, with the scale
// applied.
func (s *textStyle) scaledLineHeight() float64 {
	return s.lineHeight * s.scale()
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
	a := font.Attributes{
		Size:   s.fontSize * s.scale(),
		Lang:   s.lang,
		Italic: s.italic,
	}
	a = a.WithVariation(font.TagWght, float32(text.WeightMedium))
	for _, v := range s.variations {
		a = a.WithVariation(v.Tag, v.Value)
	}
	if forceBold {
		a = a.WithVariation(font.TagWght, float32(text.WeightBold))
	}
	for _, f := range s.features {
		a = a.WithFeature(f.Tag, f.Value)
	}
	// A false liga disables ligatures so caret positions land on byte
	// boundaries; that constraint outranks a base liga feature setting.
	if !liga {
		a = a.WithFeature(font.TagLiga, 0)
	} else if !slices.ContainsFunc(s.features, func(f font.Feature) bool { return f.Tag == font.TagLiga }) {
		a = a.WithFeature(font.TagLiga, 1)
	}
	return a
}

// setTagged sets entry in the tag-sorted settings, keeping at most one entry
// per tag.
func setTagged[T any](settings []T, entry T, tagOf func(T) text.Tag) []T {
	i, ok := slices.BinarySearchFunc(settings, tagOf(entry), func(s T, tag text.Tag) int {
		return cmp.Compare(tagOf(s), tag)
	})
	if ok {
		settings[i] = entry
		return settings
	}
	return slices.Insert(settings, i, entry)
}

// removeTagged removes tag's entry from the tag-sorted settings.
func removeTagged[T any](settings []T, tag text.Tag, tagOf func(T) text.Tag) []T {
	i, ok := slices.BinarySearchFunc(settings, tag, func(s T, tag text.Tag) int {
		return cmp.Compare(tagOf(s), tag)
	})
	if !ok {
		return settings
	}
	return slices.Delete(settings, i, i+1)
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

// SetFontFamilyInRange overrides the font family in
// [startInBytes, endInBytes). The override lasts until the value changes. A
// nil family resolves faces with the registered face source stack alone.
func (t *Text) SetFontFamilyInRange(startInBytes, endInBytes int, family *font.Family) {
	t.ensureStyleRuns().SetFamily(startInBytes, endInBytes, family)
}

// UnsetFontFamilyInRange removes the font family override in
// [startInBytes, endInBytes).
func (t *Text) UnsetFontFamilyInRange(startInBytes, endInBytes int) {
	t.ensureStyleRuns().UnsetFamily(startInBytes, endInBytes)
}

// SetScaleInRange overrides the font size in [startInBytes, endInBytes) as a
// multiplier applied to the base font size. The override lasts until the
// value changes. The line height is unaffected; the range renders on the
// line's baseline.
func (t *Text) SetScaleInRange(startInBytes, endInBytes int, scale float64) {
	t.ensureStyleRuns().SetScale(startInBytes, endInBytes, scale)
}

// UnsetScaleInRange removes the font size override in
// [startInBytes, endInBytes).
func (t *Text) UnsetScaleInRange(startInBytes, endInBytes int) {
	t.ensureStyleRuns().UnsetScale(startInBytes, endInBytes)
}

// SetLangInRange overrides the language used to select the face and its
// features when shaping [startInBytes, endInBytes). The override lasts until
// the value changes.
func (t *Text) SetLangInRange(startInBytes, endInBytes int, lang language.Tag) {
	t.ensureStyleRuns().SetLang(startInBytes, endInBytes, lang)
}

// UnsetLangInRange removes the language override in
// [startInBytes, endInBytes).
func (t *Text) UnsetLangInRange(startInBytes, endInBytes int) {
	t.ensureStyleRuns().UnsetLang(startInBytes, endInBytes)
}

// SetItalicInRange overrides the italic face selection in
// [startInBytes, endInBytes). The override lasts until the value changes.
func (t *Text) SetItalicInRange(startInBytes, endInBytes int, italic bool) {
	t.ensureStyleRuns().SetItalic(startInBytes, endInBytes, italic)
}

// UnsetItalicInRange removes the italic face selection override in
// [startInBytes, endInBytes).
func (t *Text) UnsetItalicInRange(startInBytes, endInBytes int) {
	t.ensureStyleRuns().UnsetItalic(startInBytes, endInBytes)
}

// hasMetricStyleRuns reports whether any ranged style override affects glyph
// metrics (currently the variation axes, feature settings, and face-selection
// overrides).
func (t *Text) hasMetricStyleRuns() bool {
	return t.ensureStyleRuns().HasMetricProperties()
}

// metricHashWriter adapts an FNV-1a hash to [textstyle.Writer] for
// fingerprinting the metric style properties.
type metricHashWriter struct {
	h      hash.Hash64
	buf    [8]byte
	strbuf []byte
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

func (w *metricHashWriter) WriteUint64(v uint64) {
	w.writeUint64(v)
}

func (w *metricHashWriter) WriteFloat64(v float64) {
	w.writeUint64(math.Float64bits(v))
}

func (w *metricHashWriter) WriteString(v string) {
	w.WriteInt(len(v))
	w.strbuf = append(w.strbuf[:0], v...)
	_, _ = w.h.Write(w.strbuf)
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
		if !run.Style.HasMetricProperties() {
			continue
		}
		attrs := t.faceAttributes(forceBold)
		if italic, ok := run.Style.Italic(); ok {
			attrs.Italic = italic
		}
		if scale, ok := run.Style.Scale(); ok {
			attrs.Size *= scale
		}
		if lang, ok := run.Style.Lang(); ok {
			attrs.Lang = lang
		}
		for _, v := range run.Style.Variations() {
			attrs = attrs.WithVariation(v.Tag, v.Value)
		}
		for _, f := range run.Style.Features() {
			attrs = attrs.WithFeature(f.Tag, f.Value)
		}
		family := t.baseStyle.fontFamily
		if f, ok := run.Style.Family(); ok {
			family = f
		}
		runs = append(runs, textutil.FaceRun{
			Start: run.Start,
			End:   run.End,
			Face:  font.NewFace(context, family, attrs),
		})
	}
	return runs
}
