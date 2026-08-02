// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textstyle

import (
	"image/color"
	"iter"
	"slices"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
)

// Run is a style override applied to a byte range.
type Run struct {
	// Start is the inclusive start of the range in bytes.
	Start int

	// End is the exclusive end of the range in bytes.
	End int

	// Style is the merged style overriding the range.
	Style Style
}

// Runs is a set of style overrides over byte ranges of a single text value.
// The zero value is an empty set ready for use.
//
// Runs holds byte offsets only; it does not know the text or its length.
// Callers may use ranges extending past the text's length.
type Runs struct {
	// runs is sorted by Start, with non-empty disjoint ranges, non-zero
	// styles with canonical Features and Variations, and no two adjacent
	// runs with equal styles.
	runs []Run

	// buf is scratch for rebuilding runs, cleared with a deferred
	// slices.Delete by each rebuild.
	buf []Run
}

// SetFamily overrides the font family in [start, end).
func (r *Runs) SetFamily(start, end int, family *font.Family) {
	r.apply(start, end, Style{family: opt(family)})
}

// UnsetFamily removes the font family override in [start, end).
func (r *Runs) UnsetFamily(start, end int) {
	r.unset(start, end, styleMask{family: true})
}

// SetItalic overrides the italic face selection in [start, end).
func (r *Runs) SetItalic(start, end int, italic bool) {
	r.apply(start, end, Style{italic: opt(italic)})
}

// UnsetItalic removes the italic override in [start, end).
func (r *Runs) UnsetItalic(start, end int) {
	r.unset(start, end, styleMask{italic: true})
}

// SetScale overrides the multiplier of the base font size in [start, end).
// scale must be positive.
func (r *Runs) SetScale(start, end int, scale float64) {
	r.apply(start, end, Style{scale: opt(scale)})
}

// UnsetScale removes the font size multiplier override in [start, end).
func (r *Runs) UnsetScale(start, end int) {
	r.unset(start, end, styleMask{scale: true})
}

// SetColor overrides the text color in [start, end).
func (r *Runs) SetColor(start, end int, clr color.Color) {
	r.apply(start, end, Style{color: opt(clr)})
}

// UnsetColor removes the text color override in [start, end).
func (r *Runs) UnsetColor(start, end int) {
	r.unset(start, end, styleMask{color: true})
}

// SetBackgroundColor overrides the background color in [start, end).
func (r *Runs) SetBackgroundColor(start, end int, clr color.Color) {
	r.apply(start, end, Style{backgroundColor: opt(clr)})
}

// UnsetBackgroundColor removes the background color override in [start, end).
func (r *Runs) UnsetBackgroundColor(start, end int) {
	r.unset(start, end, styleMask{backgroundColor: true})
}

// SetUnderline overrides whether an underline is drawn in [start, end).
func (r *Runs) SetUnderline(start, end int, underline bool) {
	r.apply(start, end, Style{underline: opt(underline)})
}

// UnsetUnderline removes the underline override in [start, end).
func (r *Runs) UnsetUnderline(start, end int) {
	r.unset(start, end, styleMask{underline: true})
}

// SetStrikethrough overrides whether a strikethrough is drawn in [start, end).
func (r *Runs) SetStrikethrough(start, end int, strikethrough bool) {
	r.apply(start, end, Style{strikethrough: opt(strikethrough)})
}

// UnsetStrikethrough removes the strikethrough override in [start, end).
func (r *Runs) UnsetStrikethrough(start, end int) {
	r.unset(start, end, styleMask{strikethrough: true})
}

// SetLang overrides the language used for rendering in [start, end).
func (r *Runs) SetLang(start, end int, lang language.Tag) {
	r.apply(start, end, Style{lang: opt(lang)})
}

// UnsetLang removes the language override in [start, end).
func (r *Runs) UnsetLang(start, end int) {
	r.unset(start, end, styleMask{lang: true})
}

// SetFeature overrides the OpenType feature tag in [start, end) with value.
func (r *Runs) SetFeature(start, end int, tag text.Tag, value uint32) {
	r.apply(start, end, Style{features: []font.Feature{{Tag: tag, Value: value}}})
}

// UnsetFeature removes the override of the OpenType feature tag in
// [start, end).
func (r *Runs) UnsetFeature(start, end int, tag text.Tag) {
	r.unset(start, end, styleMask{featureTags: []text.Tag{tag}})
}

// SetVariation overrides the OpenType variation axis tag in [start, end)
// with value.
func (r *Runs) SetVariation(start, end int, tag text.Tag, value float32) {
	r.apply(start, end, Style{variations: []font.Variation{{Tag: tag, Value: value}}})
}

// UnsetVariation removes the override of the OpenType variation axis tag in
// [start, end).
func (r *Runs) UnsetVariation(start, end int, tag text.Tag) {
	r.unset(start, end, styleMask{variationTags: []text.Tag{tag}})
}

