// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textutil

import (
	"image"
	"iter"
	"math"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/guigui-gui/guigui/basicwidget/internal/font"
)

// A "logical line" is a hard-break-delimited slice of the source text: it
// contains at most one hard line break, and only at its very end. The empty
// string is also a valid logical line. Logical lines are layout-independent;
// the visual lines that result from rendering them depend on the width and
// the wrap mode. The functions below are the per-logical-line counterparts
// of the whole-document Measure / Position / Index helpers in textutil.go
// and let callers shape one logical line at a time without rescanning the
// entire document.

// visualLinesFromLogicalLine yields the visual lines that result from rendering
// one logical line at the given width. Positions in the yielded values are
// relative to the start of logicalLine, not to any global document offset.
//
// If wrapMode is [WrapModeNone], exactly one visual line is yielded: logicalLine
// itself (including any trailing hard break). For other wrap modes, if the
// line fits within width, exactly one visual line is yielded as well.
// Otherwise the line is wrapped at break opportunities determined by wrapMode.
//
// An empty logicalLine yields a single empty visual line. A logicalLine that
// contains a mid-line hard break violates the contract; the iterator stops
// at the first mandatory break it encounters.
func visualLinesFromLogicalLine(width int, logicalLine string, wrapMode WrapMode, advance func(str string, indexInBytes int) float64) iter.Seq[visualLine] {
	// Fast path: a single visual line. Avoids invoking the segmenter for
	// short content that fits, including the empty-line case.
	if wrapMode == WrapModeNone || width == math.MaxInt || advance(logicalLine, len(logicalLine)) <= float64(width) {
		return func(yield func(visualLine) bool) {
			yield(visualLine{pos: 0, str: logicalLine})
		}
	}

	// This per-logical-line path does not use the layout cache: it has only an
	// advance closure (the cache is keyed on a font.Face), and it serves the
	// content the cache rejects. The cache requires valid UTF-8, so operate on
	// the sanitized copy; the yielded slices index into it.
	sanitized := sanitizedForCache(logicalLine)
	return visualLinesFromStarts(sanitized, visualLineStarts(width, sanitized, wrapMode, &recomputingRangeAdvancer{line: sanitized, advance: advance}))
}

// visualLineStarts yields the start byte offset of each visual line that line
// wraps into at the given width, in order. line must be valid UTF-8. The first
// value is always 0, the number of values equals the visual-line count, and
// visual line i spans line[start_i:start_{i+1}] (the last spans to len(line)).
// This describes exactly the same wrapping as visualLinesFromLogicalLine. ra
// measures each candidate range (see [rangeAdvancer]).
func visualLineStarts(width int, line string, wrapMode WrapMode, ra rangeAdvancer) iter.Seq[int] {
	return func(yield func(int) bool) {
		// Fast path: a single visual line. Mirrors visualLinesFromLogicalLine.
		if wrapMode == WrapModeNone || width == math.MaxInt || ra.rangeAdvance(0, len(line)) <= float64(width) {
			yield(0)
			return
		}

		if !yield(0) {
			return
		}
		var vlStart, vlEnd int
		// emit consumes the next segment, identified only by its byte length, and
		// starts a new visual line whenever the accumulated content would overflow
		// width. Returns cont=false at a mandatory break (the contract allows at
		// most one, at the very end) or once the consumer stops. A mandatory break
		// never starts a fresh visual line: its start offset is already yielded.
		emit := func(segLenInBytes int, isMandatoryBreak bool) (cont bool) {
			if vlEnd-vlStart > 0 {
				candEnd := vlEnd + segLenInBytes
				innerEnd := candEnd - vlStart - tailingLineBreakLen(line[vlStart:candEnd])
				if ra.rangeAdvance(vlStart, innerEnd) > float64(width) {
					vlStart = vlEnd
					if !yield(vlStart) {
						return false
					}
				}
			}
			vlEnd += segLenInBytes
			return !isMandatoryBreak
		}

		// WrapModeNormal wraps at line-break opportunities, WrapModeAnywhere at
		// grapheme boundaries; both feed the same packing loop. A logical line
		// has at most a trailing hard break, so the mandatory-break flag is
		// taken from each segment's own trailing line break.
		boundaries := theSegmentCache.softLineBreakBoundaries
		if wrapMode != WrapModeNormal {
			boundaries = theSegmentCache.graphemeBoundaries
		}
		var segStart int
		for end := range boundaries(line) {
			segText := line[segStart:end]
			if !emit(end-segStart, tailingLineBreakLen(segText) > 0) {
				break
			}
			segStart = end
		}
	}
}

