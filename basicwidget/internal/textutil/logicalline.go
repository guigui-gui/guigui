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
func visualLinesFromLogicalLine(width int, logicalLine string, wrapMode WrapMode, face font.Face, tabWidth float64, keepTailingSpace bool) iter.Seq[visualLine] {
	// Fast path: a single visual line. Avoids invoking the segmenter for
	// short content that fits, including the empty-line case.
	if wrapMode == WrapModeNone || width == math.MaxInt || advance(logicalLine, len(logicalLine), face.TextFace(), tabWidth, keepTailingSpace) <= float64(width) {
		return func(yield func(visualLine) bool) {
			yield(visualLine{pos: 0, str: logicalLine})
		}
	}

	// Wrapping needs valid UTF-8 for the segmenter, so operate on the sanitized
	// copy; the yielded slices index into it.
	sanitized := sanitizedForCache(logicalLine)
	ra := newRangeAdvancer(sanitized, face, tabWidth, keepTailingSpace)
	vlStarts := slices.Collect(visualLineStarts(width, sanitized, wrapMode, ra))
	return visualLinesFromStarts(sanitized, slices.Values(vlStarts))
}

// visualLineStarts yields the start byte offset of each visual line that line
// wraps into at the given width, in order. line must be valid UTF-8. The first
// value is always 0, the number of values equals the visual-line count, and
// visual line i spans line[start_i:start_{i+1}] (the last spans to len(line)).
// This describes exactly the same wrapping as visualLinesFromLogicalLine. ra
// measures each candidate range.
func visualLineStarts(width int, line string, wrapMode WrapMode, ra *rangeAdvancer) iter.Seq[int] {
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

// rangeAdvancer measures by subtracting entries of a precomputed
// advance-up-to array, so each call is a trim and one or a few subtractions, not
// a reshape. With a nonzero tab width it scans for tabs and snaps at each stop.
//
// It measures against the whole-line shaping, so a ligature or cursive join
// straddling a visual-line edge can shift the chosen break by about one glyph;
// the break is still a line-break opportunity.
type rangeAdvancer struct {
	// line is the logical line the offsets index into.
	line string
	// face shapes line when the advance-up-to array is first built.
	face text.Face
	// tabWidth is the tab stop width; 0 disables tab snapping and the break space.
	tabWidth float64
	// advanceUpTo is line's advance-up-to array, built lazily on first use; nil until then.
	advanceUpTo []float64
	// spaceAdvance is one space's advance, added for a dropped trailing break when tabWidth is nonzero.
	spaceAdvance float64
	// keepTailingSpace keeps trailing spaces in the measured width.
	keepTailingSpace bool
}

// newRangeAdvancer returns a rangeAdvancer for line whose advance-up-to array is
// built lazily on first measure (see [rangeAdvancer.buildAdvanceUpTo]). line must
// be valid UTF-8.
func newRangeAdvancer(line string, face font.Face, tabWidth float64, keepTailingSpace bool) *rangeAdvancer {
	tf := face.TextFace()
	var spaceAdvance float64
	if tabWidth != 0 {
		spaceAdvance = text.AdvanceAt(" ", 1, tf)
	}
	return &rangeAdvancer{
		line:             line,
		face:             tf,
		tabWidth:         tabWidth,
		spaceAdvance:     spaceAdvance,
		keepTailingSpace: keepTailingSpace,
	}
}

// rangeAdvance reports the wrap width of line[start:start+innerEnd]: its advance
// with the trailing hard break removed and, unless keepTailingSpace, trailing
// spaces trimmed.
func (ra *rangeAdvancer) rangeAdvance(start, innerEnd int) float64 {
	// A built advanceUpTo has length len(line)+1 ≥ 1; length 0 means it is not built
	// yet (a fresh advancer), so build it.
	if len(ra.advanceUpTo) == 0 {
		ra.buildAdvanceUpTo()
	}
	e := start + innerEnd
	var hasBreak bool
	if !ra.keepTailingSpace {
		for e > start {
			r, s := utf8.DecodeLastRuneInString(ra.line[start:e])
			if s == 0 || !unicode.IsSpace(r) {
				break
			}
			e -= s
		}
	} else if l := tailingLineBreakLen(ra.line[start:e]); l > 0 {
		e -= l
		hasBreak = true
	}
	var w float64
	if ra.tabWidth == 0 {
		w = ra.advanceUpTo[e] - ra.advanceUpTo[start]
	} else {
		// Tabs snap to the next stop measured from the visual-line left edge
		// (w = 0 at start), so accumulate each non-tab span from advanceUpTo and
		// snap at every tab; the tab's own glyph advance is skipped.
		pos := start
		for pos < e {
			i := strings.IndexByte(ra.line[pos:e], '\t')
			if i < 0 {
				w += ra.advanceUpTo[e] - ra.advanceUpTo[pos]
				break
			}
			tabIndexInBytes := pos + i
			w += ra.advanceUpTo[tabIndexInBytes] - ra.advanceUpTo[pos]
			w = nextIndentPosition(w, ra.tabWidth)
			pos = tabIndexInBytes + 1
		}
	}
	if hasBreak && ra.tabWidth != 0 {
		w += ra.spaceAdvance
	}
	return w
}

// theLazyGlyphsBuffer is scratch for shaping a line into glyphs, reused across
// builds and cleared after each (the UI is single-threaded). Clearing releases
// the per-glyph references the backing array would otherwise pin.
var theLazyGlyphsBuffer []text.LazyGlyph

// buildAdvanceUpTo builds ra.advanceUpTo from one shaping of ra.line. Serves the
// uncached callers; the cached layout path patches the last-measured line's own
// advanceUpTo in [layoutCache.relayout].
func (ra *rangeAdvancer) buildAdvanceUpTo() {
	ra.advanceUpTo = appendAdvanceUpTo(ra.advanceUpTo[:0], ra.line, ra.face)
}

// appendAdvanceUpTo appends line's advance-up-to array to dst and returns the
// extended slice: entry i of the appended run is the summed advance of every
// cluster ending at or before byte i, so run[b]-run[a] is the width of line[a:b].
// Summing advances is bidi-invariant. line must be valid UTF-8 and is shaped
// once. The appended run has length len(line)+1.
func appendAdvanceUpTo(dst []float64, line string, face text.Face) []float64 {
	theLazyGlyphsBuffer = text.AppendLazyGlyphs(theLazyGlyphsBuffer[:0], line, face, nil)
	defer func() {
		theLazyGlyphsBuffer = slices.Delete(theLazyGlyphsBuffer, 0, len(theLazyGlyphsBuffer))
	}()
	base := len(dst)
	dst = slices.Grow(dst, len(line)+1)[:base+len(line)+1]
	clear(dst[base:])
	for i := range theLazyGlyphsBuffer {
		g := &theLazyGlyphsBuffer[i]
		dst[base+g.EndIndexInBytes] += g.AdvanceX
	}
	var run float64
	for i := base; i < len(dst); i++ {
		run += dst[i]
		dst[i] = run
	}
	return dst
}

// cachedVisualLineStarts returns the visual-line start offsets for line at the
// given layout parameters, memoized by content in [theLayoutCache] and keyed by
// the resolved face's ID. ok is false for non-UTF-8 lines (whose offsets would
// index into a sanitized copy, not line) so callers fall back to their shaping
// path; in that case vlStarts is nil. face must be resolved (non-zero ID).
func cachedVisualLineStarts(width int, line string, wrapMode WrapMode, face font.Face, tabWidth float64, keepTailingSpace bool) (vlStarts []int, ok bool) {
	k := layoutKey{
		text: line,
		layoutStyleKey: layoutStyleKey{
			faceID:           face.ID(),
			width:            width,
			wrapMode:         wrapMode,
			tabWidth:         tabWidth,
			keepTailingSpace: keepTailingSpace,
		},
	}
	if k.faceID == 0 {
		panic("textutil: cachedVisualLineStarts requires a resolved face (face ID 0)")
	}
	if s, hit := theLayoutCache.get(k); hit {
		return s, true
	}
	if !utf8.ValidString(line) {
		return nil, false
	}
	vlStarts = theLayoutCache.relayout(k, face)
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
	ra := newRangeAdvancer(line, face, tabWidth, keepTailingSpace)
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
	for l := range visualLinesFromLogicalLine(width, logicalLine, wrapMode, face, tabWidth, keepTailingSpace) {
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
	return textPositionFromIndexInVisualLines(width, visualLinesFromLogicalLine(width, logicalLine, style.WrapMode, style.Face, style.TabWidth, style.KeepTailingSpace), index, style)
}

// TextIndexFromPositionInLogicalLine returns the byte offset within one logical line
// closest to the given position. The position's Y is relative to the top of
// the logical line. Counterpart of [TextIndexFromPosition].
func TextIndexFromPositionInLogicalLine(width int, position image.Point, logicalLine string, style *Style) int {
	return textIndexFromPositionInVisualLines(width, position, visualLinesFromLogicalLine(width, logicalLine, style.WrapMode, style.Face, style.TabWidth, style.KeepTailingSpace), style)
}