// Reset removes all style overrides in [start, end).
func (r *Runs) Reset(start, end int) {
	start = max(start, 0)
	if start >= end {
		return
	}

	r.buf = r.buf[:0]
	defer func() {
		r.buf = slices.Delete(r.buf, 0, len(r.buf))
	}()
	for _, run := range r.runs {
		if run.End <= start || run.Start >= end {
			r.buf = appendRun(r.buf, run)
			continue
		}
		if run.Start < start {
			r.buf = appendRun(r.buf, Run{Start: run.Start, End: start, Style: run.Style})
		}
		if run.End > end {
			r.buf = appendRun(r.buf, Run{Start: end, End: run.End, Style: run.Style})
		}
	}
	r.setRuns()
}

// Replace adjusts the runs for a replacement of the byte range [start, end)
// with newLen bytes of text: styles keep covering the text around the
// replacement. Inserted text adopts the style right before the insertion
// point, extending the run it is inserted inside or at the end of;
// replacing text adopts the style of the first replaced byte.
func (r *Runs) Replace(start, end, newLen int) {
	start = max(start, 0)
	end = max(end, start)
	newLen = max(newLen, 0)
	if start == end && newLen == 0 {
		return
	}
	delta := newLen - (end - start)

	r.buf = r.buf[:0]
	defer func() {
		r.buf = slices.Delete(r.buf, 0, len(r.buf))
	}()
	for _, run := range r.runs {
		switch {
		case start == end && run.Start < start && run.End >= start:
			r.buf = appendRun(r.buf, Run{Start: run.Start, End: run.End + delta, Style: run.Style})
		case run.End <= start:
			r.buf = appendRun(r.buf, run)
		case run.Start >= end:
			r.buf = appendRun(r.buf, Run{Start: run.Start + delta, End: run.End + delta, Style: run.Style})
		default:
			if run.Start <= start {
				r.buf = appendRun(r.buf, Run{Start: run.Start, End: start + newLen, Style: run.Style})
			}
			if run.End > end {
				r.buf = appendRun(r.buf, Run{Start: end + delta, End: run.End + delta, Style: run.Style})
			}
		}
	}
	r.setRuns()
}

// Clear removes all style overrides.
func (r *Runs) Clear() {
	r.runs = slices.Delete(r.runs, 0, len(r.runs))
}

// StyleAt returns the style overriding the byte at index, or a zero Style if
// none.
func (r *Runs) StyleAt(index int) Style {
	i, ok := slices.BinarySearchFunc(r.runs, index, func(run Run, index int) int {
		switch {
		case run.End <= index:
			return -1
		case run.Start > index:
			return 1
		default:
			return 0
		}
	})
	if !ok {
		return Style{}
	}
	return r.runs[i].Style
}

// All returns an iterator over the runs in ascending range order. The
// yielded runs never overlap or nest each other: styles applied over
// intersecting ranges have been split and merged into disjoint runs.
func (r *Runs) All() iter.Seq[Run] {
	return slices.Values(r.runs)
}

// CopyFrom replaces the run list with a copy of src's.
func (r *Runs) CopyFrom(src *Runs) {
	r.runs = slices.Delete(r.runs, 0, len(r.runs))
	r.runs = append(r.runs, src.runs...)
}

// CopyRangeFrom replaces the run list with a copy of src's overrides in
// [start, end), rebased so that start maps to 0.
func (r *Runs) CopyRangeFrom(src *Runs, start, end int) {
	start = max(start, 0)

	r.buf = r.buf[:0]
	defer func() {
		r.buf = slices.Delete(r.buf, 0, len(r.buf))
	}()
	for _, run := range src.runs {
		if run.End <= start || run.Start >= end {
			continue
		}
		r.buf = appendRun(r.buf, Run{
			Start: max(run.Start, start) - start,
			End:   min(run.End, end) - start,
			Style: run.Style,
		})
	}
	r.setRuns()
}

// ApplyStyle applies style's set properties over [start, end) on top of the
// existing overrides.
func (r *Runs) ApplyStyle(start, end int, style Style) {
	r.apply(start, end, style)
}

// ApplyAt applies src's overrides on top of the existing overrides, with
// src's byte ranges shifted by offset. src must not be r.
func (r *Runs) ApplyAt(src *Runs, offset int) {
	for _, run := range src.runs {
		r.apply(run.Start+offset, run.End+offset, run.Style)
	}
}

// ReplaceRange replaces the overrides in [start, end) with src's overrides
// in [0, end-start), shifted so that 0 maps to start. src overrides outside
// [0, end-start) are ignored. src must not be r.
func (r *Runs) ReplaceRange(src *Runs, start, end int) {
	start = max(start, 0)
	if start >= end {
		return
	}
	r.Reset(start, end)
	for _, run := range src.runs {
		s := max(run.Start, 0)
		e := min(run.End, end-start)
		if e <= s {
			continue
		}
		r.apply(s+start, e+start, run.Style)
	}
}

