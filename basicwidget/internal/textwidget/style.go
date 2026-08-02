// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
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
// changes; the ranged style overrides in [Text.ensureStyleRuns] apply on
// top, follow the text through edits and undo and redo, and are cleared when
// the value is replaced wholesale.
type textStyle struct {
	// hAlign is the horizontal alignment of the value.
	hAlign textutil.HorizontalAlign

	// vAlign is the vertical alignment of the value.
	vAlign textutil.VerticalAlign

	// scaleMinus1 is the base text scale minus 1, so the zero value means
	// scale 1.
	scaleMinus1 float64

	// style holds the base values of the properties that ranged style
	// overrides can override: the font family, italic face selection,
	// OpenType variations and features, language, and text color. The
	// effective style of a byte is style with its overrides merged on top.
	// The scale property stays unset here; ranged scale overrides multiply
	// the base font size instead of replacing a base value.
	style textstyle.Style

	// tabWidth is the tab width in pixels. A non-positive value selects the
	// default width.
	tabWidth float64

	// fontSize is the font size at scale 1; the rendered size is fontSize
	// multiplied by the widget scale.
	fontSize float64

	// lineHeight is the line height at scale 1; the rendered line height is
	// lineHeight multiplied by the widget scale.
	lineHeight float64

	// langString is the base language's string form, cached for
	// [Text.WriteStateKey].
	langString string

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

// fontFamilyID returns the base font family's ID, or 0 for a nil family.
func (s *textStyle) fontFamilyID() uint64 {
	family, _ := s.style.Family()
	if family == nil {
		return 0
	}
	return family.ID()
}

// faceAttributes returns the font attributes to shape text carrying style,
// the base style with any ranged overrides merged on top. liga sets whether
// ligatures are enabled.
func (s *textStyle) faceAttributes(style textstyle.Style, liga bool) font.Attributes {
	size := s.fontSize * s.scale()
	if scale, ok := style.Scale(); ok {
		size *= scale
	}
	a := font.Attributes{
		Size: size,
	}
	if lang, ok := style.Lang(); ok {
		a.Lang = lang
	}
	if italic, ok := style.Italic(); ok {
		a.Italic = italic
	}
	a = a.WithVariation(font.TagWght, float32(text.WeightMedium))
	for _, v := range style.Variations() {
		a = a.WithVariation(v.Tag, v.Value)
	}
	for _, f := range style.Features() {
		a = a.WithFeature(f.Tag, f.Value)
	}
	// A false liga disables ligatures so caret positions land on byte
	// boundaries; that constraint outranks a liga feature setting.
	if !liga {
		a = a.WithFeature(font.TagLiga, 0)
	} else if !slices.ContainsFunc(style.Features(), func(f font.Feature) bool { return f.Tag == font.TagLiga }) {
		a = a.WithFeature(font.TagLiga, 1)
	}
	return a
}

// ensureStyleRuns brings the ranged style overrides up to date with the
// store's content and returns the runs. Positional edits since the last call
// are replayed so the overrides keep covering the same text; mutations
// without a positional record (whole-value replacements) clear the
// overrides. Undo and redo reinstall the overrides from their history
// snapshots via [Text.restoreRangedState] instead.
func (t *Text) ensureStyleRuns() *textstyle.Runs {
	gen := t.store.Generation()
	if t.styleRunsValidGeneration == gen {
		return &t.styleRuns
	}
	defer func() {
		t.textEditsBuf = slices.Delete(t.textEditsBuf, 0, len(t.textEditsBuf))
	}()
	var covered bool
	t.textEditsBuf, covered = t.store.appendEditsSince(t.textEditsBuf, t.styleRunsValidGeneration)
	if covered {
		for _, e := range t.textEditsBuf {
			t.styleRuns.Replace(e.start, e.end, e.newLen)
		}
	} else {
		t.styleRuns.Clear()
	}
	t.styleRunsValidGeneration = gen
	return &t.styleRuns
}

// SetColorInRange overrides the text color in [startInBytes, endInBytes).
// The override follows the text through edits.
func (t *Text) SetColorInRange(startInBytes, endInBytes int, clr color.Color) {
	t.ensureStyleRuns().SetColor(startInBytes, endInBytes, clr)
}

// UnsetColorInRange removes the text color override in
// [startInBytes, endInBytes).
func (t *Text) UnsetColorInRange(startInBytes, endInBytes int) {
	t.ensureStyleRuns().UnsetColor(startInBytes, endInBytes)
}

// SetBackgroundColorInRange overrides the background color in
// [startInBytes, endInBytes). The override follows the text through edits.
func (t *Text) SetBackgroundColorInRange(startInBytes, endInBytes int, clr color.Color) {
	t.ensureStyleRuns().SetBackgroundColor(startInBytes, endInBytes, clr)
}

// UnsetBackgroundColorInRange removes the background color override in
// [startInBytes, endInBytes).
func (t *Text) UnsetBackgroundColorInRange(startInBytes, endInBytes int) {
	t.ensureStyleRuns().UnsetBackgroundColor(startInBytes, endInBytes)
}

// SetUnderlineInRange overrides whether an underline is drawn in
// [startInBytes, endInBytes). The override follows the text through edits.
func (t *Text) SetUnderlineInRange(startInBytes, endInBytes int, underline bool) {
	t.ensureStyleRuns().SetUnderline(startInBytes, endInBytes, underline)
}

// UnsetUnderlineInRange removes the underline override in
// [startInBytes, endInBytes).
func (t *Text) UnsetUnderlineInRange(startInBytes, endInBytes int) {
	t.ensureStyleRuns().UnsetUnderline(startInBytes, endInBytes)
}

// SetStrikethroughInRange overrides whether a strikethrough is drawn in
// [startInBytes, endInBytes). The override follows the text through edits.
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

// ReadStyleRuns replaces dst's runs with a copy of the ranged style
// overrides, reflecting the adjustments made for edits since the overrides
// were set.
func (t *Text) ReadStyleRuns(dst *textstyle.Runs) {
	dst.CopyFrom(t.ensureStyleRuns())
}

// CopyStyleRunsFrom replaces the ranged style overrides with a copy of
// runs.
func (t *Text) CopyStyleRunsFrom(runs *textstyle.Runs) {
	t.styleRuns.CopyFrom(runs)
	t.styleRunsValidGeneration = t.store.Generation()
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
// [startInBytes, endInBytes). The override follows the text through edits. A
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
// [startInBytes, endInBytes). The override follows the text through edits.
func (t *Text) SetItalicInRange(startInBytes, endInBytes int, italic bool) {
	t.ensureStyleRuns().SetItalic(startInBytes, endInBytes, italic)
}

// UnsetItalicInRange removes the italic face selection override in
// [startInBytes, endInBytes).
func (t *Text) UnsetItalicInRange(startInBytes, endInBytes int) {
	t.ensureStyleRuns().UnsetItalic(startInBytes, endInBytes)
}

// baseWeight returns the font weight of the base style: the wght variation
// axis value, or the default medium weight when unset.
func (t *Text) baseWeight() float32 {
	if v, ok := t.baseStyle.style.Variation(font.TagWght); ok {
		return v
	}
	return float32(text.WeightMedium)
}

// baseItalic returns the italic face selection of the base style.
func (t *Text) baseItalic() bool {
	italic, _ := t.baseStyle.style.Italic()
	return italic
}

// effectiveStyleAtCaret returns the effective style that text typed at
// textIndexInBytes adopts: the base style with the adopted ranged overrides
// merged on top.
func (t *Text) effectiveStyleAtCaret(textIndexInBytes int) textstyle.Style {
	return t.baseStyle.style.Merge(t.styleAtCaret(textIndexInBytes))
}

// styleAtCaret returns the ranged override style that text typed at
// textIndexInBytes adopts: the style overriding the byte right before the
// index, or a zero style at the start of the value.
func (t *Text) styleAtCaret(textIndexInBytes int) textstyle.Style {
	if textIndexInBytes <= 0 {
		return textstyle.Style{}
	}
	return t.ensureStyleRuns().StyleAt(textIndexInBytes - 1)
}

// IsBoldInRange reports whether the effective font weight, the ranged wght
// variation overrides applied over the base style, is the bold weight over
// every byte of [startInBytes, endInBytes). For an empty range, it reports
// the state that text typed at startInBytes would adopt.
func (t *Text) IsBoldInRange(startInBytes, endInBytes int) bool {
	if startInBytes >= endInBytes {
		v, ok := t.effectiveStyleAtCaret(startInBytes).Variation(font.TagWght)
		if !ok {
			v = float32(text.WeightMedium)
		}
		return v == float32(text.WeightBold)
	}
	v, uniform := t.ensureStyleRuns().UniformVariation(startInBytes, endInBytes, font.TagWght, t.baseWeight())
	return uniform && v == float32(text.WeightBold)
}

// IsItalicInRange reports whether the effective italic state, the ranged
// italic overrides applied over the base style, selects an italic face over
// every byte of [startInBytes, endInBytes). For an empty range, it reports
// the state that text typed at startInBytes would adopt.
func (t *Text) IsItalicInRange(startInBytes, endInBytes int) bool {
	if startInBytes >= endInBytes {
		italic, _ := t.effectiveStyleAtCaret(startInBytes).Italic()
		return italic
	}
	italic, uniform := t.ensureStyleRuns().UniformItalic(startInBytes, endInBytes, t.baseItalic())
	return uniform && italic
}

// IsUnderlineInRange reports whether an underline is drawn over every byte
// of [startInBytes, endInBytes). For an empty range, it reports the state
// that text typed at startInBytes would adopt.
func (t *Text) IsUnderlineInRange(startInBytes, endInBytes int) bool {
	if startInBytes >= endInBytes {
		underline, _ := t.effectiveStyleAtCaret(startInBytes).Underline()
		return underline
	}
	underline, uniform := t.ensureStyleRuns().UniformUnderline(startInBytes, endInBytes, false)
	return uniform && underline
}

// IsStrikethroughInRange reports whether a strikethrough is drawn over
// every byte of [startInBytes, endInBytes). For an empty range, it reports
// the state that text typed at startInBytes would adopt.
func (t *Text) IsStrikethroughInRange(startInBytes, endInBytes int) bool {
	if startInBytes >= endInBytes {
		strikethrough, _ := t.effectiveStyleAtCaret(startInBytes).Strikethrough()
		return strikethrough
	}
	strikethrough, uniform := t.ensureStyleRuns().UniformStrikethrough(startInBytes, endInBytes, false)
	return uniform && strikethrough
}

// ColorInRange returns the text color override shared by every byte of
// [startInBytes, endInBytes) and whether the range is uniform. A nil color
// with uniform true means no byte has a color override, so the range
// renders in the base text color. For an empty range, it returns the
// override that text typed at startInBytes would adopt.
func (t *Text) ColorInRange(startInBytes, endInBytes int) (clr color.Color, uniform bool) {
	if startInBytes >= endInBytes {
		c, _ := t.styleAtCaret(startInBytes).Color()
		return c, true
	}
	return t.ensureStyleRuns().UniformColor(startInBytes, endInBytes, nil)
}

// ScaleInRange returns the font size multiplier shared by every byte of
// [startInBytes, endInBytes) and whether the range is uniform. A byte
// without a scale override takes the multiplier 1. For an empty range, it
// returns the multiplier that text typed at startInBytes would adopt.
func (t *Text) ScaleInRange(startInBytes, endInBytes int) (scale float64, uniform bool) {
	if startInBytes >= endInBytes {
		if s, ok := t.styleAtCaret(startInBytes).Scale(); ok {
			return s, true
		}
		return 1, true
	}
	return t.ensureStyleRuns().UniformScale(startInBytes, endInBytes, 1)
}

// ApplyBoldInRange makes every byte of [startInBytes, endInBytes) bold or
// not bold by adjusting the ranged wght variation overrides over the base
// style. Overrides that would restate the base style are removed instead
// of set; making a range of a bold base style not bold overrides it with
// the default weight. An empty range is a no-op.
func (t *Text) ApplyBoldInRange(startInBytes, endInBytes int, bold bool) {
	if startInBytes >= endInBytes {
		return
	}
	if bold == (t.baseWeight() == float32(text.WeightBold)) {
		t.UnsetVariationInRange(startInBytes, endInBytes, font.TagWght)
		return
	}
	if bold {
		t.SetVariationInRange(startInBytes, endInBytes, font.TagWght, float32(text.WeightBold))
		return
	}
	t.SetVariationInRange(startInBytes, endInBytes, font.TagWght, float32(text.WeightMedium))
}

// ApplyItalicInRange makes every byte of [startInBytes, endInBytes) render
// with an italic face or a regular face by adjusting the ranged italic
// overrides over the base style. Overrides that would restate the base
// style are removed instead of set. An empty range is a no-op.
func (t *Text) ApplyItalicInRange(startInBytes, endInBytes int, italic bool) {
	if startInBytes >= endInBytes {
		return
	}
	if italic == t.baseItalic() {
		t.UnsetItalicInRange(startInBytes, endInBytes)
		return
	}
	t.SetItalicInRange(startInBytes, endInBytes, italic)
}

// ApplyUnderlineInRange sets whether an underline is drawn over every byte
// of [startInBytes, endInBytes). Underline false removes the underline
// overrides in the range. An empty range is a no-op.
func (t *Text) ApplyUnderlineInRange(startInBytes, endInBytes int, underline bool) {
	if startInBytes >= endInBytes {
		return
	}
	if !underline {
		t.UnsetUnderlineInRange(startInBytes, endInBytes)
		return
	}
	t.SetUnderlineInRange(startInBytes, endInBytes, true)
}

// ApplyStrikethroughInRange sets whether a strikethrough is drawn over
// every byte of [startInBytes, endInBytes). Strikethrough false removes
// the strikethrough overrides in the range. An empty range is a no-op.
func (t *Text) ApplyStrikethroughInRange(startInBytes, endInBytes int, strikethrough bool) {
	if startInBytes >= endInBytes {
		return
	}
	if !strikethrough {
		t.UnsetStrikethroughInRange(startInBytes, endInBytes)
		return
	}
	t.SetStrikethroughInRange(startInBytes, endInBytes, true)
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
// byte offsets into the committed text. Masked values append no face runs.
func (t *Text) appendFaceRunsForStyle(runs []textutil.FaceRun, context *guigui.Context, forceBold bool) []textutil.FaceRun {
	if t.masking() {
		return runs
	}
	base := t.forceBoldedBaseStyle(forceBold)
	liga := t.ligaturesEnabled()
	for run := range t.ensureStyleRuns().All() {
		if !run.Style.AffectsFaceSelection() {
			continue
		}
		style := base.Merge(run.Style)
		family, _ := style.Family()
		runs = append(runs, textutil.FaceRun{
			Start: run.Start,
			End:   run.End,
			Face:  font.NewFace(context, family, t.baseStyle.faceAttributes(style, liga)),
		})
	}
	return runs
}

// faceRunsMark records the face-run buffer lengths at an
// [Text.acquireFaceRuns] call, for [Text.releaseFaceRuns].
type faceRunsMark struct {
	committed int
	rendering int
}

// acquireFaceRuns appends the ranged style overrides' face runs to the
// reusable buffers and returns them as committed (committed-text byte
// offsets) and rendering (rendering-text byte offsets) slices.
// showComposition reports whether the caller lays out the rendering text
// with the active IME composition spliced in; without an applicable
// transform both returns are the same slice. The slices are views into
// buffers owned by t: pass mark to [Text.releaseFaceRuns] when done,
// typically deferred, and do not retain the slices past that call.
func (t *Text) acquireFaceRuns(context *guigui.Context, forceBold, showComposition bool) (committed, rendering []textutil.FaceRun, mark faceRunsMark) {
	mark = faceRunsMark{
		committed: len(t.faceRunsBuf),
		rendering: len(t.renderingFaceRunsBuf),
	}
	t.faceRunsBuf = t.appendFaceRunsForStyle(t.faceRunsBuf, context, forceBold)
	committed = t.faceRunsBuf[mark.committed:]
	rendering = committed
	if showComposition && len(committed) > 0 {
		if compLen := t.store.UncommittedTextLengthInBytes(); compLen > 0 {
			selStart, selEnd := t.store.Selection()
			t.renderingFaceRunsBuf = appendFaceRunsThroughComposition(t.renderingFaceRunsBuf, committed, selStart, selEnd, compLen)
			rendering = t.renderingFaceRunsBuf[mark.rendering:]
		}
	}
	return committed, rendering, mark
}

// releaseFaceRuns truncates the face-run buffers back to their lengths at
// the matching [Text.acquireFaceRuns] call.
func (t *Text) releaseFaceRuns(mark faceRunsMark) {
	t.faceRunsBuf = slices.Delete(t.faceRunsBuf, mark.committed, len(t.faceRunsBuf))
	t.renderingFaceRunsBuf = slices.Delete(t.renderingFaceRunsBuf, mark.rendering, len(t.renderingFaceRunsBuf))
}

// appendFaceRunsThroughComposition appends src's committed-text face runs to
// dst with their offsets transformed to the rendering text, whose composition
// splice replaces the committed byte range [selStart, selEnd) with compLen
// bytes, with the movement rules of [replaceTextRanges]. A run whose text the
// splice fully replaces is dropped.
func appendFaceRunsThroughComposition(dst, src []textutil.FaceRun, selStart, selEnd, compLen int) []textutil.FaceRun {
	if selStart > selEnd {
		selStart, selEnd = selEnd, selStart
	}
	for _, run := range src {
		r, ok := replaceTextRange(TextRange{StartInBytes: run.Start, EndInBytes: run.End}, selStart, selEnd, compLen)
		if !ok {
			continue
		}
		run.Start = r.StartInBytes
		run.End = r.EndInBytes
		dst = append(dst, run)
	}
	return dst
}