// visualLinesFromStarts yields the visual lines described by the start offsets in
// vlStarts (as produced by visualLineStarts) over line. line must be
// the same string the offsets were computed against. No shaping is performed.
func visualLinesFromStarts(line string, vlStarts iter.Seq[int]) iter.Seq[visualLine] {
	return func(yield func(visualLine) bool) {
		started := false
		var prev int
		for s := range vlStarts {
			if !started {
				prev = s
				started = true
				continue
			}
			if !yield(visualLine{pos: prev, str: line[prev:s]}) {
				return
			}
			prev = s
		}
		if started {
			yield(visualLine{pos: prev, str: line[prev:]})
		}
	}
}

// rangeAdvancer reports the wrap width of line[start:start+innerEnd]: its advance
// with the trailing hard break removed and, unless KeepTailingSpace, trailing
// spaces trimmed.
type rangeAdvancer interface {
	rangeAdvance(start, innerEnd int) float64
}

// recomputingRangeAdvancer reshapes line[start:start+innerEnd] on each call: the
// general path, used for tabs and content the layout cache rejects.
type recomputingRangeAdvancer struct {
	// line is the logical line the offsets index into.
	line string
	// advance is the package advance for a substring, with face and settings bound in.
	advance func(str string, indexInBytes int) float64
}

func (r *recomputingRangeAdvancer) rangeAdvance(start, innerEnd int) float64 {
	return r.advance(r.line[start:start+innerEnd], innerEnd)
}

// precomputedRangeAdvancer measures by subtracting two entries of a precomputed
// advance-up-to table, so each call is a trim plus a subtraction, not a reshape.
//
// It measures against the whole-line shaping, so a ligature or cursive join
// straddling a visual-line edge can shift the chosen break by about one glyph;
// the break is still a line-break opportunity.
type precomputedRangeAdvancer struct {
	// line is the logical line the offsets index into.
	line string
	// face shapes line when the advance-up-to table is first built.
	face text.Face
	// advanceUpTo is line's advance-up-to table, built lazily on first use; nil until then.
	advanceUpTo []float64
	// spaceAdvance is one space's advance, added for a dropped break when addBreakSpace is set.
	spaceAdvance float64
	// keepTailingSpace keeps trailing spaces in the measured width.
	keepTailingSpace bool
	// addBreakSpace adds spaceAdvance for a dropped trailing break.
	addBreakSpace bool
}

func (p *precomputedRangeAdvancer) rangeAdvance(start, innerEnd int) float64 {
	if p.advanceUpTo == nil {
		p.buildAdvanceUpTo()
	}
	e := start + innerEnd
	var hasBreak bool
	if !p.keepTailingSpace {
		for e > start {
			r, s := utf8.DecodeLastRuneInString(p.line[start:e])
			if s == 0 || !unicode.IsSpace(r) {
				break
			}
			e -= s
		}
	} else if l := tailingLineBreakLen(p.line[start:e]); l > 0 {
		e -= l
		hasBreak = true
	}
	w := p.advanceUpTo[e] - p.advanceUpTo[start]
	if hasBreak && p.addBreakSpace {
		w += p.spaceAdvance
	}
	return w
}

// Scratch for [precomputedRangeAdvancer.buildAdvanceUpTo], reused across calls
// (the UI is single-threaded; each table is consumed before the next build).
var (
	theLazyGlyphsBuffer  []text.LazyGlyph
	theAdvanceUpToBuffer []float64
)

