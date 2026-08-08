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
// changes; the ranged style overrides in [Text.ensureOverrideStyleRuns]
// apply on top, follow the text through edits and undo and redo, and are
// cleared when the value is replaced wholesale.
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

// ensureOverrideStyleRuns brings the ranged style overrides up to date
// with the store's content and returns the runs. Positional edits since the
// last call are replayed so the overrides keep covering the same text;
// mutations without a positional record (whole-value replacements) clear
// the overrides. Undo and redo reinstall the overrides from their history
// snapshots via [Text.restoreRangedState] instead.
func (t *Text) ensureOverrideStyleRuns() *textstyle.Runs {
	gen := t.store.Generation()
	if t.overrideStyleRunsValidGeneration == gen {
		return &t.overrideStyleRuns
	}
	defer func() {
		t.textEditsBuf = slices.Delete(t.textEditsBuf, 0, len(t.textEditsBuf))
	}()
	var covered bool
	t.textEditsBuf, covered = t.store.appendEditsSince(t.textEditsBuf, t.overrideStyleRunsValidGeneration)
	if covered {
		for _, e := range t.textEditsBuf {
			t.overrideStyleRuns.Replace(e.start, e.end, e.newLen)
		}
	} else {
		t.overrideStyleRuns.Clear()
	}
	t.overrideStyleRunsValidGeneration = gen
	return &t.overrideStyleRuns
}

// ReadOverrideStyleRuns replaces dst's runs with a copy of the ranged style
// overrides, reflecting the adjustments made for edits since the overrides
// were set.
func (t *Text) ReadOverrideStyleRuns(dst *textstyle.Runs) {
	dst.CopyFrom(t.ensureOverrideStyleRuns())
}

// CopyOverrideStyleRunsFrom replaces the ranged style overrides with a
// copy of runs. record sets whether the replacement is recorded in the undo
// history; a recorded replacement leaving the overrides unchanged records
// nothing.
func (t *Text) CopyOverrideStyleRunsFrom(runs *textstyle.Runs, record bool) {
	if record {
		if !runs.Equal(t.ensureOverrideStyleRuns()) {
			t.ensureStoreCallbacks()
			t.store.recordRangedStateChange(0, t.store.TextLengthInBytes())
		}
	}
	t.overrideStyleRuns.CopyFrom(runs)
	t.overrideStyleRunsValidGeneration = t.store.Generation()
}

// ReadBaseStyle writes the base style's overridable properties to dst.
func (t *Text) ReadBaseStyle(dst *textstyle.Style) {
	*dst = t.baseStyle.style
}

// ReadOverrideStyleRunsInRange replaces dst's runs with a copy of the
// ranged style overrides in [startInBytes, endInBytes), rebased so that
// startInBytes maps to 0.
func (t *Text) ReadOverrideStyleRunsInRange(dst *textstyle.Runs, startInBytes, endInBytes int) {
	dst.CopyRangeFrom(t.ensureOverrideStyleRuns(), startInBytes, endInBytes)
}

// ReplaceOverrideStyleRunsInRange replaces the ranged style overrides in
// [startInBytes, endInBytes) with runs' overrides in
// [0, endInBytes-startInBytes), shifted so that 0 maps to startInBytes.
// record sets whether the replacement is recorded in the undo history; a
// recorded replacement leaving the overrides unchanged records nothing.
func (t *Text) ReplaceOverrideStyleRunsInRange(runs *textstyle.Runs, startInBytes, endInBytes int, record bool) {
	current := t.ensureOverrideStyleRuns()
	if record {
		defer func() {
			t.newOverrideStyleRunsBuf.Clear()
			t.oldOverrideStyleRunsBuf.Clear()
		}()
		t.newOverrideStyleRunsBuf.CopyRangeFrom(runs, 0, endInBytes-startInBytes)
		t.oldOverrideStyleRunsBuf.CopyRangeFrom(current, startInBytes, endInBytes)
		if t.newOverrideStyleRunsBuf.Equal(&t.oldOverrideStyleRunsBuf) {
			return
		}
		t.ensureStoreCallbacks()
		t.store.recordRangedStateChange(startInBytes, endInBytes)
	}
	current.ReplaceRange(runs, startInBytes, endInBytes)
}

// SetInsertionStyle replaces the insertion style with style. Its set
// properties are applied as ranged style overrides over the next text
// inserted at the caret, on top of the adopted overrides, and the style is
// then reset. The widget also resets it without applying on other
// interactions, such as a selection change, a deletion, or an undo; neither
// setting nor resetting is recorded in the undo history.
func (t *Text) SetInsertionStyle(style textstyle.Style) {
	t.insertionStyle = style
}