// CopyMergedFrom replaces the run list with runs covering every byte of
// [0, end-start), each holding base merged with src's override of the
// corresponding byte of [start, end). Bytes whose merged style overrides
// nothing are left uncovered. An empty range clears the run list.
func (r *Runs) CopyMergedFrom(src *Runs, base Style, start, end int) {
	start = max(start, 0)

	r.buf = r.buf[:0]
	defer func() {
		r.buf = slices.Delete(r.buf, 0, len(r.buf))
	}()
	// pos is the start of the part of [start, end) not yet appended.
	pos := start
	for _, run := range src.runs {
		if run.End <= start {
			continue
		}
		if run.Start >= end {
			break
		}
		if run.Start > pos {
			r.buf = appendRun(r.buf, Run{Start: pos - start, End: run.Start - start, Style: base})
		}
		overlapStart := max(run.Start, start)
		overlapEnd := min(run.End, end)
		r.buf = appendRun(r.buf, Run{Start: overlapStart - start, End: overlapEnd - start, Style: base.Merge(run.Style)})
		pos = overlapEnd
	}
	if pos < end {
		r.buf = appendRun(r.buf, Run{Start: pos - start, End: end - start, Style: base})
	}
	r.setRuns()
}

// IsEmpty reports whether the run list holds no overrides.
func (r *Runs) IsEmpty() bool {
	return len(r.runs) == 0
}

// apply overrides [start, end) with style's set properties, on top of any
// styles applied earlier. A zero style is a no-op.
func (r *Runs) apply(start, end int, style Style) {
	start = max(start, 0)
	if start >= end {
		return
	}
	style = style.canonicalized()
	if style.IsZero() {
		return
	}

	r.buf = r.buf[:0]
	defer func() {
		r.buf = slices.Delete(r.buf, 0, len(r.buf))
	}()
	// pos is the start of the part of [start, end) not yet emitted.
	pos := start
	for _, run := range r.runs {
		if run.End <= start {
			r.buf = appendRun(r.buf, run)
			continue
		}
		if run.Start >= end {
			if pos < end {
				r.buf = appendRun(r.buf, Run{Start: pos, End: end, Style: style})
				pos = end
			}
			r.buf = appendRun(r.buf, run)
			continue
		}
		if run.Start < start {
			r.buf = appendRun(r.buf, Run{Start: run.Start, End: start, Style: run.Style})
		}
		overlapStart := max(run.Start, start)
		if pos < overlapStart {
			r.buf = appendRun(r.buf, Run{Start: pos, End: overlapStart, Style: style})
		}
		overlapEnd := min(run.End, end)
		r.buf = appendRun(r.buf, Run{Start: overlapStart, End: overlapEnd, Style: run.Style.Merge(style)})
		pos = overlapEnd
		if run.End > end {
			r.buf = appendRun(r.buf, Run{Start: end, End: run.End, Style: run.Style})
		}
	}
	if pos < end {
		r.buf = appendRun(r.buf, Run{Start: pos, End: end, Style: style})
	}
	r.setRuns()
}

// unset removes the style properties selected by mask in [start, end). A
// zero mask is a no-op.
func (r *Runs) unset(start, end int, mask styleMask) {
	start = max(start, 0)
	if start >= end {
		return
	}
	if mask.isZero() {
		return
	}

	r.buf = r.buf[:0]
	defer func() {
		r.buf = slices.Delete(r.buf, 0, len(r.buf))
	}()
	for _, run := range r.runs {
		if run.End <= start || run.Start >= end {
			r.buf = appendRun(r.buf, run)
			continue
		}
		if run.Start < start {
			r.buf = appendRun(r.buf, Run{Start: run.Start, End: start, Style: run.Style})
		}
		overlapStart := max(run.Start, start)
		overlapEnd := min(run.End, end)
		r.buf = appendRun(r.buf, Run{Start: overlapStart, End: overlapEnd, Style: run.Style.remove(mask)})
		if run.End > end {
			r.buf = appendRun(r.buf, Run{Start: end, End: run.End, Style: run.Style})
		}
	}
	r.setRuns()
}

// appendRun appends run to runs, dropping an empty or zero-styled run and
// extending the last run instead when run adjoins it with an equal style.
func appendRun(runs []Run, run Run) []Run {
	if run.Start >= run.End || run.Style.IsZero() {
		return runs
	}
	if n := len(runs); n > 0 && runs[n-1].End == run.Start && runs[n-1].Style.Equal(run.Style) {
		runs[n-1].End = run.End
		return runs
	}
	return append(runs, run)
}

// setRuns installs the runs built in r.buf as the current run list by
// copying them.
func (r *Runs) setRuns() {
	r.runs = slices.Delete(r.runs, 0, len(r.runs))
	r.runs = append(r.runs, r.buf...)
}