// buildAdvanceUpTo fills p.advanceUpTo from one shaping of p.line: advanceUpTo[i]
// is the summed advance of every cluster ending at or before byte i. Summing
// advances is bidi-invariant, so advanceUpTo[b]-advanceUpTo[a] is the width of
// p.line[a:b]. The table aliases shared scratch, valid only until the next build.
func (p *precomputedRangeAdvancer) buildAdvanceUpTo() {
	theLazyGlyphsBuffer = text.AppendLazyGlyphs(theLazyGlyphsBuffer[:0], p.line, p.face, nil)
	n := len(p.line) + 1
	if cap(theAdvanceUpToBuffer) < n {
		theAdvanceUpToBuffer = make([]float64, n)
	}
	advanceUpTo := theAdvanceUpToBuffer[:n]
	for i := range advanceUpTo {
		advanceUpTo[i] = 0
	}
	for i := range theLazyGlyphsBuffer {
		g := &theLazyGlyphsBuffer[i]
		advanceUpTo[g.EndIndexInBytes] += g.AdvanceX
	}
	var run float64
	for i := range advanceUpTo {
		run += advanceUpTo[i]
		advanceUpTo[i] = run
	}
	p.advanceUpTo = advanceUpTo
}

// cachedVisualLineStarts returns the visual-line start offsets for line at the
// given layout parameters, memoized by content in [theLayoutCache]. The face's
// recipe fingerprints the entry; a face with no recipe (zero attributes)
// computes without caching. ok is false for non-UTF-8 lines (whose offsets would
// index into a sanitized copy, not line) so callers fall back to their shaping
// path; in that case vlStarts is nil.
func cachedVisualLineStarts(width int, line string, wrapMode WrapMode, face font.Face, tabWidth float64, keepTailingSpace bool) (vlStarts []int, ok bool) {
	k := layoutKey{
		text:             line,
		faceID:           face.ID(),
		width:            width,
		wrapMode:         wrapMode,
		tabWidth:         tabWidth,
		keepTailingSpace: keepTailingSpace,
	}
	if s, hit := theLayoutCache.get(k); hit {
		return s, true
	}
	if !utf8.ValidString(line) {
		return nil, false
	}
	// line is valid UTF-8, so the segmenter can run on it directly and the
	// offsets index into line without a sanitized copy.
	tf := face.TextFace()

	// Tabs need per-visual-line stops that a single line-wide advance-up-to table
	// can't express, so a line mixing tabs with a nonzero tab width keeps the
	// reshaping path; everything else uses the precomputed table.
	var ra rangeAdvancer
	if tabWidth == 0 || !strings.ContainsRune(line, '\t') {
		var spaceAdvance float64
		addBreakSpace := tabWidth != 0
		if addBreakSpace {
			spaceAdvance = text.Advance(" ", tf)
		}
		ra = &precomputedRangeAdvancer{
			line:             line,
			face:             tf,
			spaceAdvance:     spaceAdvance,
			keepTailingSpace: keepTailingSpace,
			addBreakSpace:    addBreakSpace,
		}
	} else {
		ra = &recomputingRangeAdvancer{
			line: line,
			advance: func(s string, indexInBytes int) float64 {
				return advance(s, indexInBytes, tf, tabWidth, keepTailingSpace)
			},
		}
	}

	vlStarts = slices.Collect(visualLineStarts(width, line, wrapMode, ra))
	theLayoutCache.put(k, vlStarts)
	return vlStarts, true
}

// MeasureLogicalLineHeight returns the rendered height of one logical line
// at the given width. This is the per-logical-line counterpart of
// [MeasureHeight] and is used by virtualized layout to size lines one at a
// time without scanning the whole document.
func MeasureLogicalLineHeight(width int, logicalLine string, wrapMode WrapMode, face font.Face, lineHeight float64, tabWidth float64, keepTailingSpace bool) float64 {
	return lineHeight * float64(VisualLineCountForLogicalLine(width, logicalLine, wrapMode, face, tabWidth, keepTailingSpace))
}