// OnInsertionStyleReset sets an event handler invoked when the widget resets
// the insertion style: after applying it to inserted text, or when
// discarding it without applying, such as on a selection change. The handler
// is not invoked for [Text.SetInsertionStyle].
func (t *Text) OnInsertionStyleReset(f func(context *guigui.Context)) {
	guigui.SetEventHandler(t, textEventInsertionStyleReset, f)
}

// materializeInsertionStyle applies the insertion style as ranged overrides over
// the inserted byte span [startInBytes, startInBytes+newLenInBytes) and
// resets it. A mutation that inserts nothing resets the insertion style
// without applying it.
func (t *Text) materializeInsertionStyle(startInBytes, newLenInBytes int) {
	if t.insertionStyle.IsZero() {
		return
	}
	if newLenInBytes > 0 {
		t.ensureOverrideStyleRuns().ApplyStyle(startInBytes, startInBytes+newLenInBytes, t.insertionStyle)
	}
	t.resetInsertionStyle()
}

// resetInsertionStyle clears the insertion style and dispatches the reset
// event. Clearing an already zero insertion style dispatches nothing.
func (t *Text) resetInsertionStyle() {
	if t.insertionStyle.IsZero() {
		return
	}
	t.insertionStyle = textstyle.Style{}
	guigui.DispatchEvent(t, textEventInsertionStyleReset)
}

// styleDefaults holds the rendering defaults that unset style properties
// resolve to.
var styleDefaults = textstyle.Style{}.
	WithFamily(nil).
	WithItalic(false).
	WithScale(1).
	WithColor(nil).
	WithBackgroundColor(nil).
	WithUnderline(false).
	WithStrikethrough(false).
	WithLang(language.Tag{}).
	WithVariation(font.TagWght, float32(text.WeightMedium))

// resolvedBaseStyle returns the base style with the rendering defaults
// applied to its unset properties, so every overridable property holds a
// concrete value.
func (t *Text) resolvedBaseStyle() textstyle.Style {
	return styleDefaults.Merge(t.baseStyle.style)
}

// ReadEffectiveStyleRuns replaces dst's runs with the effective styles of
// the whole value: the resolved base style with the ranged overrides merged
// on top.
func (t *Text) ReadEffectiveStyleRuns(dst *textstyle.Runs) {
	t.ReadEffectiveStyleRunsInRange(dst, 0, t.store.TextLengthInBytes())
}

// ReadEffectiveStyleRunsInRange replaces dst's runs with the effective
// styles of [startInBytes, endInBytes): the resolved base style with the
// ranged overrides merged on top, rebased so that startInBytes maps to 0.
func (t *Text) ReadEffectiveStyleRunsInRange(dst *textstyle.Runs, startInBytes, endInBytes int) {
	dst.CopyMergedFrom(t.ensureOverrideStyleRuns(), t.resolvedBaseStyle(), startInBytes, endInBytes)
}

// EffectiveStyleAt returns the effective style that text typed at
// textIndexInBytes adopts: the resolved base style with the overrides
// adopted from the byte right before the index merged on top.
func (t *Text) EffectiveStyleAt(textIndexInBytes int) textstyle.Style {
	return t.resolvedBaseStyle().Merge(t.styleAtCaret(textIndexInBytes))
}

// styleAtCaret returns the ranged override style that text typed at
// textIndexInBytes adopts: the style overriding the byte right before the
// index, or a zero style at the start of the value.
func (t *Text) styleAtCaret(textIndexInBytes int) textstyle.Style {
	if textIndexInBytes <= 0 {
		return textstyle.Style{}
	}
	return t.ensureOverrideStyleRuns().StyleAt(textIndexInBytes - 1)
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

// metricOverrideStyleRunsFingerprint fingerprints the metric properties of
// the ranged style overrides.
func (t *Text) metricOverrideStyleRunsFingerprint() uint64 {
	h := fnv.New64a()
	w := metricHashWriter{h: h}
	t.ensureOverrideStyleRuns().WriteMetricStateKey(&w)
	return h.Sum64()
}

// invalidateSizeCacheForMetricOverrideStyleRuns resets the cached text
// sizes when the metric style overrides have changed since the last
// measurement.
func (t *Text) invalidateSizeCacheForMetricOverrideStyleRuns() {
	if fp := t.metricOverrideStyleRunsFingerprint(); fp != t.lastMetricOverrideStyleRunsFingerprint {
		t.lastMetricOverrideStyleRunsFingerprint = fp
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
	for run := range t.ensureOverrideStyleRuns().All() {
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
