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

	// lineHeightMode selects how lineHeight responds to the font sizes on a
	// visual line.
	lineHeightMode textutil.LineHeightMode

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

// SetBaseStyle replaces the base style's overridable properties with style,
// except the font family, the text color and the language, which
// [Text.SetFontFamily], [Text.SetTextColor] and [Text.SetLang] keep owning.
func (t *Text) SetBaseStyle(style textstyle.Style) {
	style = style.WithoutFamily().WithoutColor().WithoutLang()
	if family, ok := t.baseStyle.style.Family(); ok {
		style = style.WithFamily(family)
	}
	if clr, ok := t.baseStyle.style.Color(); ok {
		style = style.WithColor(clr)
	}
	if lang, ok := t.baseStyle.style.Lang(); ok {
		style = style.WithLang(lang)
	}
	t.baseStyle.style = style
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

// adoptStylesForInsertedText applies the ranged style overrides insertedText,
// which a mutation put in place of [startInBytes, endInBytes), carries: the
// style its position gives it in place of the one it took from the byte
// before it, the style of the empty line a break opens, and the insertion
// style over it.
func (t *Text) adoptStylesForInsertedText(startInBytes, endInBytes int, insertedText string) {
	if startInBytes == endInBytes && len(insertedText) > 0 {
		switch {
		case t.isLogicalLineHead(startInBytes):
			t.adoptStyleAtLineHeadIfNeeded(t.ensureOverrideStyleRuns(), startInBytes, len(insertedText), t.store.TextLengthInBytes())
		case isLineBreak(insertedText):
			t.adoptStyleForNewEmptyLineIfNeeded(startInBytes, len(insertedText))
		}
	}
	t.materializeInsertionStyle(startInBytes, len(insertedText))
}

// adoptStyleForNewEmptyLineIfNeeded gives the break ending the empty line
// that the line break inserted at [startInBytes, startInBytes+lenInBytes)
// opens the style of the character before it, replacing whatever style that
// break had, so the empty line carries what the line it was split from shows.
// A break inserted with text after it on the same line, which opens no empty
// line, styles nothing. startInBytes must not be at a logical line head.
func (t *Text) adoptStyleForNewEmptyLineIfNeeded(startInBytes, lenInBytes int) {
	insertedEnd := startInBytes + lenInBytes
	// The caret sat at the line's end, leaving an empty line after the
	// inserted break, only when the line's own break follows it directly. A
	// line ending the value has no break to carry the empty line's style.
	breakStart, breakEnd, ok := t.trailingLineBreakRange(insertedEnd)
	if !ok || breakStart != insertedEnd {
		return
	}
	runs := t.ensureOverrideStyleRuns()
	// The inserted break already carries the style of the character before
	// it, as any inserted byte does, so only the break after it is left.
	style := runs.StyleAt(startInBytes - 1)
	runs.Reset(breakStart, breakEnd)
	if style.IsZero() {
		return
	}
	runs.ApplyStyle(breakStart, breakEnd, style)
}

// adoptStyleAtLineHeadIfNeeded replaces the overrides in runs that the text
// inserted at the head of a logical line took from the byte before it with
// those of the byte after it. newLenInBytes is the length of the inserted
// text, in the offsets of the textLenInBytes-byte text runs indexes. An
// insertion with no byte after it keeps the overrides it took.
func (t *Text) adoptStyleAtLineHeadIfNeeded(runs *textstyle.Runs, startInBytes, newLenInBytes, textLenInBytes int) {
	if newLenInBytes <= 0 {
		return
	}
	insertedEnd := startInBytes + newLenInBytes
	if insertedEnd >= textLenInBytes {
		return
	}
	if !t.isLogicalLineHead(startInBytes) {
		return
	}
	style := runs.StyleAt(insertedEnd)
	runs.Reset(startInBytes, insertedEnd)
	if !style.IsZero() {
		runs.ApplyStyle(startInBytes, insertedEnd, style)
	}
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
// adopted from the neighboring byte merged on top.
func (t *Text) EffectiveStyleAt(textIndexInBytes int) textstyle.Style {
	return t.resolvedBaseStyle().Merge(t.styleAtCaret(textIndexInBytes))
}

// styleAtCaret returns the ranged override style that text typed at
// textIndexInBytes adopts: the style overriding the byte at the index at a
// logical line's head, otherwise the one overriding the byte right before
// it. A head with no byte at the index falls back to the byte before, and
// the start of an empty value has neither.
func (t *Text) styleAtCaret(textIndexInBytes int) textstyle.Style {
	if textIndexInBytes < t.store.TextLengthInBytes() && t.isLogicalLineHead(textIndexInBytes) {
		return t.ensureOverrideStyleRuns().StyleAt(textIndexInBytes)
	}
	if textIndexInBytes <= 0 {
		return textstyle.Style{}
	}
	return t.ensureOverrideStyleRuns().StyleAt(textIndexInBytes - 1)
}

// metricHashWriter adapts an FNV-1a 128-bit hash to [textstyle.Writer] for
// fingerprinting the metric style properties.
type metricHashWriter struct {
	h      hash.Hash
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

// metricStyleFingerprint fingerprints the metric properties of the ranged
// style overrides and of the insertion point, which take part in the measured
// size.
func (t *Text) metricStyleFingerprint() [16]byte {
	h := fnv.New128a()
	w := metricHashWriter{h: h}
	t.ensureOverrideStyleRuns().WriteMetricStateKey(&w)
	// The caret's position only takes part in the size where the text typed
	// there would carry a style, so a caret move within unstyled text leaves
	// the fingerprint alone.
	start, _ := t.store.Selection()
	if adopted := t.styleAtCaret(start); !adopted.IsZero() || !t.insertionStyle.IsZero() {
		w.WriteInt(start)
		adopted.WriteMetricStateKey(&w)
		t.insertionStyle.WriteMetricStateKey(&w)
	}
	var fp [16]byte
	h.Sum(fp[:0])
	return fp
}

// invalidateSizeCacheForMetricStyles resets the cached text sizes when the
// metric styles have changed since the last measurement.
func (t *Text) invalidateSizeCacheForMetricStyles() {
	if fp := t.metricStyleFingerprint(); fp != t.lastMetricStyleFingerprint {
		t.lastMetricStyleFingerprint = fp
		t.resetCachedTextSize()
	}
}

// appendFaceRunsForStyle appends the face runs derived from styleRuns' metric
// properties to runs and returns the extended slice, in styleRuns' byte
// offsets. Masked values append no face runs.
func (t *Text) appendFaceRunsForStyle(runs []textutil.FaceRun, styleRuns *textstyle.Runs, context *guigui.Context, forceBold bool) []textutil.FaceRun {
	if t.masking() {
		return runs
	}
	base := t.forceBoldedBaseStyle(forceBold)
	liga := t.ligaturesEnabled()
	for run := range styleRuns.All() {
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

// insertion returns the layout's insertion point: the face the text typed at
// a collapsed caret would use, at the caret. That is the style adopted from
// the byte before the caret with the insertion style laid over, so the caret
// and its line take the size of the text they are about to receive. The zero
// value means the layout has none, which is also the case while an IME
// composition is active, as the composition text carries the insertion style
// itself.
func (t *Text) insertion(context *guigui.Context, forceBold bool) textutil.Insertion {
	if !t.editable || t.masking() {
		return textutil.Insertion{}
	}
	if t.store.UncommittedTextLengthInBytes() > 0 {
		return textutil.Insertion{}
	}
	start, end := t.store.Selection()
	if start < 0 || start != end {
		return textutil.Insertion{}
	}
	adopted := t.styleAtCaret(start)
	if adopted.IsZero() && t.insertionStyle.IsZero() {
		return textutil.Insertion{}
	}
	base := t.forceBoldedBaseStyle(forceBold)
	style := base.Merge(adopted).Merge(t.insertionStyle)
	liga := t.ligaturesEnabled()
	attrs := t.baseStyle.faceAttributes(style, liga)
	// A face no larger than the base one leaves every line height as it is,
	// as the base face takes part in each of them.
	if attrs.Size <= t.baseStyle.faceAttributes(base, liga).Size {
		return textutil.Insertion{}
	}
	family, _ := style.Family()
	return textutil.Insertion{
		Face:         font.NewFace(context, family, attrs),
		IndexInBytes: start,
	}
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
	t.faceRunsBuf = t.appendFaceRunsForStyle(t.faceRunsBuf, t.ensureOverrideStyleRuns(), context, forceBold)
	committed = t.faceRunsBuf[mark.committed:]
	rendering = committed
	if showComposition && t.store.UncommittedTextLengthInBytes() > 0 {
		t.renderingFaceRunsBuf = t.appendFaceRunsForStyle(t.renderingFaceRunsBuf, t.renderingStyleRuns(), context, forceBold)
		rendering = t.renderingFaceRunsBuf[mark.rendering:]
	}
	return committed, rendering, mark
}

// renderingStyleRuns returns the ranged style overrides in rendering-text
// byte offsets: the committed overrides moved through the active
// composition's splice, with the insertion style applied over the composition
// so it renders as the style it will carry once committed. The returned runs
// are a buffer owned by t, valid until the next call; the caller must have an
// active composition.
func (t *Text) renderingStyleRuns() *textstyle.Runs {
	selStart, selEnd := t.store.Selection()
	if selStart > selEnd {
		selStart, selEnd = selEnd, selStart
	}
	comp := t.store.UncommittedText()
	renderingLen := t.store.TextLengthInBytes() - (selEnd - selStart) + len(comp)
	t.renderingStyleRunsBuf.CopyFrom(t.ensureOverrideStyleRuns())
	t.renderingStyleRunsBuf.Replace(selStart, selEnd, len(comp))
	if selStart == selEnd {
		t.adoptStyleAtLineHeadIfNeeded(&t.renderingStyleRunsBuf, selStart, len(comp), renderingLen)
	}
	if !t.insertionStyle.IsZero() {
		t.renderingStyleRunsBuf.ApplyStyle(selStart, selStart+len(comp), t.insertionStyle)
	}
	return &t.renderingStyleRunsBuf
}

// releaseFaceRuns truncates the face-run buffers back to their lengths at
// the matching [Text.acquireFaceRuns] call.
func (t *Text) releaseFaceRuns(mark faceRunsMark) {
	t.faceRunsBuf = slices.Delete(t.faceRunsBuf, mark.committed, len(t.faceRunsBuf))
	t.renderingFaceRunsBuf = slices.Delete(t.renderingFaceRunsBuf, mark.rendering, len(t.renderingFaceRunsBuf))
}