// VisualLineCountForLogicalLine returns the number of visual lines one
// logical line wraps into at the given width. With wrapMode set to
// [WrapModeNone] (or when the line fits) the result is always 1.
func VisualLineCountForLogicalLine(width int, logicalLine string, wrapMode WrapMode, face font.Face, tabWidth float64, keepTailingSpace bool) int {
	line := sanitizedForCache(logicalLine)
	ra := &recomputingRangeAdvancer{
		line: line,
		advance: func(s string, indexInBytes int) float64 {
			return advance(s, indexInBytes, face.TextFace(), tabWidth, keepTailingSpace)
		},
	}
	var count int
	for range visualLineStarts(width, line, wrapMode, ra) {
		count++
	}
	return count
}

// CachedVisualLineCount is [VisualLineCountForLogicalLine] backed by the
// content-keyed layout cache. For [WrapModeNone] (no packing) or a non-UTF-8
// line it falls back to the uncached count. Use this for per-tick height
// measurement of a logical line whose wrap layout the other cached paths (draw,
// caret, hit-test) also touch, so they share one cache entry.
func CachedVisualLineCount(width int, logicalLine string, wrapMode WrapMode, face font.Face, tabWidth float64, keepTailingSpace bool) int {
	if wrapMode != WrapModeNone {
		if vlStarts, ok := cachedVisualLineStarts(width, logicalLine, wrapMode, face, tabWidth, keepTailingSpace); ok {
			return len(vlStarts)
		}
	}
	return VisualLineCountForLogicalLine(width, logicalLine, wrapMode, face, tabWidth, keepTailingSpace)
}

// MeasureLogicalLine returns the rendered width and height of one logical
// line at the given width. Per-logical-line counterpart of [Measure].
func MeasureLogicalLine(width int, logicalLine string, wrapMode WrapMode, face font.Face, lineHeight float64, tabWidth float64, keepTailingSpace bool, ellipsisString string) (float64, float64) {
	var maxWidth, height float64
	for l := range visualLinesFromLogicalLine(width, logicalLine, wrapMode, func(s string, indexInBytes int) float64 {
		return advance(s, indexInBytes, face.TextFace(), tabWidth, keepTailingSpace)
	}) {
		vlStr := l.str
		if !keepTailingSpace {
			vlStr = trimTailingLineBreak(vlStr)
		}
		vlWidth := advance(vlStr, len(vlStr), face.TextFace(), tabWidth, keepTailingSpace)
		if ellipsisString != "" && vlWidth > float64(width) {
			vlStr = truncateWithEllipsis(vlStr, ellipsisString, float64(width), face.TextFace(), tabWidth)
			vlWidth = advance(vlStr, len(vlStr), face.TextFace(), tabWidth, false)
		}
		maxWidth = max(maxWidth, vlWidth)
		height += lineHeight
	}
	return maxWidth, height
}

// TextPositionFromIndexInLogicalLine returns the visual position(s) within one logical
// line corresponding to the given byte index inside that line. The Y values
// are relative to the top of the logical line (so the caller can offset them
// by the line's origin Y). Counterpart of [TextPositionFromIndex].
//
// index is a byte offset in [0, len(logicalLine)]. Out-of-range values yield
// (TextPosition{}, TextPosition{}, 0).
func TextPositionFromIndexInLogicalLine(width int, logicalLine string, index int, style *Style) (position0, position1 TextPosition, count int) {
	if index < 0 || index > len(logicalLine) {
		return TextPosition{}, TextPosition{}, 0
	}
	return textPositionFromIndexInVisualLines(width, visualLinesFromLogicalLine(width, logicalLine, style.WrapMode, func(s string, indexInBytes int) float64 {
		return advance(s, indexInBytes, style.Face.TextFace(), style.TabWidth, style.KeepTailingSpace)
	}), index, style)
}

// TextIndexFromPositionInLogicalLine returns the byte offset within one logical line
// closest to the given position. The position's Y is relative to the top of
// the logical line. Counterpart of [TextIndexFromPosition].
func TextIndexFromPositionInLogicalLine(width int, position image.Point, logicalLine string, style *Style) int {
	return textIndexFromPositionInVisualLines(width, position, visualLinesFromLogicalLine(width, logicalLine, style.WrapMode, func(s string, indexInBytes int) float64 {
		return advance(s, indexInBytes, style.Face.TextFace(), style.TabWidth, style.KeepTailingSpace)
	}), style)
}
